//
// Date: 2026-06-16
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// realEventJSON is a verbatim HERDR_PLUGIN_EVENT_JSON payload captured from herdr
// 0.7.0 for a worktree.created event. Parsing the real thing (not a hand-made
// approximation) is what keeps parseWorktreeEvent honest against herdr's wire
// format.
const realEventJSON = `{"event":"worktree_created","data":{"type":"worktree_created","workspace":{"workspace_id":"w5","number":11,"label":"WT Probe","focused":false,"pane_count":1,"tab_count":1,"active_tab_id":"w5:t1","agent_status":"unknown","worktree":{"repo_key":"/private/tmp/wt-probe-repo/.git","repo_name":"wt-probe-repo","repo_root":"/tmp/wt-probe-repo","checkout_path":"/tmp/wt-probe-repo-wt","is_linked_worktree":true}},"worktree":{"path":"/tmp/wt-probe-repo-wt","branch":"probe-wt","is_bare":false,"is_detached":false,"is_prunable":false,"is_linked_worktree":true,"open_workspace_id":"w5","label":"wt-probe-repo"}}}`

// mapEnv turns a map into a getenv-style lookup for injecting a fake environment.
func mapEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// TestParseWorktreeEventReal parses the real payload plus the HERDR_* env vars
// herdr sets, and confirms every field herdr-plus relies on is extracted.
func TestParseWorktreeEventReal(t *testing.T) {
	env := mapEnv(map[string]string{
		"HERDR_WORKSPACE_ID": "w5",
		"HERDR_TAB_ID":       "w5:t1",
		"HERDR_PANE_ID":      "w5:p1",
	})

	ev, err := parseWorktreeEvent(realEventJSON, env)
	if err != nil {
		t.Fatalf("parseWorktreeEvent: %v", err)
	}

	for _, c := range []struct{ name, got, want string }{
		{"WorkspaceID", ev.WorkspaceID, "w5"},
		{"RootTabID", ev.RootTabID, "w5:t1"},
		{"RootPaneID", ev.RootPaneID, "w5:p1"},
		{"RepoName", ev.RepoName, "wt-probe-repo"},
		{"RepoRoot", ev.RepoRoot, "/tmp/wt-probe-repo"},
		{"Branch", ev.Branch, "probe-wt"},
		{"CheckoutPath", ev.CheckoutPath, "/tmp/wt-probe-repo-wt"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// TestParseWorktreeEventEnvFallback confirms the workspace and tab ids fall back
// to the payload when the env vars are absent (the root pane id has no payload
// fallback, so it stays empty).
func TestParseWorktreeEventEnvFallback(t *testing.T) {
	ev, err := parseWorktreeEvent(realEventJSON, mapEnv(nil))
	if err != nil {
		t.Fatalf("parseWorktreeEvent: %v", err)
	}
	if ev.WorkspaceID != "w5" || ev.RootTabID != "w5:t1" {
		t.Fatalf("fallback ids = %q/%q, want w5/w5:t1", ev.WorkspaceID, ev.RootTabID)
	}
	if ev.RootPaneID != "" {
		t.Fatalf("RootPaneID = %q, want empty (no payload fallback)", ev.RootPaneID)
	}
}

// TestParseWorktreeEventBadJSON confirms malformed event JSON is a clear error,
// while an empty payload parses to a zero event (so the handler degrades to a
// no-op rather than crashing).
func TestParseWorktreeEventBadJSON(t *testing.T) {
	if _, err := parseWorktreeEvent("{not json", mapEnv(nil)); err == nil {
		t.Fatal("expected an error for malformed event JSON")
	}
	if ev, err := parseWorktreeEvent("", mapEnv(nil)); err != nil || ev.RepoName != "" {
		t.Fatalf("empty payload = %+v, %v; want zero event, no error", ev, err)
	}
}

// TestMatchWorktreeLayout covers the matching rules: repo match wins, a
// branch-specific layout beats a repo-only one, case is ignored, the repo can be
// matched against the repo-root basename, and a non-match yields nothing.
func TestMatchWorktreeLayout(t *testing.T) {
	repoOnly := WorktreeLayout{Repo: "wt-probe-repo", Tabs: []ProjectTab{{Name: "x"}}, source: "a.toml"}
	branchy := WorktreeLayout{Repo: "WT-Probe-Repo", Branch: "probe-wt", Tabs: []ProjectTab{{Name: "y"}}, source: "b.toml"}
	other := WorktreeLayout{Repo: "something-else", Tabs: []ProjectTab{{Name: "z"}}, source: "c.toml"}

	ev, err := parseWorktreeEvent(realEventJSON, mapEnv(nil))
	if err != nil {
		t.Fatalf("parseWorktreeEvent: %v", err)
	}

	// Branch-specific layout is preferred over the repo-only one (order-independent).
	if got, ok := matchWorktreeLayout([]WorktreeLayout{repoOnly, branchy, other}, ev); !ok || got.source != "b.toml" {
		t.Fatalf("match = %q, %v; want b.toml (branch-specific wins)", got.source, ok)
	}
	if got, ok := matchWorktreeLayout([]WorktreeLayout{branchy, repoOnly}, ev); !ok || got.source != "b.toml" {
		t.Fatalf("match (reordered) = %q, %v; want b.toml", got.source, ok)
	}

	// Repo-only match when no branch-specific layout is present.
	if got, ok := matchWorktreeLayout([]WorktreeLayout{repoOnly, other}, ev); !ok || got.source != "a.toml" {
		t.Fatalf("match = %q, %v; want a.toml", got.source, ok)
	}

	// No match for an unrelated repo.
	if _, ok := matchWorktreeLayout([]WorktreeLayout{other}, ev); ok {
		t.Fatal("expected no match for an unrelated repo")
	}

	// Branch-specific layout does NOT match a different branch.
	otherBranch := WorktreeLayout{Repo: "wt-probe-repo", Branch: "main", Tabs: []ProjectTab{{Name: "y"}}, source: "d.toml"}
	if _, ok := matchWorktreeLayout([]WorktreeLayout{otherBranch}, ev); ok {
		t.Fatal("branch-specific layout should not match a different branch")
	}

	// Repo matched via the basename of repo_root when repo_name is empty.
	byRoot := worktreeEvent{RepoRoot: "/Users/me/Development/options-cafe", Branch: "main"}
	if _, ok := matchWorktreeLayout([]WorktreeLayout{{Repo: "options-cafe", Tabs: []ProjectTab{{Name: "x"}}}}, byRoot); !ok {
		t.Fatal("expected a match against the repo-root basename")
	}
}

// TestWorktreeLayoutValidate confirms a layout needs a repo and at least one tab,
// and that it inherits the shared per-tab validation (a bad split is rejected).
func TestWorktreeLayoutValidate(t *testing.T) {
	cases := []struct {
		name    string
		layout  WorktreeLayout
		wantErr bool
	}{
		{"valid", WorktreeLayout{Repo: "r", Tabs: []ProjectTab{{Name: "t", Command: "ls"}}}, false},
		{"no repo", WorktreeLayout{Tabs: []ProjectTab{{Name: "t"}}}, true},
		{"no tabs", WorktreeLayout{Repo: "r"}, true},
		{"bad split", WorktreeLayout{Repo: "r", Tabs: []ProjectTab{{Name: "t", Panes: []ProjectPane{{}, {Split: "sideways"}}}}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.layout.validate(); (err != nil) != c.wantErr {
				t.Fatalf("validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

// TestLoadWorktreeLayouts round-trips real files through the loader: a missing
// directory yields nothing (the feature is opt-in), and valid layouts load with
// their tabs intact, sorted by file name.
func TestLoadWorktreeLayouts(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// No worktrees directory yet → no layouts, no error.
	got, err := loadWorktreeLayouts()
	if err != nil || got != nil {
		t.Fatalf("loadWorktreeLayouts (no dir) = %v, %v; want nil, nil", got, err)
	}

	dir := filepath.Join(tmp, "herdr-plus", "worktrees")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	layout := `repo = "options-cafe"
branch = "main"

[[tabs]]
name = "claude"
command = "claude"

[[tabs]]
name = "lazygit"
command = "lazygit"
`
	if err := os.WriteFile(filepath.Join(dir, "options-cafe.toml"), []byte(layout), 0o644); err != nil {
		t.Fatal(err)
	}

	// A second, repo-only layout to confirm multiple files load and sort by name.
	second := `repo = "bevio"

[[tabs]]
name = "terminal"
`
	if err := os.WriteFile(filepath.Join(dir, "bevio.toml"), []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err = loadWorktreeLayouts()
	if err != nil {
		t.Fatalf("loadWorktreeLayouts: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d layouts, want 2", len(got))
	}
	// Sorted by file name: bevio.toml then options-cafe.toml.
	if got[0].Repo != "bevio" || len(got[0].Tabs) != 1 {
		t.Fatalf("loaded layout[0] = %+v, want bevio with 1 tab", got[0])
	}
	if got[1].Repo != "options-cafe" || got[1].Branch != "main" || len(got[1].Tabs) != 2 {
		t.Fatalf("loaded layout[1] = %+v, want options-cafe/main with 2 tabs", got[1])
	}

	// A malformed file fails the whole load with a naming error.
	if err := os.WriteFile(filepath.Join(dir, "bad.toml"), []byte("repo = "), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadWorktreeLayouts(); err == nil {
		t.Fatal("expected an error from a malformed layout file")
	}
}
