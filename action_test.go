//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"strings"
	"testing"
)

// TestActionRender exercises command template rendering: explicit {{.Value}}
// placement, context variables, the urlquery helper, and the auto-append of a
// value when the template does not reference it.
func TestActionRender(t *testing.T) {
	cases := []struct {
		name    string
		action  Action
		ctx     RunContext
		want    string
		wantSub []string // substrings that must appear (when an exact match is brittle)
	}{
		{
			name:   "plain command unchanged",
			action: Action{Name: "GitHub", Command: "open https://github.com"},
			ctx:    RunContext{},
			want:   "open https://github.com",
		},
		{
			name:   "value substituted into template",
			action: Action{Name: "Repo", Type: TypeSelect, Command: "open https://github.com/cloudmanic/{{.Value}}"},
			ctx:    RunContext{Value: "herdr-plus"},
			want:   "open https://github.com/cloudmanic/herdr-plus",
		},
		{
			name:   "value appended when template omits it",
			action: Action{Name: "Say", Type: TypeForm, Command: "say"},
			ctx:    RunContext{Value: "hi there"},
			want:   "say 'hi there'",
		},
		{
			name:   "value with single quote is shell-safe when appended",
			action: Action{Name: "Say", Type: TypeForm, Command: "say"},
			ctx:    RunContext{Value: "it's me"},
			want:   `say 'it'\''s me'`,
		},
		{
			name:   "workdir variable",
			action: Action{Name: "Reveal", Command: "open {{.WorkDir}}"},
			ctx:    RunContext{WorkDir: "/tmp/project"},
			want:   "open /tmp/project",
		},
		{
			name:   "session title method",
			action: Action{Name: "Echo", Command: "echo {{.SessionTitle}}"},
			ctx:    RunContext{WorkspaceLabel: "herdr-plus"},
			want:   "echo herdr-plus",
		},
		{
			name:    "urlquery escapes spaces",
			action:  Action{Name: "Search", Type: TypeForm, Command: "open 'https://g.co/s?q={{.Value | urlquery}}'"},
			ctx:     RunContext{Value: "hello world"},
			wantSub: []string{"hello", "world"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.action.render(tc.ctx)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if tc.want != "" && got != tc.want {
				t.Fatalf("render = %q, want %q", got, tc.want)
			}
			for _, sub := range tc.wantSub {
				if !strings.Contains(got, sub) {
					t.Fatalf("render = %q, want substring %q", got, sub)
				}
			}
			if tc.name == "urlquery escapes spaces" && strings.Contains(got, "hello world") {
				t.Fatalf("render = %q, space was not escaped", got)
			}
		})
	}
}

// TestActionValidate confirms validation rejects incomplete or inconsistent
// action definitions and accepts well-formed ones.
func TestActionValidate(t *testing.T) {
	cases := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{"valid command", Action{Name: "A", Command: "open x"}, false},
		{"valid select", Action{Name: "A", Type: TypeSelect, Command: "open {{.Value}}", Options: []Option{{Label: "x"}}}, false},
		{"valid form", Action{Name: "A", Type: TypeForm, Command: "open {{.Value}}"}, false},
		{"missing name", Action{Command: "open x"}, true},
		{"missing command", Action{Name: "A"}, true},
		{"select without options", Action{Name: "A", Type: TypeSelect, Command: "x"}, true},
		{"unknown type", Action{Name: "A", Type: "wat", Command: "x"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.action.validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestOptionResolvedValue checks that an option falls back to its label when no
// explicit value is given.
func TestOptionResolvedValue(t *testing.T) {
	if got := (Option{Label: "Herdr Plus", Value: "herdr-plus"}).resolvedValue(); got != "herdr-plus" {
		t.Fatalf("resolvedValue = %q, want %q", got, "herdr-plus")
	}
	if got := (Option{Label: "herdr-plus"}).resolvedValue(); got != "herdr-plus" {
		t.Fatalf("resolvedValue = %q, want label fallback %q", got, "herdr-plus")
	}
}

// TestShellQuote verifies single-quote escaping produces a single shell token.
func TestShellQuote(t *testing.T) {
	if got := shellQuote("plain"); got != "'plain'" {
		t.Fatalf("shellQuote(plain) = %q", got)
	}
	if got := shellQuote("it's"); got != `'it'\''s'` {
		t.Fatalf("shellQuote(it's) = %q", got)
	}
}
