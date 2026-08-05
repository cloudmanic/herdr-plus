//
// Date: 2026-08-05
// Author: Stephane Adandedjan (Stephane.Adandedjan@goline.ch)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// This file implements the "new workspace → pick a project" flow.
//
// herdr's sidebar has a New button (and a new_workspace keybinding) that creates
// an empty workspace rooted at the current directory. herdr's plugin API has no
// hook for that button — a plugin cannot decorate it with a menu, and no event is
// cancellable — but it does fire workspace.created afterwards. So instead of
// replacing the button we react to it: when the new workspace is still empty, we
// open the projects browser over it and, once a project is chosen, we build that
// project's workspace and close the empty one it replaced.
//
// The feature is opt-in. Without `[new_workspace] mode = "picker"` in
// config.toml the handler is a quiet no-op, exactly like a worktree layout that
// matches nothing.

// New-workspace modes, the accepted values of `[new_workspace] mode` in
// config.toml. "off" (the default) leaves herdr's New button alone; "picker"
// opens the projects browser over each new, empty workspace.
const (
	newWorkspaceModeOff    = "off"
	newWorkspaceModePicker = "picker"
)

// newWorkspaceMode returns the configured, validated mode. An unset value means
// "off" so the feature stays opt-in; an unrecognized value is an error rather
// than a silent fallback, so a typo in config.toml surfaces in the plugin log
// instead of quietly disabling the feature the user asked for.
func (c PluginConfig) newWorkspaceMode() (string, error) {
	switch m := strings.ToLower(strings.TrimSpace(c.NewWorkspace.Mode)); m {
	case "":
		return newWorkspaceModeOff, nil
	case newWorkspaceModeOff, newWorkspaceModePicker:
		return m, nil
	default:
		return newWorkspaceModeOff, fmt.Errorf("invalid new_workspace.mode %q (want %q or %q)", c.NewWorkspace.Mode, newWorkspaceModeOff, newWorkspaceModePicker)
	}
}

// newWorkspaceEvent is the subset of the workspace.created payload the intercept
// decision needs: which workspace was created, how it is labeled, whether it is
// the workspace the user is now looking at, whether it is still empty, and
// whether it belongs to a git worktree (those are the worktree handler's job).
type newWorkspaceEvent struct {
	WorkspaceID string
	// ActiveTabID is the workspace's only tab at creation time. It is not part of
	// the intercept decision — it just anchors the picker pane (see
	// workspaceTargetPane).
	ActiveTabID string
	Label       string
	Focused     bool
	PaneCount   int
	TabCount    int
	IsWorktree  bool
}

// workspaceCreatedPayload mirrors the JSON herdr puts in HERDR_PLUGIN_EVENT_JSON
// for a workspace.created event. Only the fields we use are declared. worktree is
// a pointer because herdr sends null for an ordinary (non-worktree) workspace,
// and that null is precisely the signal we branch on.
type workspaceCreatedPayload struct {
	Data struct {
		Workspace struct {
			WorkspaceID string `json:"workspace_id"`
			ActiveTabID string `json:"active_tab_id"`
			Label       string `json:"label"`
			Focused     bool   `json:"focused"`
			PaneCount   int    `json:"pane_count"`
			TabCount    int    `json:"tab_count"`
			Worktree    *struct {
				RepoName string `json:"repo_name"`
			} `json:"worktree"`
		} `json:"workspace"`
	} `json:"data"`
}

// parseNewWorkspaceEvent builds a newWorkspaceEvent from the event JSON and the
// plugin environment. Unlike the worktree handler, the payload wins over
// HERDR_WORKSPACE_ID: the payload names the workspace this event is *about*,
// while the environment variable describes the caller's context — and a workspace
// created in the background is not the focused one. The env var is only a
// fallback for an id-less payload. getenv is injected so parsing is unit-testable
// without touching the real environment.
func parseNewWorkspaceEvent(eventJSON string, getenv func(string) string) (newWorkspaceEvent, error) {
	var p workspaceCreatedPayload
	if strings.TrimSpace(eventJSON) != "" {
		if err := json.Unmarshal([]byte(eventJSON), &p); err != nil {
			return newWorkspaceEvent{}, fmt.Errorf("parse HERDR_PLUGIN_EVENT_JSON: %w", err)
		}
	}
	ws := p.Data.Workspace
	return newWorkspaceEvent{
		WorkspaceID: firstNonEmpty(ws.WorkspaceID, getenv("HERDR_WORKSPACE_ID")),
		ActiveTabID: firstNonEmpty(ws.ActiveTabID, getenv("HERDR_TAB_ID")),
		Label:       ws.Label,
		Focused:     ws.Focused,
		PaneCount:   ws.PaneCount,
		TabCount:    ws.TabCount,
		IsWorktree:  ws.Worktree != nil,
	}, nil
}

// decideNewWorkspaceIntercept is the whole intercept policy in one pure function:
// given the event, the configured mode, the known project names, and whether a
// herdr-plus-created workspace was expected, it says whether to open the picker
// and — when not — why. Keeping it pure makes every guard testable without a
// running herdr, which matters because a wrong "yes" here hijacks a workspace the
// user meant to keep.
//
// The guards, in order of how cheaply they settle the question:
//
//   - mode: the feature is opt-in.
//   - a missing workspace id: nothing to act on.
//   - a worktree workspace: runOnWorktreeEvent owns those, and its layouts would
//     fight the picker for the same panes.
//   - an unfocused workspace: created in the background (by a script or the
//     socket API), so popping a full-screen picker into it would be a surprise.
//   - a workspace that already has tabs or panes: not the blank workspace the New
//     button makes, so something already filled it.
//   - suppressed: herdr-plus itself created this workspace (see
//     suppressNewWorkspaceIntercept) — intercepting it would loop forever.
//   - no projects: the picker would have nothing to offer.
//   - a label matching a project: a second, belt-and-braces guard against the
//     loop above, for the case where the suppression marker could not be written.
func decideNewWorkspaceIntercept(ev newWorkspaceEvent, mode string, projectNames []string, suppressed bool) (bool, string) {
	if mode != newWorkspaceModePicker {
		return false, "new_workspace.mode is not " + newWorkspaceModePicker
	}
	if strings.TrimSpace(ev.WorkspaceID) == "" {
		return false, "the event carries no workspace id"
	}
	if ev.IsWorktree {
		return false, "it is a worktree workspace (the worktree layout handles those)"
	}
	if !ev.Focused {
		return false, "it was created in the background"
	}
	if ev.PaneCount > 1 || ev.TabCount > 1 {
		return false, fmt.Sprintf("it is not empty (%d tab(s), %d pane(s))", ev.TabCount, ev.PaneCount)
	}
	if suppressed {
		return false, "herdr-plus created it"
	}
	if len(projectNames) == 0 {
		return false, "no projects are defined"
	}
	if matchesProjectName(ev.Label, projectNames) {
		return false, fmt.Sprintf("its label %q matches a project", ev.Label)
	}
	return true, ""
}

// matchesProjectName reports whether label is (case-insensitively) one of the
// known project names — the signature of a workspace herdr-plus opened itself.
func matchesProjectName(label string, projectNames []string) bool {
	label = strings.TrimSpace(label)
	if label == "" {
		return false
	}
	for _, n := range projectNames {
		if strings.EqualFold(strings.TrimSpace(n), label) {
			return true
		}
	}
	return false
}

// projectNames extracts the names of the loaded projects, the form
// decideNewWorkspaceIntercept wants.
func projectNames(projects []Project) []string {
	names := make([]string, 0, len(projects))
	for _, p := range projects {
		names = append(names, p.Name)
	}
	return names
}

// newWorkspaceInterceptTTL bounds how long a suppression marker stays valid. The
// marker is written immediately before herdr-plus asks herdr for a workspace, so
// the matching event follows within milliseconds; a short window is enough to
// cover it while making sure a marker whose event never arrived cannot swallow a
// New the user makes a moment later.
const newWorkspaceInterceptTTL = 15 * time.Second

// newWorkspacePaneTimeout bounds the wait for the new workspace's root pane to
// become visible to pane.list. herdr emits workspace.created before that pane is
// necessarily listed — the ordering differs between the sidebar's New button and
// an API-driven create — and the picker needs a pane to anchor to.
//
// It is deliberately generous. Creating a workspace wakes every plugin subscribed
// to workspace.created at once, and under that burst the pane has been observed
// taking several seconds to become visible to this handler even though it exists
// almost immediately. Overshooting costs a handler process sleeping in the
// background; undershooting means the browser never opens. The ceiling is herdr's
// own limit on an event command, which cuts one off well before this expires.
const newWorkspacePaneTimeout = 20 * time.Second

// newWorkspaceInterceptMarker is the marker file's name inside the plugin state
// directory.
const newWorkspaceInterceptMarker = "suppress-new-workspace-intercept"

// pluginStateDir returns the directory herdr gives the plugin for its own runtime
// state (HERDR_PLUGIN_STATE_DIR, set for every plugin command). Outside herdr —
// running the binary directly, dev, tests — it falls back to a herdr-plus folder
// in the system temp directory, which is the right lifetime for a marker that is
// only ever consumed seconds after it is written.
func pluginStateDir() string {
	if d := os.Getenv("HERDR_PLUGIN_STATE_DIR"); d != "" {
		return d
	}
	return filepath.Join(os.TempDir(), "herdr-plus")
}

// suppressNewWorkspaceIntercept records that herdr-plus is about to create a
// workspace itself, so the workspace.created event it triggers is not mistaken
// for the user pressing New. Callers write it *before* asking herdr to create the
// workspace, which is what guarantees the marker is on disk by the time the event
// handler runs.
//
// Best effort by design: a state directory we cannot write to must not stop a
// project from opening, and decideNewWorkspaceIntercept's project-name guard
// still catches the loop this prevents.
func suppressNewWorkspaceIntercept() {
	if err := writeNewWorkspaceSuppression(pluginStateDir()); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-plus: could not write the new-workspace suppression marker: %v\n", err)
	}
}

// writeNewWorkspaceSuppression drops the marker file in dir, creating dir if
// needed. The file's content is irrelevant — its modification time is the signal.
func writeNewWorkspaceSuppression(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, newWorkspaceInterceptMarker), []byte("1"), 0o644)
}

// consumeNewWorkspaceSuppression reports whether a suppression marker was waiting
// in dir and removes it, so a single marker suppresses a single event. A marker
// older than ttl is treated as stale: it is still removed (housekeeping) but does
// not suppress anything, so a create that never produced an event cannot silently
// disable the next real one.
func consumeNewWorkspaceSuppression(dir string, now time.Time, ttl time.Duration) bool {
	path := filepath.Join(dir, newWorkspaceInterceptMarker)
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	os.Remove(path)
	return now.Sub(fi.ModTime()) <= ttl
}

// runOnWorkspaceEvent is the workspace.created handler herdr invokes (via the
// [[events]] entry in herdr-plugin.toml). It fires for every workspace, including
// the ones herdr-plus makes itself, so nearly all of its work is deciding to do
// nothing — see decideNewWorkspaceIntercept. When it does decide to act it opens
// the projects browser as a zoomed pane pinned to the new workspace; that pane
// (runNewWorkspaceUI) owns everything from there.
//
// Output goes to stdout/stderr, which herdr captures in the plugin log
// (`herdr plugin log list --plugin cloudmanic.herdr-plus`).
func runOnWorkspaceEvent(_ []string) {
	cfg, err := loadPluginConfig()
	if err != nil {
		errExit(err)
	}
	mode, err := cfg.newWorkspaceMode()
	if err != nil {
		errExit(err)
	}

	// Leave the common case — the feature switched off — completely silent: this
	// handler runs for every workspace anyone creates, and logging each one would
	// bury the plugin log in noise.
	if mode != newWorkspaceModePicker {
		return
	}

	ev, err := parseNewWorkspaceEvent(os.Getenv("HERDR_PLUGIN_EVENT_JSON"), os.Getenv)
	if err != nil {
		errExit("workspace event:", err)
	}

	projects, err := loadProjects()
	if err != nil {
		errExit(err)
	}

	// Consume the marker for this event whatever we decide next: it was written
	// for exactly one workspace.created, and leaving it behind would suppress
	// somebody else's.
	suppressed := consumeNewWorkspaceSuppression(pluginStateDir(), time.Now(), newWorkspaceInterceptTTL)

	if ok, reason := decideNewWorkspaceIntercept(ev, mode, projectNames(projects), suppressed); !ok {
		fmt.Printf("herdr-plus: leaving new workspace %q alone: %s.\n", ev.WorkspaceID, reason)
		return
	}

	// The picker pane is anchored to a pane in the new workspace, so we need a
	// socket connection of our own to find one.
	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}
	started := time.Now()
	target, err := client.waitForWorkspaceTargetPane(ev.WorkspaceID, ev.ActiveTabID, newWorkspacePaneTimeout)
	if err != nil {
		errExit("could not find a pane in the new workspace:", err)
	}
	// Record a non-trivial wait: it is the one part of this handler whose timing
	// depends on how busy herdr is, so the log should say when it was close to
	// the limit.
	if waited := time.Since(started); waited > time.Second {
		fmt.Printf("herdr-plus: waited %s for the new workspace's root pane.\n", waited.Round(time.Millisecond))
	}

	if err := openNewWorkspacePicker(ev.WorkspaceID, target); err != nil {
		errExit("could not open the projects browser over the new workspace:", err)
	}
	fmt.Printf("herdr-plus: opened the projects browser over new workspace %q.\n", ev.WorkspaceID)
}

// openNewWorkspacePicker asks herdr to open the new-workspace projects browser as
// a zoomed pane inside the workspace that was just created. A zoomed plugin pane
// is positioned relative to an existing pane, not to a workspace, so it is pinned
// with --target-pane (a pane in the new workspace) rather than left to land
// wherever focus happens to be; --env hands the pane the workspace id it will
// replace so it never has to guess. It shells out to the herdr CLI, like the
// projects action does, because the event handler runs server-side with no pane of
// its own.
func openNewWorkspacePicker(workspaceID, targetPaneID string) error {
	// HERDR_BIN_PATH points at the running herdr binary; it is the portable way to
	// call back into the CLI from a plugin command.
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}

	cmd := exec.Command(herdr, "plugin", "pane", "open",
		"--plugin", pluginID,
		"--entrypoint", paneEntrypoint("new-workspace-picker"),
		"--placement", "zoomed",
		"--target-pane", targetPaneID,
		"--focus",
		"--env", newWorkspaceTargetEnv+"="+workspaceID,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// newWorkspaceTargetEnv carries the id of the empty workspace the picker was
// opened over — the one it replaces once a project is chosen. herdr sets it on
// the pane process via `plugin pane open --env`.
const newWorkspaceTargetEnv = "HERDR_PLUS_TARGET_WORKSPACE"

// runNewWorkspaceUI renders the projects browser over a brand-new, empty
// workspace and then swaps that workspace for the chosen project's. It runs
// inside the zoomed pane openNewWorkspacePicker asks herdr for.
//
// Cancelling is a first-class outcome, not a failure: esc leaves the empty
// workspace exactly as herdr's New button made it, which is how you still get a
// plain workspace with the feature enabled.
func runNewWorkspaceUI() {
	// The pane is opened with the target id in its environment; HERDR_WORKSPACE_ID
	// is the fallback, since herdr also tells a pane which workspace it lives in.
	target := firstNonEmpty(os.Getenv(newWorkspaceTargetEnv), os.Getenv("HERDR_WORKSPACE_ID"))

	choice, ok := pickProject(true)
	if !ok {
		fmt.Println("herdr-plus: no project chosen; keeping the empty workspace.")
		return
	}

	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}

	if choice.worktree {
		if err := openProjectAsWorktree(client, choice.project, choice.branch); err != nil {
			// Keep the empty workspace: it is the user's only way back.
			errExit("could not open project as worktree:", err)
		}
	} else if err := openProject(client, choice.project); err != nil {
		errExit("could not open project:", err)
	}

	fmt.Printf("herdr-plus: opened %s in place of new workspace %q.\n", choice.project.Name, target)

	// Last, because it takes this pane down with it: the picker lives in the
	// workspace being closed, so herdr kills this process as it tears the
	// workspace apart. Everything above has already completed by then.
	closeReplacedWorkspace(client, target)
}

// closeReplacedWorkspace closes the empty workspace the picker was opened over,
// once the chosen project's workspace exists and is focused. It is deliberately
// non-fatal: the project is already open, so failing to tidy up the empty
// workspace is a wart in the sidebar, not a broken outcome.
func closeReplacedWorkspace(client *herdrClient, workspaceID string) {
	if strings.TrimSpace(workspaceID) == "" {
		return
	}
	if err := client.workspaceClose(workspaceID); err != nil {
		fmt.Fprintf(os.Stderr, "herdr-plus: could not close the replaced workspace %s: %v\n", workspaceID, err)
	}
}
