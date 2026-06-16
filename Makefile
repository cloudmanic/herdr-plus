#
# Date: 2026-06-15
# Author: Spicer Matthews (spicer@cloudmanic.com)
# Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
#

BINARY := herdr-plus

.PHONY: help build test test-short vet tidy plugin-link clean site site-dev site-clean site-deps

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
	@echo ""
	@echo "  make site         Build the www/ Hugo site into www/public (mirrors CI)."
	@echo "  make site-dev     Run the site locally with live reload at http://localhost:1313/."
	@echo "  make site-clean   Remove the site's build output."

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

# ---------------------------------------------------------------- website ---
# The marketing + docs site lives in www/ as a Hugo site styled with Tailwind
# v4 (standalone binary — no Node). These targets mirror the GitHub Actions
# deploy in .github/workflows/site.yml.

# site-deps fails early with a friendly message if hugo/tailwindcss are missing.
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
	@echo "→ Tailwind --watch + Hugo dev server on http://localhost:1313/ …"
	@cd www && ( \
		tailwindcss -i assets/css/app.css -o static/css/app.css --watch & \
		TW=$$!; \
		trap "kill $$TW 2>/dev/null" EXIT INT TERM; \
		hugo server --disableFastRender \
	)

# site-clean removes generated site output.
site-clean:
	rm -rf www/public www/resources www/static/css/app.css
