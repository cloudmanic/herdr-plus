---
title: "Documentation"
description: "herdr-plus is a free, open-source extension platform for herdr — add Quick Actions and Projects to your terminal multiplexer."
---

herdr-plus is an add-on platform for [herdr](https://herdr.dev) — a place to build
extensions and plugins on top of herdr's terminal panes. It is free and open
source, built by [Cloudmanic Labs](https://github.com/cloudmanic/herdr-plus).

## The mental model

herdr-plus ships as a single binary. The same binary can run in different
**modes**, and each mode decides what to do when it talks to herdr. You bind a
mode to a herdr keybinding, press your prefix plus that key, and the mode springs
to life inside herdr.

We're in explore mode: the list of modes will grow over time. Today there are
two.

## Modes

Pick a mode with `--mode=<slug>`. With no flag, the default mode (`control`)
runs.

| Mode | Slug | Default key | What it does |
|------|------|-------------|--------------|
| Control | `control` (default) | `prefix+up` | herdr-plus's home base — a full-screen workspace for driving herdr. First feature: **Projects**. |
| Quick Actions | `quick-actions` | `prefix+down` | A fuzzy launcher: pick an action and run it in a split pane. |

Each mode has its own default key, so the two can be installed side by side.

## Where to start

- **[Quick Start](quick-start/)** — the fastest path from zero to a working
  keybinding.
- **[Installation](installation/)** — every install method (Homebrew, install
  script, from source) and how upgrades work.
- **[Control Mode & Projects](projects/)** — declarative workspace templates that
  spin up a whole herdr workspace of tabs and panes.
- **[Quick Actions](quick-actions/)** — the fuzzy launcher and per-project
  actions.

If you just want the reference, jump to
[Keybindings](keybindings/), [Modes](modes/), the
[Actions Reference](actions/), [Template Variables](variables/),
[Configuration](configuration/), the [Examples & Cookbook](examples/), or
[Troubleshooting](troubleshooting/).
