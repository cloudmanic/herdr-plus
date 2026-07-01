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

// TestConfigBaseDirPrefersManagedDir confirms herdr's managed plugin config
// directory (HERDR_PLUGIN_CONFIG_DIR) wins over the legacy location when set —
// the case that runs whenever herdr executes a plugin command.
func TestConfigBaseDirPrefersManagedDir(t *testing.T) {
	managed := filepath.Join(t.TempDir(), "herdr", "plugins", "config", "cloudmanic.herdr-plus")
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", managed)
	t.Setenv("XDG_CONFIG_HOME", "/tmp/should-be-ignored")

	got, err := configBaseDir()
	if err != nil {
		t.Fatalf("configBaseDir: %v", err)
	}
	if got != managed {
		t.Fatalf("configBaseDir = %q, want the managed dir %q", got, managed)
	}
}

// TestConfigBaseDirFallsBackToLegacy confirms that without the managed dir, config
// falls back to ~/.config/herdr-plus under $XDG_CONFIG_HOME (the path used when
// the binary runs outside herdr).
func TestConfigBaseDirFallsBackToLegacy(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got, err := configBaseDir()
	if err != nil {
		t.Fatalf("configBaseDir: %v", err)
	}
	if want := filepath.Join(xdg, "herdr-plus"); got != want {
		t.Fatalf("configBaseDir = %q, want %q", got, want)
	}
}

// TestLoadPluginConfigMissingFile confirms the optional global config is truly
// optional: no config.toml means zero values and no error.
func TestLoadPluginConfigMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", dir)

	cfg, err := loadPluginConfig()
	if err != nil {
		t.Fatalf("loadPluginConfig missing file: %v", err)
	}
	if cfg.Worktree.BranchPrefix != "" {
		t.Fatalf("BranchPrefix = %q, want empty", cfg.Worktree.BranchPrefix)
	}
}

// TestLoadPluginConfigParsesBranchPrefix confirms config.toml can set the
// optional worktree branch prefix.
func TestLoadPluginConfigParsesBranchPrefix(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", dir)

	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[worktree]\nbranch_prefix = \"dvic/\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadPluginConfig()
	if err != nil {
		t.Fatalf("loadPluginConfig: %v", err)
	}
	if cfg.Worktree.BranchPrefix != "dvic/" {
		t.Fatalf("BranchPrefix = %q, want dvic/", cfg.Worktree.BranchPrefix)
	}
}

// TestLoadPluginConfigRejectsMalformedFile confirms parse errors surface instead
// of being treated as a missing optional config.
func TestLoadPluginConfigRejectsMalformedFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", dir)

	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[worktree\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := loadPluginConfig(); err == nil {
		t.Fatal("expected malformed config.toml to error")
	}
}
