//
// Date: 2026-10-16
// Author: cgill27 (cgill27@gmail.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// isolateGit runs git in a session of its own. With no controlling terminal,
// ssh (and anything else git spawns) can't open /dev/tty to ask for a host key
// or passphrase, so it fails instead of waiting on a prompt nobody sees. The
// session is also its own process group, so a timeout kills git together with
// its ssh or credential-helper children rather than leaving them running.
func isolateGit(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
