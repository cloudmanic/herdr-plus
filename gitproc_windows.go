//
// Date: 2026-10-16
// Author: cgill27 (cgill27@gmail.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

//go:build windows

package main

import "os/exec"

// isolateGit is a no-op on Windows: exec.CommandContext's default kill of git
// itself, plus WaitDelay closing its pipes, is what bounds a timed-out call.
func isolateGit(cmd *exec.Cmd) {}
