//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// manifest mirrors the parts of herdr-plugin.toml this test asserts on: the
// top-level platform list plus every entry-point group that carries a per-OS
// `platforms` override and a `command`.
type manifest struct {
	Platforms []string       `toml:"platforms"`
	Build     []manifestStep `toml:"build"`
	Actions   []manifestStep `toml:"actions"`
	Panes     []manifestStep `toml:"panes"`
	Events    []manifestStep `toml:"events"`
}

type manifestStep struct {
	Platforms []string `toml:"platforms"`
	Command   []string `toml:"command"`
}

// TestManifestWindowsCoverage parses the real herdr-plugin.toml and enforces the
// Windows port's manifest contract: it is valid TOML, declares the windows
// platform, and every entry-point group (build, actions, panes, events) has a
// windows-targeted entry whose command invokes the herdr-plus.exe binary. This
// is the manifest half of the Definition of Done — an accidental drop of a
// windows entry (or of the .exe binary) fails here instead of silently at
// install time.
func TestManifestWindowsCoverage(t *testing.T) {
	var m manifest
	if _, err := toml.DecodeFile("herdr-plugin.toml", &m); err != nil {
		t.Fatalf("herdr-plugin.toml is not valid TOML: %v", err)
	}

	if !contains(m.Platforms, "windows") {
		t.Fatalf("top-level platforms = %v, must include windows", m.Platforms)
	}

	groups := map[string][]manifestStep{
		"build":   m.Build,
		"actions": m.Actions,
		"panes":   m.Panes,
		"events":  m.Events,
	}
	for name, steps := range groups {
		if !hasWindowsEntry(name, steps) {
			t.Errorf("%s: no windows-targeted entry found", name)
		}
	}
}

// hasWindowsEntry reports whether steps contains at least one windows entry, and
// (for non-build groups) that its command runs the herdr-plus.exe binary. The
// binary is not command[0] — the windows entries shell out through powershell
// (`powershell ... & .\bin\herdr-plus.exe <sub>`), so the .exe appears inside a
// later argument — hence the check scans every element. The build group is
// exempt: it runs `go build`, not the plugin binary.
func hasWindowsEntry(group string, steps []manifestStep) bool {
	for _, s := range steps {
		if !contains(s.Platforms, "windows") {
			continue
		}
		if group == "build" {
			return true
		}
		for _, arg := range s.Command {
			if strings.Contains(arg, "herdr-plus.exe") {
				return true
			}
		}
	}
	return false
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
