---
title: "Installation"
description: "Install herdr-plus as a herdr plugin with herdr plugin install — plus the optional standalone binary, supported platforms, and how upgrades work."
weight: 20
---

herdr-plus is a [herdr plugin](https://herdr.dev/docs/plugins/). Installing it is
one command — herdr does the rest.

> **Note:** herdr-plus is an add-on for [herdr](https://herdr.dev) and requires
> **herdr ≥ 0.7.0**. Install and set up herdr first — herdr-plus does its work by
> talking to a running herdr server.

## Install the plugin

```bash
herdr plugin install cloudmanic/herdr-plus
```

herdr clones the repository, runs the manifest's build step, and registers the
plugin's actions. The build step **prefers a local Go toolchain** (an exact build
of the cloned source) and **falls back to downloading the latest prebuilt release
binary**, so it works **with or without Go**.

After it finishes, manage the plugin with:

```bash
herdr plugin list                                        # confirm it's registered
herdr plugin action list --plugin cloudmanic.herdr-plus  # see its actions
herdr plugin uninstall cloudmanic.herdr-plus             # remove it
```

> Uninstalling removes the plugin's clone and registration but **preserves your
> config directory**, so your projects and quick actions survive a reinstall.

## How upgrades work

Re-running the install command **is** the upgrade — herdr re-clones, rebuilds, and
re-registers in place:

```bash
herdr plugin install cloudmanic/herdr-plus
```

Every merge to `main` cuts a new release, so a re-install always pulls the latest.
You can pin a specific ref with `--ref <tag>` if you need a particular version.

## Local development

To hack on herdr-plus, build the binary and link your checkout in place instead of
installing from GitHub:

```bash
make build
herdr plugin link /path/to/herdr-plus     # or: make plugin-link
```

herdr then runs the freshly built `./bin/herdr-plus` for the plugin's actions.
Undo with `herdr plugin unlink cloudmanic.herdr-plus`.

## The optional standalone binary

You don't need the `herdr-plus` binary on your `PATH` — the plugin install handles
everything. But if you'd like it there (for example to run `herdr-plus version`),
prebuilt binaries are published on every release.

**Homebrew** — the repository is its own tap:

```bash
brew tap cloudmanic/herdr-plus https://github.com/cloudmanic/herdr-plus
brew install cloudmanic/herdr-plus/herdr-plus
```

**Install script** (Linux/macOS, no Homebrew). It detects your OS/arch, downloads
the matching archive from the latest GitHub Release, and drops the static binary
into place:

```bash
curl -fsSL https://raw.githubusercontent.com/cloudmanic/herdr-plus/main/install.sh | sh
```

> The standalone binary on its own does **not** register the plugin with herdr —
> use `herdr plugin install` (above) for that.

### Install-script overrides

| Variable | Default | What it does |
|----------|---------|--------------|
| `INSTALL_DIR` | `~/.local/bin` (else `/usr/local/bin`) | Where the binary is installed. |
| `VERSION` | the latest GitHub Release | Pin a specific release tag to install. |

## Supported platforms

Releases are cross-compiled for **Linux** and **macOS** on `amd64` (x86_64) and
`arm64` (aarch64). herdr-plus is not tested on Windows.

## Checking the version

```bash
herdr-plus version
```

(`--version`, `-v`, and `-V` all work too.)

## Next steps

With the plugin installed, run an action from herdr's action menu, or
[bind it to a key](../keybindings/), then jump into the
[Quick Start](../quick-start/).
