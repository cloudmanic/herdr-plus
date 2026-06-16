//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// embeddedExamples holds the starter files baked into the binary: the projects
// empty-state example and the quick-actions starter actions (seeded into the
// quick-actions config dir on first run). Keeping them embedded makes the repo's
// examples/ tree the single source of truth.
//
//go:embed examples
var embeddedExamples embed.FS

// configBaseDir returns the root configuration directory for herdr-plus's
// projects and quick-actions.
//
// When herdr runs us as a plugin it sets HERDR_PLUGIN_CONFIG_DIR to the standard,
// herdr-managed per-plugin config directory
// (~/.config/herdr/plugins/config/cloudmanic.herdr-plus). That is the canonical
// home for our config — herdr provisions it, isolates it per plugin, and keeps it
// across uninstall/upgrade — so we prefer it whenever it is set.
//
// Outside herdr (running the binary directly, dev, tests) the variable is unset,
// so we fall back to the legacy ~/.config/herdr-plus, honoring $XDG_CONFIG_HOME.
func configBaseDir() (string, error) {
	if d := os.Getenv("HERDR_PLUGIN_CONFIG_DIR"); d != "" {
		return d, nil
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "herdr-plus"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "herdr-plus"), nil
}

// quickActionsConfigDir returns the directory that holds quick-action files,
// ~/.config/herdr-plus/quick-actions.
func quickActionsConfigDir() (string, error) {
	base, err := configBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "quick-actions"), nil
}

// projectConfigDirName is the directory a repo adds at its root to ship its own
// herdr-plus config. Quick actions for a repo live in
// <repo>/.herdr-plus/quick-actions/.
const projectConfigDirName = ".herdr-plus"

// projectQuickActionsDir returns the project-local quick-actions directory within
// workDir, e.g. <workDir>/.herdr-plus/quick-actions. It returns "" when workDir
// is unknown. Unlike the global dir, it is never created or seeded: project
// config is opt-in and read only when the repo actually provides it.
func projectQuickActionsDir(workDir string) string {
	if workDir == "" {
		return ""
	}
	return filepath.Join(workDir, projectConfigDirName, "quick-actions")
}

// ensureQuickActionsConfig makes sure the quick-actions config directory exists
// and returns its path. The very first time (when the directory does not yet
// exist) it seeds the directory with the embedded example actions. Once the
// directory exists it is left untouched, so deleting an example never causes it
// to reappear.
func ensureQuickActionsConfig() (string, error) {
	dir, err := quickActionsConfigDir()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := seedQuickActionsExamples(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// seedQuickActionsExamples copies the embedded example actions into destDir.
func seedQuickActionsExamples(destDir string) error {
	srcDir := "examples/quick-actions"
	entries, err := embeddedExamples.ReadDir(srcDir)
	if err != nil {
		// No bundled examples; nothing to seed.
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := embeddedExamples.ReadFile(srcDir + "/" + e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destDir, e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// loadActions reads, parses, and validates every *.toml action in the global
// quick-actions config directory, returning them sorted by name and tagged as
// global. A malformed or invalid file fails the whole load with a message naming
// the offending files, so config mistakes surface loudly instead of an action
// silently going missing.
func loadActions() ([]Action, error) {
	dir, err := ensureQuickActionsConfig()
	if err != nil {
		return nil, err
	}
	return loadActionsFromDir(dir, originGlobal)
}

// loadActionsFromDir reads, parses, and validates every *.toml action in dir,
// tagging each with origin and returning them sorted by name. A directory that
// does not exist yields no actions and no error, so an absent project config dir
// simply contributes nothing. A malformed or invalid file fails the whole load
// with a message naming the offending files and their directory.
func loadActionsFromDir(dir string, origin actionOrigin) ([]Action, error) {
	if dir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var actions []Action
	var problems []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		path := filepath.Join(dir, e.Name())

		var a Action
		if _, err := toml.DecodeFile(path, &a); err != nil {
			problems = append(problems, fmt.Sprintf("  %s: %v", e.Name(), err))
			continue
		}
		a.source = e.Name()
		a.origin = origin
		if err := a.validate(); err != nil {
			problems = append(problems, "  "+err.Error())
			continue
		}
		actions = append(actions, a)
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid action files in %s:\n%s", dir, strings.Join(problems, "\n"))
	}

	sort.Slice(actions, func(i, j int) bool { return actions[i].Name < actions[j].Name })
	return actions, nil
}

// loadPickerActions loads the actions to show in the picker: the global actions
// plus any project-local actions found in workDir's .herdr-plus/quick-actions/
// directory. Project actions come first and are tagged originProject, globals
// originGlobal, so the picker can group them. A repo without a .herdr-plus dir
// just yields the global set.
func loadPickerActions(workDir string) ([]Action, error) {
	global, err := loadActions()
	if err != nil {
		return nil, err
	}

	project, err := loadActionsFromDir(projectQuickActionsDir(workDir), originProject)
	if err != nil {
		return nil, err
	}

	return append(project, global...), nil
}
