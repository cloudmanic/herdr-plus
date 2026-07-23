//
// Date: 2026-07-23
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runOpen implements the headless `open <name>` subcommand: it opens a project's
// workspace by name, skipping the interactive fuzzy picker entirely. It loads the
// same project templates the picker uses, resolves the one whose name matches the
// argument, and hands it to openProject — the exact code path the picker runs when
// a project is chosen — so there is one source of truth for turning a template
// into a live workspace. This is the scriptable entry point a shell alias, a
// keybinding, another tool, or an AI agent can call to spin up a known workspace
// in a single shot. Like every herdr-plus command it reaches the running herdr
// over HERDR_SOCKET_PATH, so it must run inside herdr.
func runOpen(args []string) {
	if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
		errExit("usage: herdr-plus open <project-name>")
	}
	name := strings.TrimSpace(args[0])

	// Point config resolution at herdr's managed dir before loading templates —
	// a direct invocation has no HERDR_PLUGIN_CONFIG_DIR injected for us.
	ensureManagedConfigDir()

	// Load every project template, surfacing any config error loudly.
	projects, err := loadProjects()
	if err != nil {
		errExit(err)
	}

	// Resolve the requested name to a single project (or a helpful error).
	p, err := findProject(projects, name)
	if err != nil {
		errExit(err)
	}

	// Connect to the running herdr instance over its socket.
	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}

	// Build the live workspace via the shared picker code path.
	if err := openProject(client, p); err != nil {
		errExit("could not open project:", err)
	}

	fmt.Printf("opened %s\n", p.Name)
}

// findProject resolves a project by name within the loaded set. An exact,
// case-sensitive match wins; failing that, a case-insensitive match is accepted so
// `open Harbor-Sysadmin` still finds "harbor-sysadmin". When nothing matches it
// returns an error naming the miss and listing the available projects (or noting
// that none are defined), so a mistyped name fails loudly with a usable hint
// rather than silently doing nothing.
func findProject(projects []Project, name string) (Project, error) {
	// Prefer an exact match so names that differ only by case stay distinct.
	for _, p := range projects {
		if p.Name == name {
			return p, nil
		}
	}

	// Fall back to a case-insensitive match for CLI friendliness.
	for _, p := range projects {
		if strings.EqualFold(p.Name, name) {
			return p, nil
		}
	}

	if len(projects) == 0 {
		return Project{}, fmt.Errorf("no project named %q: no projects are defined", name)
	}

	names := make([]string, len(projects))
	for i, p := range projects {
		names[i] = p.Name
	}
	return Project{}, fmt.Errorf("no project named %q; available: %s", name, strings.Join(names, ", "))
}

// ensureManagedConfigDir makes HERDR_PLUGIN_CONFIG_DIR available to a headless
// command. herdr injects that variable whenever it runs one of our plugin commands
// (the picker path), pointing config resolution at the managed directory herdr
// stores our projects/ in. A direct invocation like `herdr-plus open <name>` from
// a shell has no such injection, so configBaseDir would fall back to the standalone
// ~/.config/herdr-plus and miss those projects. Here we ask the running herdr for
// the managed directory and export it ourselves — doing exactly what herdr would
// have done — so loadProjects reads the very same templates the picker would.
//
// Best effort: an already-set variable is left untouched, and if herdr cannot
// answer (not running, not found) we leave it unset and config keeps its standalone
// fallback. Setting a process env var here is safe: this is a short-lived one-shot
// command, and openProject drives herdr over the socket rather than spawning
// children that would inherit it.
func ensureManagedConfigDir() {
	if os.Getenv("HERDR_PLUGIN_CONFIG_DIR") != "" {
		return
	}
	if dir := herdrManagedConfigDir(); dir != "" {
		os.Setenv("HERDR_PLUGIN_CONFIG_DIR", dir)
	}
}

// herdrManagedConfigDir asks the running herdr for the directory it manages our
// plugin config in, via `herdr plugin config-dir <id>`. herdr owns that path, so
// querying it is more robust than reconstructing herdr's directory layout here. It
// returns "" when herdr is unreachable or the command fails. The herdr binary is
// located via HERDR_BIN_PATH (set for plugin commands), falling back to "herdr" on
// PATH.
func herdrManagedConfigDir() string {
	herdr := os.Getenv("HERDR_BIN_PATH")
	if herdr == "" {
		herdr = "herdr"
	}
	out, err := exec.Command(herdr, "plugin", "config-dir", pluginID).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
