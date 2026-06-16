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

	"github.com/cloudmanic/herdr-plus/internal/version"
)

// main dispatches between the binary's entry points. The bare binary (optionally
// with --mode) is the launcher: it opens a bottom split and starts the picker
// inside it. The hidden "picker" subcommand renders the fuzzy-finder TUI and, on
// exit, closes its own pane; the launcher re-invokes itself in picker mode, so
// end users only ever run the bare binary. The "on-worktree-created" subcommand
// is herdr's worktree.created event handler — herdr runs it (via the [[events]]
// entry in herdr-plugin.toml), not the user.
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "control":
			runControlCmd(os.Args[2:])
			return
		case "picker":
			runPickerCmd(os.Args[2:])
			return
		case "on-worktree-created":
			runOnWorktreeCreated(os.Args[2:])
			return
		case "version", "--version", "-v", "-V":
			fmt.Println("herdr-plus", version.Version)
			return
		}
	}
	runLauncherCmd(os.Args[1:])
}

// runLauncherCmd parses the launcher's flags (currently just --mode) and starts
// the launcher for the resolved mode.
func runLauncherCmd(args []string) {
	fs := flag.NewFlagSet("herdr-plus", flag.ExitOnError)
	modeSlug := fs.String("mode", "", "which herdr-plus mode to run (default: quick-actions)")
	_ = fs.Parse(args)

	mode, err := lookupMode(*modeSlug)
	if err != nil {
		errExit(err)
	}
	runLauncher(mode)
}

// runPickerCmd parses the internal picker invocation: --mode and --ctx flags
// plus a trailing pane id (the picker's own pane, which it closes on exit).
func runPickerCmd(args []string) {
	fs := flag.NewFlagSet("picker", flag.ExitOnError)
	modeSlug := fs.String("mode", "", "which herdr-plus mode to run")
	ctxArg := fs.String("ctx", "", "base64-encoded run context from the launcher")
	_ = fs.Parse(args)

	mode, err := lookupMode(*modeSlug)
	if err != nil {
		errExit(err)
	}
	ctx, err := decodeRunContext(*ctxArg)
	if err != nil {
		errExit("could not decode run context:", err)
	}

	// The trailing argument is the picker's own pane id so it can close itself;
	// fall back to HERDR_PANE_ID, which is this pane.
	selfPane := ""
	if rest := fs.Args(); len(rest) > 0 {
		selfPane = rest[0]
	}
	if selfPane == "" {
		selfPane = os.Getenv("HERDR_PANE_ID")
	}

	runPicker(mode, ctx, selfPane)
}

// runLauncher dispatches to the right launch behavior for the mode. Modes differ
// in how they present themselves: quick-actions opens a small split beneath the
// current pane, while control mode opens a brand-new full-screen workspace.
func runLauncher(mode Mode) {
	client, err := newHerdrClient()
	if err != nil {
		errExit(err)
	}

	switch mode.Slug {
	case ModeControl.Slug:
		launchControl(client)
	default:
		launchPicker(client, mode)
	}
}

// launchPicker splits the current herdr pane downward, focuses the new pane, and
// launches this same binary in picker mode inside it. Before splitting it
// gathers the run context (working directory and herdr session metadata) from
// the pane it was invoked in — the only pane that knows the user's real working
// directory — and hands it to the picker so the chosen action sees it. It then
// returns immediately so the original shell prompt comes back.
func launchPicker(client *herdrClient, mode Mode) {
	// HERDR_PANE_ID identifies the pane we are launched from; it is the pane we
	// split to create the picker beneath it. It is set in a pane's own shell but
	// not for a keybinding (which runs server-side), so fall back to the focused
	// pane in that case.
	paneID := os.Getenv("HERDR_PANE_ID")
	if paneID == "" {
		var err error
		paneID, err = client.focusedPaneID()
		if err != nil || paneID == "" {
			errExit("could not determine the focused pane; are you running inside herdr?")
		}
	}

	// Resolve our own absolute path so the new pane's shell can launch the
	// picker even when the binary is not on PATH.
	exe, err := os.Executable()
	if err != nil {
		errExit(err)
	}

	// Gather context from the invoking pane and encode it for the picker.
	ctx := gatherContext(client, paneID)
	encoded, err := ctx.encode()
	if err != nil {
		errExit(err)
	}

	// Create the picker pane beneath the current one and focus it so keystrokes
	// flow to the picker.
	newPane, err := client.paneSplit(paneID, "down", true)
	if err != nil {
		errExit("split failed:", err)
	}

	// Start the picker in the new pane, handing it the mode, the encoded context,
	// and the new pane's id so it can close itself when done. runCommand waits for
	// the new shell's prompt and submits with a real Enter key (not a trailing
	// newline — see sendInput), so the picker starts reliably.
	launch := fmt.Sprintf("%s picker --mode=%s --ctx=%s %s", shellQuote(exe), mode.Slug, encoded, newPane)
	if err := client.runCommand(newPane, launch); err != nil {
		errExit("failed to start picker:", err)
	}
}

// errExit prints a "herdr-plus:"-prefixed message to stderr and exits non-zero.
func errExit(args ...any) {
	fmt.Fprintln(os.Stderr, append([]any{"herdr-plus:"}, args...)...)
	os.Exit(1)
}
