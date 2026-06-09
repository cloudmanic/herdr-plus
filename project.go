//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// ProjectTab is one tab in a project's workspace, in the order it should be
// created. Name is the tab's label; Command, when set, is run in the tab's pane
// on startup (via the herdr socket, as if typed at the shell). A tab with no
// command is just an empty terminal.
type ProjectTab struct {
	Name    string `toml:"name"`
	Command string `toml:"command"`
}

// Project is a declarative herdr workspace template, loaded from one TOML file in
// ~/.config/herdr-plus/projects. Picking a project in control mode opens a new
// herdr workspace rooted at WorkingDir, labeled Name, with one tab per entry in
// Tabs (in order) running each tab's startup command. Projects replace the
// hand-written herdr-workspaces shell scripts.
type Project struct {
	Name        string       `toml:"name"`
	Description string       `toml:"description"`
	WorkingDir  string       `toml:"working_dir"`
	Tabs        []ProjectTab `toml:"tabs"`

	// source is the file the project was loaded from, used only for error
	// messages. It is not part of the on-disk format.
	source string
}

// projectsConfigDir returns the directory that holds project files,
// ~/.config/herdr-plus/projects. It hangs directly off the herdr-plus config
// root (not under a mode slug) because projects are a first-class concept that
// could one day be driven by more than one mode.
func projectsConfigDir() (string, error) {
	base, err := configBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "projects"), nil
}

// ensureProjectsDir makes sure the projects directory exists and returns its
// path. Unlike a mode's action directory, it is never seeded with examples: an
// empty directory is meaningful — it triggers control mode's onboarding
// empty-state — so we only create the (empty) folder for the user to drop files
// into.
func ensureProjectsDir() (string, error) {
	dir, err := projectsConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// loadProjects reads, parses, and validates every *.toml project in the projects
// directory, returning them sorted by name. A malformed or invalid file fails
// the whole load with a message naming the offending files, so config mistakes
// surface loudly instead of a project silently going missing. An empty directory
// returns an empty slice (not an error) so the caller can show the empty-state.
func loadProjects() ([]Project, error) {
	dir, err := ensureProjectsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var projects []Project
	var problems []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		path := filepath.Join(dir, e.Name())

		var p Project
		if _, err := toml.DecodeFile(path, &p); err != nil {
			problems = append(problems, fmt.Sprintf("  %s: %v", e.Name(), err))
			continue
		}
		p.source = e.Name()
		if err := p.validate(); err != nil {
			problems = append(problems, "  "+err.Error())
			continue
		}
		projects = append(projects, p)
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid project files in %s:\n%s", dir, strings.Join(problems, "\n"))
	}

	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
	return projects, nil
}

// validate checks that a project is internally consistent before we ever try to
// open it, turning config mistakes into clear errors at load time. The working
// directory is intentionally not checked for existence here — that is a per-open
// concern (the dir might exist on one machine but not another), reported when
// the project is actually opened.
func (p Project) validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("project %s: name is required", p.source)
	}
	if len(p.Tabs) == 0 {
		return fmt.Errorf("project %q (%s): needs at least one [[tabs]] entry", p.Name, p.source)
	}
	for i, t := range p.Tabs {
		if strings.TrimSpace(t.Name) == "" {
			return fmt.Errorf("project %q (%s): tab %d is missing a name", p.Name, p.source, i+1)
		}
	}
	return nil
}

// expandedWorkingDir resolves the project's working directory to an absolute
// path, expanding a leading ~ to the home directory and any $VARS in the path.
// An empty working_dir defaults to the user's home directory, so a minimal
// project still opens somewhere sensible.
func (p Project) expandedWorkingDir() string {
	dir := strings.TrimSpace(p.WorkingDir)
	home, _ := os.UserHomeDir()

	if dir == "" || dir == "~" {
		return home
	}
	if strings.HasPrefix(dir, "~/") {
		dir = filepath.Join(home, dir[2:])
	}
	return os.ExpandEnv(dir)
}

// tabNames returns just the tab labels in order, for the one-line layout summary
// shown in the control TUI's detail bar.
func (p Project) tabNames() []string {
	names := make([]string, len(p.Tabs))
	for i, t := range p.Tabs {
		names[i] = t.Name
	}
	return names
}
