---
title: "Projects"
description: "Projects are declarative herdr workspace templates — pick one from a full-screen fuzzy browser and herdr-plus spins up a whole workspace of tabs and panes."
weight: 50
---

**Projects** are declarative herdr workspace templates that spin up a fully
laid-out workspace with a single keypress.

## What it does

Trigger the `cloudmanic.herdr-plus.projects` action — from herdr's action menu, or
a [bound key](../keybindings/) — and herdr-plus opens a **full-screen fuzzy
browser** of your projects.

Fuzzy-find a project, press `enter` (or click it), and herdr-plus spins up a whole
workspace — every tab created, every split laid out, and every startup command
running — then drops you straight into it. Cancel with `esc` to close the browser
and return to where you were.

> **Note:** With no project files yet, the browser shows an onboarding screen
> explaining how to add your first project.

## What a project is

A **project** is a declarative herdr workspace template: a name, a description, a
working directory, and an ordered list of tabs (each with an optional startup
command, or a set of split panes). It replaces hand-written workspace shell scripts
with a simple config file.

## Where projects live

Projects live in the `projects/` subdirectory of
[herdr-plus's config directory](../configuration/), **one TOML file per project**.
The file name doesn't matter — only its contents. Find the directory with:

```bash
herdr plugin config-dir cloudmanic.herdr-plus
```

The directory **starts empty**: unlike quick-actions, it is never seeded with
examples, because an empty directory is meaningful (it triggers the onboarding
screen). To add a project, drop a `.toml` file in. To remove a project, delete its
file.

## The project schema

Here is a complete, annotated example covering every field:

```toml
# Top-level project fields.
name = "Options Cafe"                                      # required: the workspace label
description = "The main options.cafe monorepo"             # shown in the browser
group = "Cloudmanic"                                       # optional: cluster a client's projects under a heading
working_dir = "~/Development/options-cafe/options.cafe"    # ~ and $VARS expand

# Tabs open in file order. The FIRST tab reuses the workspace's root tab; the
# rest are created behind it so the first tab stays in front.

[[tabs]]
name = "claude"                                            # required: the tab label
command = "claude --dangerously-skip-permissions --chrome" # runs on startup

[[tabs]]
name = "lazygit"
command = "lazygit"

[[tabs]]
name = "terminal"                                          # no command — just an empty shell
```

### `name`

Required. The workspace label herdr-plus gives the project's workspace, and the
text you fuzzy-find in the browser.

### `description`

Optional free text shown next to the name in the browser.

### `group`

Optional. A label that clusters related projects in the browser — see
[Grouping projects by client](#grouping-projects-by-client). It has no effect on
the workspace that opens; it only changes how the browser is laid out.

### `working_dir`

The directory the workspace is rooted at. A leading `~` expands to your home
directory, and any `$VARS` in the path are expanded from your environment. An
empty (or `~`) `working_dir` defaults to your home directory, so a minimal
project still opens somewhere sensible.

> **Important:** The working directory's existence is checked when you *open* the
> project, not when the file is loaded. (The same project file might be valid on
> one machine and not another.) If the directory doesn't exist, opening the
> project fails with a clear error.

### `[[tabs]]`

A project needs **at least one** `[[tabs]]` entry. Tabs are created in file
order. Each tab has:

- `name` — **required**, the tab's label.
- `command` — optional. The startup command, run as if typed at the prompt. A
  tab with no `command` (and no panes) is just an empty shell.

The first tab reuses the workspace's root tab (it's renamed to the tab's name);
every later tab is created behind it.

## Split panes within a tab

A tab can hold up to **4 panes**. Instead of a single `command`, give the tab
`[[tabs.panes]]` entries:

```toml
[[tabs]]
name = "server"

[[tabs.panes]]
command = "php artisan serve"

[[tabs.panes]]
command = "npm run dev"
split = "down"
```

Each pane:

- `command` — optional. The command to run in that pane on startup. A pane with
  no command is an empty shell.
- `split` — how the pane is created relative to the *previous* pane in the tab:
  - `"down"` — stacked below (top/bottom).
  - `"right"` — beside it (side by side).
  - Omitted — defaults to `"down"`.

The **first pane is the tab's root**, so its `split` is ignored. Each later pane
splits off the one before it.

### `command` vs. `[[tabs.panes]]` are mutually exclusive

A tab uses *either* `command` *or* `[[tabs.panes]]`, not both. Setting both is a
config error reported when the project loads. A tab with neither is a single
empty shell.

> **Tip:** In the projects browser, split tabs are shown with a `×N` pane count
> (e.g. `server ×2`) so you can see the layout at a glance.

A tab may declare at most 4 panes; more than that is a load-time error.

## Grouping projects by client

Give a project a [`group`](#group) and the browser clusters projects that share
that label under a heading. This is built for the common case where **one client
has several projects** — tag each with the client's name and they sit together:

```toml
name = "Acme — Web"
group = "Acme Co."
working_dir = "~/Clients/acme/web"

[[tabs]]
name = "editor"
command = "spiceedit"
```

How the browser lays out:

- **Headings appear only when used.** If no project sets a `group`, the browser
  is a plain, flat list — nothing changes.
- **Named groups come first**, in case-insensitive alphabetical order by group
  name. A client's projects keep their usual name order under the heading.
- **Group-less projects** fall under a catch-all **Ungrouped** heading at the
  bottom, so a mix of grouped and ungrouped projects is always clearly labeled.
- **Search is unchanged.** Start typing and the headings drop away — you filter a
  single ranked list across every project, grouped or not.

## Adding and removing projects

The model is one file per project:

- **Add a project** — drop a new `.toml` file into the `projects/` subdirectory of
  [herdr-plus's config dir](../configuration/).
- **Remove a project** — delete its file.

Projects are sorted by name in the browser, and clustered under group headings
when any project sets a [`group`](#grouping-projects-by-client). A malformed or
invalid project file fails the whole load with a message naming the offending
file, so a config mistake surfaces loudly instead of a project silently going
missing.

## See also

- [Configuration](../configuration/) — the full directory layout.
- [Examples & Cookbook](../examples/) — ready-to-copy project files.
- [Quick Actions](../quick-actions/) — the other feature.
