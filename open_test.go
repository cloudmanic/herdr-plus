//
// Date: 2026-07-23
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindProject covers findProject's three resolution paths: an exact
// case-sensitive match, a case-insensitive fallback, and a miss that reports the
// available project names. A separate case checks the empty-set error.
func TestFindProject(t *testing.T) {
	projects := []Project{
		{Name: "app.harbor.my"},
		{Name: "harbor-sysadmin"},
	}

	// Exact match wins.
	got, err := findProject(projects, "harbor-sysadmin")
	if err != nil {
		t.Fatalf("exact match: unexpected error: %v", err)
	}
	if got.Name != "harbor-sysadmin" {
		t.Fatalf("exact match: got %q, want %q", got.Name, "harbor-sysadmin")
	}

	// Case-insensitive fallback resolves when the exact case differs.
	got, err = findProject(projects, "Harbor-Sysadmin")
	if err != nil {
		t.Fatalf("case-insensitive match: unexpected error: %v", err)
	}
	if got.Name != "harbor-sysadmin" {
		t.Fatalf("case-insensitive match: got %q, want %q", got.Name, "harbor-sysadmin")
	}

	// A miss names the bad input and lists every available project.
	_, err = findProject(projects, "nope")
	if err == nil {
		t.Fatal("no match: expected an error, got nil")
	}
	for _, want := range []string{"nope", "app.harbor.my", "harbor-sysadmin"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("no match: error %q is missing %q", err.Error(), want)
		}
	}

	// An empty project set returns a distinct, clear error.
	if _, err := findProject(nil, "anything"); err == nil {
		t.Fatal("empty set: expected an error, got nil")
	}
}

// TestHerdrManagedConfigDir confirms herdrManagedConfigDir returns herdr's
// (whitespace-trimmed) output on success and "" when the herdr binary fails,
// using a fake herdr located via HERDR_BIN_PATH so the test never needs a real one.
func TestHerdrManagedConfigDir(t *testing.T) {
	// Success: fake herdr prints a path with surrounding whitespace.
	t.Setenv("HERDR_BIN_PATH", writeFakeHerdr(t, "#!/bin/sh\necho '  /managed/dir  '\n"))
	if got := herdrManagedConfigDir(); got != "/managed/dir" {
		t.Fatalf("herdrManagedConfigDir = %q, want %q", got, "/managed/dir")
	}

	// Failure: fake herdr exits non-zero, so we get "".
	t.Setenv("HERDR_BIN_PATH", writeFakeHerdr(t, "#!/bin/sh\nexit 1\n"))
	if got := herdrManagedConfigDir(); got != "" {
		t.Fatalf("herdrManagedConfigDir on failure = %q, want empty", got)
	}
}

// TestEnsureManagedConfigDir confirms ensureManagedConfigDir exports the queried
// directory when HERDR_PLUGIN_CONFIG_DIR is unset, and leaves an already-set value
// untouched (never consulting herdr in that case).
func TestEnsureManagedConfigDir(t *testing.T) {
	t.Setenv("HERDR_BIN_PATH", writeFakeHerdr(t, "#!/bin/sh\necho /queried/dir\n"))

	// Unset → the queried directory is exported.
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", "")
	ensureManagedConfigDir()
	if got := os.Getenv("HERDR_PLUGIN_CONFIG_DIR"); got != "/queried/dir" {
		t.Fatalf("unset case: HERDR_PLUGIN_CONFIG_DIR = %q, want %q", got, "/queried/dir")
	}

	// Already set → left exactly as-is.
	t.Setenv("HERDR_PLUGIN_CONFIG_DIR", "/preset/dir")
	ensureManagedConfigDir()
	if got := os.Getenv("HERDR_PLUGIN_CONFIG_DIR"); got != "/preset/dir" {
		t.Fatalf("preset case: HERDR_PLUGIN_CONFIG_DIR = %q, want %q", got, "/preset/dir")
	}
}

// writeFakeHerdr writes an executable shell script that stands in for the herdr
// binary and returns its path. The script ignores its arguments, so it serves any
// `herdr ...` invocation the code under test makes.
func writeFakeHerdr(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "herdr")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake herdr: %v", err)
	}
	return path
}
