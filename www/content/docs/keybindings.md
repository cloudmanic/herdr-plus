---
title: "Keybindings"
description: "Bind herdr keys to herdr-plus's plugin actions with [[keys.command]] entries (type = plugin_action), or just run them from herdr's action menu."
weight: 30
---

herdr-plus registers two herdr **actions**:

| Action id | Opens |
|-----------|-------|
| `cloudmanic.herdr-plus.projects` | The Projects browser |
| `cloudmanic.herdr-plus.quick-actions` | The Quick Actions launcher |

You can run either one **from herdr's action menu** without binding anything. If
you'd rather trigger them with a keystroke, bind each to a key.

## Binding a key

Keybindings are a one-time edit to **your** herdr `config.toml` — herdr-plus never
touches it. Add a `[[keys.command]]` entry with `type = "plugin_action"` whose
`command` is the action id:

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

The keys above (`prefix+up` / `prefix+down`) are just a convention — bind whatever
you like.

### Where the config lives

herdr reads `$XDG_CONFIG_HOME/herdr/config.toml` if `XDG_CONFIG_HOME` is set,
otherwise `~/.config/herdr/config.toml`. After editing it, reload so the bindings
go live:

```bash
herdr server reload-config   # or just restart herdr
```

## The herdr prefix

herdr keybindings are *prefixed*: you press your herdr prefix (default `ctrl+b`),
release it, then press the bound key. So to launch the action bound to `prefix+up`:

> Press `ctrl+b`, then press `up`.

The `prefix+` part of the key name is herdr's placeholder for "whatever your prefix
is" — it isn't the literal text `prefix`.

## See also

- [Installation](../installation/) — installing the plugin.
- [Quick Start](../quick-start/) — the four-step path from zero.
- [Troubleshooting](../troubleshooting/) — what to check when a key does nothing.
