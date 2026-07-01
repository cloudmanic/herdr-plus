//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "testing"

// TestResolveWorktreeBranch confirms bare names get the optional prefix, names
// containing "/" are left alone, and the prefix is used verbatim.
func TestResolveWorktreeBranch(t *testing.T) {
	cases := []struct {
		input  string
		prefix string
		want   string
	}{
		{"", "dvic/", ""},
		{"foo", "dvic/", "dvic/foo"},
		{"x/foo", "dvic/", "x/foo"},
		{"foo", "", "foo"},
		{"foo", "dvic", "dvicfoo"},
	}
	for _, c := range cases {
		got := resolveWorktreeBranch(c.input, c.prefix)
		if got != c.want {
			t.Fatalf("resolveWorktreeBranch(%q, %q) = %q, want %q", c.input, c.prefix, got, c.want)
		}
	}
}
