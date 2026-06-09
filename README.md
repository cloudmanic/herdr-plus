# herdr-plus

herdr-plus is an add-on platform for [herdr](https://herdr.dev) — a place to build
extensions and plugins on top of herdr's terminal panes. The same binary can run
in different **modes**; each mode decides what to do when it talks to herdr.

We're in explore mode: the list of modes will grow over time. Today there is one.

## Modes

Pick a mode with `--mode=<slug>`. With no flag, the default mode runs.

| Mode | Slug | What it does |
|------|------|--------------|
| Quick Actions | `quick-actions` (default) | A fuzzy launcher: pick an action and run it. |

```bash
herdr-plus                       # default mode (quick-actions)
herdr-plus --mode=quick-actions  # explicit
```

The bare command opens a focused split beneath your current pane and runs the
picker there. Choose an action, and the pane closes itself when the action runs.

## Install the keybinding

```bash
herdr-plus install                 # binds prefix+down -> quick-actions
herdr-plus install --key=prefix+a  # pick a different key
herdr-plus install --mode=<slug>   # bind a specific mode
```

`install` adds a `[[keys.command]]` entry to herdr's `config.toml` that runs the
**absolute path** of the binary you invoked, then reloads the running herdr
server. It is idempotent (it won't duplicate an existing herdr-plus binding) and
refuses to overwrite a key already bound to something else. After installing,
press your herdr prefix (default `ctrl+b`) followed by the bound key.

## Configuration

All config lives under `~/.config/herdr-plus/` (honoring `$XDG_CONFIG_HOME`), with
one subdirectory per mode:

```
~/.config/herdr-plus/
  quick-actions/
    github.toml
    google.toml
    open-repo.toml
    ...
```

Each `*.toml` file in a mode's directory defines **one action**. Add a file to
add an action; delete a file to remove it. The first time you run a mode, the
directory is seeded with a set of example actions you can edit or delete — they
won't come back once the directory exists.

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
