//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
)

// herdrClient talks to the running herdr instance over its unix domain socket.
// The protocol is newline-delimited JSON: one request object per line, one
// response object per line. Each call opens a short-lived connection, writes a
// single request, and reads a single response.
type herdrClient struct {
	socketPath string
}

// newHerdrClient builds a client from the HERDR_SOCKET_PATH environment
// variable. It returns an error when the process is not running inside herdr.
func newHerdrClient() (*herdrClient, error) {
	path := os.Getenv("HERDR_SOCKET_PATH")
	if path == "" {
		return nil, errors.New("HERDR_SOCKET_PATH is not set; are you running inside herdr?")
	}
	return &herdrClient{socketPath: path}, nil
}

// request is one JSON-RPC-style message sent to herdr.
type request struct {
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}

// herdrError carries the code and human message herdr returns on failure.
type herdrError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// response is one JSON line returned by herdr. Exactly one of Result or Error
// is populated.
type response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *herdrError     `json:"error"`
}

// call sends a single request over a fresh connection and decodes the result
// into out (which may be nil when the caller does not care about the payload).
func (c *herdrClient) call(method string, params map[string]any, out any) error {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("connect herdr socket: %w", err)
	}
	defer conn.Close()

	// json.Encoder.Encode appends a trailing newline, which is exactly the
	// framing herdr expects for each request.
	if err := json.NewEncoder(conn).Encode(request{ID: "qa", Method: method, Params: params}); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	var resp response
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&resp); err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("herdr error %s: %s", resp.Error.Code, resp.Error.Message)
	}
	if out != nil {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("decode result: %w", err)
		}
	}
	return nil
}

// paneSplit splits the target pane in the given direction ("down" for a new pane
// beneath it, "right" for one beside it), creating a new pane, and returns the
// new pane's id. When focus is true the new pane becomes the focused pane (the
// socket API does not focus new panes by default).
func (c *herdrClient) paneSplit(targetPaneID, direction string, focus bool) (string, error) {
	var out struct {
		Pane struct {
			PaneID string `json:"pane_id"`
		} `json:"pane"`
	}
	err := c.call("pane.split", map[string]any{
		"target_pane_id": targetPaneID,
		"direction":      direction,
		"focus":          focus,
	}, &out)
	if err != nil {
		return "", err
	}
	return out.Pane.PaneID, nil
}

// sendInput writes text into a pane as if it were typed at the keyboard.
// Include a trailing newline to submit a shell command.
func (c *herdrClient) sendInput(paneID, text string) error {
	return c.call("pane.send_input", map[string]any{
		"pane_id": paneID,
		"text":    text,
	}, nil)
}

// closePane terminates a pane and frees its terminal. Closing the focused pane
// returns focus to an adjacent pane.
func (c *herdrClient) closePane(paneID string) error {
	return c.call("pane.close", map[string]any{
		"pane_id": paneID,
	}, nil)
}

// paneInfo is the subset of herdr's pane metadata we expose to actions.
type paneInfo struct {
	PaneID        string `json:"pane_id"`
	TabID         string `json:"tab_id"`
	WorkspaceID   string `json:"workspace_id"`
	TerminalID    string `json:"terminal_id"`
	Cwd           string `json:"cwd"`
	ForegroundCwd string `json:"foreground_cwd"`
	Agent         string `json:"agent"`
	AgentSession  struct {
		Value string `json:"value"`
	} `json:"agent_session"`
}

// focusedPaneID returns the id of the currently focused pane. It is used when
// herdr-plus is launched outside a pane's own shell — for example from a
// keybinding, which runs server-side and does not set HERDR_PANE_ID.
func (c *herdrClient) focusedPaneID() (string, error) {
	var out struct {
		Panes []struct {
			PaneID  string `json:"pane_id"`
			Focused bool   `json:"focused"`
		} `json:"panes"`
	}
	if err := c.call("pane.list", map[string]any{}, &out); err != nil {
		return "", err
	}
	for _, p := range out.Panes {
		if p.Focused {
			return p.PaneID, nil
		}
	}
	return "", errors.New("no focused pane")
}

// paneGet fetches metadata for a single pane, including its working directory
// and the tab/workspace it belongs to.
func (c *herdrClient) paneGet(paneID string) (paneInfo, error) {
	var out struct {
		Pane paneInfo `json:"pane"`
	}
	err := c.call("pane.get", map[string]any{"pane_id": paneID}, &out)
	return out.Pane, err
}

// tabInfo is the subset of herdr's tab metadata we expose to actions.
type tabInfo struct {
	TabID string `json:"tab_id"`
	Label string `json:"label"`
}

// tabGet fetches metadata for a single tab, notably its human label.
func (c *herdrClient) tabGet(tabID string) (tabInfo, error) {
	var out struct {
		Tab tabInfo `json:"tab"`
	}
	err := c.call("tab.get", map[string]any{"tab_id": tabID}, &out)
	return out.Tab, err
}

// workspaceInfo is the subset of herdr's workspace metadata we expose to
// actions.
type workspaceInfo struct {
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
}

// workspaceGet fetches metadata for a single workspace, notably its label —
// which herdr derives from the repo or folder name and is our best stand-in for
// a "session title".
func (c *herdrClient) workspaceGet(workspaceID string) (workspaceInfo, error) {
	var out struct {
		Workspace workspaceInfo `json:"workspace"`
	}
	err := c.call("workspace.get", map[string]any{"workspace_id": workspaceID}, &out)
	return out.Workspace, err
}

// workspaceCreate makes a brand-new workspace rooted at cwd with the given
// label, and returns the ids of the new workspace, its single root tab, and
// that tab's root pane. When focus is true the workspace becomes the active one
// (the user is switched to it); pass false to create it in the background.
// Control mode uses this both to open its own "Herdr Plus" workspace and to
// build a project's workspace.
func (c *herdrClient) workspaceCreate(cwd, label string, focus bool) (workspaceID, tabID, paneID string, err error) {
	var out struct {
		Workspace struct {
			WorkspaceID string `json:"workspace_id"`
		} `json:"workspace"`
		Tab struct {
			TabID string `json:"tab_id"`
		} `json:"tab"`
		RootPane struct {
			PaneID string `json:"pane_id"`
		} `json:"root_pane"`
	}
	err = c.call("workspace.create", map[string]any{
		"cwd":   cwd,
		"label": label,
		"focus": focus,
	}, &out)
	if err != nil {
		return "", "", "", err
	}
	return out.Workspace.WorkspaceID, out.Tab.TabID, out.RootPane.PaneID, nil
}

// tabCreate adds a tab to an existing workspace and returns the new tab's id and
// its root pane's id. focus controls whether the new tab is brought to the front
// — we create a project's later tabs with focus=false so the first tab stays
// active while the rest spin up behind it.
func (c *herdrClient) tabCreate(workspaceID, label string, focus bool) (tabID, paneID string, err error) {
	var out struct {
		Tab struct {
			TabID string `json:"tab_id"`
		} `json:"tab"`
		RootPane struct {
			PaneID string `json:"pane_id"`
		} `json:"root_pane"`
	}
	err = c.call("tab.create", map[string]any{
		"workspace_id": workspaceID,
		"label":        label,
		"focus":        focus,
	}, &out)
	if err != nil {
		return "", "", err
	}
	return out.Tab.TabID, out.RootPane.PaneID, nil
}

// tabRename changes a tab's human label. A freshly created workspace's root tab
// is named "1"; we rename it to the project's first tab name (or "projects" for
// the control workspace itself).
func (c *herdrClient) tabRename(tabID, label string) error {
	return c.call("tab.rename", map[string]any{
		"tab_id": tabID,
		"label":  label,
	}, nil)
}

// workspaceClose tears down a whole workspace and all of its tabs and panes.
// Control mode calls this on its own ephemeral "Herdr Plus" workspace once a
// project has been opened (or the picker is cancelled).
func (c *herdrClient) workspaceClose(workspaceID string) error {
	return c.call("workspace.close", map[string]any{
		"workspace_id": workspaceID,
	}, nil)
}
