---
title: "Worktree Auto-Layout"
description: "Automatically lay a project-style tab layout into a git worktree the moment herdr creates it, driven by the plugin's worktree.created event — with a per-layout on/off switch."
weight: 65
---

herdr-plus can open a project-style tab layout **automatically** when herdr
creates a git worktree — no keypress, no picker. It's the plugin system's
[`[[events]]`](https://herdr.dev/docs/plugins/) hook put to work: herdr-plus
subscribes to the `worktree.created` event and fills the new worktree's workspace
for you.

## How it works

When you run `herdr worktree create` (or `herdr worktree open`), herdr:

1. Creates the git worktree.
2. Makes a fresh herdr **workspace** for it, rooted at the worktree's directory.
3. Fires a `worktree.created` event.

herdr-plus catches that event, looks at the worktree's **repo** (and branch),
finds a matching layout, and opens that layout's tabs and panes in the workspace
herdr just made — running every startup command. Because herdr has already
created the workspace and its first tab, herdr-plus only has to fill it in.

This reuses the exact same tab/pane model as [Projects](../projects/), so a
worktree layout can do everything a project tab can, including multi-pane splits.

## Configuring a layout

Layouts live in `~/.config/herdr-plus/worktrees/` (honoring `$XDG_CONFIG_HOME`),
one TOML file per layout — the file name doesn't matter. A layout is a `repo`
matcher plus an ordered list of `[[tabs]]`:

```toml
repo = "options-cafe"          # matches the worktree's repo name (case-insensitive)

[[tabs]]
name = "claude"
command = "claude --dangerously-skip-permissions --chrome"

[[tabs]]
name = "lazygit"
command = "lazygit"

[[tabs]]
name = "terminal"              # no command — just an empty shell
```

Create a worktree of `options-cafe` and you land in a workspace with three tabs —
`claude` running, `lazygit` running, and an empty `terminal` — every time.

## Turning the feature on and off

There are two layers of "on":

1. **The directory itself is opt-in.** With no `worktrees/` directory — or an
   empty one — herdr-plus does nothing. The event still fires for every worktree;
   herdr-plus just has no layout to apply, so worktree creation is unchanged.

2. **Each layout has its own switch.** A layout deploys when `enabled` is omitted
   or set to `true`. Set **`enabled = false`** to keep the file on disk but stop it
   firing — creating a worktree of that repo then just makes a plain workspace,
   exactly as if the layout weren't there. It's the clean way to pause a layout
   without deleting your tab list.

```toml
repo = "options-cafe"
enabled = false                # keep the layout, but don't deploy it

[[tabs]]
name = "claude"
command = "claude"
```

A disabled layout is still validated on load (so a typo can't hide behind
`enabled = false`), and it never suppresses another matching layout — if you have
an enabled repo-only layout and a disabled branch-specific one, the enabled one
still wins.

## Matching rules

- **`repo`** (required) is matched case-insensitively against the new worktree's
  repository name (the repo's basename, e.g. `options-cafe`). It also matches the
  basename of the repo's root path, so it works regardless of how the worktree was
  created.
- **`branch`** (optional) narrows a layout to worktrees created on exactly that
  branch (case-insensitive). Leave it off to apply to every branch of the repo.
- When **more than one enabled layout matches** the same worktree, a
  branch-specific layout wins over a repo-only one; otherwise the first by file
  name is used.

```toml
# A branch-specific layout: only worktrees on the "release" branch get this one.
repo = "options-cafe"
branch = "release"

[[tabs]]
name = "deploy"
command = "./scripts/release.sh"
```

## Tabs and split panes

The `[[tabs]]` format is identical to a project's. A tab can run a single
`command`, or hold up to four panes via `[[tabs.panes]]` with `split = "down"` or
`"right"`. See [Split panes within a tab](../projects/#split-panes-within-a-tab)
for the full vocabulary.

## When nothing matches

Every worktree creation fires the event, but if no enabled layout matches the
repo, herdr-plus does nothing — the feature is opt-in and silent. With no
`worktrees/` directory at all, it's simply inert.

## Where it runs

The handler is the plugin's `worktree.created` event, declared in
`herdr-plugin.toml`. herdr runs it for you (you never invoke it by hand), and its
output is captured in the plugin log:

```bash
herdr plugin log list --plugin cloudmanic.herdr-plus
```

A line like `applied worktree layout "options-cafe.toml" to repo "options-cafe"`
confirms a layout fired. `no worktree layout matches repo …` means nothing did,
and `worktree layout "options-cafe.toml" matches repo "options-cafe" but is
disabled` means a layout matched but you switched it off.

## See also

- [Projects](../projects/) — the on-demand cousin: pick a project and spin up its
  workspace by hand.
- [Configuration](../configuration/) — where herdr-plus config lives.
- [Troubleshooting](../troubleshooting/) — what to check when a layout doesn't fire.
