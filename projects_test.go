//
// Date: 2026-07-05
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// writeArgRecordingHerdr writes a fake herdr binary that appends its argv to
// recordPath (one arg per line, invocations separated by a blank line) and
// exits 0. Used to confirm launchProjects/launchQuickActions pass the
// expected --placement value through to `herdr plugin pane open`.
func writeArgRecordingHerdr(t *testing.T, recordPath string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "herdr")
	script := "#!/bin/sh\nfor a in \"$@\"; do echo \"$a\" >> \"" + recordPath + "\"; done\necho >> \"" + recordPath + "\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake herdr: %v", err)
	}
	return path
}

// TestLaunchProjectsPlacement confirms launchProjects defaults to zoomed with
// no config, and passes through a valid [projects].placement override.
func TestLaunchProjectsPlacement(t *testing.T) {
	cases := []struct {
		name       string
		configToml string
		want       string
	}{
		{"no config defaults to zoomed", "", "zoomed"},
		{"valid override is passed through", "[projects]\nplacement = \"popup\"\n", "popup"},
		{"invalid value falls back to zoomed", "[projects]\nplacement = \"bogus\"\n", "zoomed"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			configDir := t.TempDir()
			t.Setenv("HERDR_PLUGIN_CONFIG_DIR", configDir)
			if c.configToml != "" {
				if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(c.configToml), 0o644); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}

			record := filepath.Join(t.TempDir(), "args.txt")
			t.Setenv("HERDR_BIN_PATH", writeArgRecordingHerdr(t, record))

			launchProjects()

			got, err := os.ReadFile(record)
			if err != nil {
				t.Fatalf("read recorded args: %v", err)
			}
			args := strings.Split(string(got), "\n")
			if !containsPlacement(args, c.want) {
				t.Fatalf("recorded args %v do not contain --placement %q", args, c.want)
			}
		})
	}
}

// containsPlacement reports whether args contains "--placement" immediately
// followed by want.
func containsPlacement(args []string, want string) bool {
	for i, a := range args {
		if a == "--placement" && i+1 < len(args) && args[i+1] == want {
			return true
		}
	}
	return false
}
