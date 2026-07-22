//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

//go:build !windows

package main

import (
	"io"
	"net"
)

// dialHerdr opens a connection to herdr's IPC endpoint. On unix that endpoint is
// the unix domain socket at socketPath (the value herdr injects as
// HERDR_SOCKET_PATH). The returned connection is used only as an
// io.ReadWriteCloser by call(), so the Windows twin can return an *os.File.
func dialHerdr(socketPath string) (io.ReadWriteCloser, error) {
	return net.Dial("unix", socketPath)
}
