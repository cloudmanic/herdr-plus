---
title: "Quick Start"
description: "Go from zero to a working herdr-plus keybinding in four steps: install herdr, install herdr-plus, bind a key, press it."
weight: 10
---

This is the fastest path from zero to a working herdr-plus keybinding. Four
steps, a couple of minutes.

## 1. Make sure herdr is installed and running

herdr-plus is an add-on for [herdr](https://herdr.dev). You need a working herdr
install first — follow the [herdr install guide](https://herdr.dev) — and you
need to be running inside a herdr session, because herdr-plus talks to the
running herdr server over a local socket.

> **Note:** herdr-plus only does something useful from inside herdr. If you run
> it outside herdr it can't find a pane to work with.

## 2. Install herdr-plus

Pick whichever you prefer. **Homebrew** (the repo is its own tap):

```bash
brew tap cloudmanic/herdr-plus https://github.com/cloudmanic/herdr-plus
brew install cloudmanic/herdr-plus/herdr-plus
```

**Install script** (Linux/macOS, no Homebrew):

```bash
curl -fsSL https://raw.githubusercontent.com/cloudmanic/herdr-plus/main/install.sh | sh
```

See [Installation](../installation/) for from-source builds, install-location
overrides, and how upgrades work.

## 3. Install the keybindings

`herdr-plus install` wires herdr-plus into herdr's `config.toml` as a keybinding
and reloads the running herdr server, so the binding is live immediately.

```bash
herdr-plus install                       # binds prefix+up   -> control (default mode)
herdr-plus install --mode=quick-actions  # binds prefix+down -> quick-actions
```

Each mode claims its own default key, so the two coexist. Override any key with
`--key=prefix+a`. See [Keybindings](../keybindings/) for the full story.

## 4. Press your prefix, then the key

In herdr, press your prefix (default `ctrl+b`) followed by the bound key:

- `prefix` then `up` opens **Control mode** — a full-screen "Herdr Plus"
  workspace with the Projects browser.
- `prefix` then `down` opens the **Quick Actions** launcher — a fuzzy finder in a
  split beneath your current pane.

That's it. The first time you open Quick Actions, herdr-plus seeds your config
with editable example actions so you have something to try right away.

## Next steps

- [Control Mode & Projects](../projects/) — build workspace templates.
- [Quick Actions](../quick-actions/) — the launcher and per-project actions.
- [Actions Reference](../actions/) — write your own actions.
- [Configuration](../configuration/) — where everything lives on disk.
