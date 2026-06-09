//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"os"
	"time"
)

// main dispatches between the two modes of the binary. With no arguments it
// runs the launcher, which opens a bottom split and starts the picker inside
// it. With "picker <pane-id>" it renders the fuzzy-finder TUI and, on exit,
// closes the given pane.
func main() {
	if len(os.Args) > 1 && os.Args[1] == "picker" {
		runPicker(os.Args[2:])
		return
	}
	runLauncher()
}

// runLauncher splits the current herdr pane downward, focuses the new pane, and
// launches this same binary in picker mode inside it. It passes the new pane's
// id to the picker so the picker can close exactly the right pane on exit. The
// launcher then returns immediately so the original shell prompt comes back.
func runLauncher() {
	client, err := newHerdrClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-quick-actions:", err)
		os.Exit(1)
	}

	// HERDR_PANE_ID identifies the pane we are launched from; it is the pane we
	// split to create the picker beneath it.
	paneID := os.Getenv("HERDR_PANE_ID")
	if paneID == "" {
		fmt.Fprintln(os.Stderr, "herdr-quick-actions: HERDR_PANE_ID is not set; are you running inside herdr?")
		os.Exit(1)
	}

	// Resolve our own absolute path so the new pane's shell can launch the
	// picker even when the binary is not on PATH.
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-quick-actions:", err)
		os.Exit(1)
	}

	// Create the picker pane beneath the current one and focus it so keystrokes
	// flow to the picker.
	newPane, err := client.splitDown(paneID, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, "herdr-quick-actions: split failed:", err)
		os.Exit(1)
	}

	// Give the freshly spawned shell a brief moment to initialize before we
	// send the command that starts the picker.
	time.Sleep(250 * time.Millisecond)

	// Start the picker in the new pane, handing it the new pane's id so it can
	// close itself when done.
	if err := client.sendInput(newPane, fmt.Sprintf("%s picker %s\n", exe, newPane)); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-quick-actions: failed to start picker:", err)
		os.Exit(1)
	}
}
