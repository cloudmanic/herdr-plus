//
// Date: 2026-09-14
// Author: Brendan Lees (43487974+brendanlees@users.noreply.github.com)
// Copyright: 2026 Brendan Lees. All rights reserved.
//

package main

import (
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Each test owns its renderer; never change Lip Gloss's package-wide profile.
func titleTestRenderer(profile termenv.Profile) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(profile)
	return r
}

// Check the entire SGR parameter set rather than just excluding RGB: even an
// ANSI foreground/background would defeat terminal-default inverse colours.
func assertInverseTitle(t *testing.T, rendered string) {
	t.Helper()
	sgr := regexp.MustCompile(`\x1b\[([0-9;]*)m`)
	reversed := false
	for _, match := range sgr.FindAllStringSubmatch(rendered, -1) {
		for _, code := range strings.Split(match[1], ";") {
			switch code {
			case "0", "1": // reset and bold
			case "7":
				reversed = true
			default:
				t.Fatalf("unexpected SGR %q in inverse title %q", code, rendered)
			}
		}
	}
	if !reversed {
		t.Fatalf("missing inverse SGR in %q", rendered)
	}
	if strings.Contains(sgr.ReplaceAllString(rendered, ""), "\x1b") || strings.Contains(rendered, "#") {
		t.Fatalf("unexpected escape or literal colour in %q", rendered)
	}
}

func TestConfiguredTitleStyle(t *testing.T) {
	for _, profile := range []termenv.Profile{termenv.TrueColor, termenv.ANSI256, termenv.ANSI, termenv.Ascii} {
		t.Run(profile.Name(), func(t *testing.T) {
			t.Parallel()
			r := titleTestRenderer(profile)
			// The released purple style, independently specified to catch drift.
			original := r.NewStyle().Bold(true).
				Foreground(lipgloss.Color("#11111B")).
				Background(lipgloss.Color("#A78BFA")).Padding(0, 1)
			var cfg PluginConfig
			cfg.UI.TerminalTitle = true
			inverse := configuredTitleStyle(cfg).Renderer(r)
			if inverse.GetForeground() != (lipgloss.NoColor{}) || inverse.GetBackground() != (lipgloss.NoColor{}) || !inverse.GetReverse() {
				t.Fatal("inverse must have reverse set and neither colour set")
			}
			for _, width := range []int{0, 40} {
				got := inverse.Width(width).Render("Title")
				plain := r.NewStyle().Padding(0, 1).Width(width).Render("Title")
				if lipgloss.Width(got) != lipgloss.Width(plain) {
					t.Fatalf("inverse changed title width: %q", got)
				}
				if profile == termenv.Ascii {
					if got != plain {
						t.Fatalf("ASCII should omit all styling: got %q, want %q", got, plain)
					}
				} else {
					assertInverseTitle(t, got)
				}
				// Selecting/rendering the opt-in style must not contaminate defaults.
				gotDefault := configuredTitleStyle(PluginConfig{}).Renderer(r).Width(width).Render("Title")
				if want := original.Width(width).Render("Title"); gotDefault != want {
					t.Fatalf("default changed: got %q, want %q", gotDefault, want)
				}
			}
		})
	}
}

func TestTerminalTitleNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("CLICOLOR_FORCE", "1") // NO_COLOR wins in the pinned termenv stack.
	r := lipgloss.NewRenderer(io.Discard)
	if r.ColorProfile() != termenv.Ascii {
		t.Fatalf("NO_COLOR profile = %v, want ASCII", r.ColorProfile())
	}
	var cfg PluginConfig
	cfg.UI.TerminalTitle = true
	if got := configuredTitleStyle(cfg).Renderer(r).Render("Title"); got != " Title " {
		t.Fatalf("NO_COLOR must suppress reverse too: %q", got)
	}
}
