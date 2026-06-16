//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"os"
	"strings"
)

// errExit prints a "herdr-plus:"-prefixed message to stderr and exits non-zero.
func errExit(args ...any) {
	fmt.Fprintln(os.Stderr, append([]any{"herdr-plus:"}, args...)...)
	os.Exit(1)
}

// firstNonEmpty returns the first argument that is not the empty string, or ""
// when all are empty.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// shellQuote wraps a string in single quotes so the shell treats it as one
// literal argument, escaping any embedded single quotes the usual POSIX way
// ('\'' closes the quote, adds an escaped quote, and reopens it).
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
