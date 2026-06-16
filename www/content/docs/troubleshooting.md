---
title: "Troubleshooting & FAQ"
description: "Concrete fixes for common herdr-plus issues: the plugin not registering, dead keybindings, config not loading, per-project actions, and template errors."
weight: 110
---

Concrete answers to the things that go wrong. Each item lists what to check, in
order.

## The plugin didn't install or its actions don't show up

1. **Is it registered?** Run `herdr plugin list` and look for
   `cloudmanic.herdr-plus`. If it's missing, re-run
   `herdr plugin install cloudmanic/herdr-plus`.
2. **Did the build step fail?** The install builds the binary (Go, or a prebuilt
   download as a fallback). Check the plugin's log for build output:
   `herdr plugin log list --plugin cloudmanic.herdr-plus`.
3. **Is herdr new enough?** herdr-plus requires **herdr ≥ 0.7.0**. Check with
   `herdr --version`.
4. **Are the actions there?** `herdr plugin action list --plugin cloudmanic.herdr-plus`
   should list `projects` and `quick-actions`.

## Nothing happens when I press the key

Work through these:

1. **Did you bind the action?** A key does nothing until you bind it. Add the
   `[[keys.command]]` entries (`type = "plugin_action"`) to your herdr
   `config.toml` — see [Keybindings](../keybindings/) — or just run the action
   from herdr's action menu instead.
2. **Did you reload herdr?** After editing `config.toml`, run
   `herdr server reload-config` (or restart herdr) so the binding goes live.
3. **Are you inside herdr?** herdr-plus runs as a herdr plugin; the actions only
   exist inside a herdr session.
4. **Are you pressing the prefix correctly?** herdr keybindings are prefixed:
   press your prefix (default `ctrl+b`), release, then press the bound key (e.g.
   `up`). See [Keybindings](../keybindings/).

## A key opens the wrong thing, or two actions collide

You manage the bindings yourself in herdr's `config.toml`, so a conflict means two
`[[keys.command]]` entries claim the same `key`. Give each action a distinct key
and reload with `herdr server reload-config`.

## My config / action / project isn't picked up

1. **Right directory?** Both live under herdr-plus's config dir — find it with
   `herdr plugin config-dir cloudmanic.herdr-plus`. Quick actions go in its
   `quick-actions/` subdir; projects go in `projects/`. Per-project quick actions
   go in a repo's `.herdr-plus/quick-actions/`. See [Configuration](../configuration/).
2. **Is the file `*.toml`?** Only files ending in `.toml` are loaded.
3. **Is the TOML valid?** A malformed or invalid file fails the **whole** load for
   that directory, with an error naming the offending file. Fix the named file.
4. **Required fields present?** Actions need a `name` and a non-empty `command`;
   `select` actions need at least one option with a label. Projects need a `name`
   and at least one `[[tabs]]` entry, and each tab needs a `name`.

## My per-project actions don't show up

1. **`.herdr-plus/quick-actions/` at the repo root?** The directory must mirror the
   global layout and sit at the root of the repo.
2. **Launched from inside the repo?** Project actions appear only when the pane's
   working directory is inside that repo — that's the launch directory herdr-plus
   uses.
3. **The directory is never auto-created.** herdr-plus won't make it for you; the
   repo has to provide it.

See [Quick Actions](../quick-actions/) for the full behavior.

## A project's working directory error

Opening a project fails with "working directory does not exist" when its
`working_dir` doesn't resolve to a real directory on this machine. The path is
checked at open time, not load time, so the same file can be valid elsewhere. Fix
the `working_dir` (remember `~` and `$VARS` expand). See [Projects](../projects/).

## My worktree didn't get its layout

Creating or opening a git worktree gave you a plain workspace instead of your
tabs. The [worktree auto-layout](../worktrees/) handler runs on herdr's
`worktree.created` and `worktree.opened` events and logs why it did or didn't act
— check the plugin log first:

```bash
herdr plugin log list --plugin cloudmanic.herdr-plus
```

Common causes:

- **The worktree wasn't created through herdr.** herdr only fires these events for
  worktrees it creates or opens itself (`herdr worktree create`/`open` or its
  right-click worktree dialog). A worktree made with plain `git worktree add` — or
  from lazygit or another tool — is invisible to herdr, so no event fires and no
  layout applies. The plugin log will have **no** `on-worktree` entry at all.
- **No file for that repo.** A layout is on only if a file for it exists in
  `worktrees/`. Confirm the file is there (not in `projects/`) and hasn't been
  deleted or moved.
- **No matching layout.** The log says `no worktree layout matches repo …`. The
  `repo` in your layout must match the worktree's repo name (its basename),
  case-insensitively.
- **A branch mismatch.** A layout with a `branch` only fires for worktrees created
  on exactly that branch. Drop the `branch` line to apply to every branch.
- **A config typo.** An invalid file fails the whole load with an error naming the
  file — fix it and create the worktree again.

## Template errors in a command

The `command` is a Go `text/template`. A bad field name or malformed `{{...}}`
produces a parse or render error when the action runs (printed to stderr).

- Check your field names against [Template Variables](../variables/) — they're
  case-sensitive (`{{.WorkDir}}`, not `{{.workdir}}`).
- Make sure braces are balanced: `{{.Value}}`, not `{{.Value}` or `{.Value}}`.

## My action's output flashed by before I could read it

The Quick Actions launcher closes itself once the command finishes. To hold it
open, end the command with a wait that reads the terminal directly (its stdin is
`/dev/null`):

```text
read _ </dev/tty            # wait for Enter, then close
read -t 30 _ </dev/tty      # ...but auto-close after 30s
sleep 5                     # just linger N seconds
```

See the [Examples & Cookbook](../examples/) for full action snippets.

## "command not found: herdr-plus"

You don't normally need the `herdr-plus` binary on your `PATH` — the plugin install
handles everything, and the actions run the plugin's own binary. You'll only see
this if you try to run `herdr-plus` at a prompt without having installed the
[optional standalone binary](../installation/#the-optional-standalone-binary).

## How do I check the version?

```bash
herdr-plus version
```

(`--version`, `-v`, and `-V` work too — this needs the optional standalone binary.)
To upgrade the plugin, re-run `herdr plugin install cloudmanic/herdr-plus`; see
[Installation](../installation/).

## Still stuck?

herdr-plus is open source. File an issue or read the source on
[GitHub](https://github.com/cloudmanic/herdr-plus).
