# herdr-plus

herdr-plus is an add-on for [herdr](https://herdr.dev), built as a first-class
[herdr plugin](https://herdr.dev/docs/plugins/). Its flagship feature is
**Projects**: declarative herdr-workspace templates you fuzzy-pick to spin up a
whole workspace — every tab and pane created, every startup command running — in
one keypress.

> This is a clean, plugin-first rebuild. The previous standalone-binary
> implementation lives under [`old/`](old/) as reference and is not built.

## Install

herdr-plus is a herdr plugin (requires **herdr ≥ 0.7.0**). Installing it registers
the plugin's actions with herdr — no editing of your `config.toml`.

```bash
herdr plugin install cloudmanic/herdr-plus
```

herdr clones the repo, runs the manifest's `[[build]]` step (`go build`, needs a
Go toolchain), and registers the actions. Manage it with `herdr plugin list`,
`herdr plugin action list --plugin cloudmanic.herdr-plus`, and
`herdr plugin uninstall cloudmanic.herdr-plus`.

**Local development:** build the binary and link your checkout in place:

```bash
make build
herdr plugin link /path/to/herdr-plus     # or: make plugin-link
```

## Projects

Pick a project from a full-screen fuzzy browser and herdr-plus builds its whole
workspace. Trigger it from herdr's plugin action menu, or
[bind a key](#binding-a-key) — the action is `cloudmanic.herdr-plus.projects`.

A project is one TOML file in `~/.config/herdr-plus/projects/` (honoring
`$XDG_CONFIG_HOME`). The file name doesn't matter; add a file to add a project,
delete it to remove it. With no files there, the browser shows an onboarding card.

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

## Binding a key

Binding a key to the Projects action is an optional, one-time edit to **your**
herdr `config.toml` (`~/.config/herdr/config.toml`). Add a `[[keys.command]]`
entry with `type = "plugin_action"` whose `command` is the action id:

```toml
[[keys.command]]
key = "prefix+up"
type = "plugin_action"
command = "cloudmanic.herdr-plus.projects"
description = "herdr-plus: projects"
```

Then `herdr server reload-config` (or restart herdr) and press your herdr prefix
(default `ctrl+b`) followed by the bound key.

## Building

```bash
make build     # build ./bin/herdr-plus
make test      # go test -race ./...
make vet       # go vet ./...
```

The repo root is the active Go module; `old/` is a separate nested module and is
ignored by `go ... ./...`.
