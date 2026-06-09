#
# Date: 2026-06-09
# Author: Spicer Matthews (spicer@cloudmanic.com)
# Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
#

BINARY := herdr-plus

.PHONY: run build install-bin install-keybind build-linux test test-short coverage tidy clean help

# help is the default target so `make` with no args prints what's available.
help:
	@echo "herdr-plus — an add-on platform for the herdr terminal multiplexer"
	@echo ""
	@echo "Targets:"
	@echo "  make run            Run the launcher (inside a herdr pane)."
	@echo "  make build          Build the binary into ./bin/$(BINARY)."
	@echo "  make install-bin    Install ./bin/$(BINARY) into /usr/local/bin."
	@echo "  make install-keybind Build, then bind the herdr keybinding (prefix+down)."
	@echo "  make build-linux    Cross-compile a static linux/amd64 binary."
	@echo "  make test           Run the full test suite with -race."
	@echo "  make test-short     Skip slow tests (-short) — quick iteration loop."
	@echo "  make coverage       Generate coverage.out + an HTML report at coverage.html."
	@echo "  make tidy           Run 'go mod tidy'."
	@echo "  make clean          Remove ./bin and coverage artifacts."

# run starts the launcher via 'go run'. Must be run inside a herdr pane.
run:
	go run .

# build produces a single binary at ./bin/$(BINARY).
build:
	mkdir -p bin
	go build -o bin/$(BINARY) .

# install-bin copies the binary into /usr/local/bin so you can launch it as
# `herdr-plus`.
install-bin: build
	install -m 0755 bin/$(BINARY) /usr/local/bin/$(BINARY)

# install-keybind builds the binary and registers the herdr keybinding, using
# the freshly built binary's absolute path.
install-keybind: build
	./bin/$(BINARY) install

# build-linux cross-compiles a fully static linux/amd64 binary.
build-linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o bin/$(BINARY)-linux-amd64 .

# test runs the full suite with the race detector — the same command CI runs
# (.github/workflows/test.yml). Keep them in lockstep.
test:
	go test -race ./...

# test-short is the quick local iteration loop: skip anything tagged slow.
test-short:
	go test -short ./...

# coverage produces a coverage profile and a rendered HTML report.
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out | tail -n 1

# tidy keeps go.mod / go.sum in sync with what's actually imported.
tidy:
	go mod tidy

# clean removes build artifacts and coverage output.
clean:
	rm -rf bin coverage.out coverage.html
