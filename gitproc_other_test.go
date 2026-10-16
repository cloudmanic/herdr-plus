//
// Date: 2026-10-16
// Author: cgill27 (cgill27@gmail.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

//go:build !windows

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestExecGitTimeoutKillsChildren confirms a timed-out git is killed together
// with what it spawned (as ssh or a credential helper would be), not just git.
func TestExecGitTimeoutKillsChildren(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	fakeGitOnPath(t, "sleep 30 &\necho $! > "+pidFile+"\nwait")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, err := execGit(ctx, t.TempDir(), "fetch"); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("execGit err = %v, want a timeout", err)
	}

	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("fake git never started its child: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatalf("bad child pid %q: %v", raw, err)
	}
	// The child may linger briefly as a zombie until it is reaped; that counts
	// as gone.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) || isZombie(pid) {
			return
		}
		if time.Now().After(deadline) {
			syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("child %d of the timed-out git is still running", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// isZombie reports whether pid has exited but not been reaped (Linux only;
// false where /proc isn't available).
func isZombie(pid int) bool {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return false
	}
	_, after, ok := strings.Cut(string(stat), ") ")
	return ok && strings.HasPrefix(after, "Z")
}

// TestExecGitHasNoTerminal confirms git runs without a controlling terminal, so
// ssh can't stop to prompt for a host key or passphrase.
func TestExecGitHasNoTerminal(t *testing.T) {
	fakeGitOnPath(t, "if (: < /dev/tty) 2>/dev/null; then echo tty; else echo notty; fi")
	out, err := execGit(context.Background(), t.TempDir(), "fetch")
	if err != nil {
		t.Fatalf("execGit: %v", err)
	}
	if out != "notty" {
		t.Fatalf("git could open /dev/tty (%q); want no controlling terminal", out)
	}
}
