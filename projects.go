//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// launchProjects is the Projects action's entry point. herdr runs it server-side
// (from the plugin action / keybinding), so it has no terminal of its own. It
// asks herdr to open the projects browser as a pane (the `picker` entrypoint in
// herdr-plugin.toml) — zoomed by default, or the placement set by
// [projects].placement in config.toml. herdr creates and tears down that pane for
// us, so — unlike the old design — there is no throwaway workspace to manage.
func launchProjects() {
	cfg, err := loadPluginConfig()
	if err != nil {
		errExit(err)
	}
	placement := resolvePlacement(cfg.Projects.Placement, "zoomed")

	// HERDR_BIN_PATH points at the running herdr binary; it is the portable way to
	// call back into the CLI from a plugin command.
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}

	cmd := exec.Command(herdr, "plugin", "pane", "open",
		"--plugin", pluginID,
		"--entrypoint", paneEntrypoint("picker"),
		"--placement", placement,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errExit("could not open the projects browser:", err)
	}
}

// runProjectsUI renders the full-screen projects browser. It runs inside the
// zoomed pane herdr opens for the `picker` entrypoint (which has a real
// terminal), loads the projects, and — when one is chosen — spins up its
// workspace. On cancel it simply exits and herdr tears the pane down.
func runProjectsUI() {
	projects, err := loadProjects()
	if err != nil {
		// Leave the pane open so the user can read the config error.
		errExit(err)
	}

	cfg, err := loadPluginConfig()
	if err != nil {
		errExit(err)
	}

	dir, _ := projectsConfigDir()

	// WithMouseCellMotion enables click/release/wheel events so a project can be
	// opened with the mouse. herdr forwards these to us once we ask for them;
	// until then it keeps the mouse for its own pane focus/selection.
	p := tea.NewProgram(newProjectsModel(projects, dir, cfg.Worktree.BranchPrefix), tea.WithAltScreen(), tea.WithMouseCellMotion())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-plus:", err)
	}

	m, ok := result.(projectsModel)
	if !ok || m.chosen == nil {
		// Cancelled (or the program never produced a model) — nothing to do; herdr
		// closes the pane when this process exits.
		return
	}

	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}
	if m.worktree {
		if err := openProjectAsWorktree(client, *m.chosen, m.branch); err != nil {
			errExit("could not open project as worktree:", err)
		}
		return
	}
	if err := openProject(client, *m.chosen); err != nil {
		errExit("could not open project:", err)
	}
}

// openProject turns a project into a live herdr workspace: it creates a focused
// workspace rooted at the project's working directory and lays out its tabs and
// panes, running each startup command. Creating the focused workspace switches
// the user to it; the picker pane this was launched from is then torn down by
// herdr when runProjectsUI exits.
func openProject(client *herdrClient, p Project) error {
	dir, err := p.expandedWorkingDir()
	if err != nil {
		return err
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Errorf("working directory does not exist: %s", dir)
	}

	ws, rootTab, rootPane, err := client.workspaceCreate(dir, p.Name, true)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	// Lay the project's tabs into the new workspace.
	return layoutTabs(client, ws, rootTab, rootPane, p.Tabs)
}

// openProjectAsWorktree creates a git worktree from the project's working
// directory instead of opening the project as a normal workspace. It resolves and
// validates the directory, confirms it is inside a git work tree (worktrees
// require one — a clearer error than herdr's if not), and asks herdr to create the
// worktree, focused. The new workspace's tabs are not laid out here: herdr's
// worktree.created event drives the worktree auto-layout, so a matching file in
// worktrees/ fills the workspace. Without one the worktree opens bare — the
// project's own [[tabs]] are not used for this path.
func openProjectAsWorktree(client *herdrClient, p Project, branch string) error {
	// filepath.Abs already returns an absolute path (or an error), so no separate
	// IsAbs check is needed after it.
	expanded, err := p.expandedWorkingDir()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	dir, err := filepath.Abs(expanded)
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Errorf("working directory does not exist: %s", dir)
	}
	if !isInsideGitWorkTree(dir) {
		return fmt.Errorf("not a git repository: %s (opening as a worktree requires one)", dir)
	}
	return client.worktreeCreate(dir, branch, true)
}

// isInsideGitWorkTree reports whether dir is inside a git work tree, so
// openProjectAsWorktree can give a clear error before asking herdr to create a
// worktree there. It shells out to git: when git runs and reports the directory is
// not a work tree it returns false, but when git cannot be run at all (not
// installed / not on PATH) it returns true, deferring the real check to herdr
// rather than blocking on git's absence.
func isInsideGitWorkTree(dir string) bool {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false // git ran and said this is not a work tree
		}
		return true // git could not run — let herdr make the call
	}
	return strings.TrimSpace(string(out)) == "true"
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
			if lbl := strings.TrimSpace(pane.Label); lbl != "" {
				if err := client.paneRename(paneID, lbl); err != nil {
					// Labeling is cosmetic — warn but keep building the workspace.
					fmt.Fprintf(os.Stderr, "herdr-plus: failed to label pane %d in tab %q: %v\n", j+1, t.Name, err)
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
