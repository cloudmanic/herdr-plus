---
title: "Installation"
description: "Install herdr-plus via Homebrew, the install.sh one-liner, or from source — with supported platforms, install locations, and upgrades."
weight: 20
---

herdr-plus is a single static binary. There are three ways to install it:
Homebrew, the install script, or from source. All of them are free and open
source.

> **Note:** herdr-plus is an add-on for [herdr](https://herdr.dev). Install and
> set up herdr first — herdr-plus does its work by talking to a running herdr
> server.

## Homebrew

The herdr-plus repository is its own Homebrew tap. Tap it, then install:

```bash
brew tap cloudmanic/herdr-plus https://github.com/cloudmanic/herdr-plus
brew install cloudmanic/herdr-plus/herdr-plus
```

To upgrade later:

```bash
brew upgrade cloudmanic/herdr-plus/herdr-plus
```

## Install script

A POSIX `sh` installer detects your OS and architecture, downloads the matching
archive from the latest GitHub Release, extracts the static binary, and drops it
into place. It works under plain `sh`, so it's fine on Alpine/BusyBox and minimal
SSH targets, not just bash.

```bash
curl -fsSL https://raw.githubusercontent.com/cloudmanic/herdr-plus/main/install.sh | sh
```

Re-running the script performs an upgrade.

### Environment overrides

The script honors two environment variables:

| Variable | Default | What it does |
|----------|---------|--------------|
| `INSTALL_DIR` | `~/.local/bin` (else `/usr/local/bin`) | Where the binary is installed. |
| `VERSION` | the latest GitHub Release | Pin a specific release tag to install. |

Examples:

```bash
# Install into a custom directory.
curl -fsSL https://raw.githubusercontent.com/cloudmanic/herdr-plus/main/install.sh | INSTALL_DIR=/opt/bin sh

# Pin a specific version (tags are prefixed with "v").
curl -fsSL https://raw.githubusercontent.com/cloudmanic/herdr-plus/main/install.sh | VERSION=v0.0.1 sh
```

## From source

Clone the [repository](https://github.com/cloudmanic/herdr-plus) and build with
the Makefile:

```bash
make build         # build the herdr-plus binary
make install-bin   # build, then install the binary onto your PATH
```

Or use the Go toolchain directly:

```bash
go build -o herdr-plus .
go test ./...
```

## Supported platforms

Releases are cross-compiled for:

- **Operating systems:** Linux and macOS.
- **Architectures:** `amd64` (x86_64) and `arm64` (aarch64).

The install script maps `uname` output onto those tokens and refuses to run on
anything else, telling you exactly what it detected.

## Where the binary lands

The install script chooses, in order:

1. The directory in `INSTALL_DIR`, if you set it.
2. `~/.local/bin` — preferred, because it needs no `sudo`.
3. `/usr/local/bin` — the fallback, which may prompt for `sudo`.

> **Tip:** If the chosen directory isn't on your `$PATH`, the script prints the
> exact `export PATH=...` line to add to your shell rc. Without that, your shell
> won't find the `herdr-plus` command.

## Checking the version

Confirm what's installed at any time:

```bash
herdr-plus version
```

(`--version`, `-v`, and `-V` all work too.) It prints `herdr-plus` followed by
the release version.

## How upgrades work

Every merge to `main` auto-bumps the patch version and cuts a new GitHub Release
with cross-compiled binaries. To pull the latest:

- **Homebrew:** `brew upgrade cloudmanic/herdr-plus/herdr-plus`
- **Install script:** re-run the `curl ... | sh` one-liner.
- **From source:** `git pull` and `make install-bin` again.

## Next steps

Once herdr-plus is on your PATH, bind it to a key:
[Keybindings](../keybindings/), then jump into the
[Quick Start](../quick-start/).
