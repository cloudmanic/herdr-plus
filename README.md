# herdr-plus

herdr-plus is an add-on platform for [herdr](https://herdr.dev) — a place to build
extensions and plugins on top of herdr's terminal panes. It ships as a
[herdr plugin](https://herdr.dev/docs/plugins/): herdr registers it from the
[`herdr-plugin.toml`](herdr-plugin.toml) manifest and exposes its **modes** as
plugin actions you trigger from a keybinding or herdr's action menu.

We're in explore mode: the list of modes will grow over time.

## Modes

Each mode is a herdr plugin action under the plugin id `cloudmanic.herdr-plus`.

| Mode | Plugin action | Slug | What it does |
|------|---------------|------|--------------|
| Control | `cloudmanic.herdr-plus.control` | `control` (default) | herdr-plus's home base — a full-screen workspace for driving herdr. First feature: **Projects**. |
| Quick Actions | `cloudmanic.herdr-plus.quick-actions` | `quick-actions` | A fuzzy launcher: pick an action and run it in a split. |

The same actions back the binary directly, which is handy for scripting and
debugging the modes outside herdr's action plumbing:

```bash
herdr-plus                       # default mode (control)
herdr-plus --mode=quick-actions  # the fuzzy launcher
herdr-plus version               # print the version and exit
```

## Control mode & Projects

Triggering the control action (bind it to `prefix+up`, or run it from herdr's
action menu — see [Installing](#installing)) opens a brand-new, full-screen herdr
workspace titled **Herdr Plus** with a `projects` tab, and runs the projects
browser there. This is control mode — over time it will gain more features; today
it has Projects.

A **project** is a declarative herdr workspace template: a name, a description, a
working directory, and an ordered list of tabs (each with an optional startup
command). Fuzzy-find a project, press `enter`, and herdr-plus spins up a whole
workspace — every tab created and every command running — then closes the ephemeral
"Herdr Plus" workspace. It replaces hand-written workspace shell scripts with simple
config files.

Projects live in `~/.config/herdr-plus/projects/` (honoring `$XDG_CONFIG_HOME`), one
TOML file per project (the file name doesn't matter):

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

Tabs open in file order; the first tab reuses the workspace's root tab and the rest
are created behind it. A tab with no `command` is just an empty shell. With no
project files yet, control mode shows an onboarding screen explaining all of this.

### Grouping projects

A project may set an optional `group` — a label that clusters related projects in
the browser. It is handy when one client has several projects:

```toml
name = "Acme — Web"
group = "Acme Co."
working_dir = "~/Clients/acme/web"

[[tabs]]
name = "editor"
command = "spiceedit"
```

Projects that share a `group` are shown together under that heading, in
case-insensitive alphabetical order by group name. Any project without a `group`
falls under a catch-all **Ungrouped** heading at the bottom. Grouping only kicks
in when at least one project sets a `group` — if none do, the browser stays a
plain, heading-less list exactly as before. Filtering is unchanged: start typing
and the headings drop away to a single ranked list.

### Split panes within a tab

A tab can hold up to **4 panes**. Instead of a single `command`, give the tab
`[[tabs.panes]]` entries. Each pane after the first sets `split` to `"down"`
(stacked, top/bottom) or `"right"` (side by side) — the direction it splits off the
previous pane. An omitted `split` defaults to `"down"`.

```toml
[[tabs]]
name = "server"

[[tabs.panes]]
command = "php artisan serve"

[[tabs.panes]]
command = "npm run dev"
split = "down"
```

A tab uses *either* `command` *or* `[[tabs.panes]]`, not both. In the projects list,
split tabs are shown with a `×N` pane count (e.g. `server ×2`).

## Worktree auto-layout

herdr-plus can lay a project-style tab layout into a git **worktree** the moment
herdr creates it. When you run `herdr worktree create` (or open one), herdr makes a
fresh workspace for the worktree and fires a `worktree.created` event; herdr-plus
catches it, finds a layout matching the worktree's repo, and opens that layout's
tabs and panes in the new workspace — every command running — with no keypress.
This is the plugin system's `[[events]]` hook (declared in
[`herdr-plugin.toml`](herdr-plugin.toml)) put to work.

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

With no files in `worktrees/`, the feature is inert — every worktree fires the
event, and herdr-plus does nothing when nothing matches. The handler's output
shows up in `herdr plugin log list --plugin cloudmanic.herdr-plus`.

## Installing

herdr-plus is a herdr plugin (requires **herdr ≥ 0.7.0**). Installing it registers
the plugin's actions with herdr — no editing of your `config.toml` and no separate
keybinding installer.

**As a herdr plugin** (recommended). herdr clones this repo, runs the manifest's
`[[build]]` step to compile the binary, and registers the `control` and
`quick-actions` actions. Needs a Go toolchain on the machine:

```bash
herdr plugin install cloudmanic/herdr-plus
```

**For local development**, link a checkout in place (build the binary first so the
actions have something to run):

```bash
make build
herdr plugin link /path/to/herdr-plus
```

From inside a checkout, `make plugin-link` does both steps in one shot.

Manage it with the usual plugin commands — `herdr plugin list`,
`herdr plugin action list --plugin cloudmanic.herdr-plus`,
`herdr plugin uninstall cloudmanic.herdr-plus` (or `unlink` for a linked checkout).

### Just the binary (Homebrew / install script)

If you'd rather have `herdr-plus` on your `PATH` to run directly, the standalone
binary is still distributed on its own:

```bash
# Homebrew (the repo is its own tap)
brew tap cloudmanic/herdr-plus https://github.com/cloudmanic/herdr-plus
brew install cloudmanic/herdr-plus/herdr-plus

# or the install script (Linux/macOS, no Homebrew)
curl -fsSL https://raw.githubusercontent.com/cloudmanic/herdr-plus/main/install.sh | sh
```

Every merge to `main` auto-bumps the patch version and cuts a new GitHub Release
with cross-compiled binaries; `brew upgrade` / re-running the install script pulls
the latest.

## Binding a key

The plugin registers the actions; binding a key to one is an optional, one-time
edit to **your** herdr `config.toml` (`~/.config/herdr/config.toml`). Add a
`[[keys.command]]` entry with `type = "plugin_action"` whose `command` is the
fully-qualified action id:

```toml
[[keys.command]]
key = "prefix+up"
type = "plugin_action"
command = "cloudmanic.herdr-plus.control"
description = "herdr-plus: control / projects"

[[keys.command]]
key = "prefix+down"
type = "plugin_action"
command = "cloudmanic.herdr-plus.quick-actions"
description = "herdr-plus: quick actions"
```

Then `herdr server reload-config` (or restart herdr) and press your herdr prefix
(default `ctrl+b`) followed by the bound key. You can also run any action from
herdr's plugin action menu without binding a key at all.

## Configuration

All config lives under `~/.config/herdr-plus/` (honoring `$XDG_CONFIG_HOME`):

```
~/.config/herdr-plus/
  projects/          # one *.toml per project (control mode)
    options-cafe.toml
    bevio.toml
    ...
  quick-actions/     # one *.toml per action (quick-actions mode)
    github.toml
    google.toml
    ...
```

For **quick-actions**, each `*.toml` defines one action; the directory is seeded
with editable examples the first time you run the mode. For **projects** (see
[Control mode & Projects](#control-mode--projects)), each `*.toml` defines one
project and the directory starts empty — control mode's onboarding screen explains
how to add your first one. In both cases: add a file to add an entry, delete a file
to remove it.

### Per-project quick actions

A repo can ship its own quick actions. Add a `.herdr-plus/` directory at the repo
root that mirrors the global layout, and drop one `*.toml` per action into its
`quick-actions/` subdirectory — same format as your global actions:

```
your-repo/
  .herdr-plus/
    quick-actions/
      make-build.toml
      make-test.toml
```

When you launch the quick-actions picker from inside that repo, its project
actions appear **grouped under a `Project` heading**, above your `Global` ones, so
it is always clear which is which. (Start typing to filter and the two groups
merge into a single ranked list.) Launch from a repo with no `.herdr-plus`
directory and the picker looks exactly as before — a single, ungrouped list. The
directory is read-only and never auto-created: it shows up only when a repo
actually provides it. This repo ships one as a live example (`make build` /
`make test`).

## Actions

An action has a `name`, a `description`, a `command`, and a `type`. The command
is run through `sh -c`, in the working directory you launched from, with the
context exported as `HERDR_PLUS_*` environment variables.

The `command` is a [Go text/template](https://pkg.go.dev/text/template) rendered
against the run context (see [Variables](#variables)).

### Type: `command` (default)

Runs immediately when selected.

```toml
name = "GitHub"
description = "Open https://github.com"
command = "open https://github.com"
```

### Type: `select`

Shows a second fuzzy list of options. The chosen option's `value` becomes
`{{.Value}}`. If `value` is omitted, the `label` is used. An optional
`description` shows dim text next to the label (the `value` itself is never
shown, so you can encode data into it without cluttering the list).

```toml
name = "Open Repo on GitHub"
description = "Pick a repo and open it"
type = "select"
command = "open https://github.com/cloudmanic/{{.Value}}"

[[options]]
label = "Herdr Plus"
value = "herdr-plus"
description = "cloudmanic/herdr-plus"

[[options]]
label = "Options Cafe"
value = "options-cafe"
description = "cloudmanic/options-cafe"
```

To visually group options, add a separator: an option with **no `label`**. Give
it a `heading` to show a dim group title, or leave it blank for a plain spacer.
Separators are not selectable, are skipped when navigating, and disappear while
you filter.

```toml
[[options]]
heading = "Cascade"   # a labeled group header

[[options]]
label = "Options Cafe"
value = "cascade https://github.com/users/cloudmanic/projects/8"

[[options]]               # a blank spacer (no label, no heading)

[[options]]
label = "Options Cafe (Rager)"
value = "rager https://github.com/users/cloudmanic/projects/8"
```

### Type: `form`

Shows a text field. What you type becomes `{{.Value}}`. The `[form]` table is
optional.

```toml
name = "Search Google"
description = "Type a query and open the results"
type = "form"
command = "open 'https://www.google.com/search?q={{.Value | urlquery}}'"

[form]
prompt = "Search Google for"
placeholder = "e.g. herdr terminal multiplexer"
```

### Passing the value

If your command references `{{.Value}}`, the value is placed exactly there. If it
doesn't, the value is appended as a single shell-quoted final argument — so
`command = "my-script"` becomes `my-script 'the value'`.

## Variables

Every action's command template can use these fields. The same values are also
exported to the command's environment with a `HERDR_PLUS_` prefix (e.g.
`{{.WorkDir}}` is also `$HERDR_PLUS_WORKDIR`).

| Template | Env var | Meaning |
|----------|---------|---------|
| `{{.Value}}` | `HERDR_PLUS_VALUE` | Selected option / entered text (select & form). |
| `{{.WorkDir}}` | `HERDR_PLUS_WORKDIR` | Directory you launched herdr-plus from. |
| `{{.SessionTitle}}` | `HERDR_PLUS_SESSION_TITLE` | herdr workspace label (often the repo name). |
| `{{.SessionId}}` | `HERDR_PLUS_SESSION_ID` | herdr workspace id. |
| `{{.WorkspaceLabel}}` | `HERDR_PLUS_WORKSPACE_LABEL` | Same as SessionTitle. |
| `{{.WorkspaceId}}` | `HERDR_PLUS_WORKSPACE_ID` | Same as SessionId. |
| `{{.TabLabel}}` | `HERDR_PLUS_TAB_LABEL` | herdr tab label. |
| `{{.TabId}}` | `HERDR_PLUS_TAB_ID` | herdr tab id. |
| `{{.PaneId}}` | `HERDR_PLUS_PANE_ID` | Pane you launched from. |
| `{{.TerminalId}}` | `HERDR_PLUS_TERMINAL_ID` | herdr terminal id. |
| `{{.Agent}}` | `HERDR_PLUS_AGENT` | Agent running in the pane, if any. |
| `{{.AgentSessionId}}` | `HERDR_PLUS_AGENT_SESSION_ID` | That agent's session id. |
| `{{.Home}}` | — | Your home directory. |

## Building

```bash
go build -o herdr-plus .
go test ./...
```

## Adding a mode

1. Add a `Mode` value and register it in `orderedModes` in `mode.go`.
2. Add an `[[actions]]` entry to `herdr-plugin.toml` that runs
   `./bin/herdr-plus --mode=<slug>`, so herdr exposes the mode as a plugin action.
3. (Optional) Add bundled example actions under `examples/<slug>/`.
4. Teach the launcher/picker how the mode behaves where it differs.
