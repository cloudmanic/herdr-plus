---
title: "Documentation"
description: "herdr-plus is a free, open-source herdr plugin that adds Quick Actions and Projects to your terminal multiplexer."
---

herdr-plus is an add-on for [herdr](https://herdr.dev), built as a first-class
[herdr plugin](https://herdr.dev/docs/plugins/). It is free and open source, built
by [Cloudmanic Labs](https://github.com/cloudmanic/herdr-plus).

## The mental model

You install herdr-plus once with `herdr plugin install`. herdr clones the repo,
builds it, and registers the plugin's **actions** with herdr. From then on you run
those actions — from herdr's action menu, or from a key you bind to them — and they
spring to life inside herdr.

Two actions ship today:

| Action id | What it does |
|-----------|--------------|
| `cloudmanic.herdr-plus.projects` | Opens a full-screen fuzzy browser of your **Projects** — declarative templates that spin up a whole herdr workspace. |
| `cloudmanic.herdr-plus.quick-actions` | Opens the **Quick Actions** launcher — a fuzzy finder that runs a one-off action in the directory you launched from. |

Everything herdr-plus does is driven by plain TOML files you own, kept in herdr's
managed plugin config directory. We expect the list of features to grow.

## Where to start

- **[Quick Start](quick-start/)** — the fastest path from zero to a working
  keybinding.
- **[Installation](installation/)** — installing the plugin, the optional
  standalone binary, and how upgrades work.
- **[Projects](projects/)** — declarative workspace templates that spin up a whole
  herdr workspace of tabs and panes.
- **[Quick Actions](quick-actions/)** — the fuzzy launcher and per-project actions.
- **[Worktree Auto-Layout](worktrees/)** — auto-open a tab layout whenever herdr
  creates a matching git worktree, with a per-layout on/off switch.

If you just want the reference, jump to
[Keybindings](keybindings/), the
[Actions Reference](actions/), [Template Variables](variables/),
[Configuration](configuration/), the [Examples & Cookbook](examples/), or
[Troubleshooting](troubleshooting/).
