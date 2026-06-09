# herdr-plus

herdr-plus is an add-on platform for [herdr](https://herdr.dev) — a place to build
extensions and plugins on top of herdr's terminal panes. The same binary can run
in different **modes**; each mode decides what to do when it talks to herdr.

We're in explore mode: the list of modes will grow over time.

## Modes

Pick a mode with `--mode=<slug>`. With no flag, the default mode runs.

| Mode | Slug | Key | What it does |
|------|------|-----|--------------|
| Control | `control` (default) | `prefix+up` | herdr-plus's home base — a full-screen workspace for driving herdr. First feature: **Projects**. |
| Quick Actions | `quick-actions` | `prefix+down` | A fuzzy launcher: pick an action and run it in a split. |

```bash
herdr-plus                       # default mode (control)
herdr-plus --mode=quick-actions  # the fuzzy launcher
herdr-plus version               # print the version and exit
```

## Control mode & Projects

Pressing `prefix+up` opens a brand-new, full-screen herdr workspace titled
**Herdr Plus** with a `projects` tab, and runs the projects browser there. This is
control mode — over time it will gain more features; today it has Projects.

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

## Installing

**Homebrew** (the repo is its own tap):

```bash
brew tap cloudmanic/herdr-plus https://github.com/cloudmanic/herdr-plus
brew install cloudmanic/herdr-plus/herdr-plus
```

**Install script** (Linux/macOS, no Homebrew):

```bash
curl -fsSL https://raw.githubusercontent.com/cloudmanic/herdr-plus/main/install.sh | sh
```

**From source:**

```bash
make build && make install-bin
```

Every merge to `main` auto-bumps the patch version and cuts a new GitHub Release
with cross-compiled binaries; `brew upgrade` / re-running the install script
pulls the latest.

The bare command opens a focused split beneath your current pane and runs the
picker there. Choose an action, and the pane closes itself when the action runs.

## Install the keybinding

```bash
herdr-plus install                      # binds prefix+up -> control (the default mode)
herdr-plus install --mode=quick-actions # binds prefix+down -> quick-actions
herdr-plus install --key=prefix+a       # override the key for any mode
```

Each mode has its own default key (control → `prefix+up`, quick-actions →
`prefix+down`), so the two can be installed side by side. Pass `--key` to override.

`install` adds a `[[keys.command]]` entry to herdr's `config.toml` that runs the
**absolute path** of the binary you invoked, then reloads the running herdr
server. It is idempotent (it won't duplicate an existing herdr-plus binding) and
refuses to overwrite a key already bound to something else. After installing,
press your herdr prefix (default `ctrl+b`) followed by the bound key.

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

1. Add a `Mode` value and register it in `modes` in `mode.go`.
2. (Optional) Add bundled example actions under `examples/<slug>/`.
3. Teach the launcher/picker how the mode behaves where it differs.
