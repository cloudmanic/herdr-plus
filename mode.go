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
// to herdr. We expect this list to grow.
type Mode struct {
	// Slug is the stable identifier used on the command line (--mode=<slug>) and
	// as the per-mode configuration subdirectory name under ~/.config/herdr-plus.
	Slug string
	// Title is the human-facing heading shown at the top of the mode's UI.
	Title string
	// DefaultKey is the herdr keybinding `herdr-plus install` binds for this mode
	// when the user does not pass --key. Each mode claims its own key so the two
	// modes can coexist (quick-actions on prefix+down, control on prefix+up).
	DefaultKey string
}

// ModeControl is herdr-plus's home base: a full-screen workspace ("Herdr Plus")
// from which you drive herdr. Its first feature is Projects — declarative
// workspace templates you fuzzy-pick to spin up a fully laid-out workspace. It
// is the default mode, so the bare binary lands here.
var ModeControl = Mode{Slug: "control", Title: "Herdr Plus · Control · Projects", DefaultKey: "prefix+up"}

// ModeQuickActions is the fuzzy launcher: pick an action from your config and
// run it in a split beneath the pane you launched from. This was herdr-plus's
// original behavior.
var ModeQuickActions = Mode{Slug: "quick-actions", Title: "⚡ Quick Actions", DefaultKey: "prefix+down"}

// defaultMode is the mode used when --mode is omitted. Control mode is the
// front door of herdr-plus, so the bare binary opens it.
var defaultMode = ModeControl

// modes is every mode herdr-plus knows about, keyed by slug. Add new modes here.
var modes = map[string]Mode{
	ModeControl.Slug:      ModeControl,
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
