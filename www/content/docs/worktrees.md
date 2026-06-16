---
title: "Worktree Auto-Layout"
description: "Automatically lay a project-style tab layout into a git worktree the moment herdr creates or opens it, driven by the plugin's worktree.created and worktree.opened events."
weight: 65
---

herdr-plus can open a project-style tab layout **automatically** when herdr
creates or opens a git worktree — no keypress, no picker. It's the plugin
system's [`[[events]]`](https://herdr.dev/docs/plugins/) hook put to work:
herdr-plus subscribes to the `worktree.created` and `worktree.opened` events and
fills the worktree's workspace for you.

## How it works

When you run `herdr worktree create` (or `herdr worktree open`, or use herdr's
right-click worktree dialog), herdr:

1. Creates or opens the git worktree.
2. Makes a fresh herdr **workspace** for it, rooted at the worktree's directory.
3. Fires a `worktree.created` event (for a new worktree) or `worktree.opened`
   event (for an existing one).

herdr-plus catches either event, looks at the worktree's **repo** (and branch),
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

## Turning a layout on and off

The switch is simply **whether the file exists**:

- **On:** a `*.toml` file in `worktrees/` for the repo. Create a worktree of that
  repo and the layout opens.
- **Off:** delete the file (or move it out of `worktrees/`). Worktrees of that repo
  then open as a plain workspace, unchanged.

With no `worktrees/` directory — or an empty one — herdr-plus does nothing at all.
The event still fires for every worktree; herdr-plus just has no layout to apply.
There's no separate enable/disable flag to keep in sync: the file's presence *is*
the switch.

## Matching rules

- **`repo`** (required) is matched case-insensitively against the new worktree's
  repository name (the repo's basename, e.g. `options-cafe`). It also matches the
  basename of the repo's root path, so it works regardless of how the worktree was
  created.
- **`branch`** (optional) narrows a layout to worktrees created on exactly that
  branch (case-insensitive). Leave it off to apply to every branch of the repo.
- When **more than one layout matches** the same worktree, a branch-specific
  layout wins over a repo-only one; otherwise the first by file name is used.

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

Every worktree creation fires the event, but if no layout matches the repo,
herdr-plus does nothing — the feature is opt-in and silent. With no `worktrees/`
directory at all, it's simply inert.

## Where it runs

The handler is wired to the plugin's `worktree.created` and `worktree.opened`
events, declared in `herdr-plugin.toml`. herdr runs it for you (you never invoke
it by hand), and its output is captured in the plugin log:

```bash
herdr plugin log list --plugin cloudmanic.herdr-plus
```

A line like `applied worktree layout "options-cafe.toml" to repo "options-cafe"`
confirms a layout fired. `no worktree layout matches repo …` means nothing did —
either there's no file for that repo, or its `repo`/`branch` didn't match.

## See also

- [Projects](../projects/) — the on-demand cousin: pick a project and spin up its
  workspace by hand.
- [Configuration](../configuration/) — where herdr-plus config lives.
- [Troubleshooting](../troubleshooting/) — what to check when a layout doesn't fire.
