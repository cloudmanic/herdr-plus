//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"strings"
	"testing"
	"time"
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

	// Run a command in the new pane the exact way openProject does — through
	// runCommand — and confirm it actually executed by reading back its output.
	// This is the regression guard for the bug where the command was only typed at
	// the prompt (a trailing "\n" never submitted), so the app never started until
	// the user pressed Enter. runCommand owns the startup-race handling (wait for
	// prompt, type, wait for echo, real Enter), so the test needs no manual delays.
	const marker = "herdr_plus_run_marker_8842"
	if err := client.runCommand(pane2, "echo "+marker); err != nil {
		t.Fatalf("runCommand: %v", err)
	}

	// Give the shell a moment to run the command, then read the pane. The marker
	// must appear twice: once as the typed command line and once as echo's output.
	// A single occurrence means it was typed but never submitted — the bug.
	time.Sleep(750 * time.Millisecond)
	out, err := client.paneRead(pane2, "visible", 20)
	if err != nil {
		t.Fatalf("paneRead: %v", err)
	}
	if got := strings.Count(out, marker); got < 2 {
		t.Fatalf("command did not run: marker %q appeared %d time(s), want >= 2 (echoed command + its output). pane:\n%s", marker, got, out)
	}
}
