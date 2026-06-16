//
// Date: 2026-06-15
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
// It is gathered when the quick-actions action fires — from the focused pane herdr
// reports — then serialized and handed to the picker. The picker substitutes
// these values into the chosen command (as Go template fields) and also exports
// them to the command's environment.
type RunContext struct {
	// Value is the dynamic input for the action: the chosen option's value for a
	// "select" action, or the entered string for a "form" action. It is empty for
	// a plain "command" action.
	Value string `json:"value"`

	// WorkDir is the working directory the user invoked herdr-plus from. Actions
	// run with this as their working directory and can read it as {{.WorkDir}}.
	WorkDir string `json:"work_dir"`

	// herdr session/pane metadata.
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

// encode serializes the context to a base64 JSON blob so the launching action can
// pass it to the picker pane as a single, shell-safe environment variable.
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

// pluginContext mirrors the subset of HERDR_PLUGIN_CONTEXT_JSON herdr-plus reads.
// herdr injects this when it runs a plugin action, describing the pane that was
// focused when the action fired — exactly the context a quick action wants.
type pluginContext struct {
	WorkspaceID      string `json:"workspace_id"`
	WorkspaceLabel   string `json:"workspace_label"`
	WorkspaceCwd     string `json:"workspace_cwd"`
	TabID            string `json:"tab_id"`
	TabLabel         string `json:"tab_label"`
	FocusedPaneID    string `json:"focused_pane_id"`
	FocusedPaneCwd   string `json:"focused_pane_cwd"`
	FocusedPaneAgent string `json:"focused_pane_agent"`
}

// contextFromPluginEnv builds a RunContext from HERDR_PLUGIN_CONTEXT_JSON, which
// herdr sets when it runs the quick-actions action. The working directory is the
// focused pane's cwd (the user's real directory), falling back to the workspace
// cwd. Any field herdr does not supply is left empty — a partial context is far
// better than refusing to launch.
func contextFromPluginEnv() RunContext {
	var pc pluginContext
	if raw := os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &pc)
	}
	return RunContext{
		WorkDir:        firstNonEmpty(pc.FocusedPaneCwd, pc.WorkspaceCwd),
		PaneId:         pc.FocusedPaneID,
		TabId:          pc.TabID,
		TabLabel:       pc.TabLabel,
		WorkspaceId:    pc.WorkspaceID,
		WorkspaceLabel: pc.WorkspaceLabel,
		Agent:          pc.FocusedPaneAgent,
	}
}
