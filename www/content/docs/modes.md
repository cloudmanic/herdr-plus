---
title: "Modes"
description: "One herdr-plus binary, multiple modes. How --mode works, the available modes, and how to invoke each."
weight: 40
---

herdr-plus is an add-on platform for [herdr](https://herdr.dev): the same binary
can be invoked many times in different herdr panes, and a `--mode` flag tells
each invocation what to do when it talks to herdr. We expect this list of modes
to grow.

## How modes work

One binary, multiple behaviors. You pick a mode with `--mode=<slug>`. With no
flag, the **default mode (`control`)** runs — it's the front door of herdr-plus,
so the bare binary lands there.

An unrecognized slug is an error (so typos fail loudly instead of silently doing
the wrong thing). For example, `herdr-plus --mode=quikactions` exits with an
`unknown mode` message.

## Available modes

| Mode | Slug | Default key | What it does |
|------|------|-------------|--------------|
| Control | `control` (default) | `prefix+up` | herdr-plus's home base — a full-screen "Herdr Plus" workspace for driving herdr. First feature: **Projects**. |
| Quick Actions | `quick-actions` | `prefix+down` | A fuzzy launcher: pick an action and run it in a split pane. |

Each mode has its own default key, so the two can be installed side by side. The
slug is also the name of the mode's per-mode config subdirectory under
`~/.config/herdr-plus/` (see [Configuration](../configuration/)).

## Invoking herdr-plus

```bash
herdr-plus                       # default mode (control)
herdr-plus --mode=quick-actions  # the fuzzy launcher
herdr-plus version               # print the version and exit
```

In day-to-day use you don't run these by hand — you bind them to a herdr key with
`herdr-plus install` and press the key instead. See [Keybindings](../keybindings/).

## How adding a mode works

herdr-plus is open source and designed to grow. Adding a mode is a small change
in the [repository](https://github.com/cloudmanic/herdr-plus): register a new
`Mode` value (giving it a slug, title, and default key) in `mode.go`, optionally
add bundled example actions under `examples/<slug>/`, and teach the launcher how
the mode should present itself. If you'd like to build one, the repo is the place
to start.

## Go deeper

- [Control Mode & Projects](../projects/) — the full guide to control mode.
- [Quick Actions](../quick-actions/) — the full guide to the launcher.
