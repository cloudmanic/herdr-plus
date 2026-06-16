#
# Date: 2026-06-15
# Author: Spicer Matthews (spicer@cloudmanic.com)
# Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
#

BINARY := herdr-plus

.PHONY: help build test test-short vet tidy plugin-link clean

# help is the default target so `make` with no args prints what's available.
help:
	@echo "herdr-plus — a herdr plugin"
	@echo ""
	@echo "Targets:"
	@echo "  make build        Build the binary into ./bin/$(BINARY)."
	@echo "  make test         Run the test suite with -race."
	@echo "  make test-short   Skip slow tests (-short) — quick iteration loop."
	@echo "  make vet          Run go vet."
	@echo "  make tidy         Run 'go mod tidy'."
	@echo "  make plugin-link  Build, then link this checkout as a herdr plugin (dev)."
	@echo "  make clean        Remove ./bin and coverage artifacts."

# build produces a single binary at ./bin/$(BINARY) — the same path the plugin
# manifest's [[build]] step and entry points use.
build:
	mkdir -p bin
	go build -o bin/$(BINARY) .

# test runs the full suite with the race detector — the same command CI runs.
test:
	go test -race ./...

# test-short is the quick local iteration loop: skip anything tagged slow.
test-short:
	go test -short ./...

# vet runs go vet across the module.
vet:
	go vet ./...

# tidy keeps go.mod / go.sum in sync with what's actually imported.
tidy:
	go mod tidy

# plugin-link builds the binary and links this checkout with herdr as a local
# development plugin, so its entry points run the freshly built ./bin/herdr-plus.
# Undo with `herdr plugin unlink cloudmanic.herdr-plus`.
plugin-link: build
	herdr plugin link $(CURDIR)

# clean removes build artifacts and coverage output.
clean:
	rm -rf bin coverage.out coverage.html
