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

// splitDown splits the target pane horizontally, creating a new pane beneath it,
// and returns the new pane's id. When focus is true the new pane becomes the
// focused pane (the socket API does not focus new panes by default).
func (c *herdrClient) splitDown(targetPaneID string, focus bool) (string, error) {
	var out struct {
		Pane struct {
			PaneID string `json:"pane_id"`
		} `json:"pane"`
	}
	err := c.call("pane.split", map[string]any{
		"target_pane_id": targetPaneID,
		"direction":      "down",
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
