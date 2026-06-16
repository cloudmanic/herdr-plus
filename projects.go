//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// pickerWorkspaceLabel is the title herdr-plus gives the ephemeral workspace it
// opens to host the projects browser.
const pickerWorkspaceLabel = "Herdr Plus"

// pickerTabLabel is the name of the tab inside that workspace where the browser
// runs.
const pickerTabLabel = "projects"

// launchProjects is the Projects action's entry point. herdr runs it server-side
// (from the plugin action / keybinding), so it has no terminal of its own; it
// opens a brand-new full-screen workspace to host the browser, renames its root
// tab to "projects", and starts the browser UI in that pane. It returns
// immediately so the pane the user triggered it from keeps its prompt.
func launchProjects() {
	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		errExit(err)
	}

	// Resolve our own absolute path so the new pane's shell can launch the browser
	// even when the binary is not on PATH.
	exe, err := os.Executable()
	if err != nil {
		errExit(err)
	}

	// Open the focused host workspace rooted at the home directory.
	ws, tab, pane, err := client.workspaceCreate(home, pickerWorkspaceLabel, true)
	if err != nil {
		errExit("could not create projects workspace:", err)
	}

	// Rename the root tab to "projects". Best effort: if it fails the tab simply
	// keeps its default numbered name.
	_ = client.tabRename(tab, pickerTabLabel)

	// Start the browser in the new pane, handing it the host workspace id so it
	// can tear the whole workspace down when the user picks a project or quits.
	// runCommand waits for the new shell's prompt and submits with a real Enter
	// key (not a trailing newline — see sendInput), so the UI starts reliably.
	launch := fmt.Sprintf("%s projects-ui %s", shellQuote(exe), ws)
	if err := client.runCommand(pane, launch); err != nil {
		errExit("failed to start the projects browser:", err)
	}
}

// runProjectsUICmd parses the internal `projects-ui` invocation. Its single
// positional argument is the id of the ephemeral host workspace to close on exit.
func runProjectsUICmd(args []string) {
	fs := flag.NewFlagSet("projects-ui", flag.ExitOnError)
	_ = fs.Parse(args)

	pickerWS := ""
	if rest := fs.Args(); len(rest) > 0 {
		pickerWS = rest[0]
	}

	runProjectsUI(pickerWS)
}

// runProjectsUI loads the projects, renders the full-screen browser, and acts on
// the result: opening the chosen project's workspace, or — on cancel — tearing
// down the ephemeral host workspace. pickerWS is the workspace this UI runs in,
// which is closed once we are done with it.
func runProjectsUI(pickerWS string) {
	projects, err := loadProjects()
	if err != nil {
		// Leave the pane open so the user can read the config error.
		errExit(err)
	}

	dir, _ := projectsConfigDir()

	// WithMouseCellMotion enables click/release/wheel events so a project can be
	// opened with the mouse. herdr forwards these to us once we ask for them;
	// until then it keeps the mouse for its own pane focus/selection.
	p := tea.NewProgram(newProjectsModel(projects, dir), tea.WithAltScreen(), tea.WithMouseCellMotion())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-plus:", err)
	}

	m, ok := result.(projectsModel)
	if !ok || m.chosen == nil {
		// Cancelled (or the program never produced a model) — remove the ephemeral
		// host workspace and return focus to where the user was.
		closeWorkspace(pickerWS)
		return
	}

	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}
	if err := openProject(client, *m.chosen, pickerWS); err != nil {
		// Leave the host workspace open so the error stays on screen.
		errExit("could not open project:", err)
	}
}

// openProject turns a project into a live herdr workspace: it creates a focused
// workspace rooted at the project's working directory, lays out its tabs and
// panes, runs each startup command, and finally closes the ephemeral host
// workspace the browser ran in.
func openProject(client *herdrClient, p Project, pickerWS string) error {
	dir := p.expandedWorkingDir()
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Errorf("working directory does not exist: %s", dir)
	}

	ws, rootTab, rootPane, err := client.workspaceCreate(dir, p.Name, true)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	// Lay the project's tabs into the new workspace.
	if err := layoutTabs(client, ws, rootTab, rootPane, p.Tabs); err != nil {
		return err
	}

	// Tear down the ephemeral host workspace. Focus is already on the new project
	// workspace, so this only removes the browser. This also closes the pane this
	// process is running in, so it is the last thing we do.
	if pickerWS != "" {
		_ = client.workspaceClose(pickerWS)
	}
	return nil
}

// layoutTabs lays an ordered list of tabs — each with its panes and optional
// startup commands — into an existing workspace whose root tab and root pane are
// rootTab and rootPane. tab[0] reuses the root tab (renamed) and root pane; each
// later tab is created without focus so the first stays in front while the rest
// spin up. Within a tab the first pane is the tab's root and each later pane is
// split off the previous one. Every startup command is run last, once all panes
// exist, paced to its freshly spawned shell.
func layoutTabs(client *herdrClient, ws, rootTab, rootPane string, tabs []ProjectTab) error {
	// pendingRun pairs a pane with the command it should run once all panes exist.
	type pendingRun struct {
		pane    string
		command string
	}
	var runs []pendingRun
	var err error

	for i, t := range tabs {
		tabRoot := rootPane
		if i == 0 {
			if err = client.tabRename(rootTab, t.Name); err != nil {
				return fmt.Errorf("rename root tab: %w", err)
			}
		} else {
			_, tabRoot, err = client.tabCreate(ws, t.Name, false)
			if err != nil {
				return fmt.Errorf("create tab %q: %w", t.Name, err)
			}
		}

		prev := tabRoot
		for j, pane := range t.effectivePanes() {
			paneID := tabRoot
			if j > 0 {
				paneID, err = client.paneSplit(prev, pane.Split, false)
				if err != nil {
					return fmt.Errorf("split pane %d in tab %q: %w", j+1, t.Name, err)
				}
			}
			if strings.TrimSpace(pane.Command) != "" {
				runs = append(runs, pendingRun{pane: paneID, command: pane.Command})
			}
			prev = paneID
		}
	}

	// Run each tab's startup command. runCommand paces itself to each freshly
	// spawned shell — waiting for its prompt, typing the command, then submitting
	// with a real Enter key — so the apps (claude, lazygit, …) actually start
	// instead of sitting unsubmitted at the prompt for the user to press Enter.
	for _, r := range runs {
		if err := client.runCommand(r.pane, r.command); err != nil {
			return fmt.Errorf("run command in pane %s: %w", r.pane, err)
		}
	}
	return nil
}

// closeWorkspace asks herdr to close a workspace. Failures are ignored: from a
// pane that is about to go away, there is nothing useful to do if the socket is
// unreachable.
func closeWorkspace(workspaceID string) {
	if workspaceID == "" {
		return
	}
	client, err := newHerdrClient()
	if err != nil {
		return
	}
	_ = client.workspaceClose(workspaceID)
}
