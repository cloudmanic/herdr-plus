---
title: "Keybindings"
description: "How herdr-plus install binds a herdr key to a mode: per-mode defaults, overriding the key, idempotency, and how to trigger an action."
weight: 30
---

You launch herdr-plus by pressing a herdr keybinding. The `herdr-plus install`
command wires those bindings into herdr for you.

## What `herdr-plus install` does

A bare `herdr-plus install` installs **every mode at once**, each on its own
default key (control → `prefix+up`, quick-actions → `prefix+down`). For each one
it adds a `[[keys.command]]` entry to herdr's `config.toml`, then reloads the
running herdr server so the bindings are live immediately. Pass `--mode` to
install just a single mode. One entry looks like this:

```toml
# herdr-plus — added by `herdr-plus install`
[[keys.command]]
key = "prefix+up"
type = "shell"
command = "'/absolute/path/to/herdr-plus' --mode=control"
description = "herdr-plus: control"
```

A few important details:

- **It binds the absolute path of the binary you invoked.** That means the
  keybinding works no matter where herdr-plus lives or what your current
  directory is. If the binary was reached through a symlink, the symlink is
  resolved to its real target first.
- **It writes to herdr's config.** That's `$XDG_CONFIG_HOME/herdr/config.toml`
  if `XDG_CONFIG_HOME` is set, otherwise `~/.config/herdr/config.toml`. The file
  (and its directory) are created if they don't exist.
- **It reloads herdr.** After writing, it runs `herdr server reload-config` so
  you don't have to restart herdr. If the reload fails it tells you to run that
  command yourself or restart herdr.

### It's idempotent

Running `install` again for the same mode won't duplicate the binding. herdr-plus
detects an existing binding that runs the exact same command (same binary, same
`--mode`) and simply reports where it lives instead of adding another. Because
each mode's command carries its own `--mode` flag, the two modes never collide
with each other — installing `control` doesn't trip over an existing
`quick-actions` binding.

### It refuses to clobber other bindings

If the key you're asking for is already bound to something *else*, `install`
stops and tells you what's there, suggesting you pick a different key with
`--key`. It never overwrites a binding it didn't create.

## Per-mode default keys

Each mode claims its own conventional key, so the two can be installed side by
side without conflicting:

| Mode | Slug | Default key |
|------|------|-------------|
| Control | `control` | `prefix+up` |
| Quick Actions | `quick-actions` | `prefix+down` |

When you run `install` for a single mode without `--key`, it uses that mode's
default.

## Installing both, side by side

A bare `install` does this in one shot — it walks every mode and binds each to
its default key:

```bash
herdr-plus install   # prefix+up -> control AND prefix+down -> quick-actions
```

Now `prefix+up` opens Control mode and `prefix+down` opens Quick Actions. (You
can still install them one at a time with `--mode` if you only want one.)

## Overriding the key

Pass `--key` to bind any herdr key you like, for any mode:

```bash
herdr-plus install --key=prefix+a                       # control on prefix+a
herdr-plus install --mode=quick-actions --key=prefix+space
```

If the chosen key is taken, `install` will refuse and ask you to pick another.

## The herdr prefix, and triggering an action

herdr keybindings are *prefixed*: you press your herdr prefix (default
`ctrl+b`), release it, then press the bound key. So to launch a mode bound to
`prefix+up`:

> Press `ctrl+b`, then press `up`.

The `prefix+` part of the key name is herdr's placeholder for "whatever your
prefix is" — it isn't the literal text `prefix`. After a successful install,
herdr-plus prints the exact reminder, e.g. *"Press your prefix, then up, to
launch."*

## See also

- [Modes](../modes/) — the mode concept and the `--mode` flag.
- [Installation](../installation/) — getting the binary onto your PATH.
- [Troubleshooting](../troubleshooting/) — what to check when a key does nothing.
