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
// manifest entry point. Today there is only "ping", a smoke test that proves the
// plugin loop end to end. Real features (starting with Projects) land in later
// phases; until then the bare binary just prints usage.
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "ping":
			runPing()
			return
		case "version", "--version", "-v", "-V":
			fmt.Println("herdr-plus", version.Version)
			return
		}
	}
	errExit("a herdr plugin; run its actions through herdr (e.g. `herdr plugin action invoke cloudmanic.herdr-plus.ping`) or `herdr-plus version`.")
}
