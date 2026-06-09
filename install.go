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

// runInstallCmd wires herdr-plus into herdr's config.toml as one or more
// keybindings so a single keypress launches it. The bound command uses the
// absolute path of the running binary, so it works no matter where herdr-plus
// lives or what the current directory is. Re-running is safe: it detects an
// existing herdr-plus binding and refuses to clobber a key already taken by
// something else.
//
// With neither --mode nor --key, it installs EVERY mode on its own default key
// (control → prefix+up, quick-actions → prefix+down) — the common case where a
// bare `herdr-plus install` should wire up the whole add-on in one shot. An
// explicit --mode (optionally with --key) installs just that single mode.
func runInstallCmd(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	key := fs.String("key", "", "herdr keybinding to bind (default: the mode's own key, e.g. prefix+up for control)")
	modeSlug := fs.String("mode", "", "which herdr-plus mode the keybinding launches (default: install every mode on its own key)")
	_ = fs.Parse(args)

	// The absolute path of this very binary — what every keybinding will run.
	self, err := selfBinaryPath()
	if err != nil {
		errExit(err)
	}

	cfgPath, err := herdrConfigPath()
	if err != nil {
		errExit(err)
	}

	// Bare install: bind each mode to its own default key in one pass, then
	// reload once. A conflict on one mode is reported but does not stop the
	// others, so installing both is as forgiving as possible.
	if *modeSlug == "" && *key == "" {
		wroteAny := false
		for _, m := range orderedModes {
			if wrote, _ := installMode(m, m.DefaultKey, self, cfgPath); wrote {
				wroteAny = true
			}
		}
		if wroteAny {
			reloadHerdrConfig("")
		}
		return
	}

	// Single-mode install: resolve the requested mode and bind it on its own
	// default key unless --key overrides.
	mode, err := lookupMode(*modeSlug)
	if err != nil {
		errExit(err)
	}
	k := *key
	if k == "" {
		k = mode.DefaultKey
	}

	wrote, conflict := installMode(mode, k, self, cfgPath)
	if conflict {
		// installMode already explained the clash on stderr; exit non-zero so
		// scripts notice the explicit install did not take.
		os.Exit(1)
	}
	if wrote {
		reloadHerdrConfig(k)
	}
}

// installMode binds a single herdr-plus mode to key in herdr's config.toml. It
// is idempotent per mode (each mode's command carries its own --mode, so two
// modes never match each other) and never clobbers a key already taken by
// something else. It returns wrote=true when it actually appended a new binding
// — telling the caller the config needs a reload — and conflict=true when key
// was already occupied by an unrelated command. It reads the file fresh on each
// call, so looping over modes correctly sees bindings written earlier in the
// same run.
func installMode(mode Mode, key, self, cfgPath string) (wrote bool, conflict bool) {
	command := shellQuote(self) + " --mode=" + mode.Slug
	description := "herdr-plus: " + mode.Slug

	// Read existing bindings (a missing file just means "none yet").
	existing, _ := readKeybindings(cfgPath)

	// Idempotent per mode: if THIS mode is already bound, report where and stop.
	if b, ok := existingBinding(existing, command); ok {
		if b.Key == key {
			fmt.Printf("herdr-plus: %s already installed — press %s to launch it.\n", mode.Slug, b.Key)
		} else {
			fmt.Printf("herdr-plus: %s already installed at %s. Remove that binding in %s to rebind to %s.\n", mode.Slug, b.Key, cfgPath, key)
		}
		return false, false
	}

	// Don't clobber a key already used by anything else (including the other mode).
	if b, ok := conflictBinding(existing, key, command); ok {
		fmt.Fprintf(os.Stderr, "herdr-plus: %s not installed: key %q is already bound to: %s\n  Choose a different key with --key (e.g. --key=prefix+a).\n", mode.Slug, key, b.Command)
		return false, true
	}

	if err := appendToFile(cfgPath, keybindBlock(key, command, description)); err != nil {
		errExit("could not write to", cfgPath+":", err)
	}
	fmt.Printf("herdr-plus: bound %s -> %s\n  in %s\n", key, command, cfgPath)
	return true, false
}

// reloadHerdrConfig asks the running herdr server to reload its config so any
// freshly added binding is live without a restart. It is best effort: a failure
// just prints how to reload manually. keyHint, when non-empty, names the single
// key to press in the success message; empty means several were bound, so the
// message stays generic.
func reloadHerdrConfig(keyHint string) {
	if out, err := exec.Command("herdr", "server", "reload-config").CombinedOutput(); err != nil {
		fmt.Printf("herdr-plus: saved, but reload failed (%v). Run `herdr server reload-config` or restart herdr.\n", strings.TrimSpace(string(out)))
		return
	}
	if keyHint != "" {
		fmt.Printf("herdr-plus: reloaded herdr config. Press your prefix, then %s, to launch.\n", strings.TrimPrefix(keyHint, "prefix+"))
	} else {
		fmt.Println("herdr-plus: reloaded herdr config. Press your prefix, then a bound key (e.g. up or down), to launch.")
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

// existingBinding finds a binding that already runs this exact herdr-plus
// command — same binary and same --mode. Because each mode's command carries its
// own --mode flag, two herdr-plus modes never match each other here, so
// installing one mode is idempotent without disturbing the other.
func existingBinding(bindings []keybinding, command string) (keybinding, bool) {
	for _, b := range bindings {
		if b.Command == command {
			return b, true
		}
	}
	return keybinding{}, false
}

// conflictBinding finds a binding that occupies key with a different command —
// anything other than this mode's own binding (including herdr-plus's other
// mode). That is a real conflict we must not clobber.
func conflictBinding(bindings []keybinding, key, command string) (keybinding, bool) {
	for _, b := range bindings {
		if b.Key == key && b.Command != command {
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
