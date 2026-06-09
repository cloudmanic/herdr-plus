//
// Date: 2026-06-09
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// keybinding is the slice of a herdr [[keys.command]] entry we care about for
// install: which key it binds and what command it runs.
type keybinding struct {
	Key     string
	Command string
}

// runInstallCmd wires herdr-plus into herdr's config.toml as a keybinding so a
// single keypress launches it. The bound command uses the absolute path of the
// running binary, so it works no matter where herdr-plus lives or what the
// current directory is. Re-running is safe: it detects an existing herdr-plus
// binding and refuses to clobber a key already taken by something else.
func runInstallCmd(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	key := fs.String("key", "prefix+down", "herdr keybinding to bind (e.g. prefix+down)")
	modeSlug := fs.String("mode", "", "which herdr-plus mode the keybinding launches (default: quick-actions)")
	_ = fs.Parse(args)

	mode, err := lookupMode(*modeSlug)
	if err != nil {
		errExit(err)
	}

	// The absolute path of this very binary — what the keybinding will run.
	self, err := selfBinaryPath()
	if err != nil {
		errExit(err)
	}

	cfgPath, err := herdrConfigPath()
	if err != nil {
		errExit(err)
	}

	command := shellQuote(self) + " --mode=" + mode.Slug
	description := "herdr-plus: " + mode.Slug

	// Read existing bindings (a missing file just means "none yet").
	existing, _ := readKeybindings(cfgPath)

	// Idempotent: if herdr-plus is already bound, report where and stop.
	if b, ok := existingBinding(existing, self); ok {
		if b.Key == *key {
			fmt.Printf("herdr-plus: already installed — %s launches herdr-plus.\n", b.Key)
		} else {
			fmt.Printf("herdr-plus: already installed at %s. Remove that binding in %s to rebind to %s.\n", b.Key, cfgPath, *key)
		}
		return
	}

	// Don't clobber a key someone else is using.
	if b, ok := conflictBinding(existing, *key, self); ok {
		errExit(fmt.Sprintf("key %q is already bound to: %s\nChoose a different key with --key (e.g. --key=prefix+up or --key=prefix+a).", *key, b.Command))
	}

	if err := appendToFile(cfgPath, keybindBlock(*key, command, description)); err != nil {
		errExit("could not write to", cfgPath+":", err)
	}
	fmt.Printf("herdr-plus: bound %s -> %s\n  in %s\n", *key, command, cfgPath)

	// Best effort: reload the running server so the binding is live immediately.
	if out, err := exec.Command("herdr", "server", "reload-config").CombinedOutput(); err != nil {
		fmt.Printf("herdr-plus: saved, but reload failed (%v). Run `herdr server reload-config` or restart herdr.\n", strings.TrimSpace(string(out)))
	} else {
		fmt.Printf("herdr-plus: reloaded herdr config. Press your prefix, then %s, to launch.\n", strings.TrimPrefix(*key, "prefix+"))
	}
}

// selfBinaryPath returns the absolute, symlink-resolved path of the running
// herdr-plus binary.
func selfBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// herdrConfigPath returns the path to herdr's config.toml, honoring
// $XDG_CONFIG_HOME and otherwise using ~/.config/herdr/config.toml.
func herdrConfigPath() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "herdr", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "herdr", "config.toml"), nil
}

// readKeybindings parses the [[keys.command]] entries from herdr's config.toml.
func readKeybindings(path string) ([]keybinding, error) {
	var cfg struct {
		Keys struct {
			Command []struct {
				Key     string `toml:"key"`
				Command string `toml:"command"`
			} `toml:"command"`
		} `toml:"keys"`
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	out := make([]keybinding, 0, len(cfg.Keys.Command))
	for _, c := range cfg.Keys.Command {
		out = append(out, keybinding{Key: c.Key, Command: c.Command})
	}
	return out, nil
}

// existingBinding finds the binding that already launches the given binary path,
// identified by the absolute path appearing anywhere in its command.
func existingBinding(bindings []keybinding, selfPath string) (keybinding, bool) {
	for _, b := range bindings {
		if strings.Contains(b.Command, selfPath) {
			return b, true
		}
	}
	return keybinding{}, false
}

// conflictBinding finds a binding that occupies key but is not our own
// herdr-plus binding.
func conflictBinding(bindings []keybinding, key, selfPath string) (keybinding, bool) {
	for _, b := range bindings {
		if b.Key == key && !strings.Contains(b.Command, selfPath) {
			return b, true
		}
	}
	return keybinding{}, false
}

// keybindBlock renders a herdr [[keys.command]] block for appending. The %q
// verb produces double-quoted strings whose escaping is compatible with TOML
// basic strings.
func keybindBlock(key, command, description string) string {
	return fmt.Sprintf(
		"\n# herdr-plus — added by `herdr-plus install`\n[[keys.command]]\nkey = %q\ntype = \"shell\"\ncommand = %q\ndescription = %q\n",
		key, command, description,
	)
}

// appendToFile appends text to a file, creating it (and its directory) if
// needed. When the path is a symlink, the write follows it to the target.
func appendToFile(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(text)
	return err
}
