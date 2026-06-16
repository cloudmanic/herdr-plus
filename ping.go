//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"os"
)

// runPing is a tiny end-to-end smoke test of the plugin loop: it talks to herdr
// over the socket and prints what it sees about the focused pane. herdr captures
// stdout in the plugin log (`herdr plugin log list --plugin cloudmanic.herdr-plus`),
// so a successful ping proves the plugin is registered, the socket is reachable,
// and invocation works — before any real feature exists. It is the seed the
// Projects feature grows from in the next phase.
func runPing() {
	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}

	// When herdr runs us as a plugin action it sets HERDR_PANE_ID to the focused
	// pane; fall back to asking herdr for the focused pane if it is absent.
	paneID := os.Getenv("HERDR_PANE_ID")
	if paneID == "" {
		paneID, err = client.focusedPaneID()
		if err != nil {
			errExit("could not determine the focused pane:", err)
		}
	}

	pane, err := client.paneGet(paneID)
	if err != nil {
		errExit("pane.get:", err)
	}

	label := ""
	if ws, err := client.workspaceGet(pane.WorkspaceID); err == nil {
		label = ws.Label
	}

	fmt.Printf("herdr-plus ping ok — plugin %q\n", os.Getenv("HERDR_PLUGIN_ID"))
	fmt.Printf("  pane=%s tab=%s workspace=%s (%q)\n", pane.PaneID, pane.TabID, pane.WorkspaceID, label)
	fmt.Printf("  cwd=%s\n", firstNonEmpty(pane.ForegroundCwd, pane.Cwd))
}
