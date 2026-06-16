//
// Date: 2026-06-09
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

// controlWorkspaceLabel is the title herdr-plus gives the workspace it opens for
// control mode. It is the home base from which you drive herdr.
const controlWorkspaceLabel = "Herdr Plus"

// controlProjectsTabLabel is the name of the tab inside the control workspace
// where the projects browser runs.
const controlProjectsTabLabel = "projects"

// launchControl is control mode's launcher. Unlike quick-actions (which splits
// the current pane), it opens a brand-new, full-screen workspace titled
// "Herdr Plus", renames its root tab to "projects", and starts the control UI
// inside that pane. It returns immediately so the pane the user pressed the
// keybinding from keeps its prompt.
func launchControl(client *herdrClient) {
	home, err := os.UserHomeDir()
	if err != nil {
		errExit(err)
	}

	// Resolve our own absolute path so the new pane's shell can launch the
	// control UI even when the binary is not on PATH.
	exe, err := os.Executable()
	if err != nil {
		errExit(err)
	}

	// Open the focused control workspace rooted at the home directory.
	ws, tab, pane, err := client.workspaceCreate(home, controlWorkspaceLabel, true)
	if err != nil {
		errExit("could not create control workspace:", err)
	}

	// Rename the root tab to "projects". Best effort: if it fails the tab simply
	// keeps its default numbered name.
	_ = client.tabRename(tab, controlProjectsTabLabel)

	// Start the control UI in the new pane, handing it the control workspace id so
	// it can tear the whole workspace down when the user picks a project or quits.
	// runCommand waits for the new shell's prompt and submits with a real Enter
	// key (not a trailing newline — see sendInput), so the UI starts reliably.
	launch := fmt.Sprintf("%s control %s", shellQuote(exe), ws)
	if err := client.runCommand(pane, launch); err != nil {
		errExit("failed to start control UI:", err)
	}
}

// runControlCmd parses the internal `control` invocation. Its single positional
// argument is the id of the control workspace to close on exit.
func runControlCmd(args []string) {
	fs := flag.NewFlagSet("control", flag.ExitOnError)
	_ = fs.Parse(args)

	controlWS := ""
	if rest := fs.Args(); len(rest) > 0 {
		controlWS = rest[0]
	}

	runControl(controlWS)
}

// runControl loads the projects, renders the full-screen browser, and acts on
// the result: opening the chosen project's workspace, or — on cancel — tearing
// down the ephemeral control workspace. controlWS is the workspace this UI runs
// in, which is closed once we are done with it.
func runControl(controlWS string) {
	projects, err := loadProjects()
	if err != nil {
		// Leave the pane open so the user can read the config error.
		errExit(err)
	}

	dir, _ := projectsConfigDir()

	// WithMouseCellMotion enables click/release/wheel events so a project can be
	// opened with the mouse. herdr forwards these to us once we ask for them;
	// until then it keeps the mouse for its own pane focus/selection.
	p := tea.NewProgram(newControlModel(projects, dir), tea.WithAltScreen(), tea.WithMouseCellMotion())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-plus:", err)
	}

	m, ok := result.(controlModel)
	if !ok || m.chosen == nil {
		// Cancelled (or the program never produced a model) — remove the
		// ephemeral control workspace and return focus to where the user was.
		closeControlWorkspace(controlWS)
		return
	}

	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}
	if err := openProject(client, *m.chosen, controlWS); err != nil {
		// Leave the control workspace open so the error stays on screen.
		errExit("could not open project:", err)
	}
}

// openProject turns a project into a live herdr workspace: it creates a focused
// workspace rooted at the project's working directory, lays out one tab per
// entry (the first reusing the workspace's root tab, the rest created behind
// it), runs each tab's startup command, and finally closes the control
// workspace we were launched from.
func openProject(client *herdrClient, p Project, controlWS string) error {
	dir := p.expandedWorkingDir()
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Errorf("working directory does not exist: %s", dir)
	}

	ws, rootTab, rootPane, err := client.workspaceCreate(dir, p.Name, true)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	// Lay the project's tabs into the freshly created workspace.
	if err := layoutTabs(client, ws, rootTab, rootPane, p.Tabs); err != nil {
		return err
	}

	// Tear down the ephemeral control workspace. Focus is already on the new
	// project workspace, so this only removes the launcher. This also closes the
	// pane this process is running in, so it is the last thing we do.
	if controlWS != "" {
		_ = client.workspaceClose(controlWS)
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
//
// It is shared by control mode — which creates the workspace first, then fills
// it — and the worktree.created handler, where herdr has already made the
// worktree's workspace and we only need to populate it.
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

// closeControlWorkspace asks herdr to close the control workspace. Failures are
// ignored: from a pane that is about to go away, there is nothing useful to do
// if the socket is unreachable.
func closeControlWorkspace(workspaceID string) {
	if workspaceID == "" {
		return
	}
	client, err := newHerdrClient()
	if err != nil {
		return
	}
	_ = client.workspaceClose(workspaceID)
}
