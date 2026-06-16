---
title: "Quick Start"
description: "Go from zero to a working herdr-plus plugin in four steps: install herdr, install the plugin, run an action, and optionally bind a key."
weight: 10
---

This is the fastest path from zero to running herdr-plus. Four steps, a couple of
minutes.

## 1. Make sure herdr is installed and running

herdr-plus is a plugin for [herdr](https://herdr.dev) and needs **herdr ≥ 0.7.0**.
Follow the [herdr install guide](https://herdr.dev/docs/install/), and make sure
you're inside a herdr session — herdr-plus does its work by talking to the running
herdr server.

## 2. Install the plugin

```bash
herdr plugin install cloudmanic/herdr-plus
```

herdr clones the repo, builds it (with or without Go), and registers the plugin's
actions. That's the whole install — see [Installation](../installation/) for local
development, the optional standalone binary, and how upgrades work.

## 3. Run an action

herdr-plus registers two actions. You can run either one straight from **herdr's
action menu** — no configuration required:

- `cloudmanic.herdr-plus.projects` opens the **Projects** browser — a full-screen
  fuzzy picker. With no project files yet, it shows an onboarding card explaining
  how to add your first one.
- `cloudmanic.herdr-plus.quick-actions` opens the **Quick Actions** launcher — a
  fuzzy finder over your actions. The first time you open it, herdr-plus seeds your
  config with editable example actions so you have something to try right away.

## 4. (Optional) Bind a key

If you'd rather not open the action menu each time, bind each action to a herdr
key. Add two `[[keys.command]]` entries with `type = "plugin_action"` to your herdr
`config.toml`:

```toml
[[keys.command]]
key = "prefix+up"
type = "plugin_action"
command = "cloudmanic.herdr-plus.projects"
description = "herdr-plus: projects"

[[keys.command]]
key = "prefix+down"
type = "plugin_action"
command = "cloudmanic.herdr-plus.quick-actions"
description = "herdr-plus: quick actions"
```

Then reload with `herdr server reload-config` (or restart herdr). Now press your
prefix (default `ctrl+b`), release it, and press the bound key — `up` for Projects,
`down` for Quick Actions. See [Keybindings](../keybindings/) for the full story.

## Next steps

- [Projects](../projects/) — build workspace templates.
- [Quick Actions](../quick-actions/) — the launcher and per-project actions.
- [Actions Reference](../actions/) — write your own actions.
- [Configuration](../configuration/) — where everything lives on disk.
