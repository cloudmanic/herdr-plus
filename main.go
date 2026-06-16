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
//   - "projects" is the Projects action herdr runs from a keybinding: it asks
//     herdr to open the browser as a zoomed plugin pane.
//   - "projects-ui" is the browser itself; herdr runs it inside that zoomed pane
//     (the `picker` entrypoint), so end users never run it directly.
//   - "ping" is a smoke test that proves the plugin loop end to end.
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
		case "ping":
			runPing()
			return
		case "version", "--version", "-v", "-V":
			fmt.Println("herdr-plus", version.Version)
			return
		}
	}
	errExit("a herdr plugin; run its actions through herdr (e.g. `herdr plugin action invoke cloudmanic.herdr-plus.projects`) or `herdr-plus version`.")
}
