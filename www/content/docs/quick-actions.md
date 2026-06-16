---
title: "Quick Actions Mode"
description: "The fuzzy launcher: pick an action and run it in a split. Covers global actions and per-project actions shipped in a repo's .herdr-plus directory."
weight: 60
---

Quick Actions is a fuzzy launcher. Trigger the quick-actions action (run it from
herdr's action menu, or bind it to a key like `prefix+down` — see
[Keybindings](../keybindings/)), type a few characters to filter, pick an action,
and it runs in a split pane.

## What it does

Triggering the quick-actions action opens a focused split *beneath* your current
pane and runs the picker there. The picker is a fuzzy finder over your actions:

- `↑`/`↓` (or `ctrl+p`/`ctrl+n`, or the mouse wheel) move the highlight.
- Type to filter the list.
- `enter` or a left-click runs the highlighted action.
- `esc` (or `ctrl+c`) cancels.

When you choose an action, the picker runs it in that split, then **closes its
own pane** so focus returns to where you launched from. (The launch directory is
the working directory of the pane you launched from, so actions run in the right
place.)

> **Tip:** A fast command's output would flash by before the pane closes. To hold
> the pane open so you can read it, end the command with a wait — see the
> [Actions Reference](../actions/) and [Examples](../examples/) for the
> keep-open patterns (`read </dev/tty`, a timed read, or `sleep`).

## Where global actions live

Global actions live in `~/.config/herdr-plus/quick-actions/` (honoring
`$XDG_CONFIG_HOME`), **one `*.toml` file per action**. The **first time** you run
the mode, this directory is seeded with editable example actions so you have a
working set to learn from and edit. After that the directory is left alone, so
deleting an example never makes it reappear.

Add a file to add an action; delete a file to remove it. Actions are sorted by
name in the picker.

## Per-project quick actions

A repo can ship its own quick actions. Add a `.herdr-plus/` directory at the repo
root that mirrors the global layout, and drop one `*.toml` per action into its
`quick-actions/` subdirectory — the **exact same format** as your global actions:

```text
your-repo/
  .herdr-plus/
    quick-actions/
      make-build.toml
      make-test.toml
```

When you launch the picker **from inside that repo**, its project actions appear
**grouped under a `Project` heading**, above your `Global` ones, so it's always
clear which is which. Start typing to filter and the two groups merge into a
single ranked list.

Launch from a repo with no `.herdr-plus` directory and the picker looks exactly
as before — a single, ungrouped list.

Key properties of the per-project directory:

- **Read-only and never auto-created.** It shows up only when a repo actually
  provides it. herdr-plus never writes into it.
- **Scoped to the launch directory.** Project actions appear only when you launch
  from inside the repo (the working directory of your pane).
- **Same format as global actions.** Anything you can do in a global action you
  can do in a project action.

> **Note:** This very repository ships a per-project set as a live example —
> `make build` and `make test` — so you can see the `Project` grouping in action.

## What goes in an action file

Every action — global or per-project — is a TOML file with a `name`,
`description`, `command`, and `type`. There are three types: `command`,
`select`, and `form`. The full format is documented in the
[Actions Reference](../actions/), and the variables available to a command are in
[Template Variables](../variables/).

## See also

- [Actions Reference](../actions/) — the complete action file format.
- [Template Variables](../variables/) — context you can use in a command.
- [Examples & Cookbook](../examples/) — ready-to-copy actions.
- [Configuration](../configuration/) — the on-disk layout.
