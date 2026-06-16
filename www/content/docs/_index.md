---
title: "Documentation"
description: "herdr-plus is a free, open-source extension platform for herdr — add Quick Actions and Projects to your terminal multiplexer."
---

herdr-plus is an add-on platform for [herdr](https://herdr.dev) — a place to build
extensions and plugins on top of herdr's terminal panes. It is free and open
source, built by [Cloudmanic Labs](https://github.com/cloudmanic/herdr-plus).

## The mental model

herdr-plus ships as a [herdr plugin](https://herdr.dev/docs/plugins/). herdr
registers it from the [`herdr-plugin.toml`](https://github.com/cloudmanic/herdr-plus/blob/main/herdr-plugin.toml)
manifest and exposes its **modes** as plugin actions you trigger from herdr's
action menu or a keybinding you choose.

We're in explore mode: the list of modes will grow over time. Today there are
two.

## Modes

Each mode is a herdr plugin action under the plugin id `cloudmanic.herdr-plus`.

| Mode | Plugin action | Slug | Recommended key | What it does |
|------|---------------|------|-----------------|--------------|
| Control | `cloudmanic.herdr-plus.control` | `control` (default) | `prefix+up` | herdr-plus's home base — a full-screen workspace for driving herdr. First feature: **Projects**. |
| Quick Actions | `cloudmanic.herdr-plus.quick-actions` | `quick-actions` | `prefix+down` | A fuzzy launcher: pick an action and run it in a split pane. |

Bind a key to either action or trigger it from herdr's action menu — the
recommended keys differ, so the two coexist side by side.

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
