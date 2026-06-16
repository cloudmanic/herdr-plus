---
title: "Actions Reference"
description: "The complete action file format: the command, command/select/form types, separators and headings, and how the value is passed."
weight: 70
---

An action is one entry in the Quick Actions picker, loaded from a TOML file in
herdr-plus's `quick-actions/` config directory (see
[Configuration](../configuration/)). This page documents the complete file format.

## The basics

Every action has four fields:

| Field | Required | Meaning |
|-------|----------|---------|
| `name` | yes | Shown in the picker; the text you fuzzy-find. |
| `description` | no | Dim text shown next to the name. |
| `command` | yes | The shell command run when the action completes. |
| `type` | no | `command` (default), `select`, or `form`. |

`name` and a non-empty `command` are required; an action missing either fails to
load with a clear error. An unknown `type` is also a load-time error.

## How the command runs

The `command` is run through `sh -c`, in the working directory you launched
herdr-plus from, with the run context exported as `HERDR_PLUS_*` environment
variables. Running through `sh -c` means commands can use pipes, arguments, and
full scripts.

The `command` is also a [Go text/template](https://pkg.go.dev/text/template)
rendered against the run context — so it can reference fields like `{{.Value}}`,
`{{.WorkDir}}`, and `{{.SessionTitle}}` (see [Template Variables](../variables/)).

## Type: `command` (default)

Runs immediately when selected — no further input. This is the simplest possible
action: just a name, a description, and a command. `type` defaults to `command`
when omitted.

```toml
name = "GitHub"
description = "Open https://github.com"
command = "open https://github.com"
```

## Type: `select`

Shows a second fuzzy list of options. The chosen option's `value` becomes
`{{.Value}}` in the command.

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

Each option:

- `label` — what the user sees in the list. **Required for a selectable option.**
- `value` — what gets substituted into the command. **If omitted, the `label` is
  used as the value too.**
- `description` — optional dim text shown next to the label. The `value` itself
  is never shown, so you can encode data into it (e.g. a host and URL) without
  cluttering the list.

A `select` action needs at least one selectable option (one with a label), or it
fails to load.

### Separators and headings

To visually group options, add a **separator**: an option with **no `label`**.
Give it a `heading` to show a dim group title, or leave it blank for a plain
spacer.

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

Separators are **not selectable**, are **skipped** when navigating, and
**disappear** while you filter.

## Type: `form`

Shows a single text field. Whatever you type becomes `{{.Value}}` in the command.
The `[form]` table is optional.

```toml
name = "Search Google"
description = "Type a query and open the results"
type = "form"
command = "open 'https://www.google.com/search?q={{.Value | urlquery}}'"

[form]
prompt = "Search Google for"
placeholder = "e.g. herdr terminal multiplexer"
```

The `[form]` table customizes the field:

- `prompt` — the label rendered above the input. Defaults to `Enter a value`.
- `placeholder` — the greyed-out hint in the empty field. Defaults to
  `Type a value…`.

Your input is trimmed of surrounding whitespace before it becomes `{{.Value}}`.

## Passing the value

For `select` and `form` actions there's a resolved value (the option's value or
your typed text). herdr-plus passes it to the command one of two ways:

- **If your command references `{{.Value}}`,** the value is substituted exactly
  there.
- **If it doesn't,** the value is appended as a **single shell-quoted final
  argument** — so `command = "my-script"` becomes `my-script 'the value'`.

This lets a command either position the value precisely with `{{.Value}}` or just
receive it as its last argument.

> **Note:** A plain `command` action has no value, so nothing is appended.

## Template functions

Because the command is a Go `text/template`, the standard template functions are
available. The most useful for actions is `urlquery`, which escapes text so it's
safe inside a URL:

```toml
command = "open 'https://www.google.com/search?q={{.Value | urlquery}}'"
```

## See also

- [Template Variables](../variables/) — every field you can reference.
- [Examples & Cookbook](../examples/) — ready-to-copy actions.
- [Quick Actions](../quick-actions/) — how actions are launched.
