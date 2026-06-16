---
title: "Keybindings"
description: "How to bind a herdr key to a herdr-plus plugin action: the recommended per-mode keys, choosing your own key, reloading herdr, and triggering an action from the action menu."
weight: 30
---

herdr-plus ships as a [herdr plugin](https://herdr.dev/docs/plugins/). Once the
plugin is installed (see [Installation](../installation/)), herdr registers its
two **plugin actions** and you can trigger them straight from herdr's plugin
action menu. Binding a key to an action is an optional, one-time convenience —
it lives in **your own** herdr config, not in herdr-plus.

## The plugin actions

Each mode is exposed as a herdr plugin action under the plugin id
`cloudmanic.herdr-plus`. You reference an action by its fully-qualified id:

| Mode | Plugin action | Recommended key |
|------|---------------|-----------------|
| Control | `cloudmanic.herdr-plus.control` | `prefix+up` |
| Quick Actions | `cloudmanic.herdr-plus.quick-actions` | `prefix+down` |

You can confirm the plugin and its actions are registered with:

```bash
herdr plugin list
herdr plugin action list --plugin cloudmanic.herdr-plus
```

## Binding a key

Binding a key to an action is a one-time edit to **your** herdr config —
`$XDG_CONFIG_HOME/herdr/config.toml` if `XDG_CONFIG_HOME` is set, otherwise
`~/.config/herdr/config.toml`. herdr-plus never edits this file for you; you own
it.

Add a `[[keys.command]]` entry with `type = "plugin_action"` whose `command` is
the action id. The recommended convention is `prefix+up` for control and
`prefix+down` for quick-actions, side by side:

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

You can add just one of these, or both. The two modes use different keys, so they
coexist happily.

## Reloading herdr

After editing your `config.toml`, make the new binding live with:

```bash
herdr server reload-config
```

(or restart herdr). Until you reload, herdr is still running the previous config
and the key won't do anything.

## Choosing your own key

`prefix+up` and `prefix+down` are recommendations, not requirements — the
`key` field takes any herdr key you like. To put control on `prefix+a` instead,
just change the `key`:

```toml
[[keys.command]]
key = "prefix+a"
type = "plugin_action"
command = "cloudmanic.herdr-plus.control"
description = "herdr-plus: control / projects"
```

Pick a key that isn't already bound to something else in your herdr config. If a
key is already taken, herdr's own config rules apply — choose a free one.

## Triggering without a key

You don't have to bind anything at all. Both actions are always available from
herdr's **plugin action menu**, so you can run control or quick-actions
on demand without touching your config. A key binding is purely a shortcut for
the actions you reach for often.

## The herdr prefix

herdr keybindings are *prefixed*: you press your herdr prefix (default
`ctrl+b`), release it, then press the bound key. So to launch a mode bound to
`prefix+up`:

> Press `ctrl+b`, then press `up`.

The `prefix+` part of the key name is herdr's placeholder for "whatever your
prefix is" — it isn't the literal text `prefix`.

## See also

- [Modes](../modes/) — the mode concept and the plugin actions.
- [Installation](../installation/) — installing the plugin (and the standalone binary).
- [Troubleshooting](../troubleshooting/) — what to check when a key does nothing.
