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
// git worktree is created. Where a Project is opened on demand and creates its
// own workspace, a worktree layout reacts to herdr's `worktree.created` event:
// herdr has already made the worktree's workspace, so the layout just fills it
// with tabs. It reuses the same ProjectTab model, so worktree layouts get the
// full single-command / split-pane vocabulary that projects have.
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

	// Enabled is the per-layout on/off switch. It is a pointer so an omitted
	// `enabled` key is distinguishable from an explicit `enabled = false`: omitted
	// (nil) means on, matching the rule that simply having a layout file turns the
	// feature on. Set `enabled = false` to keep the layout on disk but stop it
	// deploying — creating a worktree of its repo then just makes a plain
	// workspace, as if no layout existed.
	Enabled *bool `toml:"enabled"`

	// Tabs is the ordered list of tabs to open in the worktree's workspace,
	// identical in shape to a Project's tabs (a single `command`, or multiple
	// [[tabs.panes]] splits).
	Tabs []ProjectTab `toml:"tabs"`

	// source is the file the layout was loaded from, used only for error and log
	// messages. It is not part of the on-disk format.
	source string
}

// isEnabled reports whether the layout should deploy. A layout with no `enabled`
// key (Enabled == nil) is on by default — the presence of the file is the opt-in
// — so only an explicit `enabled = false` turns it off.
func (l WorktreeLayout) isEnabled() bool {
	return l.Enabled == nil || *l.Enabled
}

// validate checks that a layout is internally consistent before we ever act on a
// worktree event, turning config mistakes into clear errors at load time. A
// disabled layout is still validated so a typo never hides behind `enabled =
// false`, surfacing only later when the layout is switched back on.
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
// worktree's branch to match. It does not consider the enabled switch — that is
// the caller's concern, so a disabled layout can still be recognized for logging.
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

// matchWorktreeLayout returns the best enabled layout for an event, if any.
// Disabled layouts are skipped entirely — they neither deploy nor suppress an
// otherwise-matching enabled layout. Among the enabled matches a branch-specific
// one wins over a repo-only one (it is more specific); ties break by file name,
// which loadWorktreeLayouts already sorts by.
func matchWorktreeLayout(layouts []WorktreeLayout, ev worktreeEvent) (WorktreeLayout, bool) {
	var best WorktreeLayout
	found := false
	for _, l := range layouts {
		if !l.isEnabled() || !l.matches(ev) {
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

// disabledMatch returns a layout that would have matched the event but is turned
// off (enabled = false). It exists only so the handler can log the difference
// between "no layout for this repo" and "a layout exists but you switched it off"
// — a useful breadcrumb when a worktree opens plain and you expected tabs.
func disabledMatch(layouts []WorktreeLayout, ev worktreeEvent) (WorktreeLayout, bool) {
	for _, l := range layouts {
		if !l.isEnabled() && l.matches(ev) {
			return l, true
		}
	}
	return WorktreeLayout{}, false
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
// for a `worktree.created` event. Only the fields we use are declared.
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

// runOnWorktreeCreated is the `worktree.created` event handler herdr invokes (via
// the [[events]] entry in herdr-plugin.toml). It finds the enabled layout that
// matches the new worktree and lays its tabs into the workspace herdr already
// created. With no matching layout it does nothing — every worktree fires this, so
// a quiet no-op is the common, correct case. Output goes to stdout/stderr, which
// herdr captures in the plugin log (`herdr plugin log list --plugin
// cloudmanic.herdr-plus`).
func runOnWorktreeCreated(_ []string) {
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
		// No enabled layout for this repo/branch — the expected case for most
		// worktrees. Distinguish "nothing configured" from "configured but switched
		// off" so the plugin log explains why a worktree opened plain.
		if off, disabled := disabledMatch(layouts, ev); disabled {
			fmt.Printf("herdr-plus: worktree layout %q matches repo %q but is disabled (enabled = false); nothing to do.\n", off.source, ev.RepoName)
			return
		}
		fmt.Printf("herdr-plus: no worktree layout matches repo %q (branch %q); nothing to do.\n", ev.RepoName, ev.Branch)
		return
	}

	// We need the workspace herdr made for the worktree and its root tab/pane to
	// build on. They should always be present for a worktree.created event; bail
	// loudly if not so the failure is visible in the plugin log.
	if ev.WorkspaceID == "" || ev.RootTabID == "" || ev.RootPaneID == "" {
		errExit(fmt.Sprintf("worktree event missing ids (workspace=%q tab=%q pane=%q)", ev.WorkspaceID, ev.RootTabID, ev.RootPaneID))
	}

	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}

	if err := layoutTabs(client, ev.WorkspaceID, ev.RootTabID, ev.RootPaneID, layout.Tabs); err != nil {
		errExit("apply worktree layout:", err)
	}

	fmt.Printf("herdr-plus: applied worktree layout %q to repo %q (branch %q): %d tab(s).\n", layout.source, ev.RepoName, ev.Branch, len(layout.Tabs))
}
