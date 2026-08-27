//
// Date: 2026-08-10
// Author: idoga
//

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// TestLaunchQuickActionsPlacement mirrors TestLaunchProjectsPlacement for the
// quick-actions picker, whose built-in default is overlay rather than zoomed.
func TestLaunchQuickActionsPlacement(t *testing.T) {
	cases := []struct {
		name       string
		configToml string
		want       string
	}{
		{"no config defaults to overlay", "", "overlay"},
		{"valid override is passed through", "[quick_actions]\nplacement = \"popup\"\n", "popup"},
		{"invalid value falls back to overlay", "[quick_actions]\nplacement = \"bogus\"\n", "overlay"},
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

			launchQuickActions()

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
