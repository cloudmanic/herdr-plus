//
// Date: 2026-07-05
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

// TestPaneSplitParams confirms the pane.split contract: the target, direction and
// focus always ride along, while a zero ratio is left out so herdr splits evenly.
func TestPaneSplitParams(t *testing.T) {
	cases := []struct {
		name    string
		ratio   float64
		wantKey bool
	}{
		{"unset", 0, false},
		{"negative", -0.5, false},
		{"set", 0.75, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := paneSplitParams("w1:p1", SplitRight, c.ratio, "", false)
			if got["target_pane_id"] != "w1:p1" {
				t.Fatalf("target_pane_id = %v, want w1:p1", got["target_pane_id"])
			}
			if got["direction"] != SplitRight {
				t.Fatalf("direction = %v, want %q", got["direction"], SplitRight)
			}
			if got["focus"] != false {
				t.Fatalf("focus = %v, want false", got["focus"])
			}
			ratio, ok := got["ratio"]
			if ok != c.wantKey {
				t.Fatalf("ratio key present = %v, want %v (params=%v)", ok, c.wantKey, got)
			}
			if ok && ratio != c.ratio {
				t.Fatalf("ratio = %v, want %v", ratio, c.ratio)
			}
		})
	}
}

// TestPaneSplitParamsCwd confirms a pane's directory rides along only when it has
// one: an empty cwd is left out of the payload so the new pane inherits the
// directory of the pane it was split from, which is what a config that says
// nothing about directories has always done.
func TestPaneSplitParamsCwd(t *testing.T) {
	if _, ok := paneSplitParams("w1:p1", SplitDown, 0, "", false)["cwd"]; ok {
		t.Fatal("an empty cwd should be omitted from the pane.split payload")
	}
	if _, ok := paneSplitParams("w1:p1", SplitDown, 0, "   ", false)["cwd"]; ok {
		t.Fatal("a blank cwd should be omitted from the pane.split payload")
	}
	got := paneSplitParams("w1:p1", SplitDown, 0, "/src/web", false)
	if got["cwd"] != "/src/web" {
		t.Fatalf("cwd = %v, want /src/web", got["cwd"])
	}
}

// TestTabCreateParams confirms the tab.create contract: the workspace, label and
// focus always ride along, and a directory is sent only when the tab has one so
// an unset working_dir keeps inheriting the workspace's.
func TestTabCreateParams(t *testing.T) {
	got := tabCreateParams("w1", "editor", "", true)
	if got["workspace_id"] != "w1" || got["label"] != "editor" || got["focus"] != true {
		t.Fatalf("params = %v, want the workspace, label and focus carried through", got)
	}
	if _, ok := got["cwd"]; ok {
		t.Fatal("an empty cwd should be omitted from the tab.create payload")
	}
	if got := tabCreateParams("w1", "editor", "/src/api", false); got["cwd"] != "/src/api" {
		t.Fatalf("cwd = %v, want /src/api", got["cwd"])
	}
}
