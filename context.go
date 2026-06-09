//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
)

// RunContext is the bag of variables herdr-plus exposes to an action's command.
// It is gathered by the launcher — which runs in the pane the user invoked us
// from, so it sees the real working directory — then serialized and handed to
// the picker. The picker substitutes these values into the chosen command (as
// Go template fields) and also exports them to the command's environment.
type RunContext struct {
	// Value is the dynamic input for the action: the chosen option's value for a
	// "select" action, or the entered string for a "form" action. It is empty for
	// a plain "command" action.
	Value string `json:"value"`

	// WorkDir is the working directory the user invoked herdr-plus from. Actions
	// run with this as their working directory and can read it as {{.WorkDir}}.
	WorkDir string `json:"work_dir"`

	// herdr session/pane metadata, straight from the socket API.
	PaneId         string `json:"pane_id"`
	TabId          string `json:"tab_id"`
	TabLabel       string `json:"tab_label"`
	WorkspaceId    string `json:"workspace_id"`
	WorkspaceLabel string `json:"workspace_label"`
	TerminalId     string `json:"terminal_id"`
	Agent          string `json:"agent"`
	AgentSessionId string `json:"agent_session_id"`
}

// SessionTitle is a friendly alias for the workspace label — herdr's closest
// notion of "what am I working on". Templates can use {{.SessionTitle}}.
func (c RunContext) SessionTitle() string { return c.WorkspaceLabel }

// SessionId is a friendly alias for the workspace id, available as {{.SessionId}}.
func (c RunContext) SessionId() string { return c.WorkspaceId }

// Home returns the current user's home directory, available as {{.Home}}.
func (c RunContext) Home() string {
	h, _ := os.UserHomeDir()
	return h
}

// envPairs renders the context as KEY=VALUE strings so any spawned command can
// read the same variables from its environment (handy for scripts that would
// rather not bother with templating). Every field is prefixed HERDR_PLUS_ to
// avoid colliding with herdr's own HERDR_ variables.
func (c RunContext) envPairs() []string {
	return []string{
		"HERDR_PLUS_VALUE=" + c.Value,
		"HERDR_PLUS_WORKDIR=" + c.WorkDir,
		"HERDR_PLUS_PANE_ID=" + c.PaneId,
		"HERDR_PLUS_TAB_ID=" + c.TabId,
		"HERDR_PLUS_TAB_LABEL=" + c.TabLabel,
		"HERDR_PLUS_WORKSPACE_ID=" + c.WorkspaceId,
		"HERDR_PLUS_WORKSPACE_LABEL=" + c.WorkspaceLabel,
		"HERDR_PLUS_SESSION_TITLE=" + c.WorkspaceLabel,
		"HERDR_PLUS_SESSION_ID=" + c.WorkspaceId,
		"HERDR_PLUS_TERMINAL_ID=" + c.TerminalId,
		"HERDR_PLUS_AGENT=" + c.Agent,
		"HERDR_PLUS_AGENT_SESSION_ID=" + c.AgentSessionId,
	}
}

// encode serializes the context to a base64 JSON blob so the launcher can pass
// it to the picker as a single, shell-safe command-line argument.
func (c RunContext) encode() (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// decodeRunContext is the inverse of encode. An empty string yields a zero
// context rather than an error so the picker can still run with no metadata.
func decodeRunContext(s string) (RunContext, error) {
	var c RunContext
	if s == "" {
		return c, nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}

// gatherContext collects everything herdr-plus knows about the invoking pane:
// the local working directory (from the launcher's own process, which is the
// most accurate source) plus the pane, tab, and workspace metadata herdr
// reports over the socket. Any field herdr cannot supply is left empty — a
// partial context is far better than refusing to launch.
func gatherContext(client *herdrClient, paneID string) RunContext {
	ctx := RunContext{PaneId: paneID}

	// The launcher runs in the user's pane, so its own cwd is exactly the
	// directory the action should run in.
	if wd, err := os.Getwd(); err == nil {
		ctx.WorkDir = wd
	}

	pane, err := client.paneGet(paneID)
	if err != nil {
		return ctx
	}
	ctx.TabId = pane.TabID
	ctx.WorkspaceId = pane.WorkspaceID
	ctx.TerminalId = pane.TerminalID
	ctx.Agent = pane.Agent
	ctx.AgentSessionId = pane.AgentSession.Value
	if ctx.WorkDir == "" {
		ctx.WorkDir = pane.ForegroundCwd
	}

	// Tab and workspace labels are best-effort; ignore their errors.
	if tab, err := client.tabGet(pane.TabID); err == nil {
		ctx.TabLabel = tab.Label
	}
	if ws, err := client.workspaceGet(pane.WorkspaceID); err == nil {
		ctx.WorkspaceLabel = ws.Label
	}
	return ctx
}
