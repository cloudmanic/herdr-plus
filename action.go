//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

// Action types. An action's Type decides what happens between selecting it in
// the picker and running its command.
const (
	// TypeCommand runs the command immediately with no further input.
	TypeCommand = "command"
	// TypeSelect shows a list of Options; the chosen option's Value becomes the
	// action's Value (available to the command as {{.Value}}).
	TypeSelect = "select"
	// TypeForm shows a single text field; the entered string becomes the action's
	// Value.
	TypeForm = "form"
)

// Option is one choice in a "select" action. Label is what the user sees in the
// list; Value is what gets handed to the command. When Value is empty the Label
// is used as the value too. Description, when set, is shown as dim text next to
// the label — useful when the label alone isn't self-explanatory, or to keep a
// long value out of the list.
//
// An option with no Label is a non-selectable separator used to visually group
// the choices: with a Heading it renders as a dim group title (preceded by a
// blank line); without one it renders as a plain blank spacer.
type Option struct {
	Label       string `toml:"label"`
	Value       string `toml:"value"`
	Description string `toml:"description"`
	Heading     string `toml:"heading"`
}

// isSeparator reports whether the option is a non-selectable spacer/heading
// rather than a real choice. An option needs a label to be selectable.
func (o Option) isSeparator() bool {
	return o.Label == ""
}

// resolvedValue returns the value to hand to the command for this option.
func (o Option) resolvedValue() string {
	if o.Value != "" {
		return o.Value
	}
	return o.Label
}

// actionOrigin records where an action was loaded from. The picker uses it to
// group project-local actions separately from the user's global actions so it is
// always clear which is which.
type actionOrigin int

const (
	// originGlobal is an action from the user's global config dir
	// (~/.config/herdr-plus/quick-actions/). It is the zero value, so any Action
	// built without an explicit origin is treated as global.
	originGlobal actionOrigin = iota
	// originProject is an action from a repo's own .herdr-plus/quick-actions/
	// directory, available only when herdr-plus is launched from inside that repo.
	originProject
)

// FormConfig customizes the text field shown for a "form" action.
type FormConfig struct {
	// Prompt is the label rendered above the input field.
	Prompt string `toml:"prompt"`
	// Placeholder is the greyed-out hint shown in the empty field.
	Placeholder string `toml:"placeholder"`
}

// Action is one entry in the quick-action picker, loaded from a TOML file in the
// quick-actions config directory. Name and Description are shown in the list;
// Command is the shell command run when the action completes. Command is a Go
// text/template rendered against a RunContext, so it can reference {{.Value}},
// {{.WorkDir}}, {{.SessionTitle}}, and the other context fields.
type Action struct {
	Name        string     `toml:"name"`
	Description string     `toml:"description"`
	Type        string     `toml:"type"`
	Command     string     `toml:"command"`
	Options     []Option   `toml:"options"`
	Form        FormConfig `toml:"form"`

	// source is the file the action was loaded from, used only for error
	// messages. It is not part of the on-disk format.
	source string

	// origin marks whether the action came from the global config or a project's
	// .herdr-plus directory. Set at load time; not part of the on-disk format.
	origin actionOrigin
}

// effectiveType returns the action's type, defaulting to TypeCommand when the
// file omits it so the simplest actions need only a name and a command.
func (a Action) effectiveType() string {
	if a.Type == "" {
		return TypeCommand
	}
	return a.Type
}

// validate checks that an action is internally consistent before we ever try to
// run it, turning config mistakes into clear errors at load time.
func (a Action) validate() error {
	if a.Name == "" {
		return fmt.Errorf("action %s: name is required", a.source)
	}
	if strings.TrimSpace(a.Command) == "" {
		return fmt.Errorf("action %q (%s): command is required", a.Name, a.source)
	}
	switch a.effectiveType() {
	case TypeCommand, TypeForm:
		// nothing extra to check
	case TypeSelect:
		selectable := 0
		for _, o := range a.Options {
			if !o.isSeparator() {
				selectable++
			}
		}
		if selectable == 0 {
			return fmt.Errorf("action %q (%s): select actions need at least one selectable option (one with a label)", a.Name, a.source)
		}
	default:
		return fmt.Errorf("action %q (%s): unknown type %q (want command, select, or form)", a.Name, a.source, a.Type)
	}
	return nil
}

// render turns the action's command template into the final shell command. The
// chosen Value (option value or form input) is placed in ctx.Value. When the
// template does not explicitly reference .Value but a value is present, the
// value is appended as a single shell-quoted argument — so a command can either
// position the value precisely with {{.Value}} or just receive it as its last
// argument.
func (a Action) render(ctx RunContext) (string, error) {
	tmpl, err := template.New(a.Name).Parse(a.Command)
	if err != nil {
		return "", fmt.Errorf("parse command for %q: %w", a.Name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("render command for %q: %w", a.Name, err)
	}
	cmdline := buf.String()

	if ctx.Value != "" && !strings.Contains(a.Command, ".Value") {
		cmdline += " " + shellQuote(ctx.Value)
	}
	return cmdline, nil
}

// run renders and executes the action's command through the shell, in the
// invoking pane's working directory and with the context exported as
// HERDR_PLUS_* environment variables. Running through "sh -c" lets commands use
// pipes, arguments, and full scripts.
func (a Action) run(ctx RunContext) error {
	cmdline, err := a.render(ctx)
	if err != nil {
		return err
	}

	cmd := exec.Command("sh", "-c", cmdline)
	if ctx.WorkDir != "" {
		cmd.Dir = ctx.WorkDir
	}
	cmd.Env = append(os.Environ(), ctx.envPairs()...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
