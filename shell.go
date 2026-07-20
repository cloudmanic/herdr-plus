//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os/exec"
	"runtime"
	"strings"
)

// This file is the one place that knows which shell herdr-plus's quick actions
// run under, and how to quote a value for it. Everything OS-specific about
// running a command string lives here — the rest of the plugin calls
// shellCommand / shellQuote and stays free of runtime.GOOS checks.
//
// It matters only for quick actions, which the plugin executes itself (see
// action.go). Project and worktree tab commands are typed straight into a herdr
// pane and run by that pane's own shell, so they never pass through here.

// platform bundles everything about the host OS that quick-action execution
// varies on: the shell commands run under, how to quote a value for that shell,
// the pane-entrypoint suffix, and the default "open this" command. Selecting it
// once (currentPlatform) keeps a single OS switch here instead of one per
// concern — adding an OS is a new case, not four scattered edits.
type platform struct {
	shellPrefix      []string            // argv prefix; the command string is appended
	quote            func(string) string // wrap a value as one literal shell argument
	entrypointSuffix string              // appended to a base pane id (see paneEntrypoint)
	opener           string              // command that opens a URL/file in its default handler
}

// currentPlatform is the descriptor for the OS this binary was built for. GOOS is
// fixed per build, so it is resolved once at init rather than on every call.
var currentPlatform = func() platform {
	switch runtime.GOOS {
	case "windows":
		// Windows PowerShell (always present, unlike pwsh). The "-windows" suffix
		// selects the .exe pane twins; Windows cannot spawn the extensionless PE.
		return platform{
			shellPrefix:      []string{"powershell", "-NoProfile", "-NonInteractive", "-Command"},
			quote:            powershellQuote,
			entrypointSuffix: "-windows",
			opener:           "Start-Process",
		}
	case "darwin":
		return platform{shellPrefix: []string{"sh", "-c"}, quote: posixQuote, opener: "open"}
	default: // linux and other unixes
		return platform{shellPrefix: []string{"sh", "-c"}, quote: posixQuote, opener: "xdg-open"}
	}
}()

// shellCommand builds an *exec.Cmd that runs cmdline through the platform's
// default shell. Running through a shell — rather than splitting the string
// ourselves — is deliberate: a quick action's command may use pipes,
// redirection, and multiple arguments.
func shellCommand(cmdline string) *exec.Cmd {
	argv := append(append([]string{}, currentPlatform.shellPrefix...), cmdline)
	return exec.Command(argv[0], argv[1:]...)
}

// shellQuote wraps s so the platform shell treats it as a single literal
// argument — no word splitting, no variable expansion. It uses the quoting rules
// of the shell shellCommand runs under, so the two always agree.
func shellQuote(s string) string {
	return currentPlatform.quote(s)
}

// posixQuote wraps s in single quotes, escaping any embedded single quote the
// usual POSIX way ('\'' closes the quote, adds an escaped quote, and reopens
// it). Single quotes make the shell treat everything inside literally.
func posixQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// powershellQuote wraps s in single quotes for PowerShell, escaping an embedded
// single quote by doubling it ('' is a literal quote inside a single-quoted
// string). A PowerShell single-quoted string is fully literal — no $var or
// backtick interpretation — which is exactly what we want for an injected value.
func powershellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// paneEntrypoint maps a base pane id to the entrypoint id registered for the
// current OS. On Windows every pane is declared twice in herdr-plugin.toml — the
// unix id and a `-windows` twin that runs the .exe binary (Windows cannot spawn
// the extensionless PE by its bare path). Callers that open a pane pass the base
// id through here so the "-windows" suffix lives in one place, not scattered
// runtime.GOOS checks at each call site.
func paneEntrypoint(base string) string {
	return base + currentPlatform.entrypointSuffix
}

// opener returns the platform command that opens a URL, file, or directory in
// its default handler: `open` on macOS, `xdg-open` on Linux, and PowerShell's
// `Start-Process` on Windows. Quick actions reach it through the {{opener}}
// template function (see templateFuncs), so one example — `{{opener}} <target>`
// — works on every OS instead of hardcoding the macOS-only `open`.
func opener() string {
	return currentPlatform.opener
}
