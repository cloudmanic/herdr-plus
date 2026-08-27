//
// Date: 2026-08-27
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
