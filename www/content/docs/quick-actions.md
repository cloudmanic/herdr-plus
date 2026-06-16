---
title: "Quick Actions"
description: "The fuzzy launcher: pick an action and run it in the directory you launched from. Covers global actions and per-project actions shipped in a repo's .herdr-plus directory."
weight: 60
---

**Quick Actions** is a fuzzy launcher. Trigger it, type a few characters to filter,
pick an action, and it runs — in the directory you launched from.

## What it does

Trigger the `cloudmanic.herdr-plus.quick-actions` action — from herdr's action
menu, or a [bound key](../keybindings/) — and herdr-plus opens a focused launcher
over your workspace. The launcher is a fuzzy finder over your actions:

- `↑`/`↓` (or `ctrl+p`/`ctrl+n`, or the mouse wheel) move the highlight.
- Type to filter the list.
- `enter` or a left-click runs the highlighted action.
- `esc` (or `ctrl+c`) cancels.

When you choose an action, herdr-plus runs it and then **closes the launcher**,
handing focus back to where you were. The command runs in the **working directory
of the pane you launched from**, so actions act on the right place.

> **Tip:** A fast command's output would flash by before the launcher closes. To
> hold it open so you can read it, end the command with a wait — see the
> [Actions Reference](../actions/) and [Examples](../examples/) for the keep-open
> patterns (`read </dev/tty`, a timed read, or `sleep`).

## Where global actions live

Global actions live in the `quick-actions/` subdirectory of
[herdr-plus's config dir](../configuration/), **one `*.toml` file per action**. The
**first time** you open the launcher, this directory is seeded with editable
example actions so you have a working set to learn from and edit. After that the
directory is left alone, so deleting an example never makes it reappear.

Add a file to add an action; delete a file to remove it. Actions are sorted by
name in the picker.

## Per-project quick actions

A repo can ship its own quick actions. Add a `.herdr-plus/` directory at the repo
root and drop one `*.toml` per action into its `quick-actions/` subdirectory — the
**exact same format** as your global actions:

```text
your-repo/
  .herdr-plus/
    quick-actions/
      make-build.toml
      make-test.toml
```

When you launch the picker **from inside that repo**, its project actions appear
**grouped under a `Project` heading**, above your `Global` ones, so it's always
clear which is which. Start typing to filter and the two groups merge into a single
ranked list.

Launch from a repo with no `.herdr-plus` directory and the picker looks exactly as
before — a single, ungrouped list.

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
`description`, `command`, and `type`. There are three types: `command`, `select`,
and `form`. The full format is documented in the
[Actions Reference](../actions/), and the variables available to a command are in
[Template Variables](../variables/).

## See also

- [Actions Reference](../actions/) — the complete action file format.
- [Template Variables](../variables/) — context you can use in a command.
- [Examples & Cookbook](../examples/) — ready-to-copy actions.
- [Configuration](../configuration/) — the on-disk layout.
