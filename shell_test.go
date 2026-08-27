//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"runtime"
	"testing"
)

// TestPosixQuote verifies POSIX single-quote escaping produces a single token,
// escaping an embedded quote the '\'' way. Tested directly (not via shellQuote)
// so it runs on every platform, not just Unix.
func TestPosixQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "'plain'"},
		{"it's", `'it'\''s'`},
		{"a b c", "'a b c'"},
		{"", "''"},
	}
	for _, c := range cases {
		if got := posixQuote(c.in); got != c.want {
			t.Fatalf("posixQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestPowershellQuote verifies PowerShell single-quote escaping produces a single
// literal token, escaping an embedded quote by doubling it ('' -> '). Tested
// directly so it runs on every platform, not just Windows.
func TestPowershellQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "'plain'"},
		{"it's", "'it''s'"},
		{"a b c", "'a b c'"},
		{`$env:PATH`, `'$env:PATH'`}, // single quotes keep it literal, no expansion
		{"", "''"},
	}
	for _, c := range cases {
		if got := powershellQuote(c.in); got != c.want {
			t.Fatalf("powershellQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestShellQuoteDispatch confirms shellQuote picks the rules for the host shell,
// so it agrees with the shell shellCommand runs.
func TestShellQuoteDispatch(t *testing.T) {
	got := shellQuote("it's")
	want := posixQuote("it's")
	if runtime.GOOS == "windows" {
		want = powershellQuote("it's")
	}
	if got != want {
		t.Fatalf("shellQuote(it's) = %q, want %q for GOOS=%s", got, want, runtime.GOOS)
	}
}

// TestShellCommand confirms the invoked shell and its arguments match the host:
// PowerShell -Command on Windows, sh -c elsewhere, with the command string as
// the final argument either way.
func TestShellCommand(t *testing.T) {
	cmd := shellCommand("echo hi")
	args := cmd.Args
	if len(args) == 0 {
		t.Fatal("shellCommand produced no args")
	}
	if got := args[len(args)-1]; got != "echo hi" {
		t.Fatalf("last arg = %q, want the command string", got)
	}
	if runtime.GOOS == "windows" {
		if args[0] != "powershell" {
			t.Fatalf("shell = %q, want powershell", args[0])
		}
	} else if args[0] != "sh" || args[1] != "-c" {
		t.Fatalf("shell = %v, want [sh -c ...]", args[:2])
	}
}

// TestPaneEntrypoint confirms the base pane id gets the "-windows" suffix on
// Windows and is returned unchanged elsewhere, matching the paired manifest
// entries (picker / picker-windows). This is what keeps the runtime.GOOS branch
// out of the pane-open call sites.
func TestPaneEntrypoint(t *testing.T) {
	got := paneEntrypoint("picker")
	want := "picker"
	if runtime.GOOS == "windows" {
		want = "picker-windows"
	}
	if got != want {
		t.Fatalf("paneEntrypoint(picker) = %q, want %q for GOOS=%s", got, want, runtime.GOOS)
	}
}

// TestOpener confirms opener returns the current platform's open command, so the
// {{opener}} template helper renders to something runnable on the host.
func TestOpener(t *testing.T) {
	want := map[string]string{"windows": "Start-Process", "darwin": "open"}[runtime.GOOS]
	if want == "" {
		want = "xdg-open"
	}
	if got := opener(); got != want {
		t.Fatalf("opener() = %q, want %q for GOOS=%s", got, want, runtime.GOOS)
	}
}
