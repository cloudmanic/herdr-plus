---
title: "Quick Start"
description: "Go from zero to a working herdr-plus plugin in four steps: install herdr, install the plugin, bind a key (optional), trigger an action."
weight: 10
---

This is the fastest path from zero to a working herdr-plus. Four
steps, a couple of minutes.

## 1. Make sure herdr is installed and running

herdr-plus is an add-on for [herdr](https://herdr.dev) and ships as a herdr
plugin, so it needs **herdr ≥ 0.7.0**. You need a working herdr install first —
follow the [herdr install guide](https://herdr.dev) — and you need to be running
inside a herdr session, because herdr-plus talks to the running herdr server over
a local socket.

> **Note:** herdr-plus only does something useful from inside herdr. If you run
> it outside herdr it can't find a pane to work with.

## 2. Install the plugin

herdr clones the repo, runs the manifest's build step to compile the binary
(it needs a Go toolchain), and registers the `control` and `quick-actions`
actions:

```bash
herdr plugin install cloudmanic/herdr-plus
```

Confirm it registered:

```bash
herdr plugin list
herdr plugin action list --plugin cloudmanic.herdr-plus
```

See [Installation](../installation/) for local-development linking, the
standalone binary (Homebrew / install script), and how upgrades work.

## 3. Bind a key (optional)

The plugin's actions are already available from herdr's plugin action menu — no
config needed. For a shortcut, add a `[[keys.command]]` entry to **your** herdr
config (`~/.config/herdr/config.toml`) with `type = "plugin_action"`. The
recommended convention binds `prefix+up` to control and `prefix+down` to
quick-actions:

```toml
[[keys.command]]
key = "prefix+up"
type = "plugin_action"
command = "cloudmanic.herdr-plus.control"
description = "herdr-plus: control / projects"

[[keys.command]]
key = "prefix+down"
type = "plugin_action"
command = "cloudmanic.herdr-plus.quick-actions"
description = "herdr-plus: quick actions"
```

Then reload herdr so the bindings are live:

```bash
herdr server reload-config
```

See [Keybindings](../keybindings/) for the full story, including choosing your
own keys.

## 4. Trigger an action

Run either action from herdr's plugin action menu, or — if you bound keys above —
press your prefix (default `ctrl+b`) followed by the bound key:

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
