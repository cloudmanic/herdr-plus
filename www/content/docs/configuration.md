---
title: "Configuration"
description: "The herdr-plus config directory layout: projects/, quick-actions/, worktrees/, the file-per-entry model, per-repo overrides, and XDG_CONFIG_HOME."
weight: 90
---

All herdr-plus configuration lives under `~/.config/herdr-plus/`, honoring
`$XDG_CONFIG_HOME`. There's no central config file — everything is a file per
entry.

## Directory layout

```text
~/.config/herdr-plus/
  projects/          # one *.toml per project (control mode)
    options-cafe.toml
    bevio.toml
    ...
  quick-actions/     # one *.toml per action (quick-actions mode)
    github.toml
    google.toml
    ...
  worktrees/         # one *.toml per worktree auto-layout
    options-cafe.toml
    ...
```

- **`projects/`** holds your [project templates](../projects/) for control mode.
  Each `*.toml` defines one project. This directory **starts empty** — control
  mode's onboarding screen explains how to add your first one.
- **`quick-actions/`** holds your [quick actions](../quick-actions/). Each
  `*.toml` defines one action. This directory is **seeded with editable
  examples** the first time you run the mode.
- **`worktrees/`** holds your [worktree auto-layouts](../worktrees/). Each
  `*.toml` defines one layout that fires when herdr creates a matching git
  worktree. This directory is never created or seeded — add it yourself to opt in.

The per-mode subdirectory name is the mode's slug (`quick-actions`), so future
modes get their own folder. `projects/` and `worktrees/` hang directly off the
config root (not under a mode slug) because they're first-class concepts.

## The file-per-entry model

In both directories the rule is the same: **add a file to add an entry, delete a
file to remove it.** File names don't matter — only the contents. Entries are
sorted by their `name` in the UI.

> **Important:** A malformed or invalid file fails the whole load for that
> directory, with an error naming the offending file. This is deliberate: a typo
> surfaces loudly instead of an entry silently going missing.

### Seeding behavior

- `quick-actions/` is seeded with bundled examples **only** when the directory
  doesn't yet exist. Once it exists, herdr-plus leaves it alone — so deleting an
  example won't make it reappear.
- `projects/` is never seeded. An empty directory is meaningful: it triggers
  control mode's onboarding empty-state. herdr-plus only ever creates the empty
  folder for you to drop files into.

## Per-project (per-repo) overrides

A repo can ship its own quick actions. Add a `.herdr-plus/` directory at the repo
root that mirrors the global layout, with one `*.toml` per action in its
`quick-actions/` subdirectory:

```text
your-repo/
  .herdr-plus/
    quick-actions/
      make-build.toml
      make-test.toml
```

When you launch the quick-actions picker from inside that repo, these actions
appear grouped under a `Project` heading above your `Global` ones. The directory
is **read-only and never auto-created** — it's read only when a repo actually
provides it. See [Quick Actions](../quick-actions/) for the full behavior.

## `XDG_CONFIG_HOME` behavior

herdr-plus follows the XDG convention:

- If `XDG_CONFIG_HOME` is set, config lives in
  `$XDG_CONFIG_HOME/herdr-plus/`.
- Otherwise it falls back to `~/.config/herdr-plus/`, so the location is the same
  on macOS and Linux.

> **Note:** herdr's own config (`config.toml`, where you optionally add a
> keybinding for a herdr-plus plugin action) is a separate file that follows the
> same XDG rule under `herdr/` rather than `herdr-plus/`. herdr-plus never edits
> it — you own it. See [Keybindings](../keybindings/).

## See also

- [Control Mode & Projects](../projects/) — the project file format.
- [Actions Reference](../actions/) — the action file format.
- [Examples & Cookbook](../examples/) — ready-to-copy files.
