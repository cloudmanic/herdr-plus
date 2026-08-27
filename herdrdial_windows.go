//
// Date: 2026-06-15
// Author: Spicer Matthews (spicer@cloudmanic.com)
// Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
//

//go:build windows

package main

import (
	"errors"
	"io"
	"os"
	"syscall"
	"time"
)

// On Windows herdr has no unix domain socket. It listens on a named pipe whose
// name is the HERDR_SOCKET_PATH value, exposed under the pipe namespace as
// \\.\pipe\<socketPath>. The on-disk file at socketPath is only a PID:nonce
// liveness marker — not the endpoint — so we must open the pipe, not the file.
const pipePrefix = `\\.\pipe\`

// errPipeBusy is Win32 ERROR_PIPE_BUSY (231): every instance of the pipe is
// currently serving another client. It is transient — herdr creates a new
// instance as soon as one frees up — so dialHerdr retries briefly instead of
// failing the whole call.
const errPipeBusy = syscall.Errno(231)

// dialHerdr opens a connection to herdr's IPC endpoint. On Windows that is the
// named pipe \\.\pipe\<socketPath>, opened for read/write. The *os.File it
// returns satisfies the io.ReadWriteCloser that call() needs; the newline-
// delimited JSON protocol is identical to the unix socket. It retries on
// ERROR_PIPE_BUSY for up to ~1s so a momentarily saturated pipe does not fail
// an otherwise-healthy request.
func dialHerdr(socketPath string) (io.ReadWriteCloser, error) {
	pipePath := pipePrefix + socketPath

	deadline := time.Now().Add(time.Second)
	for {
		f, err := os.OpenFile(pipePath, os.O_RDWR, 0)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, errPipeBusy) || time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(50 * time.Millisecond)
	}
}
