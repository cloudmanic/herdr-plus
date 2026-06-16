//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"embed"
	"os"
	"path/filepath"
)

// embeddedExamples holds the starter files baked into the binary. Today it backs
// the projects empty-state card (the example shown when no project files exist),
// keeping the repo's examples/ tree the single source of truth.
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
