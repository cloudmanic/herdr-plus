//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "testing"

// TestLookupModeDefaultIsControl confirms the bare binary (empty --mode) resolves
// to control mode, while explicit slugs resolve to their modes and typos error.
func TestLookupModeDefaultIsControl(t *testing.T) {
	if defaultMode.Slug != ModeControl.Slug {
		t.Fatalf("default mode = %q, want control", defaultMode.Slug)
	}

	m, err := lookupMode("")
	if err != nil || m.Slug != ModeControl.Slug {
		t.Fatalf("lookupMode(\"\") = %q, %v; want control", m.Slug, err)
	}

	if m, err := lookupMode("quick-actions"); err != nil || m.Slug != ModeQuickActions.Slug {
		t.Fatalf("lookupMode(quick-actions) = %q, %v", m.Slug, err)
	}

	if _, err := lookupMode("nope"); err == nil {
		t.Fatal("lookupMode(nope) should error on an unknown slug")
	}
}
