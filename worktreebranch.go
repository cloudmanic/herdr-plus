//
// Date: 2026-07-05
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

package main

import "strings"

// resolveWorktreeBranch turns the branch name typed in the projects browser into
// the branch handed to herdr. A blank input stays blank, letting herdr generate
// its native worktree/<name> branch. A name already containing "/" is treated as
// fully qualified and used as-is. Otherwise the optional prefix is prepended
// verbatim (the caller's config supplies any trailing "/").
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
