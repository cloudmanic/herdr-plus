---
title: "Template Variables"
description: "Every context field an action's command can use, both as a Go template field and as a HERDR_PLUS_* environment variable."
weight: 80
---

Every action's command template can use these fields. The same values are also
exported to the command's environment with a `HERDR_PLUS_` prefix — so a script
that would rather not bother with templating can read them straight from the
environment.

## The variables

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

## Two ways to use them

Every field above is available **both** as a Go template field in the `command`
and as a `HERDR_PLUS_` environment variable when the command runs. For example,
`{{.WorkDir}}` is also `$HERDR_PLUS_WORKDIR`.

> **Note:** `{{.Home}}` is the one exception — it is available as a template
> field but has **no** environment variable.

## Examples

Using a variable in the command template — open the launch directory in your
file manager:

```toml
name = "Reveal Working Dir"
description = "Open the launch directory"
type = "command"
command = "open {{.WorkDir}}"
```

Using the environment variable instead — a command that prefers `$HERDR_PLUS_*`
over templating:

```toml
name = "Print Context"
description = "Echo the workspace label and working dir"
type = "command"
command = 'echo "workspace=$HERDR_PLUS_WORKSPACE_LABEL dir=$HERDR_PLUS_WORKDIR"'
```

> **Tip:** Any field herdr can't supply is left empty rather than failing — a
> partial context is better than refusing to launch. Fields like `{{.Agent}}` or
> `{{.AgentSessionId}}` will be blank when no agent is running in the pane.

## See also

- [Actions Reference](../actions/) — where these variables are used.
- [Examples & Cookbook](../examples/) — actions that use context variables.
