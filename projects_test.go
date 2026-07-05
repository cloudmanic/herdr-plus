//
// Date: 2026-07-05
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os/exec"
	"testing"
)

// TestIsInsideGitWorkTree confirms the pre-flight check openProjectAsWorktree uses
// before creating a worktree: a plain directory is not a work tree, while a
// git-initialized one is.
func TestIsInsideGitWorkTree(t *testing.T) {
	// A fresh temp dir with no git repo is not inside a work tree.
	plain := t.TempDir()
	if isInsideGitWorkTree(plain) {
		t.Fatalf("expected %s to not be inside a git work tree", plain)
	}

	// A git-initialized dir is. Skip (rather than fail) if git can't init here, so
	// the suite still runs in an environment without a usable git.
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init").CombinedOutput(); err != nil {
		t.Skipf("git init unavailable, skipping: %v (%s)", err, out)
	}
	if !isInsideGitWorkTree(repo) {
		t.Fatalf("expected %s to be inside a git work tree", repo)
	}
}
