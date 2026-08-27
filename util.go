//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import (
	"fmt"
	"os"
)

// pluginID is herdr-plus's plugin id, matching the `id` field in
// herdr-plugin.toml. It is how herdr identifies the plugin — used when opening
// plugin-owned panes and when asking herdr for our managed config directory.
const pluginID = "cloudmanic.herdr-plus"

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
