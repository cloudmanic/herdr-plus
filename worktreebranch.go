//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "strings"

func resolveWorktreeBranch(input, prefix string) string {
	branch := strings.TrimSpace(input)
	if branch == "" {
		return ""
	}
	if prefix == "" || strings.Contains(branch, "/") {
		return branch
	}
	return prefix + branch
}
