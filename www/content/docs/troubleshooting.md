---
title: "Troubleshooting & FAQ"
description: "Concrete fixes for the common herdr-plus issues: dead keybindings, command not found, key conflicts, config not loading, and template errors."
weight: 110
---

Concrete answers to the things that go wrong. Each item lists what to check, in
order.

## Nothing happens when I press the key

Work through these:

1. **Is the plugin installed?** Run `herdr plugin list` and confirm
   `cloudmanic.herdr-plus` is there, and `herdr plugin action list --plugin
   cloudmanic.herdr-plus` shows `control` and `quick-actions`. If not, install it
   — `herdr plugin install cloudmanic/herdr-plus`. See
   [Installation](../installation/).
2. **Did you bind the key?** The action menu works without any binding, but a
   *key* only fires after you add a `[[keys.command]]` entry to **your** herdr
   config (`~/.config/herdr/config.toml`) with `type = "plugin_action"` and the
   right action id. See [Keybindings](../keybindings/).
3. **Did the config reload?** After editing `config.toml`, run
   `herdr server reload-config` (or restart herdr) so the new binding is live.
4. **Are you inside herdr?** herdr-plus talks to a running herdr server over a
   local socket. If you're not in a herdr session, there's no pane to act on.
5. **Are you pressing the prefix correctly?** herdr keybindings are prefixed:
   press your prefix (default `ctrl+b`), release, then press the bound key (e.g.
   `up`). See [Keybindings](../keybindings/).

## "command not found: herdr-plus"

The binary isn't on your `$PATH`.

- The install script installs to `~/.local/bin` (preferred) or `/usr/local/bin`.
  If that directory isn't on your `$PATH`, the script printed the exact
  `export PATH=...` line to add to your shell rc — add it and restart your shell.
- You can also pass `INSTALL_DIR=...` to the script to install somewhere already
  on your PATH. See [Installation](../installation/).

> **Note:** When you install the plugin with `herdr plugin install`, herdr runs
> the action's binary for you — you don't need `herdr-plus` on your `$PATH`. The
> "command not found" error only bites when you install the standalone binary and
> type `herdr-plus` at a prompt yourself.

## My key is bound to something else

If the key you chose already runs another command, herdr's own config rules
apply — a key can only do one thing. Pick a free key in your
`[[keys.command]]` entry:

```toml
[[keys.command]]
key = "prefix+a"
type = "plugin_action"
command = "cloudmanic.herdr-plus.control"
description = "herdr-plus: control / projects"
```

After changing the `key`, run `herdr server reload-config` (or restart herdr) so
the new binding takes effect. See [Keybindings](../keybindings/).

## My config / action / project isn't picked up

1. **Right directory?** Quick actions go in
   `~/.config/herdr-plus/quick-actions/`; projects go in
   `~/.config/herdr-plus/projects/`. Per-project quick actions go in a repo's
   `.herdr-plus/quick-actions/`.
2. **`XDG_CONFIG_HOME` set?** If it is, config lives under
   `$XDG_CONFIG_HOME/herdr-plus/`, not `~/.config/herdr-plus/`. See
   [Configuration](../configuration/).
3. **Is the file `*.toml`?** Only files ending in `.toml` are loaded.
4. **Is the TOML valid?** A malformed or invalid file fails the **whole** load
   for that directory, with an error naming the offending file. herdr-plus leaves
   the pane open so you can read the error. Fix the named file.
5. **Required fields present?** Actions need a `name` and a non-empty `command`;
   `select` actions need at least one option with a label. Projects need a `name`
   and at least one `[[tabs]]` entry, and each tab needs a `name`.

## My per-project actions don't show up

1. **`.herdr-plus/quick-actions/` at the repo root?** The directory must mirror
   the global layout and sit at the root of the repo.
2. **Launched from inside the repo?** Project actions appear only when the pane's
   working directory is inside that repo — that's the launch directory
   herdr-plus uses.
3. **The directory is never auto-created.** herdr-plus won't make it for you; the
   repo has to provide it.

See [Quick Actions](../quick-actions/) for the full behavior.

## A project's working directory error

Opening a project fails with "working directory does not exist" when its
`working_dir` doesn't resolve to a real directory on this machine. The path is
checked at open time, not load time, so the same file can be valid elsewhere.
Fix the `working_dir` (remember `~` and `$VARS` expand). See
[Control Mode & Projects](../projects/).

## Template errors in a command

The `command` is a Go `text/template`. A bad field name or malformed `{{...}}`
produces a parse or render error when the action runs (printed to stderr).

- Check your field names against [Template Variables](../variables/) — they're
  case-sensitive (`{{.WorkDir}}`, not `{{.workdir}}`).
- Make sure braces are balanced: `{{.Value}}`, not `{{.Value}` or `{.Value}}`.

## My action's output flashed by before I could read it

The quick-actions pane closes itself once the command finishes. To hold it open,
end the command with a wait that reads the terminal directly (its stdin is
`/dev/null`):

```text
read _ </dev/tty            # wait for Enter, then close
read -t 30 _ </dev/tty      # ...but auto-close after 30s
sleep 5                     # just linger N seconds
```

See the [Examples & Cookbook](../examples/) for full action snippets.

## How do I check the version?

```bash
herdr-plus version
```

(`--version`, `-v`, and `-V` work too.) To upgrade, see
[Installation](../installation/).

## Still stuck?

herdr-plus is open source. File an issue or read the source on
[GitHub](https://github.com/cloudmanic/herdr-plus).
