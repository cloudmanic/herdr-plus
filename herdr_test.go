//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "testing"

// TestWorktreeCreateParams confirms the worktree.create contract: cwd/focus are
// always sent, while a blank branch is omitted so herdr can generate its default.
func TestWorktreeCreateParams(t *testing.T) {
	cases := []struct {
		name       string
		branch     string
		wantBranch string
		wantKey    bool
	}{
		{"empty", "", "", false},
		{"whitespace", "   \t", "", false},
		{"trimmed", "  feature/x  ", "feature/x", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := worktreeCreateParams("/srv/repo", c.branch, true)
			if got["cwd"] != "/srv/repo" {
				t.Fatalf("cwd = %v, want /srv/repo", got["cwd"])
			}
			if got["focus"] != true {
				t.Fatalf("focus = %v, want true", got["focus"])
			}
			branch, ok := got["branch"]
			if ok != c.wantKey {
				t.Fatalf("branch key present = %v, want %v (params=%v)", ok, c.wantKey, got)
			}
			if ok && branch != c.wantBranch {
				t.Fatalf("branch = %v, want %q", branch, c.wantBranch)
			}
		})
	}
}
