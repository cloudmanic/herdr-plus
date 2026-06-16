//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"strings"
	"testing"
)

// TestRunContextRoundTrip checks that a context survives encode/decode intact,
// since the launching action and the picker are separate processes that
// communicate through this serialized blob.
func TestRunContextRoundTrip(t *testing.T) {
	want := RunContext{
		Value:          "some value",
		WorkDir:        "/Users/spicer/Development/cloudmanic/herdr-plus",
		PaneId:         "p_172",
		TabId:          "w1:1",
		TabLabel:       "claude",
		WorkspaceId:    "w1",
		WorkspaceLabel: "herdr-plus",
		TerminalId:     "term_1",
		Agent:          "claude",
		AgentSessionId: "abc-123",
	}

	encoded, err := want.encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	got, err := decodeRunContext(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != want {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

// TestDecodeEmptyContext confirms an empty string decodes to a zero context
// rather than an error, so the picker can still run with no metadata.
func TestDecodeEmptyContext(t *testing.T) {
	got, err := decodeRunContext("")
	if err != nil {
		t.Fatalf("decode empty: %v", err)
	}
	if (got != RunContext{}) {
		t.Fatalf("decode empty = %+v, want zero context", got)
	}
}

// TestContextAliases verifies the friendly accessors map to the workspace
// fields.
func TestContextAliases(t *testing.T) {
	c := RunContext{WorkspaceLabel: "herdr-plus", WorkspaceId: "w1"}
	if c.SessionTitle() != "herdr-plus" {
		t.Fatalf("SessionTitle = %q", c.SessionTitle())
	}
	if c.SessionId() != "w1" {
		t.Fatalf("SessionId = %q", c.SessionId())
	}
}

// TestEnvPairs confirms the context is exported with the HERDR_PLUS_ prefix and
// includes the session-title alias.
func TestEnvPairs(t *testing.T) {
	c := RunContext{Value: "v", WorkDir: "/wd", WorkspaceLabel: "herdr-plus"}
	pairs := c.envPairs()

	want := map[string]string{
		"HERDR_PLUS_VALUE":         "v",
		"HERDR_PLUS_WORKDIR":       "/wd",
		"HERDR_PLUS_SESSION_TITLE": "herdr-plus",
	}
	for key, val := range want {
		found := false
		for _, p := range pairs {
			if p == key+"="+val {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("env pairs missing %s=%s in %v", key, val, pairs)
		}
	}
	for _, p := range pairs {
		if !strings.HasPrefix(p, "HERDR_PLUS_") {
			t.Fatalf("env pair %q is not HERDR_PLUS_ prefixed", p)
		}
	}
}

// TestContextFromPluginEnv confirms the run context is built from the
// HERDR_PLUGIN_CONTEXT_JSON herdr injects, mapping the focused pane's cwd to the
// working directory.
func TestContextFromPluginEnv(t *testing.T) {
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", `{"workspace_id":"w3","workspace_label":"herdr-plus","workspace_cwd":"/ws","tab_id":"w3:t1","tab_label":"shell","focused_pane_id":"w3:p2","focused_pane_cwd":"/Users/spicer/code","focused_pane_agent":"claude"}`)

	c := contextFromPluginEnv()
	if c.WorkDir != "/Users/spicer/code" {
		t.Fatalf("WorkDir = %q, want the focused pane cwd", c.WorkDir)
	}
	if c.WorkspaceId != "w3" || c.WorkspaceLabel != "herdr-plus" || c.PaneId != "w3:p2" || c.Agent != "claude" {
		t.Fatalf("context = %+v", c)
	}
}

// TestContextFromPluginEnvWorkspaceFallback confirms the workspace cwd is used
// when the focused pane has none.
func TestContextFromPluginEnvWorkspaceFallback(t *testing.T) {
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", `{"workspace_id":"w3","workspace_cwd":"/ws","focused_pane_id":"w3:p2"}`)
	if got := contextFromPluginEnv().WorkDir; got != "/ws" {
		t.Fatalf("WorkDir = %q, want the workspace cwd fallback", got)
	}
}
