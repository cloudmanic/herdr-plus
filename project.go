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
// defaults to "down" when omitted. Ratio is the share of the split this pane
// takes, between 0 and 1; omitting it splits the space evenly, and it is ignored
// for the first pane, which has nothing to split off.
type ProjectPane struct {
	Command string  `toml:"command"`
	Split   string  `toml:"split"`
	Label   string  `toml:"label"`
	Ratio   float64 `toml:"ratio"`
}

// splitRatio translates the pane's authored ratio — the share of the split it
// takes — into the ratio herdr's pane.split expects, which is the share kept by
// the pane being split. A zero ratio means the pane did not ask for one, and is
// passed through as zero so the caller can leave it out of the request.
func (p ProjectPane) splitRatio() float64 {
	if p.Ratio <= 0 {
		return 0
	}
	return 1 - p.Ratio
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
// authoring forms into one list. The first pane is the tab's root (its split and
// ratio are cleared); each later pane carries the direction it splits off the
// previous one, defaulting to "down".
func (t ProjectTab) effectivePanes() []ProjectPane {
	if len(t.Panes) == 0 {
		return []ProjectPane{{Command: t.Command}}
	}
	panes := make([]ProjectPane, len(t.Panes))
	for i, p := range t.Panes {
		panes[i] = p
		if i == 0 {
			panes[i].Split = ""
			panes[i].Ratio = 0
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
// split is "down" or "right" with a ratio inside 0..1. label and source identify
// the owning config in error messages — a project's name or a layout's repo, and
// the file it came from.
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
			// A zero ratio is an omitted one — an even split. herdr silently clamps
			// anything outside its own range, so a typo like ratio = 30 would open a
			// workspace that looks nothing like the config; catch it at load time.
			if pane.Ratio < 0 || pane.Ratio >= 1 {
				return fmt.Errorf("%q (%s): tab %q pane %d has ratio %v; must be greater than 0 and less than 1", label, source, t.Name, j+1, pane.Ratio)
			}
		}
	}
	return nil
}

// promptDirSentinel is the working_dir value that makes a project ask for its
// directory when opened instead of pinning one in the TOML. One such project is
// a universal template: pick it, type a path, and any repo opens with the
// project's tabs.
const promptDirSentinel = "{prompt}"

// promptsForDir reports whether this project asks for its working directory at
// open time (working_dir = "{prompt}").
func (p Project) promptsForDir() bool {
	return strings.TrimSpace(p.WorkingDir) == promptDirSentinel
}

// expandedWorkingDir resolves the project's working directory to an absolute
// path, expanding a leading ~ to the home directory and any $VARS in the path.
// An empty working_dir defaults to the user's home directory, so a minimal
// project still opens somewhere sensible.
func (p Project) expandedWorkingDir() (string, error) {
	dir, err := expandPath(p.WorkingDir)
	if err != nil {
		return "", fmt.Errorf("resolve home directory for working_dir %q: %w", p.WorkingDir, err)
	}
	return dir, nil
}

// expandPath resolves a user-authored path the way working_dir is resolved: a
// leading ~ becomes the home directory, $VAR / ${VAR} expand (not Windows
// %VAR%), and an empty value defaults to home. Shared with the picker's path
// prompt so a typed path and a configured one behave identically.
//
// Home comes from os.UserHomeDir (USERPROFILE on Windows, HOME elsewhere), and
// the result is filepath.Clean'd so a path written with forward slashes lands
// on native separators — both so this works on Windows.
func expandPath(s string) (string, error) {
	dir := strings.TrimSpace(s)

	home, err := os.UserHomeDir()
	needsHome := dir == "" || dir == "~" || strings.HasPrefix(dir, "~/")
	if err != nil && needsHome {
		// Only a fatal problem when the path actually references home; an absolute
		// or relative path resolves fine without it.
		return "", err
	}

	if dir == "" || dir == "~" {
		return home, nil
	}
	if strings.HasPrefix(dir, "~/") {
		dir = filepath.Join(home, dir[2:])
	}
	return filepath.Clean(os.ExpandEnv(dir)), nil
}

// displayWorkingDir resolves the working directory for UI contexts that have no
// way to surface an error (the picker's detail bar). On failure it falls back to
// the raw working_dir so the picker still renders something rather than breaking.
func (p Project) displayWorkingDir() string {
	dir, err := p.expandedWorkingDir()
	if err != nil {
		return strings.TrimSpace(p.WorkingDir)
	}
	return dir
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
