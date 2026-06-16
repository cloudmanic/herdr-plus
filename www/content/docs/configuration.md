---
title: "Configuration"
description: "The herdr-plus config directory: herdr's managed plugin dir, the projects/, quick-actions/, and worktrees/ subdirs, the file-per-entry model, and per-repo overrides."
weight: 90
---

herdr-plus keeps its configuration in herdr's **managed plugin directory**. There's
no central config file — everything is a file per entry.

## Finding the config directory

Ask herdr where it is:

```bash
herdr plugin config-dir cloudmanic.herdr-plus
# → ~/.config/herdr/plugins/config/cloudmanic.herdr-plus
```

herdr provisions this directory for the plugin and **keeps it across uninstall and
upgrade**, so your projects and quick actions survive a reinstall.

> **Running outside herdr:** if you run the `herdr-plus` binary directly (not
> through herdr), there's no managed directory to use, so it falls back to
> `~/.config/herdr-plus/`, honoring `$XDG_CONFIG_HOME`. Inside herdr — the normal
> case — the managed directory above always wins.

## Directory layout

```text
<config-dir>/
  projects/          # one *.toml per project
    options-cafe.toml
    bevio.toml
    ...
  quick-actions/     # one *.toml per action
    github.toml
    google.toml
    ...
  worktrees/         # one *.toml per worktree auto-layout
    options-cafe.toml
    ...
```

- **`projects/`** holds your [project templates](../projects/). Each `*.toml`
  defines one project. This directory **starts empty** — the Projects browser's
  onboarding screen explains how to add your first one.
- **`quick-actions/`** holds your [quick actions](../quick-actions/). Each `*.toml`
  defines one action. This directory is **seeded with editable examples** the first
  time you open the launcher.
- **`worktrees/`** holds your [worktree auto-layouts](../worktrees/). Each `*.toml`
  defines one layout that fires when herdr creates a matching git worktree. This
  directory is **never created or seeded** — add it yourself to opt in. A layout
  is on simply by existing; delete the file to turn it off.

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
- `projects/` is never seeded. An empty directory is meaningful: it triggers the
  Projects onboarding empty-state.
- `worktrees/` is never created or seeded. It exists only if you add it, which is
  how the [worktree auto-layout](../worktrees/) feature stays opt-in.

## Per-project (per-repo) overrides

A repo can ship its own quick actions. Add a `.herdr-plus/` directory at the repo
root with one `*.toml` per action in its `quick-actions/` subdirectory:

```text
your-repo/
  .herdr-plus/
    quick-actions/
      make-build.toml
      make-test.toml
```

When you launch the Quick Actions picker from inside that repo, these actions
appear grouped under a `Project` heading above your `Global` ones. The directory is
**read-only and never auto-created** — it's read only when a repo actually provides
it. See [Quick Actions](../quick-actions/) for the full behavior.

## See also

- [Projects](../projects/) — the project file format.
- [Actions Reference](../actions/) — the action file format.
- [Examples & Cookbook](../examples/) — ready-to-copy files.
