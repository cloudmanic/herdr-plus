//
// Date: 2026-08-05
// Author: Stephane Adandedjan (Stephane.Adandedjan@goline.ch)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// plainWorkspaceEventJSON is a HERDR_PLUGIN_EVENT_JSON payload for the
// workspace.created herdr fires when the sidebar's New button makes an empty
// workspace: focused, one tab, one pane, and a null worktree. Its shape mirrors
// the captured worktree.created payload in worktree_test.go, which is what keeps
// parseNewWorkspaceEvent honest against herdr's wire format.
const plainWorkspaceEventJSON = `{"event":"workspace_created","data":{"type":"workspace_created","workspace":{"workspace_id":"w7","number":3,"label":"src","focused":true,"pane_count":1,"tab_count":1,"active_tab_id":"w7:t1","agent_status":"unknown","worktree":null}}}`

// worktreeWorkspaceEventJSON is the same event for a workspace herdr created for
// a git worktree — the case the worktree layout handler owns.
const worktreeWorkspaceEventJSON = `{"event":"workspace_created","data":{"type":"workspace_created","workspace":{"workspace_id":"w8","number":4,"label":"wt-probe-repo","focused":true,"pane_count":1,"tab_count":1,"active_tab_id":"w8:t1","agent_status":"unknown","worktree":{"repo_key":"/tmp/wt-probe-repo/.git","repo_name":"wt-probe-repo","repo_root":"/tmp/wt-probe-repo","checkout_path":"/tmp/wt-probe-repo-wt","is_linked_worktree":true}}}}`

// TestParseNewWorkspaceEvent parses the real payload and confirms every field the
// intercept decision relies on is extracted, including the null worktree that
// distinguishes a plain workspace from a worktree one.
func TestParseNewWorkspaceEvent(t *testing.T) {
	ev, err := parseNewWorkspaceEvent(plainWorkspaceEventJSON, mapEnv(nil))
	if err != nil {
		t.Fatalf("parseNewWorkspaceEvent: %v", err)
	}

	if ev.WorkspaceID != "w7" {
		t.Errorf("WorkspaceID = %q, want %q", ev.WorkspaceID, "w7")
	}
	if ev.ActiveTabID != "w7:t1" {
		t.Errorf("ActiveTabID = %q, want %q", ev.ActiveTabID, "w7:t1")
	}
	if ev.Label != "src" {
		t.Errorf("Label = %q, want %q", ev.Label, "src")
	}
	if !ev.Focused {
		t.Error("Focused = false, want true")
	}
	if ev.PaneCount != 1 || ev.TabCount != 1 {
		t.Errorf("TabCount/PaneCount = %d/%d, want 1/1", ev.TabCount, ev.PaneCount)
	}
	if ev.IsWorktree {
		t.Error("IsWorktree = true, want false for a null worktree")
	}
}

// TestParseNewWorkspaceEventWorktree confirms a populated worktree object is
// recognized, since that alone sends the handler back to sleep.
func TestParseNewWorkspaceEventWorktree(t *testing.T) {
	ev, err := parseNewWorkspaceEvent(worktreeWorkspaceEventJSON, mapEnv(nil))
	if err != nil {
		t.Fatalf("parseNewWorkspaceEvent: %v", err)
	}
	if !ev.IsWorktree {
		t.Error("IsWorktree = false, want true")
	}
}

// TestParseNewWorkspaceEventIDFallback confirms the workspace id falls back to
// HERDR_WORKSPACE_ID only when the payload has none — the payload names the
// workspace the event is about, the environment merely describes the caller.
func TestParseNewWorkspaceEventIDFallback(t *testing.T) {
	env := mapEnv(map[string]string{"HERDR_WORKSPACE_ID": "w1"})

	ev, err := parseNewWorkspaceEvent(plainWorkspaceEventJSON, env)
	if err != nil {
		t.Fatalf("parseNewWorkspaceEvent: %v", err)
	}
	if ev.WorkspaceID != "w7" {
		t.Errorf("WorkspaceID = %q, want the payload's %q", ev.WorkspaceID, "w7")
	}

	ev, err = parseNewWorkspaceEvent("", env)
	if err != nil {
		t.Fatalf("parseNewWorkspaceEvent (empty payload): %v", err)
	}
	if ev.WorkspaceID != "w1" {
		t.Errorf("WorkspaceID = %q, want the env fallback %q", ev.WorkspaceID, "w1")
	}
}

// TestParseNewWorkspaceEventInvalidJSON confirms malformed event JSON is an error
// rather than a zero-value event that could be acted on.
func TestParseNewWorkspaceEventInvalidJSON(t *testing.T) {
	if _, err := parseNewWorkspaceEvent("{not json", mapEnv(nil)); err == nil {
		t.Fatal("parseNewWorkspaceEvent accepted malformed JSON, want an error")
	}
}

// TestDecideNewWorkspaceIntercept walks the whole policy: the one case that
// intercepts, and every guard that must not. A wrong "yes" here hijacks a
// workspace the user meant to keep, so each guard gets its own case.
func TestDecideNewWorkspaceIntercept(t *testing.T) {
	// empty is the workspace herdr's New button produces: focused, blank, plain.
	empty := newWorkspaceEvent{WorkspaceID: "w7", Label: "src", Focused: true, PaneCount: 1, TabCount: 1}
	names := []string{"Options Cafe", "Harbor"}

	cases := []struct {
		name       string
		ev         newWorkspaceEvent
		mode       string
		names      []string
		suppressed bool
		want       bool
	}{
		{name: "empty focused workspace", ev: empty, mode: newWorkspaceModePicker, names: names, want: true},
		{name: "mode off", ev: empty, mode: newWorkspaceModeOff, names: names},
		{name: "no workspace id", ev: newWorkspaceEvent{Focused: true, PaneCount: 1, TabCount: 1}, mode: newWorkspaceModePicker, names: names},
		{
			name:  "worktree workspace",
			ev:    newWorkspaceEvent{WorkspaceID: "w8", Focused: true, PaneCount: 1, TabCount: 1, IsWorktree: true},
			mode:  newWorkspaceModePicker,
			names: names,
		},
		{
			name:  "background workspace",
			ev:    newWorkspaceEvent{WorkspaceID: "w9", PaneCount: 1, TabCount: 1},
			mode:  newWorkspaceModePicker,
			names: names,
		},
		{
			name:  "workspace already filled",
			ev:    newWorkspaceEvent{WorkspaceID: "w9", Focused: true, PaneCount: 3, TabCount: 2},
			mode:  newWorkspaceModePicker,
			names: names,
		},
		{name: "suppressed", ev: empty, mode: newWorkspaceModePicker, names: names, suppressed: true},
		{name: "no projects", ev: empty, mode: newWorkspaceModePicker, names: nil},
		{
			name:  "label matches a project",
			ev:    newWorkspaceEvent{WorkspaceID: "w7", Label: "options cafe", Focused: true, PaneCount: 1, TabCount: 1},
			mode:  newWorkspaceModePicker,
			names: names,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, reason := decideNewWorkspaceIntercept(c.ev, c.mode, c.names, c.suppressed)
			if got != c.want {
				t.Fatalf("decideNewWorkspaceIntercept = %v (%s), want %v", got, reason, c.want)
			}
			if !got && reason == "" {
				t.Error("declined without a reason; the plugin log would say nothing useful")
			}
		})
	}
}

// TestNewWorkspaceSuppressionConsumedOnce confirms one marker suppresses exactly
// one event: the first read reports it and removes it, the next sees nothing.
func TestNewWorkspaceSuppressionConsumedOnce(t *testing.T) {
	dir := t.TempDir()

	if err := writeNewWorkspaceSuppression(dir); err != nil {
		t.Fatalf("writeNewWorkspaceSuppression: %v", err)
	}
	if !consumeNewWorkspaceSuppression(dir, time.Now(), newWorkspaceInterceptTTL) {
		t.Fatal("a freshly written marker did not suppress")
	}
	if consumeNewWorkspaceSuppression(dir, time.Now(), newWorkspaceInterceptTTL) {
		t.Fatal("the marker suppressed twice; it must be consumed on read")
	}
}

// TestNewWorkspaceSuppressionExpires confirms a marker older than the TTL neither
// suppresses nor lingers — a create whose event never arrived must not swallow the
// next New the user presses.
func TestNewWorkspaceSuppressionExpires(t *testing.T) {
	dir := t.TempDir()

	if err := writeNewWorkspaceSuppression(dir); err != nil {
		t.Fatalf("writeNewWorkspaceSuppression: %v", err)
	}
	path := filepath.Join(dir, newWorkspaceInterceptMarker)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("os.Chtimes: %v", err)
	}

	if consumeNewWorkspaceSuppression(dir, time.Now(), newWorkspaceInterceptTTL) {
		t.Fatal("a stale marker suppressed")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a stale marker was left behind")
	}
}

// TestNewWorkspaceSuppressionMissing confirms the common case — no marker at all —
// does not suppress.
func TestNewWorkspaceSuppressionMissing(t *testing.T) {
	if consumeNewWorkspaceSuppression(t.TempDir(), time.Now(), newWorkspaceInterceptTTL) {
		t.Fatal("an absent marker suppressed")
	}
}

// TestNewWorkspaceMode covers the config surface: unset means off (the feature is
// opt-in), the two documented values pass through case-insensitively, and a typo
// is an error instead of a silent "off".
func TestNewWorkspaceMode(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: newWorkspaceModeOff},
		{in: "off", want: newWorkspaceModeOff},
		{in: "picker", want: newWorkspaceModePicker},
		{in: "  PICKER  ", want: newWorkspaceModePicker},
		{in: "pickr", want: newWorkspaceModeOff, wantErr: true},
	}

	for _, c := range cases {
		var cfg PluginConfig
		cfg.NewWorkspace.Mode = c.in

		got, err := cfg.newWorkspaceMode()
		if c.wantErr && err == nil {
			t.Errorf("newWorkspaceMode(%q) = %q, want an error", c.in, got)
			continue
		}
		if !c.wantErr && err != nil {
			t.Errorf("newWorkspaceMode(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("newWorkspaceMode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestPickTargetPane covers the anchor a zoomed picker pane is opened against.
// herdr rejects a zoomed plugin pane that names no target pane, so getting this
// wrong means the picker never appears — the pane list always includes other
// workspaces' panes, and a new workspace may already hold another plugin's pane.
func TestPickTargetPane(t *testing.T) {
	panes := []paneCandidate{
		{PaneID: "w1:p1", TabID: "w1:t1", WorkspaceID: "w1", Focused: true},
		{PaneID: "w7:p2", TabID: "w7:t1", WorkspaceID: "w7"},
		{PaneID: "w7:p3", TabID: "w7:t2", WorkspaceID: "w7", Focused: true},
	}

	cases := []struct {
		name        string
		panes       []paneCandidate
		workspaceID string
		tabID       string
		want        string
	}{
		{name: "pane in the named tab wins", panes: panes, workspaceID: "w7", tabID: "w7:t1", want: "w7:p2"},
		{name: "focused pane when the tab is unknown", panes: panes, workspaceID: "w7", want: "w7:p3"},
		{name: "unknown tab falls back rather than failing", panes: panes, workspaceID: "w7", tabID: "w7:t9", want: "w7:p3"},
		{
			name:        "first pane when none is focused",
			panes:       []paneCandidate{{PaneID: "w7:p2", TabID: "w7:t1", WorkspaceID: "w7"}},
			workspaceID: "w7",
			want:        "w7:p2",
		},
		{name: "no panes in the workspace", panes: panes, workspaceID: "w9", want: ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pickTargetPane(c.panes, c.workspaceID, c.tabID); got != c.want {
				t.Errorf("pickTargetPane = %q, want %q", got, c.want)
			}
		})
	}
}

// TestNewWorkspacePickerPresentation confirms the new-workspace presentation
// changes the wording — and only the wording — so esc reads as "keep the empty
// workspace" rather than a dead end.
func TestNewWorkspacePickerPresentation(t *testing.T) {
	base := newProjectsModel([]Project{{Name: "Options Cafe"}}, "/tmp/projects", "")
	picker := base.asNewWorkspacePicker()

	if base.headerTitle() != projectsTitle {
		t.Errorf("default headerTitle = %q, want %q", base.headerTitle(), projectsTitle)
	}
	if picker.headerTitle() != newWorkspaceTitle {
		t.Errorf("new-workspace headerTitle = %q, want %q", picker.headerTitle(), newWorkspaceTitle)
	}
	if base.cancelHint() == picker.cancelHint() {
		t.Errorf("cancelHint is %q in both presentations; the new-workspace one must say what esc keeps", base.cancelHint())
	}
	if len(picker.projects) != len(base.projects) {
		t.Error("asNewWorkspacePicker changed the project list; it must only change wording")
	}
}
