//
// Date: 2026-06-15
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

// Split directions herdr's pane.split understands. "down" stacks the new pane
// below the previous one (top/bottom); "right" puts it beside (side by side).
const (
	SplitDown  = "down"
	SplitRight = "right"
)

// maxPanesPerTab caps how many panes a single tab may declare. A tab is split
// into at most this many panes.
const maxPanesPerTab = 4

// ProjectPane is one pane within a tab. Command, when set, runs in the pane on
// startup. Split is how the pane is created relative to the previous pane in the
// tab — "down" or "right"; it is ignored for the first pane (the tab's root) and
// defaults to "down" when omitted.
type ProjectPane struct {
	Command string `toml:"command"`
	Split   string `toml:"split"`
}

// ProjectTab is one tab in a project's workspace, in the order it should be
// created. Name is the tab's label. A tab is authored in one of two forms: the
// single-pane shorthand sets Command directly; a split tab instead lists
// [[tabs.panes]] (up to maxPanesPerTab of them). A tab with neither is an empty
// terminal.
type ProjectTab struct {
	Name    string        `toml:"name"`
	Command string        `toml:"command"`
	Panes   []ProjectPane `toml:"panes"`
}

// effectivePanes returns the tab's panes in creation order, normalizing the two
// authoring forms into one list. The first pane is the tab's root (its split is
// cleared); each later pane carries the direction it splits off the previous
// one, defaulting to "down".
func (t ProjectTab) effectivePanes() []ProjectPane {
	if len(t.Panes) == 0 {
		return []ProjectPane{{Command: t.Command}}
	}
	panes := make([]ProjectPane, len(t.Panes))
	for i, p := range t.Panes {
		panes[i] = p
		if i == 0 {
			panes[i].Split = ""
			continue
		}
		if panes[i].Split == "" {
			panes[i].Split = SplitDown
		}
	}
	return panes
}

// Project is a declarative herdr workspace template, loaded from one TOML file in
// ~/.config/herdr-plus/projects. Opening a project creates a new herdr workspace
// rooted at WorkingDir, labeled Name, with one tab per entry in Tabs (in order)
// running each tab's startup command. Projects replace hand-written
// herdr-workspace shell scripts with simple config files.
type Project struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`

	// Group is an optional label that clusters projects in the browser. Projects
	// sharing a Group are shown together under a heading — for example, every
	// project belonging to one client. It is purely a browsing aid and has no
	// effect on the workspace that opens. Leaving it empty drops the project into
	// the catch-all "Ungrouped" heading when any other project sets a Group; when
	// no project sets one, the browser stays a plain, heading-less list.
	Group string `toml:"group"`

	WorkingDir string       `toml:"working_dir"`
	Tabs       []ProjectTab `toml:"tabs"`

	// source is the file the project was loaded from, used only for error
	// messages. It is not part of the on-disk format.
	source string
}

// projectsConfigDir returns the directory that holds project files,
// ~/.config/herdr-plus/projects. It hangs directly off the herdr-plus config
// root because projects are a first-class concept.
func projectsConfigDir() (string, error) {
	base, err := configBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "projects"), nil
}

// ensureProjectsDir makes sure the projects directory exists and returns its
// path. Unlike a config that ships examples, it is never seeded: an empty
// directory is meaningful — it triggers the projects browser's onboarding
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
// directory, returning them sorted by name. A malformed or invalid file fails the
// whole load with a message naming the offending files, so config mistakes
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
	return validateTabs(p.Name, p.source, p.Tabs)
}

// validateTabs checks the per-tab rules shared by projects and worktree layouts:
// every tab needs a name; a tab uses either a single command or [[tabs.panes]],
// never both; a tab holds at most maxPanesPerTab panes; and every non-root pane's
// split is "down" or "right". label and source identify the owning config in
// error messages — a project's name or a layout's repo, and the file it came from.
func validateTabs(label, source string, tabs []ProjectTab) error {
	for i, t := range tabs {
		if strings.TrimSpace(t.Name) == "" {
			return fmt.Errorf("%q (%s): tab %d is missing a name", label, source, i+1)
		}
		if len(t.Panes) > 0 && strings.TrimSpace(t.Command) != "" {
			return fmt.Errorf("%q (%s): tab %q sets both command and [[tabs.panes]]; use one or the other", label, source, t.Name)
		}
		if len(t.Panes) > maxPanesPerTab {
			return fmt.Errorf("%q (%s): tab %q has %d panes; at most %d are allowed", label, source, t.Name, len(t.Panes), maxPanesPerTab)
		}
		for j, pane := range t.Panes {
			if j == 0 {
				continue // the first pane is the tab's root; its split is ignored
			}
			switch pane.Split {
			case "", SplitDown, SplitRight:
				// ok — an empty split defaults to "down"
			default:
				return fmt.Errorf("%q (%s): tab %q pane %d has split %q; must be %q or %q", label, source, t.Name, j+1, pane.Split, SplitDown, SplitRight)
			}
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

// tabLabels returns the tab names in order for the browser's detail bar,
// annotating split tabs with a "×N" pane count so the layout is visible at a
// glance (e.g. "server ×2").
func (p Project) tabLabels() []string {
	labels := make([]string, len(p.Tabs))
	for i, t := range p.Tabs {
		if n := len(t.effectivePanes()); n > 1 {
			labels[i] = fmt.Sprintf("%s ×%d", t.Name, n)
		} else {
			labels[i] = t.Name
		}
	}
	return labels
}
