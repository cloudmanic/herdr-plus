//
// Date: 2026-06-09
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

// embeddedExamples holds the starter action files baked into the binary. They
// are copied into a mode's config directory the first time herdr-plus runs that
// mode, giving the user a working set of actions to learn from and edit. Keeping
// them embedded means the repo's examples/ tree is the single source of truth.
//
//go:embed examples
var embeddedExamples embed.FS

// configBaseDir returns the root configuration directory, ~/.config/herdr-plus.
// It honors $XDG_CONFIG_HOME when set (the cross-platform convention) and
// otherwise falls back to ~/.config so the location is the same on macOS and
// Linux.
func configBaseDir() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "herdr-plus"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "herdr-plus"), nil
}

// modeConfigDir returns the per-mode subdirectory that holds a mode's action
// files, e.g. ~/.config/herdr-plus/quick-actions.
func modeConfigDir(mode Mode) (string, error) {
	base, err := configBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, mode.Slug), nil
}

// ensureModeConfig makes sure a mode's config directory exists and returns its
// path. The very first time (when the directory does not yet exist) it seeds the
// directory with the embedded example actions. Once the directory exists it is
// left untouched, so deleting an example never causes it to reappear.
func ensureModeConfig(mode Mode) (string, error) {
	dir, err := modeConfigDir(mode)
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
	if err := seedExamples(mode, dir); err != nil {
		return "", err
	}
	return dir, nil
}

// seedExamples copies the embedded example actions for a mode into destDir. A
// mode without bundled examples seeds nothing, which is fine.
func seedExamples(mode Mode, destDir string) error {
	srcDir := "examples/" + mode.Slug
	entries, err := embeddedExamples.ReadDir(srcDir)
	if err != nil {
		// No bundled examples for this mode; nothing to seed.
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

// loadActions reads, parses, and validates every *.toml action in the mode's
// config directory, returning them sorted by name. A malformed or invalid file
// fails the whole load with a message naming the offending files, so config
// mistakes surface loudly instead of an action silently going missing.
func loadActions(mode Mode) ([]Action, error) {
	dir, err := ensureModeConfig(mode)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
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
