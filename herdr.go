//
// Date: 2026-06-15
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
	"strings"
	"time"
)

// herdrClient talks to the running herdr instance over its unix domain socket.
// The protocol is newline-delimited JSON: one request object per line, one
// response object per line. Each call opens a short-lived connection, writes a
// single request, and reads a single response. herdr injects HERDR_SOCKET_PATH
// into every plugin command, so this works whenever herdr runs us.
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
	if err := json.NewEncoder(conn).Encode(request{ID: "herdr-plus", Method: method, Params: params}); err != nil {
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

// sendInput types text into a pane and then presses the given keys, as if at the
// keyboard. The keys are herdr key names (e.g. "Enter") delivered as real key
// events. To RUN a shell command, pass the command as text and "Enter" as the
// sole key — do not embed a trailing newline in text. herdr's send_input treats
// text as a paste: once the shell's line editor (zsh ZLE) is active it inserts an
// embedded "\n" literally instead of executing the line, so the command would
// just sit at the prompt until the user pressed Enter by hand. A real Enter key
// always submits, which is also how herdr's own `pane run` works. Pass no keys
// for plain typing with no submission.
func (c *herdrClient) sendInput(paneID, text string, keys ...string) error {
	params := map[string]any{
		"pane_id": paneID,
		"text":    text,
	}
	if len(keys) > 0 {
		params["keys"] = keys
	}
	return c.call("pane.send_input", params, nil)
}

// paneRead returns the text currently shown in a pane. source selects which slice
// of the terminal to read ("visible" for the on-screen rows, "recent" for the
// recent scrollback); lines caps how many trailing lines come back. It exists so
// callers can confirm a command actually ran — its output appeared — rather than
// merely being typed at the prompt.
func (c *herdrClient) paneRead(paneID, source string, lines int) (string, error) {
	var out struct {
		Read struct {
			Text string `json:"text"`
		} `json:"read"`
	}
	err := c.call("pane.read", map[string]any{
		"pane_id": paneID,
		"source":  source,
		"lines":   lines,
	}, &out)
	if err != nil {
		return "", err
	}
	return out.Read.Text, nil
}

// runCommand types command into a freshly created pane and submits it, pacing
// itself to the shell's startup so the command actually runs instead of sitting
// unsubmitted at the prompt. There are two startup races it has to dodge:
//
//   - Typing too early: a pane created moments ago may not have an interactive
//     shell yet; keystrokes sent into that gap are dropped. So we first wait for
//     the shell to draw its prompt.
//   - Submitting too early: even once typing lands, pressing Enter before the
//     shell's line editor has the text races startup and the line is lost. So we
//     wait until the command visibly echoes before pressing Enter.
//
// Submission is a real Enter key, never a trailing "\n" in the text — herdr pastes
// text, and an embedded newline is inserted literally once zsh's line editor is
// active rather than running the line (see sendInput). Every wait is best effort:
// on timeout we proceed anyway, so a slow or unusual shell degrades to the old
// blind behavior rather than hanging.
func (c *herdrClient) runCommand(paneID, command string) error {
	// 1. Wait for the shell to be ready to receive input (its prompt is drawn).
	c.waitForPaneReady(paneID, 5*time.Second)

	// 2. Type the command (no trailing newline).
	if err := c.sendInput(paneID, command); err != nil {
		return err
	}

	// 3. Wait until the command echoes back, proving the line editor accepted it.
	c.waitForPaneText(paneID, commandEchoProbe(command), 5*time.Second)

	// 4. Submit with a real Enter key.
	return c.sendInput(paneID, "", "Enter")
}

// commandEchoProbe returns a short, stable fragment of a command to look for when
// confirming it was typed at the prompt: the first line, capped to a few
// characters so it stays on a single terminal row. A long command wraps across
// rows, so matching the whole string against the rendered screen would fail; a
// short leading fragment does not wrap and is specific enough on an otherwise
// empty fresh pane.
func commandEchoProbe(command string) string {
	probe := command
	if i := strings.IndexByte(probe, '\n'); i >= 0 {
		probe = probe[:i]
	}
	if len(probe) > 12 {
		probe = probe[:12]
	}
	return strings.TrimSpace(probe)
}

// waitForPaneReady blocks until the pane shows any non-blank content — its shell
// prompt — or the timeout elapses. A fresh pane is blank until its shell starts
// and prints a prompt, so non-blank content is a good "ready for input" signal.
// Best effort: a timeout just means we stop waiting and proceed.
func (c *herdrClient) waitForPaneReady(paneID string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if text, err := c.paneRead(paneID, "visible", 5); err == nil && strings.TrimSpace(text) != "" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// waitForPaneText blocks until the pane's visible text contains probe or the
// timeout elapses. An empty probe returns immediately. Best effort, like
// waitForPaneReady: a timeout returns without error and the caller proceeds.
func (c *herdrClient) waitForPaneText(paneID, probe string, timeout time.Duration) {
	if probe == "" {
		return
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if text, err := c.paneRead(paneID, "visible", 20); err == nil && strings.Contains(text, probe) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// closePane terminates a pane and frees its terminal. Closing the focused pane
// returns focus to an adjacent pane.
func (c *herdrClient) closePane(paneID string) error {
	return c.call("pane.close", map[string]any{
		"pane_id": paneID,
	}, nil)
}

// paneInfo is the subset of herdr's pane metadata herdr-plus uses.
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
// keybinding or action, which run server-side and may not set HERDR_PANE_ID.
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

// workspacePaneCount returns how many panes currently live in the given
// workspace. The worktree handler uses it as an idempotency guard: a freshly
// created or opened worktree workspace has exactly one (root) pane, so a count
// above one means the layout was already applied — and we should not apply it
// again. On any socket error it returns 0 so the caller fails open (proceeds) and
// degrades to the old, unguarded behavior rather than skipping wrongly.
func (c *herdrClient) workspacePaneCount(workspaceID string) (int, error) {
	var out struct {
		Panes []struct {
			WorkspaceID string `json:"workspace_id"`
		} `json:"panes"`
	}
	if err := c.call("pane.list", map[string]any{}, &out); err != nil {
		return 0, err
	}
	n := 0
	for _, p := range out.Panes {
		if p.WorkspaceID == workspaceID {
			n++
		}
	}
	return n, nil
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

// tabInfo is the subset of herdr's tab metadata herdr-plus uses.
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

// workspaceInfo is the subset of herdr's workspace metadata herdr-plus uses.
type workspaceInfo struct {
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
}

// workspaceGet fetches metadata for a single workspace, notably its label —
// which herdr derives from the repo or folder name.
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
// — a project's later tabs are created with focus=false so the first tab stays
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
// is named "1"; callers rename it to the first tab's name.
func (c *herdrClient) tabRename(tabID, label string) error {
	return c.call("tab.rename", map[string]any{
		"tab_id": tabID,
		"label":  label,
	}, nil)
}

// workspaceClose tears down a whole workspace and all of its tabs and panes.
func (c *herdrClient) workspaceClose(workspaceID string) error {
	return c.call("workspace.close", map[string]any{
		"workspace_id": workspaceID,
	}, nil)
}
