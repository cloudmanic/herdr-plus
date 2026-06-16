//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"os"

	"github.com/cloudmanic/herdr-plus/internal/version"
)

// main is the plugin binary's entry point. herdr-plus is a herdr plugin: herdr
// registers it from herdr-plugin.toml and runs this binary with a subcommand per
// manifest entry point.
//
//   - "projects" / "quick-actions" are the actions herdr runs from a keybinding:
//     each asks herdr to open its UI as a plugin pane.
//   - "projects-ui" / "quick-actions-ui" are those UIs; herdr runs them inside the
//     pane it opens (the `picker` / `quick-actions-picker` entrypoints), so end
//     users never run them directly.
//   - "ping" is a smoke test that proves the plugin loop end to end.
//   - "on-worktree" is herdr's worktree event handler, run for both worktree.created
//     and worktree.opened (via the [[events]] entries in herdr-plugin.toml), not the
//     user.
//
// The bare binary has no launcher of its own, so it just prints usage.
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "projects":
			launchProjects()
			return
		case "projects-ui":
			runProjectsUI()
			return
		case "quick-actions":
			launchQuickActions()
			return
		case "quick-actions-ui":
			runQuickActionsUI()
			return
		case "ping":
			runPing()
			return
		case "on-worktree":
			runOnWorktreeEvent(os.Args[2:])
			return
		case "version", "--version", "-v", "-V":
			fmt.Println("herdr-plus", version.Version)
			return
		}
	}
	errExit("a herdr plugin; run its actions through herdr (e.g. `herdr plugin action invoke cloudmanic.herdr-plus.projects`) or `herdr-plus version`.")
}
