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
		{"ok ratio", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Panes: []ProjectPane{{}, {Ratio: 0.3}}}}}, false},
		{"ratio as percentage", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Panes: []ProjectPane{{}, {Ratio: 30}}}}}, true},
		{"ratio of one", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Panes: []ProjectPane{{}, {Ratio: 1}}}}}, true},
		{"negative ratio", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Panes: []ProjectPane{{}, {Ratio: -0.2}}}}}, true},
		{"first pane ratio ignored", Project{Name: "A", Tabs: []ProjectTab{{Name: "t", Panes: []ProjectPane{{Ratio: 30}}}}}, false},
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

	// Labels pass through effectivePanes unchanged.
	labeled := ProjectTab{Name: "services", Panes: []ProjectPane{
		{Command: "make run", Label: "Server"},
		{Command: "make minio", Split: "right", Label: "Minio"},
		{Command: "", Label: "Empty"},
	}}.effectivePanes()
	if labeled[0].Label != "Server" {
		t.Fatalf("pane 0 label = %q, want Server", labeled[0].Label)
	}
	if labeled[1].Label != "Minio" {
		t.Fatalf("pane 1 label = %q, want Minio", labeled[1].Label)
	}
	if labeled[2].Label != "Empty" {
		t.Fatalf("pane 2 label = %q, want Empty", labeled[2].Label)
	}

	// The root pane has nothing to split off, so its ratio is dropped; later panes
	// keep theirs.
	sized := ProjectTab{Name: "editor", Panes: []ProjectPane{
		{Command: "nvim", Ratio: 0.4},
		{Command: "lazygit", Split: "right", Ratio: 0.3},
	}}.effectivePanes()
	if sized[0].Ratio != 0 {
		t.Fatalf("root pane ratio = %v, want 0", sized[0].Ratio)
	}
	if sized[1].Ratio != 0.3 {
		t.Fatalf("pane 2 ratio = %v, want 0.3", sized[1].Ratio)
	}
}

// TestSplitRatio confirms an authored ratio is flipped into the share herdr's
// pane.split keeps for the pane being split, and that an unset ratio stays zero
// so the request omits it.
func TestSplitRatio(t *testing.T) {
	cases := []struct {
		name string
		pane ProjectPane
		want float64
	}{
		{"unset", ProjectPane{}, 0},
		{"third", ProjectPane{Ratio: 0.25}, 0.75},
		{"half", ProjectPane{Ratio: 0.5}, 0.5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.pane.splitRatio(); got != c.want {
				t.Fatalf("splitRatio() = %v, want %v", got, c.want)
			}
		})
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
	// os.UserHomeDir reads USERPROFILE on Windows and HOME elsewhere; set both so
	// the ~ cases resolve to our temp home on every platform. HOME also feeds the
	// $HOME expansion case below.
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cases := []struct {
		in   string
		want string
	}{
		{"", home},
		{"~", home},
		{"~/code/x", filepath.Join(home, "code", "x")},
		{"$HOME/code/y", filepath.Join(home, "code", "y")},
		{"/srv/abs", filepath.Clean("/srv/abs")},
	}
	for _, c := range cases {
		got, err := Project{WorkingDir: c.in}.expandedWorkingDir()
		if err != nil {
			t.Fatalf("expandedWorkingDir(%q) returned error: %v", c.in, err)
		}
		// expandedWorkingDir normalizes separators (filepath.Clean); compare against
		// a cleaned want so the $VAR case matches on Windows too.
		if want := filepath.Clean(c.want); got != want {
			t.Fatalf("expandedWorkingDir(%q) = %q, want %q", c.in, got, want)
		}
	}
}

// TestEffectivePanesInheritsTabWorkingDir confirms the directory fallback inside a
// tab: a pane with no working_dir of its own takes the tab's, a pane that sets one
// keeps it, and the single-command shorthand carries the tab's directory too.
func TestEffectivePanesInheritsTabWorkingDir(t *testing.T) {
	split := ProjectTab{Name: "app", WorkingDir: "~/src/web", Panes: []ProjectPane{
		{Command: "npm run dev"},
		{Command: "npm test", Split: SplitDown},
		{Command: "psql", Split: SplitRight, WorkingDir: "~/src/db"},
	}}.effectivePanes()

	if split[0].WorkingDir != "~/src/web" || split[1].WorkingDir != "~/src/web" {
		t.Fatalf("panes 1-2 dirs = %q/%q, want the tab's ~/src/web", split[0].WorkingDir, split[1].WorkingDir)
	}
	if split[2].WorkingDir != "~/src/db" {
		t.Fatalf("pane 3 dir = %q, want its own ~/src/db", split[2].WorkingDir)
	}

	// The single-command shorthand is one pane, and it belongs in the tab's dir.
	shorthand := ProjectTab{Name: "api", Command: "make run", WorkingDir: "~/src/api"}.effectivePanes()
	if len(shorthand) != 1 || shorthand[0].WorkingDir != "~/src/api" {
		t.Fatalf("shorthand panes = %+v, want one pane in ~/src/api", shorthand)
	}

	// A tab that says nothing about directories still produces empty ones, so the
	// workspace's own directory is inherited exactly as before.
	plain := ProjectTab{Name: "shell"}.effectivePanes()
	if plain[0].WorkingDir != "" {
		t.Fatalf("plain tab pane dir = %q, want empty", plain[0].WorkingDir)
	}
}

// TestResolveNestedDir covers how a tab or pane's working_dir is resolved against
// the workspace root: an empty value stays empty (inherit), a relative one is
// joined onto the root so "web" means the repo's web/ directory, and an absolute
// or ~-rooted one is used as written.
func TestResolveNestedDir(t *testing.T) {
	root := t.TempDir()

	if got, err := resolveNestedDir("", root); err != nil || got != "" {
		t.Fatalf("resolveNestedDir(\"\") = %q, %v; want an empty dir and no error", got, err)
	}
	if got, err := resolveNestedDir("   ", root); err != nil || got != "" {
		t.Fatalf("resolveNestedDir(blank) = %q, %v; want an empty dir and no error", got, err)
	}

	got, err := resolveNestedDir("web", root)
	if err != nil {
		t.Fatalf("resolveNestedDir: %v", err)
	}
	if want := filepath.Join(root, "web"); got != want {
		t.Fatalf("relative dir = %q, want %q", got, want)
	}

	abs := filepath.Join(root, "api")
	if got, err := resolveNestedDir(abs, root); err != nil || got != abs {
		t.Fatalf("absolute dir = %q, %v; want %q unchanged", got, err, abs)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory on this machine")
	}
	if got, err := resolveNestedDir("~/src", root); err != nil || got != filepath.Join(home, "src") {
		t.Fatalf("~-rooted dir = %q, %v; want %q", got, err, filepath.Join(home, "src"))
	}
}

// TestResolvePaneDirs confirms the pre-flight pass layoutTabs runs before it
// touches a workspace: it resolves each pane's directory in the order the tabs are
// laid out, leaves an unset one empty so the workspace's is inherited, and refuses
// a directory that does not exist rather than building half a workspace.
func TestResolvePaneDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "web"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	tabs := []ProjectTab{
		{Name: "shell"},
		{Name: "app", WorkingDir: "web", Panes: []ProjectPane{{Command: "a"}, {Command: "b", Split: SplitDown}}},
	}
	dirs, err := resolvePaneDirs(root, tabs)
	if err != nil {
		t.Fatalf("resolvePaneDirs: %v", err)
	}
	if len(dirs) != 2 || len(dirs[0]) != 1 || len(dirs[1]) != 2 {
		t.Fatalf("dirs shape = %v, want one entry per pane per tab", dirs)
	}
	if dirs[0][0] != root {
		t.Fatalf("tab with no working_dir = %q, want the workspace root %q", dirs[0][0], root)
	}
	web := filepath.Join(root, "web")
	if dirs[1][0] != web || dirs[1][1] != web {
		t.Fatalf("tab dirs = %q/%q, want both %q", dirs[1][0], dirs[1][1], web)
	}

	// With no root to anchor to, an undeclared directory stays empty so herdr's own
	// inheritance keeps working — the behavior every config had before this key.
	rootless, err := resolvePaneDirs("", []ProjectTab{{Name: "shell"}})
	if err != nil {
		t.Fatalf("resolvePaneDirs with no root: %v", err)
	}
	if rootless[0][0] != "" {
		t.Fatalf("dir with no root = %q, want empty", rootless[0][0])
	}

	// A directory that is not there fails up front, naming the tab and pane.
	missing := []ProjectTab{{Name: "api", WorkingDir: "nope"}}
	if _, err := resolvePaneDirs(root, missing); err == nil {
		t.Fatal("expected a missing working_dir to be rejected")
	}

	// So does a path that exists but is a file rather than a directory.
	file := filepath.Join(root, "README")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := resolvePaneDirs(root, []ProjectTab{{Name: "api", WorkingDir: "README"}}); err == nil {
		t.Fatal("expected a file working_dir to be rejected")
	}
}
