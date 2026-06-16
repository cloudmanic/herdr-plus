//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// projectsDirIn returns the projects directory under a temp XDG config root and
// makes sure it exists, mirroring how the real config layout is rooted.
func projectsDirIn(t *testing.T, tmp string) string {
	t.Helper()
	// Pin config to the temp XDG dir even if these tests run inside a herdr
	// plugin context (where HERDR_PLUGIN_CONFIG_DIR would otherwise win).
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", "")
	dir := filepath.Join(tmp, "herdr-plus", "projects")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir projects dir: %v", err)
	}
	return dir
}

// TestLoadProjectsParsesAndSorts confirms valid project files are parsed, sorted
// by name, and have their tabs preserved in file order.
func TestLoadProjectsParsesAndSorts(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	dir := projectsDirIn(t, tmp)

	bravo := `name = "Bravo"
description = "second alphabetically"
working_dir = "~/code/bravo"

[[tabs]]
name = "edit"
command = "vim"

[[tabs]]
name = "shell"
`
	alpha := `name = "Alpha"
working_dir = "/srv/alpha"

[[tabs]]
name = "run"
command = "make serve"
`
	if err := os.WriteFile(filepath.Join(dir, "bravo.toml"), []byte(bravo), 0o644); err != nil {
		t.Fatalf("write bravo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "alpha.toml"), []byte(alpha), 0o644); err != nil {
		t.Fatalf("write alpha: %v", err)
	}

	projects, err := loadProjects()
	if err != nil {
		t.Fatalf("loadProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(projects))
	}
	if projects[0].Name != "Alpha" || projects[1].Name != "Bravo" {
		t.Fatalf("projects not sorted by name: %q, %q", projects[0].Name, projects[1].Name)
	}

	b := projects[1]
	if len(b.Tabs) != 2 || b.Tabs[0].Name != "edit" || b.Tabs[0].Command != "vim" {
		t.Fatalf("bravo tabs wrong: %+v", b.Tabs)
	}
	if b.Tabs[1].Command != "" {
		t.Fatalf("bravo second tab should have no command, got %q", b.Tabs[1].Command)
	}
}

// TestLoadProjectsParsesSplitPanes confirms a tab authored with [[tabs.panes]]
// loads with its panes, commands, and split directions intact.
func TestLoadProjectsParsesSplitPanes(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	dir := projectsDirIn(t, tmp)

	content := `name = "Rental Notice"
working_dir = "/srv/rental"

[[tabs]]
name = "claude"
command = "claude"

[[tabs]]
name = "server"

[[tabs.panes]]
command = "php artisan serve"

[[tabs.panes]]
command = "npm run dev"
split = "down"
`
	if err := os.WriteFile(filepath.Join(dir, "rental.toml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	projects, err := loadProjects()
	if err != nil {
		t.Fatalf("loadProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1", len(projects))
	}

	server := projects[0].Tabs[1]
	if len(server.Panes) != 2 {
		t.Fatalf("server panes = %d, want 2", len(server.Panes))
	}
	if server.Panes[0].Command != "php artisan serve" || server.Panes[1].Command != "npm run dev" {
		t.Fatalf("pane commands wrong: %+v", server.Panes)
	}
	if server.Panes[1].Split != "down" {
		t.Fatalf("pane 2 split = %q, want down", server.Panes[1].Split)
	}
}

// TestLoadProjectsEmptyDirIsNotAnError confirms an empty projects directory
// yields no projects (and no error), so the caller can show the empty-state.
func TestLoadProjectsEmptyDirIsNotAnError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	projectsDirIn(t, tmp)

	projects, err := loadProjects()
	if err != nil {
		t.Fatalf("loadProjects on empty dir: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected no projects, got %d", len(projects))
	}
}

// TestLoadProjectsRejectsInvalidFile confirms a structurally-invalid project
// (here: no tabs) fails the load loudly rather than silently disappearing.
func TestLoadProjectsRejectsInvalidFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	dir := projectsDirIn(t, tmp)

	noTabs := "name = \"No Tabs\"\nworking_dir = \"/tmp\"\n"
	if err := os.WriteFile(filepath.Join(dir, "notabs.toml"), []byte(noTabs), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := loadProjects(); err == nil {
		t.Fatal("expected error for project with no tabs, got nil")
	}
}

// TestProjectValidate exercises each validation rule directly.
func TestProjectValidate(t *testing.T) {
	twoPanes := []ProjectPane{{Command: "a"}, {Command: "b", Split: "down"}}
	fivePanes := []ProjectPane{{}, {}, {}, {}, {}}
	cases := []struct {
		name    string
		project Project
		wantErr bool
	}{
		{"ok", Project{Name: "A", Tabs: []ProjectTab{{Name: "t"}}}, false},
		{"missing name", Project{Tabs: []ProjectTab{{Name: "t"}}}, true},
		{"no tabs", Project{Name: "A"}, true},
		{"tab missing name", Project{Name: "A", Tabs: []ProjectTab{{Command: "ls"}}}, true},
		{"ok multi-pane", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Panes: twoPanes}}}, false},
		{"command and panes", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Command: "ls", Panes: twoPanes}}}, true},
		{"too many panes", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Panes: fivePanes}}}, true},
		{"bad split", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Panes: []ProjectPane{{}, {Split: "sideways"}}}}}, true},
		{"first pane split ignored", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Panes: []ProjectPane{{Split: "sideways"}}}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.project.validate()
			if (err != nil) != c.wantErr {
				t.Fatalf("validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

// TestEffectivePanes confirms the two authoring forms normalize correctly: a
// single-pane tab yields one pane, and a multi-pane tab clears the first pane's
// split while defaulting later panes to "down".
func TestEffectivePanes(t *testing.T) {
	single := ProjectTab{Name: "claude", Command: "claude"}.effectivePanes()
	if len(single) != 1 || single[0].Command != "claude" || single[0].Split != "" {
		t.Fatalf("single-pane = %+v", single)
	}

	multi := ProjectTab{Name: "server", Panes: []ProjectPane{
		{Command: "php artisan serve", Split: "right"}, // split on root is ignored
		{Command: "npm run dev"},                       // omitted split defaults to down
		{Command: "tail -f log", Split: "right"},
	}}.effectivePanes()
	if len(multi) != 3 {
		t.Fatalf("got %d panes, want 3", len(multi))
	}
	if multi[0].Split != "" {
		t.Fatalf("root pane split = %q, want empty", multi[0].Split)
	}
	if multi[1].Split != SplitDown {
		t.Fatalf("pane 2 split = %q, want down (default)", multi[1].Split)
	}
	if multi[2].Split != SplitRight {
		t.Fatalf("pane 3 split = %q, want right", multi[2].Split)
	}
}

// TestTabLabels confirms split tabs are annotated with a "×N" pane count while
// single-pane tabs show just their name.
func TestTabLabels(t *testing.T) {
	p := Project{Tabs: []ProjectTab{
		{Name: "claude", Command: "claude"},
		{Name: "server", Panes: []ProjectPane{{Command: "a"}, {Command: "b"}}},
	}}
	got := p.tabLabels()
	if got[0] != "claude" {
		t.Fatalf("label[0] = %q, want claude", got[0])
	}
	if got[1] != "server ×2" {
		t.Fatalf("label[1] = %q, want \"server ×2\"", got[1])
	}
}

// TestExpandedWorkingDir confirms ~, $VARS, absolute paths, and an empty value
// all resolve sensibly relative to the home directory.
func TestExpandedWorkingDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := []struct {
		in   string
		want string
	}{
		{"", home},
		{"~", home},
		{"~/code/x", filepath.Join(home, "code", "x")},
		{"$HOME/code/y", filepath.Join(home, "code", "y")},
		{"/srv/abs", "/srv/abs"},
	}
	for _, c := range cases {
		got := Project{WorkingDir: c.in}.expandedWorkingDir()
		if got != c.want {
			t.Fatalf("expandedWorkingDir(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
