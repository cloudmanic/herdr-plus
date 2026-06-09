//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"testing"
)

// TestLiveSocketWorkspaceLifecycle exercises the real workspace/tab socket
// methods (the exact calls openProject makes) against a running herdr instance.
// It is skipped unless HERDR_PLUS_LIVE=1 and a socket is present, so ordinary
// `go test` and CI never touch herdr. Everything is created in the background
// (focus=false) and torn down at the end, so it never disturbs the user's view.
func TestLiveSocketWorkspaceLifecycle(t *testing.T) {
	if os.Getenv("HERDR_PLUS_LIVE") != "1" || os.Getenv("HERDR_SOCKET_PATH") == "" {
		t.Skip("live herdr socket test; set HERDR_PLUS_LIVE=1 inside herdr to run")
	}

	client, err := newHerdrClient()
	if err != nil {
		t.Fatalf("newHerdrClient: %v", err)
	}

	home, _ := os.UserHomeDir()

	ws, rootTab, rootPane, err := client.workspaceCreate(home, "herdr-plus-verify", false)
	if err != nil {
		t.Fatalf("workspaceCreate: %v", err)
	}
	// Always clean up, even if a later step fails.
	defer func() {
		if err := client.workspaceClose(ws); err != nil {
			t.Errorf("workspaceClose: %v", err)
		}
	}()

	if ws == "" || rootTab == "" || rootPane == "" {
		t.Fatalf("workspaceCreate returned empty ids: ws=%q tab=%q pane=%q", ws, rootTab, rootPane)
	}

	if err := client.tabRename(rootTab, "first"); err != nil {
		t.Fatalf("tabRename: %v", err)
	}

	_, pane2, err := client.tabCreate(ws, "second", false)
	if err != nil {
		t.Fatalf("tabCreate: %v", err)
	}
	if pane2 == "" {
		t.Fatal("tabCreate returned empty pane id")
	}

	// Run a harmless command in the new pane the same way openProject does.
	if err := client.sendInput(pane2, "true\n"); err != nil {
		t.Fatalf("sendInput: %v", err)
	}
}
