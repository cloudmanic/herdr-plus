//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"os/exec"
)

// launchQuickActions is the Quick Actions action's entry point. herdr runs it
// server-side (from the plugin action / keybinding), so it has no terminal of its
// own. It captures the focused pane's context (working directory, workspace) from
// the env herdr injects, then asks herdr to open the action picker as an overlay
// pane — passing the encoded context along so the chosen command runs in the
// directory you launched from, not the picker's. herdr creates the overlay and,
// when the picker exits, tears it down and restores your previous focus.
func launchQuickActions() {
	ctx := contextFromPluginEnv()
	enc, err := ctx.encode()
	if err != nil {
		errExit("could not encode run context:", err)
	}

	// HERDR_BIN_PATH points at the running herdr binary; it is the portable way to
	// call back into the CLI from a plugin command.
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}

	args := []string{
		"plugin", "pane", "open",
		"--plugin", "cloudmanic.herdr-plus",
		"--entrypoint", "quick-actions-picker",
		"--placement", "overlay",
		// Hand the launch context to the picker as a single shell-safe env var.
		"--env", "HERDR_PLUS_CTX=" + enc,
	}
	// IMPORTANT: do not add --cwd here. The manifest registers this pane with a
	// relative command (./bin/herdr-plus), which herdr resolves against the pane's
	// working directory — so the pane must run in the plugin's own install dir for
	// that path to resolve. Passing --cwd <launch dir> made herdr look for
	// ./bin/herdr-plus inside the launch directory and fail to spawn the picker.
	// The launch directory still reaches the picker — and the action it runs —
	// through HERDR_PLUS_CTX (ctx.WorkDir), which sets each command's cmd.Dir and
	// the per-repo action lookup. So --cwd was purely cosmetic; dropping it loses
	// nothing and matches how the projects pane (which never set it) already works.

	cmd := exec.Command(herdr, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errExit("could not open the quick-actions picker:", err)
	}
}
