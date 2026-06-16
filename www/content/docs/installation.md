---
title: "Installation"
description: "Install herdr-plus as a herdr plugin (recommended), or get the standalone binary via Homebrew, the install.sh one-liner, or from source."
weight: 20
---

herdr-plus ships as a [herdr plugin](https://herdr.dev/docs/plugins/). The
recommended way to install it is `herdr plugin install`, which registers the
plugin's actions with herdr. If you'd rather just have the standalone binary on
your `PATH`, it's also distributed on its own via Homebrew, an install script, or
from source. All of them are free and open source.

> **Note:** herdr-plus is an add-on for [herdr](https://herdr.dev) and the plugin
> needs **herdr ≥ 0.7.0**. Install and set up herdr first — herdr-plus does its
> work by talking to a running herdr server.

## As a herdr plugin (recommended)

herdr clones the repo, runs the manifest's `[[build]]` step to compile the binary
(it needs a Go toolchain on the machine), and registers the `control` and
`quick-actions` actions:

```bash
herdr plugin install cloudmanic/herdr-plus
```

Confirm it registered, and manage it, with the usual plugin commands:

```bash
herdr plugin list                                            # is it installed?
herdr plugin action list --plugin cloudmanic.herdr-plus      # its actions
herdr plugin uninstall cloudmanic.herdr-plus                 # remove it
```

To upgrade, re-run `herdr plugin install` so herdr re-clones and rebuilds the
latest.

### For local development

To work on herdr-plus, build the binary and link your checkout in place so the
actions have something to run:

```bash
make build
herdr plugin link /path/to/herdr-plus
```

Unlink it with `herdr plugin unlink cloudmanic.herdr-plus`.

Once the plugin is installed, the actions are available from herdr's plugin
action menu, and you can optionally bind keys — see [Keybindings](../keybindings/).

## Just the binary

If you'd rather have `herdr-plus` on your `PATH` to run directly (handy for
scripting or debugging the modes outside herdr's action plumbing), the standalone
binary is distributed on its own. This never touches your keybindings.

### Homebrew

The herdr-plus repository is its own Homebrew tap. Tap it, then install:

```bash
brew tap cloudmanic/herdr-plus https://github.com/cloudmanic/herdr-plus
brew install cloudmanic/herdr-plus/herdr-plus
```

To upgrade later:

```bash
brew upgrade cloudmanic/herdr-plus/herdr-plus
```

### Install script

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

### From source

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

### Supported platforms

Releases are cross-compiled for:

- **Operating systems:** Linux and macOS.
- **Architectures:** `amd64` (x86_64) and `arm64` (aarch64).

The install script maps `uname` output onto those tokens and refuses to run on
anything else, telling you exactly what it detected.

### Where the binary lands

The install script chooses, in order:

1. The directory in `INSTALL_DIR`, if you set it.
2. `~/.local/bin` — preferred, because it needs no `sudo`.
3. `/usr/local/bin` — the fallback, which may prompt for `sudo`.

> **Tip:** If the chosen directory isn't on your `$PATH`, the script prints the
> exact `export PATH=...` line to add to your shell rc. Without that, your shell
> won't find the `herdr-plus` command.

### Checking the version

Confirm what's installed at any time:

```bash
herdr-plus version
```

(`--version`, `-v`, and `-V` all work too.) It prints `herdr-plus` followed by
the release version.

### How standalone-binary upgrades work

Every merge to `main` auto-bumps the patch version and cuts a new GitHub Release
with cross-compiled binaries. To pull the latest:

- **Homebrew:** `brew upgrade cloudmanic/herdr-plus/herdr-plus`
- **Install script:** re-run the `curl ... | sh` one-liner.
- **From source:** `git pull` and `make install-bin` again.

## Next steps

Once the plugin is installed, optionally bind a key —
[Keybindings](../keybindings/) — then jump into the
[Quick Start](../quick-start/).
