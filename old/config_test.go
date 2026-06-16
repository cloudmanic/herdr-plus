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

// TestProjectConfigDir confirms the project dir mirrors the global per-mode
// layout (<repo>/.herdr-plus/<slug>) and that an unknown working directory yields
// no path.
func TestProjectConfigDir(t *testing.T) {
	got := projectConfigDir(ModeQuickActions, "/repo")
	want := filepath.Join("/repo", ".herdr-plus", ModeQuickActions.Slug)
	if got != want {
		t.Fatalf("projectConfigDir = %q, want %q", got, want)
	}
	if got := projectConfigDir(ModeQuickActions, ""); got != "" {
		t.Fatalf("projectConfigDir(\"\") = %q, want empty", got)
	}
}

// TestLoadActionsFromDirMissingIsEmpty confirms a non-existent directory yields no
// actions and no error, so an absent project config is simply ignored rather than
// failing the picker.
func TestLoadActionsFromDirMissingIsEmpty(t *testing.T) {
	actions, err := loadActionsFromDir(filepath.Join(t.TempDir(), "nope"), originProject)
	if err != nil {
		t.Fatalf("loadActionsFromDir on missing dir: %v", err)
	}
	if actions != nil {
		t.Fatalf("expected nil actions for missing dir, got %v", actions)
	}
}

// TestLoadPickerActionsCombinesGlobalAndProject confirms loadPickerActions returns
// the seeded global actions tagged global, that an absent .herdr-plus contributes
// nothing, and that a project action added under .herdr-plus/quick-actions then
// appears tagged project.
func TestLoadPickerActionsCombinesGlobalAndProject(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	work := t.TempDir()

	// No .herdr-plus yet: only the seeded globals come back, all tagged global.
	globalOnly, err := loadPickerActions(ModeQuickActions, work)
	if err != nil {
		t.Fatalf("loadPickerActions (no project): %v", err)
	}
	if len(globalOnly) == 0 {
		t.Fatal("expected seeded global actions, got none")
	}
	for _, a := range globalOnly {
		if a.origin != originGlobal {
			t.Fatalf("action %q has origin %v, want global", a.Name, a.origin)
		}
	}

	// Add a project action and confirm it now appears, tagged project.
	projDir := filepath.Join(work, projectConfigDirName, ModeQuickActions.Slug)
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	actionTOML := "name = \"make build\"\ncommand = \"make build\"\n"
	if err := os.WriteFile(filepath.Join(projDir, "make-build.toml"), []byte(actionTOML), 0o644); err != nil {
		t.Fatalf("write project action: %v", err)
	}

	combined, err := loadPickerActions(ModeQuickActions, work)
	if err != nil {
		t.Fatalf("loadPickerActions (with project): %v", err)
	}
	if len(combined) != len(globalOnly)+1 {
		t.Fatalf("got %d actions, want %d", len(combined), len(globalOnly)+1)
	}

	var found *Action
	for i := range combined {
		if combined[i].Name == "make build" {
			found = &combined[i]
		}
	}
	if found == nil {
		t.Fatal("project action \"make build\" missing from combined set")
	}
	if found.origin != originProject {
		t.Fatalf("project action origin = %v, want project", found.origin)
	}
}

// TestLoadPickerActionsRejectsBadProjectFile confirms a malformed project action
// fails the load just like a malformed global one, instead of silently dropping.
func TestLoadPickerActionsRejectsBadProjectFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	work := t.TempDir()
	projDir := filepath.Join(work, projectConfigDirName, ModeQuickActions.Slug)
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// A select action with no options is invalid.
	bad := "name = \"Broken\"\ntype = \"select\"\ncommand = \"echo {{.Value}}\"\n"
	if err := os.WriteFile(filepath.Join(projDir, "broken.toml"), []byte(bad), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := loadPickerActions(ModeQuickActions, work); err == nil {
		t.Fatal("expected error for invalid project action file, got nil")
	}
}
