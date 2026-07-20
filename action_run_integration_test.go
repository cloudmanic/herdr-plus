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

// TestActionRunQuotingRoundTrip is the empirical half of the quoting contract: it
// renders and runs a real action through the host's actual shell (PowerShell on
// Windows, sh elsewhere) and asserts a hostile Value — spaces, a single quote, a
// double quote, and a $ that PowerShell would otherwise expand as a variable —
// comes back out verbatim. The unit tests prove shellQuote's output; this proves
// that output actually survives the shell shellCommand launches, which is the DoD
// guarantee for quick-action value injection on Windows.
//
// It exercises render()'s auto-append path: a command with no {{.Value}} gets the
// value appended as one shell-quoted argument (via shellQuote). That is the path
// herdr-plus is responsible for making injection-safe. A command that hand-quotes
// {{.Value}} itself (e.g. echo '{{.Value}}') is the author's responsibility — the
// raw value is substituted verbatim there — so it is intentionally not asserted.
func TestActionRunQuotingRoundTrip(t *testing.T) {
	// A value that breaks naive concatenation on both shells at once: spaces, both
	// quote styles, and a PowerShell variable sigil.
	const nasty = `a b 'c" $x`

	// echo with no {{.Value}} → render appends shellQuote(value). `echo` is
	// Write-Output on PowerShell and the echo builtin on sh; both print their
	// argument verbatim.
	a := Action{Name: "roundtrip", Type: TypeForm, Command: "echo"}
	cmdline, err := a.render(RunContext{Value: nasty})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out, err := shellCommand(cmdline).Output()
	if err != nil {
		t.Fatalf("run %q: %v", cmdline, err)
	}

	if got := strings.TrimRight(string(out), "\r\n"); got != nasty {
		t.Fatalf("round-trip through shell = %q, want %q (cmdline: %q)", got, nasty, cmdline)
	}
}
