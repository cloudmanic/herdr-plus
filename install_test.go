//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

// TestKeybindBlockIsValidToml confirms the generated block parses as TOML and
// carries the expected key, type, command, and description.
func TestKeybindBlockIsValidToml(t *testing.T) {
	block := keybindBlock("prefix+down", "'/abs/herdr-plus' --mode=quick-actions", "herdr-plus: quick-actions")

	var cfg struct {
		Keys struct {
			Command []struct {
				Key         string `toml:"key"`
				Type        string `toml:"type"`
				Command     string `toml:"command"`
				Description string `toml:"description"`
			} `toml:"command"`
		} `toml:"keys"`
	}
	if _, err := toml.Decode(block, &cfg); err != nil {
		t.Fatalf("block is not valid TOML: %v\n%s", err, block)
	}
	if len(cfg.Keys.Command) != 1 {
		t.Fatalf("got %d command entries, want 1", len(cfg.Keys.Command))
	}
	c := cfg.Keys.Command[0]
	if c.Key != "prefix+down" || c.Type != "shell" {
		t.Fatalf("key/type = %q/%q", c.Key, c.Type)
	}
	if c.Command != "'/abs/herdr-plus' --mode=quick-actions" {
		t.Fatalf("command = %q", c.Command)
	}
	if c.Description != "herdr-plus: quick-actions" {
		t.Fatalf("description = %q", c.Description)
	}
}

// TestReadKeybindings confirms existing bindings are parsed out of a config.
func TestReadKeybindings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[[keys.command]]
key = "prefix+u"
type = "shell"
command = "$HOME/bin/setup.sh"

[[keys.command]]
key = "prefix+down"
type = "shell"
command = "/abs/herdr-plus --mode=quick-actions"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := readKeybindings(path)
	if err != nil {
		t.Fatalf("readKeybindings: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d bindings, want 2", len(got))
	}
	if got[1].Key != "prefix+down" || got[1].Command != "/abs/herdr-plus --mode=quick-actions" {
		t.Fatalf("binding[1] = %+v", got[1])
	}
}

// TestExistingAndConflictBinding checks the per-mode idempotency and conflict
// detection used by install: a binding matches only when both the binary and the
// --mode match, so the two modes can be installed side by side.
func TestExistingAndConflictBinding(t *testing.T) {
	quickCmd := "/abs/herdr-plus --mode=quick-actions"
	controlCmd := "/abs/herdr-plus --mode=control"
	bindings := []keybinding{
		{Key: "prefix+u", Command: "$HOME/bin/setup.sh"},
		{Key: "prefix+down", Command: quickCmd},
	}

	// The quick-actions binding is found by its exact command.
	if b, ok := existingBinding(bindings, quickCmd); !ok || b.Key != "prefix+down" {
		t.Fatalf("existingBinding(quick-actions) = %+v, %v; want prefix+down", b, ok)
	}

	// Control mode is NOT installed yet: same binary, different --mode.
	if b, ok := existingBinding(bindings, controlCmd); ok {
		t.Fatalf("existingBinding(control) = %+v; want not found (different mode)", b)
	}

	// Installing control on prefix+down would clobber quick-actions -> conflict.
	if b, ok := conflictBinding(bindings, "prefix+down", controlCmd); !ok || b.Command != quickCmd {
		t.Fatalf("conflictBinding(prefix+down, control) = %+v, %v; want the quick-actions conflict", b, ok)
	}

	// prefix+down holds quick-actions' own command, so it is not a self-conflict.
	if _, ok := conflictBinding(bindings, "prefix+down", quickCmd); ok {
		t.Fatal("conflictBinding(prefix+down, quick-actions) reported a conflict for its own binding")
	}

	// A free key (prefix+up) has no conflict for control.
	if _, ok := conflictBinding(bindings, "prefix+up", controlCmd); ok {
		t.Fatal("conflictBinding(prefix+up, control) reported a conflict for a free key")
	}
}

// TestAppendToFileFollowsSymlink confirms appending through a symlink writes to
// the link target (herdr's config.toml is a dotfiles symlink).
func TestAppendToFileFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.toml")
	link := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(target, []byte("onboarding = false\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if err := appendToFile(link, "\nADDED\n"); err != nil {
		t.Fatalf("appendToFile: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !filepath.IsAbs(target) || string(data) != "onboarding = false\n\nADDED\n" {
		t.Fatalf("target content = %q", string(data))
	}
}
