//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

// loadTitleTestConfig uses only a temporary plugin config directory. An empty
// input leaves config.toml absent, exercising the normal default path.
func loadTitleTestConfig(t *testing.T, contents string) PluginConfig {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", dir)
	if contents != "" {
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := loadPluginConfig()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestLoadPluginConfigTerminalTitle(t *testing.T) {
	for _, tc := range []struct {
		name, contents string
		want           bool
	}{
		{"missing file", "", false},
		{"missing option", "[worktree]\nbranch_prefix = 'dev/'\n", false},
		{"explicit false", "[ui]\nterminal_title = false\n", false},
		{"opt in", "[ui]\nterminal_title = true\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadTitleTestConfig(t, tc.contents)
			if cfg.UI.TerminalTitle != tc.want {
				t.Fatalf("TerminalTitle = %v, want %v", cfg.UI.TerminalTitle, tc.want)
			}
			if got := configuredTitleStyle(cfg); !tc.want && (got.GetForeground() != titleStyle.GetForeground() || got.GetBackground() != titleStyle.GetBackground() || got.GetReverse()) {
				t.Fatal("absent/false terminal_title must retain the original title style")
			}
		})
	}
}

func TestTerminalTitleConfigRoundTrip(t *testing.T) {
	cfg := loadTitleTestConfig(t, "[ui]\nterminal_title = true\n")
	var encoded bytes.Buffer
	if err := toml.NewEncoder(&encoded).Encode(cfg); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded.Bytes(), []byte("terminal_title = true")) {
		t.Fatalf("encoded config lost the option: %s", &encoded)
	}
	var decoded PluginConfig
	if _, err := toml.Decode(encoded.String(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != cfg {
		t.Fatalf("round trip = %+v, want %+v", decoded, cfg)
	}
}

func TestTerminalTitleConfigRejectsWrongType(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[ui]\nterminal_title = 'true'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPluginConfig(); err == nil {
		t.Fatal("expected a string terminal_title to fail decoding")
	}
}

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

// TestLoadPluginConfigParsesPlacements confirms config.toml can set the optional
// per-picker pane placement overrides.
func TestLoadPluginConfigParsesPlacements(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", dir)

	toml := "[projects]\nplacement = \"popup\"\n\n[quick_actions]\nplacement = \"split\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(toml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadPluginConfig()
	if err != nil {
		t.Fatalf("loadPluginConfig: %v", err)
	}
	if cfg.Projects.Placement != "popup" {
		t.Fatalf("Projects.Placement = %q, want popup", cfg.Projects.Placement)
	}
	if cfg.QuickActions.Placement != "split" {
		t.Fatalf("QuickActions.Placement = %q, want split", cfg.QuickActions.Placement)
	}
}

// TestResolvePlacement covers the fallback rules: empty config keeps the
// default, a valid override is used as-is, and an invalid value falls back
// rather than being passed through to herdr.
func TestResolvePlacement(t *testing.T) {
	cases := []struct {
		name       string
		configured string
		fallback   string
		want       string
	}{
		{"empty uses fallback", "", "overlay", "overlay"},
		{"valid override is used", "popup", "overlay", "popup"},
		{"another valid override is used", "zoomed", "overlay", "zoomed"},
		{"invalid value falls back", "not-a-placement", "zoomed", "zoomed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolvePlacement(c.configured, c.fallback); got != c.want {
				t.Fatalf("resolvePlacement(%q, %q) = %q, want %q", c.configured, c.fallback, got, c.want)
			}
		})
	}
}
