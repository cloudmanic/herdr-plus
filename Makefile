#
# Date: 2026-06-09
# Author: Spicer Matthews (spicer@cloudmanic.com)
# Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
#

BINARY := herdr-plus

.PHONY: run build install-bin plugin-link build-linux test test-short coverage tidy clean help site site-dev site-clean

# help is the default target so `make` with no args prints what's available.
help:
	@echo "herdr-plus — an add-on platform for the herdr terminal multiplexer"
	@echo ""
	@echo "Targets:"
	@echo "  make run            Run the launcher (inside a herdr pane)."
	@echo "  make build          Build the binary into ./bin/$(BINARY)."
	@echo "  make install-bin    Install ./bin/$(BINARY) into /usr/local/bin."
	@echo "  make plugin-link    Build, then link this checkout as a herdr plugin (dev)."
	@echo "  make build-linux    Cross-compile a static linux/amd64 binary."
	@echo "  make test           Run the full test suite with -race."
	@echo "  make test-short     Skip slow tests (-short) — quick iteration loop."
	@echo "  make coverage       Generate coverage.out + an HTML report at coverage.html."
	@echo "  make tidy           Run 'go mod tidy'."
	@echo "  make clean          Remove ./bin and coverage artifacts."
	@echo ""
	@echo "  make site           Build the www/ Hugo site into www/public (mirrors CI)."
	@echo "  make site-dev       Run the site locally with live reload at http://localhost:1313/."
	@echo "  make site-clean     Remove the site's build output."

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

# plugin-link builds the binary and links this checkout with herdr as a local
# development plugin, so its actions run the freshly built ./bin/herdr-plus.
# Undo with `herdr plugin unlink cloudmanic.herdr-plus`.
plugin-link: build
	herdr plugin link $(CURDIR)

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

# ---------------------------------------------------------------- website ---
# The marketing + docs site lives in www/ as a Hugo site styled with Tailwind
# v4 (standalone binary — no Node). These targets mirror the GitHub Actions
# deploy in .github/workflows/site.yml.

# site-deps fails early with a friendly message if hugo/tailwindcss are missing.
.PHONY: site-deps
site-deps:
	@command -v hugo >/dev/null 2>&1 || { echo "✗ hugo not found — install with: brew install hugo"; exit 1; }
	@command -v tailwindcss >/dev/null 2>&1 || { echo "✗ tailwindcss not found — install with: brew install tailwindcss"; exit 1; }

# site builds the production static site into www/public.
site: site-deps
	@echo "→ Compiling Tailwind CSS…"
	@cd www && tailwindcss -i assets/css/app.css -o static/css/app.css --minify
	@echo "→ Building Hugo site…"
	@cd www && hugo --minify --gc
	@echo ""
	@echo "✓ Built www/public — preview the whole thing with: make site-dev"

# site-dev runs Tailwind in --watch alongside Hugo's live-reload dev server.
site-dev: site-deps
	@echo "→ Tailwind --watch + Hugo dev server on http://localhost:1313/herdr-plus/ …"
	@cd www && ( \
		tailwindcss -i assets/css/app.css -o static/css/app.css --watch & \
		TW=$$!; \
		trap "kill $$TW 2>/dev/null" EXIT INT TERM; \
		hugo server --disableFastRender \
	)

# site-clean removes generated site output.
site-clean:
	rm -rf www/public www/resources www/static/css/app.css
