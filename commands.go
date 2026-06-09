//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

// Command is a single selectable action in the quick-action picker. Name and
// Description are shown in the list; Run is the shell command executed when the
// command is chosen.
type Command struct {
	Name        string
	Description string
	Run         string
}

// commands returns the built-in quick actions. This list is intentionally tiny
// for now and will grow over time.
func commands() []Command {
	return []Command{
		{Name: "GitHub", Description: "Open https://github.com", Run: "open https://github.com"},
		{Name: "Google", Description: "Open https://google.com", Run: "open https://google.com"},
		{Name: "Options Cafe", Description: "Open https://options.cafe", Run: "open https://options.cafe"},
	}
}
