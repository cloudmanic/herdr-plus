#!/bin/sh
#
# Date: 2026-06-15
# Author: Spicer Matthews (spicer@cloudmanic.com)
# Copyright: 2026 Cloudmanic Labs, LLC. All rights reserved.
#
# Build herdr-plus for `herdr plugin install`. herdr runs this as the manifest's
# [[build]] step after cloning the repo, in the plugin root, with no plugin
# context and no guaranteed Go toolchain.
#
# Prefer a local Go toolchain — it builds the exact cloned source. When Go is
# absent, fall back to downloading the latest prebuilt release binary via
# install.sh, so installing the plugin works without Go. Either way the result is
# ./bin/herdr-plus, which the manifest's actions and panes invoke.

set -eu

mkdir -p bin

if command -v go >/dev/null 2>&1; then
	echo "herdr-plus: building from source (go build)…" >&2
	exec go build -o bin/herdr-plus .
fi

echo "herdr-plus: no Go toolchain found — downloading the latest prebuilt binary…" >&2
INSTALL_DIR="$(pwd)/bin" sh install.sh
