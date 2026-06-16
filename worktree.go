//
// Date: 2026-06-16
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// WorktreeLayout is a declarative tab layout applied automatically when a herdr
// git worktree is created or opened. Where a Project is opened on demand and
// creates its own workspace, a worktree layout reacts to herdr's worktree.created
// and worktree.opened events: herdr has already made the worktree's workspace, so
// the layout just fills it with tabs. It reuses the same ProjectTab model, so
// worktree layouts get the full single-command / split-pane vocabulary that
// projects have.
//
// Layouts live in ~/.config/herdr-plus/worktrees/, one TOML file per layout (the
// file name does not matter). With no files there, the feature is simply inert.
type WorktreeLayout struct {
	// Repo selects which worktrees this layout applies to, matched
	// case-insensitively against the new worktree's repo name (the repository's
	// basename, e.g. "options-cafe"). Required.
	Repo string `toml:"repo"`

	// Branch, when set, narrows the layout to worktrees created on exactly this
	// branch (case-insensitive). Empty applies to every branch of the repo, and a
	// branch-specific layout is preferred over a repo-only one when both match.
	Branch string `toml:"branch"`

	// Tabs is the ordered list of tabs to open in the worktree's workspace,
	// identical in shape to a Project's tabs (a single `command`, or multiple
	// [[tabs.panes]] splits).
	Tabs []ProjectTab `toml:"tabs"`

	// source is the file the layout was loaded from, used only for error and log
	// messages. It is not part of the on-disk format.
	source string
}

// validate checks that a layout is internally consistent before we ever act on a
// worktree event, turning config mistakes into clear errors at load time.
func (l WorktreeLayout) validate() error {
	if strings.TrimSpace(l.Repo) == "" {
		return fmt.Errorf("worktree layout %s: repo is required", l.source)
	}
	if len(l.Tabs) == 0 {
		return fmt.Errorf("worktree layout %q (%s): needs at least one [[tabs]] entry", l.Repo, l.source)
	}
	return validateTabs(l.Repo, l.source, l.Tabs)
}

// matches reports whether this layout applies to the given worktree event. The
// repo must match (against either the repo name or the basename of the repo
// root, case-insensitively); a layout with a Branch additionally requires the
// worktree's branch to match.
func (l WorktreeLayout) matches(ev worktreeEvent) bool {
	repo := strings.TrimSpace(l.Repo)
	if !strings.EqualFold(repo, ev.RepoName) && !strings.EqualFold(repo, filepath.Base(ev.RepoRoot)) {
		return false
	}
	if l.Branch != "" && !strings.EqualFold(strings.TrimSpace(l.Branch), ev.Branch) {
		return false
	}
	return true
}

// worktreesConfigDir returns the directory that holds worktree layout files,
// ~/.config/herdr-plus/worktrees. Like projects, it hangs directly off the
// herdr-plus config root rather than under a mode slug.
func worktreesConfigDir() (string, error) {
	base, err := configBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "worktrees"), nil
}

// loadWorktreeLayouts reads, parses, and validates every *.toml layout in the
// worktrees directory, returning them sorted by file name. A missing directory
// yields no layouts and no error, so the feature is opt-in: with nothing there,
// the worktree handler is a no-op. A malformed or invalid file fails the whole
// load with a message naming the offending files, so config mistakes surface in
// the plugin log instead of a layout silently going missing.
func loadWorktreeLayouts() ([]WorktreeLayout, error) {
	dir, err := worktreesConfigDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var layouts []WorktreeLayout
	var problems []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		path := filepath.Join(dir, e.Name())

		var l WorktreeLayout
		if _, err := toml.DecodeFile(path, &l); err != nil {
			problems = append(problems, fmt.Sprintf("  %s: %v", e.Name(), err))
			continue
		}
		l.source = e.Name()
		if err := l.validate(); err != nil {
			problems = append(problems, "  "+err.Error())
			continue
		}
		layouts = append(layouts, l)
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid worktree layout files in %s:\n%s", dir, strings.Join(problems, "\n"))
	}

	sort.Slice(layouts, func(i, j int) bool { return layouts[i].source < layouts[j].source })
	return layouts, nil
}

// matchWorktreeLayout returns the best matching layout for an event, if any.
// Among all matching layouts a branch-specific one wins over a repo-only one (it
// is more specific); ties break by file name, which loadWorktreeLayouts already
// sorts by.
func matchWorktreeLayout(layouts []WorktreeLayout, ev worktreeEvent) (WorktreeLayout, bool) {
	var best WorktreeLayout
	found := false
	for _, l := range layouts {
		if !l.matches(ev) {
			continue
		}
		if !found {
			best, found = l, true
			continue
		}
		// A branch-specific match beats the repo-only match we already have.
		if best.Branch == "" && l.Branch != "" {
			best = l
		}
	}
	return best, found
}

// worktreeEvent is the subset of the `worktree.created` payload herdr-plus acts
// on: the repo and branch (for matching a layout) plus the ids of the workspace,
// root tab, and root pane herdr already created for the worktree (to lay the
// layout into).
type worktreeEvent struct {
	WorkspaceID  string
	RootTabID    string
	RootPaneID   string
	RepoName     string
	RepoRoot     string
	Branch       string
	CheckoutPath string
}

// worktreeCreatedPayload mirrors the JSON herdr puts in HERDR_PLUGIN_EVENT_JSON
// for a worktree.created or worktree.opened event (the two share this shape).
// Only the fields we use are declared.
type worktreeCreatedPayload struct {
	Data struct {
		Workspace struct {
			WorkspaceID string `json:"workspace_id"`
			ActiveTabID string `json:"active_tab_id"`
			Worktree    struct {
				RepoName     string `json:"repo_name"`
				RepoRoot     string `json:"repo_root"`
				CheckoutPath string `json:"checkout_path"`
			} `json:"worktree"`
		} `json:"workspace"`
		Worktree struct {
			Path   string `json:"path"`
			Branch string `json:"branch"`
		} `json:"worktree"`
	} `json:"data"`
}

// parseWorktreeEvent builds a worktreeEvent from the event JSON and the plugin
// environment. herdr provides the workspace/tab/pane ids both as HERDR_* env vars
// and (for workspace and tab) inside the payload; we prefer the env vars and fall
// back to the payload. The root pane id comes only from HERDR_PANE_ID. getenv is
// injected so the parsing is unit-testable without touching the real environment.
func parseWorktreeEvent(eventJSON string, getenv func(string) string) (worktreeEvent, error) {
	var p worktreeCreatedPayload
	if strings.TrimSpace(eventJSON) != "" {
		if err := json.Unmarshal([]byte(eventJSON), &p); err != nil {
			return worktreeEvent{}, fmt.Errorf("parse HERDR_PLUGIN_EVENT_JSON: %w", err)
		}
	}
	return worktreeEvent{
		WorkspaceID:  firstNonEmpty(getenv("HERDR_WORKSPACE_ID"), p.Data.Workspace.WorkspaceID),
		RootTabID:    firstNonEmpty(getenv("HERDR_TAB_ID"), p.Data.Workspace.ActiveTabID),
		RootPaneID:   getenv("HERDR_PANE_ID"),
		RepoName:     p.Data.Workspace.Worktree.RepoName,
		RepoRoot:     p.Data.Workspace.Worktree.RepoRoot,
		Branch:       p.Data.Worktree.Branch,
		CheckoutPath: firstNonEmpty(p.Data.Worktree.Path, p.Data.Workspace.Worktree.CheckoutPath),
	}, nil
}

// runOnWorktreeEvent is the worktree event handler herdr invokes (via the
// [[events]] entries in herdr-plugin.toml). herdr runs it for both
// worktree.created (a new worktree) and worktree.opened (an existing worktree
// reopened into a workspace); both hand us a fresh workspace that wants tabs, so
// they share one handler. It finds the layout matching the worktree and lays its
// tabs into the workspace herdr already created. With no matching layout it does
// nothing — every worktree fires this, so a quiet no-op is the common, correct
// case. Output goes to stdout/stderr, which herdr captures in the plugin log
// (`herdr plugin log list --plugin cloudmanic.herdr-plus`).
func runOnWorktreeEvent(_ []string) {
	ev, err := parseWorktreeEvent(os.Getenv("HERDR_PLUGIN_EVENT_JSON"), os.Getenv)
	if err != nil {
		errExit("worktree event:", err)
	}

	layouts, err := loadWorktreeLayouts()
	if err != nil {
		errExit(err)
	}

	layout, ok := matchWorktreeLayout(layouts, ev)
	if !ok {
		// No layout for this repo/branch — the expected case for most worktrees.
		// The feature is opt-in: a layout exists only if you put a file in
		// worktrees/, so a quiet no-op here is the common, correct path.
		fmt.Printf("herdr-plus: no worktree layout matches repo %q (branch %q); nothing to do.\n", ev.RepoName, ev.Branch)
		return
	}

	// We need the workspace herdr made for the worktree and its root tab/pane to
	// build on. They should always be present for a worktree event; bail loudly if
	// not so the failure is visible in the plugin log.
	if ev.WorkspaceID == "" || ev.RootTabID == "" || ev.RootPaneID == "" {
		errExit(fmt.Sprintf("worktree event missing ids (workspace=%q tab=%q pane=%q)", ev.WorkspaceID, ev.RootTabID, ev.RootPaneID))
	}

	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}

	// Idempotency guard. We subscribe to both worktree.created and worktree.opened,
	// and herdr may also reopen a worktree workspace across sessions — so the
	// handler can fire for a workspace we already laid out. A freshly created or
	// opened worktree workspace has exactly one (root) pane, so more than one pane
	// means the layout is already in place and we skip rather than stack a second
	// copy of the tabs on top. A pane.list error returns 0, which fails open
	// (proceeds) rather than wrongly skipping.
	if n, err := client.workspacePaneCount(ev.WorkspaceID); err == nil && n > 1 {
		fmt.Printf("herdr-plus: worktree workspace %q already has %d panes; skipping layout %q (already applied).\n", ev.WorkspaceID, n, layout.source)
		return
	}

	if err := layoutTabs(client, ev.WorkspaceID, ev.RootTabID, ev.RootPaneID, layout.Tabs); err != nil {
		errExit("apply worktree layout:", err)
	}

	fmt.Printf("herdr-plus: applied worktree layout %q to repo %q (branch %q): %d tab(s).\n", layout.source, ev.RepoName, ev.Branch, len(layout.Tabs))
}
