//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "fmt"

// Mode identifies which herdr-plus behavior to run. herdr-plus is an add-on
// platform for herdr: the same binary can be invoked many times in different
// herdr panes, and a --mode flag tells each invocation what to do when it talks
// to herdr. We expect this list to grow; for now quick-actions is the only mode.
type Mode struct {
	// Slug is the stable identifier used on the command line (--mode=<slug>) and
	// as the per-mode configuration subdirectory name under ~/.config/herdr-plus.
	Slug string
	// Title is the human-facing heading shown at the top of the mode's picker.
	Title string
}

// ModeQuickActions is the fuzzy launcher: pick an action from your config and
// run it. This is the original (and currently only) herdr-plus behavior.
var ModeQuickActions = Mode{Slug: "quick-actions", Title: "⚡ Quick Actions"}

// defaultMode is the mode used when --mode is omitted, so the bare binary keeps
// working the way it always has.
var defaultMode = ModeQuickActions

// modes is every mode herdr-plus knows about, keyed by slug. Add new modes here.
var modes = map[string]Mode{
	ModeQuickActions.Slug: ModeQuickActions,
}

// lookupMode resolves a --mode slug to its Mode. An empty slug selects the
// default mode; an unrecognized slug is an error so typos fail loudly instead of
// silently doing the wrong thing.
func lookupMode(slug string) (Mode, error) {
	if slug == "" {
		return defaultMode, nil
	}
	m, ok := modes[slug]
	if !ok {
		return Mode{}, fmt.Errorf("unknown mode %q", slug)
	}
	return m, nil
}
