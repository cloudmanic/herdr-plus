# herdr-plus

herdr-plus is an add-on for [herdr](https://herdr.dev), built as a first-class
[herdr plugin](https://herdr.dev/docs/plugins/). It adds two things:

- **[Projects](#projects)** — declarative herdr-workspace templates you fuzzy-pick
  to spin up a whole workspace (every tab and pane, every startup command) in one
  keypress.
- **[Quick Actions](#quick-actions)** — a fuzzy launcher for one-off
  actions/scripts, run in the directory you launched from.

## Install

herdr-plus is a herdr plugin (requires **herdr ≥ 0.7.0**). Installing it registers
the plugin's actions with herdr — no editing of your `config.toml`.

```bash
herdr plugin install cloudmanic/herdr-plus
```

herdr clones the repo, runs the manifest's `[[build]]` step, and registers the
actions. That step **prefers a local Go toolchain** (an exact build of the source)
and **falls back to downloading the latest prebuilt release binary**, so it works
**with or without Go**. Manage it with `herdr plugin list`,
`herdr plugin action list --plugin cloudmanic.herdr-plus`, and
`herdr plugin uninstall cloudmanic.herdr-plus`.

**Local development:** build the binary and link your checkout in place:

```bash
make build
herdr plugin link /path/to/herdr-plus     # or: make plugin-link
```

### Just the binary

If you'd rather have `herdr-plus` on your `PATH` (e.g. to run `herdr-plus version`),
prebuilt binaries are published on every release:

```bash
# Homebrew (the repo is its own tap)
brew tap cloudmanic/herdr-plus https://github.com/cloudmanic/herdr-plus
brew install cloudmanic/herdr-plus/herdr-plus

# or the install script (Linux/macOS, no Homebrew)
curl -fsSL https://raw.githubusercontent.com/cloudmanic/herdr-plus/main/install.sh | sh
```

The binary on its own doesn't register the plugin with herdr — use
`herdr plugin install` (above) for that. Every merge to `main` cuts a new release
with cross-compiled binaries.

## Configuration

herdr-plus keeps its config in herdr's managed plugin directory — find it with:

```bash
herdr plugin config-dir cloudmanic.herdr-plus
# → ~/.config/herdr/plugins/config/cloudmanic.herdr-plus
```

Inside it, `projects/` holds your [project templates](#projects) and
`quick-actions/` your [actions](#quick-actions). herdr provisions this directory
and keeps it across uninstall/upgrade. (Running the binary *outside* herdr falls
back to `~/.config/herdr-plus/`, honoring `$XDG_CONFIG_HOME`.)

## Projects

Pick a project from a full-screen fuzzy browser and herdr-plus builds its whole
workspace. Trigger it from herdr's plugin action menu, or
[bind a key](#binding-a-key) — the action is `cloudmanic.herdr-plus.projects`.

A project is one TOML file in the `projects/` subdir of
[herdr-plus's config dir](#configuration). The file name doesn't matter; add a
file to add a project, delete it to remove it. With no files there, the browser
shows an onboarding card.

```toml
name = "Options Cafe"
description = "The main options.cafe monorepo"
working_dir = "~/Development/options-cafe/options.cafe"   # ~ and $VARS expand

[[tabs]]
name = "claude"
command = "claude --dangerously-skip-permissions --chrome"

[[tabs]]
name = "lazygit"
command = "lazygit"

[[tabs]]
name = "terminal"   # no command — just an empty shell
```

Tabs open in file order. The first tab reuses the workspace's root tab; the rest
are created behind it. A tab with no `command` is just an empty shell.

### Grouping

A project may set an optional `group` to cluster related projects under a heading
in the browser (handy when one client has several). Projects sharing a `group` are
shown together; group-less ones fall under an **Ungrouped** heading. Grouping only
engages when at least one project sets a `group` — otherwise the list is plain.
Filtering ignores headings: start typing and it collapses to one ranked list.

### Split panes within a tab

A tab can hold up to **4 panes**. Instead of a single `command`, give it
`[[tabs.panes]]` entries. Each pane after the first sets `split` to `"down"`
(stacked) or `"right"` (side by side) — how it splits off the previous pane. An
omitted `split` defaults to `"down"`.

```toml
[[tabs]]
name = "server"

[[tabs.panes]]
command = "php artisan serve"

[[tabs.panes]]
command = "npm run dev"
split = "down"
```

A tab uses *either* `command` *or* `[[tabs.panes]]`, not both.

## Quick Actions

A fuzzy launcher for one-off commands. Trigger it (action
`cloudmanic.herdr-plus.quick-actions`), fuzzy-pick an action, and it runs in the
directory you launched from. Actions are TOML files in the `quick-actions/` subdir
of [herdr-plus's config dir](#configuration) (seeded with editable examples on
first run). A repo can also ship its own in `<repo>/.herdr-plus/quick-actions/`, shown
under a **Project** heading above your **Global** ones — this repo ships
`make build` / `make test` as a live example.

There are three action types:

```toml
# command (default) — runs immediately
name = "GitHub"
command = "open https://github.com"
```

```toml
# select — pick from a second fuzzy list; the choice becomes {{.Value}}
name = "Open Repo"
type = "select"
command = "open https://github.com/cloudmanic/{{.Value}}"

[[options]]
label = "Herdr Plus"
value = "herdr-plus"
```

```toml
# form — type a value that becomes {{.Value}}
name = "Search Google"
type = "form"
command = "open 'https://www.google.com/search?q={{.Value | urlquery}}'"

[form]
prompt = "Search Google for"
```

The `command` is a [Go template](https://pkg.go.dev/text/template) rendered against
the launch context: `{{.WorkDir}}` (where you launched from), `{{.SessionTitle}}`
(the workspace label), `{{.Value}}` (select/form input), and more — also exported
as `HERDR_PLUS_*` environment variables. If a command doesn't reference
`{{.Value}}`, the value is appended as a final shell-quoted argument.

## Worktree auto-layout

herdr-plus can lay a project-style tab layout into a git **worktree** the moment
herdr creates *or opens* it. When you run `herdr worktree create`/`open` (or use
herdr's right-click worktree dialog), herdr makes a fresh workspace for the
worktree and fires a `worktree.created` event (new worktree) or `worktree.opened`
event (existing one); herdr-plus catches either, finds a layout matching the
worktree's repo, and opens that layout's tabs and panes in the new workspace —
every command running — with no keypress. This is the plugin system's `[[events]]`
hook (declared in [`herdr-plugin.toml`](herdr-plugin.toml)) put to work.

Layouts live in `~/.config/herdr-plus/worktrees/`, one TOML file per layout (the
file name doesn't matter). A layout is a `repo` matcher plus the same `[[tabs]]`
format projects use:

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

- **`repo`** (required) matches the new worktree's repository name — the repo's
  basename, e.g. `options-cafe` — case-insensitively.
- **`branch`** (optional) narrows a layout to worktrees created on exactly that
  branch. When more than one layout matches, a branch-specific one wins over a
  repo-only one.
- **`[[tabs]]`** is identical to a project's tabs, including multi-pane
  `[[tabs.panes]]` splits (see [Split panes within a tab](#split-panes-within-a-tab)).

### Turning a layout on and off

The switch is simply **whether the file exists**. A layout in `worktrees/` is on;
to turn one off, delete the file (or move it out of the directory). With no files
in `worktrees/` at all, the feature is inert — every worktree fires the event, and
herdr-plus does nothing when nothing matches.

The handler's output shows up in `herdr plugin log list --plugin
cloudmanic.herdr-plus`, so you can confirm whether a layout fired.

## Binding a key

Binding keys to the actions is an optional, one-time edit to **your** herdr
`config.toml` (`~/.config/herdr/config.toml`). Add `[[keys.command]]` entries with
`type = "plugin_action"` whose `command` is the action id:

```toml
[[keys.command]]
key = "prefix+up"
type = "plugin_action"
command = "cloudmanic.herdr-plus.projects"
description = "herdr-plus: projects"

[[keys.command]]
key = "prefix+down"
type = "plugin_action"
command = "cloudmanic.herdr-plus.quick-actions"
description = "herdr-plus: quick actions"
```

Then `herdr server reload-config` (or restart herdr) and press your herdr prefix
(default `ctrl+b`) followed by the bound key.

## Building

```bash
make build     # build ./bin/herdr-plus
make test      # go test -race ./...
make vet       # go vet ./...
```

The marketing + docs site lives in `www/` (Hugo + Tailwind). Build it with
`make site`, or run it locally with live reload via `make site-dev`.
