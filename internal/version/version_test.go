//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

// Tests for the tiny version package. The Version constant is the single source
// of truth a release pipeline bumps, so we fail loudly if its shape ever drifts
// away from semver.

package version

import (
	"strconv"
	"strings"
	"testing"
)

// TestVersionNotEmpty makes sure the constant has a value at all.
func TestVersionNotEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("Version is empty")
	}
}

// TestVersionIsSemver verifies the constant is in major.minor.patch form so a
// release pipeline's auto-bump logic stays correct.
func TestVersionIsSemver(t *testing.T) {
	parts := strings.Split(Version, ".")
	if len(parts) != 3 {
		t.Fatalf("Version %q is not in x.y.z form (got %d parts)", Version, len(parts))
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			t.Fatalf("Version %q part %d (%q) is not an integer: %v", Version, i, p, err)
		}
		if n < 0 {
			t.Fatalf("Version %q part %d (%q) is negative", Version, i, p)
		}
	}
}

// TestVersionPreOneZero pins the major version at 0 while we are still pre-1.0.
// Bumping past 0 is a deliberate event, so update this test on purpose rather
// than letting the constant drift.
func TestVersionPreOneZero(t *testing.T) {
	major, err := strconv.Atoi(strings.Split(Version, ".")[0])
	if err != nil {
		t.Fatalf("Version %q has non-numeric major: %v", Version, err)
	}
	if major != 0 {
		t.Fatalf("Version %q major is %d, expected 0 (pre-1.0). Update this test deliberately when shipping 1.0.", Version, major)
	}
}
