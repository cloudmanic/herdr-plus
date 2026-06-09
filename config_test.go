//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadActionsSeedsAndParses points the config dir at a temp directory and
// confirms loadActions seeds the embedded examples on first run and that every
// bundled example parses and validates. This doubles as a guard that the shipped
// example TOML files stay correct.
func TestLoadActionsSeedsAndParses(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	actions, err := loadActions(ModeQuickActions)
	if err != nil {
		t.Fatalf("loadActions: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected seeded example actions, got none")
	}

	// The directory should now exist with the seeded files.
	dir := filepath.Join(tmp, "herdr-plus", ModeQuickActions.Slug)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("mode config dir not created: %v", err)
	}

	// At least one example of each type should be present and valid.
	types := map[string]bool{}
	for _, a := range actions {
		if err := a.validate(); err != nil {
			t.Fatalf("seeded action %q failed validation: %v", a.Name, err)
		}
		types[a.effectiveType()] = true
	}
	for _, want := range []string{TypeCommand, TypeSelect, TypeForm} {
		if !types[want] {
			t.Fatalf("seeded examples missing a %q action; have types %v", want, types)
		}
	}
}

// TestLoadActionsRejectsBadFile confirms a malformed action file fails the load
// with an error rather than silently disappearing.
func TestLoadActionsRejectsBadFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "herdr-plus", ModeQuickActions.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// A select action with no options is invalid.
	bad := "name = \"Broken\"\ntype = \"select\"\ncommand = \"echo {{.Value}}\"\n"
	if err := os.WriteFile(filepath.Join(dir, "broken.toml"), []byte(bad), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := loadActions(ModeQuickActions); err == nil {
		t.Fatal("expected error for invalid action file, got nil")
	}
}

// TestSeedingIsNotRepeatedOnExistingDir confirms an existing (even empty) mode
// directory is left untouched, so a user who deletes an example does not see it
// return.
func TestSeedingIsNotRepeatedOnExistingDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "herdr-plus", ModeQuickActions.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if _, err := ensureModeConfig(ModeQuickActions); err != nil {
		t.Fatalf("ensureModeConfig: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected existing dir left empty, found %d files", len(entries))
	}
}
