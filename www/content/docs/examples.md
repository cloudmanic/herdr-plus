---
title: "Examples & Cookbook"
description: "A copy-pasteable gallery of complete quick actions and project templates — commands, selects, forms, splits, and context variables."
weight: 100
---

A gallery of complete, copy-pasteable examples. Drop a quick action into the
`quick-actions/` subdirectory of [herdr-plus's config dir](../configuration/), or a
project into `projects/` — one file per entry, file name up to you. Find the
directory with `herdr plugin config-dir cloudmanic.herdr-plus`.

## Quick actions

### A plain command

The simplest action: run a command immediately.

```toml
# google.toml
name = "Google"
description = "Open https://google.com"
type = "command"
command = "open https://google.com"
```

### Open a URL

`type` defaults to `command`, so you can omit it.

```toml
# github.toml
name = "GitHub"
description = "Open https://github.com"
command = "open https://github.com"
```

### Open the current repo on GitHub

Uses the launch context. This opens the `origin` remote of whatever repo your
pane is in, in your browser.

```toml
# open-repo-on-github.toml
name = "Open This Repo on GitHub"
description = "Open the current repo's GitHub page"
type = "command"
command = "open \"$(git -C {{.WorkDir}} remote get-url origin | sed -E 's#git@github.com:#https://github.com/#; s#\\.git$##')\""
```

### A `select` with grouped options and separators

A second fuzzy list. Headings group the options; a blank separator adds a spacer.

```toml
# open-repo.toml
name = "Open Repo on GitHub"
description = "Pick one of our repos and open it"
type = "select"
command = "open https://github.com/cloudmanic/{{.Value}}"

[[options]]
heading = "Apps"

[[options]]
label = "Options Cafe"
value = "options-cafe"
description = "cloudmanic/options-cafe"

[[options]]
label = "Skyclerk"
value = "skyclerk"
description = "cloudmanic/skyclerk"

[[options]]            # blank spacer

[[options]]
heading = "Tools"

[[options]]
label = "Herdr Plus"
value = "herdr-plus"
description = "cloudmanic/herdr-plus"
```

### A `form` with `urlquery`

Type a query; `urlquery` escapes it so it's URL-safe.

```toml
# google-search.toml
name = "Search Google"
description = "Type a query and open the results"
type = "form"
command = "open 'https://www.google.com/search?q={{.Value | urlquery}}'"

[form]
prompt = "Search Google for"
placeholder = "e.g. herdr terminal multiplexer"
```

### A "search docs" form

Same idea, pointed at a docs site.

```toml
# search-docs.toml
name = "Search herdr Docs"
description = "Open the docs search for a term"
type = "form"
command = "open 'https://herdr.dev/?q={{.Value | urlquery}}'"

[form]
prompt = "Search the docs for"
placeholder = "e.g. keybindings"
```

### Using a context variable

Open the directory you launched from in your file manager.

```toml
# reveal-working-dir.toml
name = "Reveal Working Dir"
description = "Open the launch directory in Finder"
type = "command"
command = "open {{.WorkDir}}"
```

### A project-local action that keeps the pane open

For per-project actions (in a repo's `.herdr-plus/quick-actions/`), end the
command with a wait so its output doesn't flash by before the picker pane closes.
The action's stdin is `/dev/null`, so the wait must read the terminal directly
via `</dev/tty`.

```toml
# .herdr-plus/quick-actions/make-build.toml
name = "make build"
description = "Build the herdr-plus binary (make build)"
command = 'make build; printf "\n— done · press Enter to close (auto-closes in 30s) —\n"; read -t 30 _ </dev/tty || true'
```

Keep-open variants you can use in place of the `read -t 30 ...` tail:

```text
read _ </dev/tty            # wait for Enter, then close
read -t 30 _ </dev/tty      # ...but auto-close after 30s if you walk away
sleep 5                     # simplest: just linger N seconds, then close
```

The `|| true` keeps a read timeout (or a missing tty) from looking like an error.

## Projects

### A simple 3-tab dev project

Three tabs, each running a startup command (the last is just an empty shell).

```toml
# myapp.toml
name = "My App"
description = "Day-to-day dev workspace"
working_dir = "~/Development/my-app"

[[tabs]]
name = "claude"
command = "claude --dangerously-skip-permissions"

[[tabs]]
name = "lazygit"
command = "lazygit"

[[tabs]]
name = "terminal"   # no command — just an empty shell
```

### A project with a split-pane server tab

The `server` tab is split into two stacked panes; it shows as `server ×2` in the
browser.

```toml
# options-cafe.toml
name = "Options Cafe"
description = "The main options.cafe monorepo"
working_dir = "~/Development/options-cafe/options.cafe"   # ~ and $VARS expand

[[tabs]]
name = "claude"
command = "claude --dangerously-skip-permissions --chrome"

[[tabs]]
name = "editor"
command = "spiceedit"

# A split tab: instead of `command`, list [[tabs.panes]].
[[tabs]]
name = "server"

[[tabs.panes]]
command = "php artisan serve"

[[tabs.panes]]
command = "npm run dev"
split = "down"
```

### lazygit + editor + server, with a side-by-side split

A four-tab project whose server tab splits side by side (`right`) instead of
stacked.

```toml
# webapp.toml
name = "Web App"
description = "lazygit, editor, and a side-by-side server tab"
working_dir = "$HOME/Development/web-app"   # $VARS expand too

[[tabs]]
name = "lazygit"
command = "lazygit"

[[tabs]]
name = "editor"
command = "nvim ."

[[tabs]]
name = "server"

[[tabs.panes]]
command = "make api"

[[tabs.panes]]
command = "make web"
split = "right"   # beside the first pane, not below

[[tabs]]
name = "shell"   # an empty terminal to poke around in
```

### A minimal project using `~`

`working_dir` expands a leading `~`; omit it entirely and it defaults to your
home directory.

```toml
# notes.toml
name = "Notes"
description = "Quick scratch workspace in my notes dir"
working_dir = "~/Documents/notes"

[[tabs]]
name = "edit"
command = "nvim ."
```

## Worktree auto-layouts

These live in `worktrees/` and fire automatically when herdr creates a git
worktree of a matching repo. See [Worktree Auto-Layout](../worktrees/) for the
full behavior.

### Auto-open tabs for a repo's worktrees

```toml
# options-cafe.toml
repo = "options-cafe"          # matches the worktree's repo, case-insensitive

[[tabs]]
name = "claude"
command = "claude --dangerously-skip-permissions --chrome"

[[tabs]]
name = "lazygit"
command = "lazygit"

[[tabs]]
name = "terminal"              # no command — just an empty shell
```

### A branch-specific layout

When more than one layout matches, a branch-specific one wins over a repo-only one.

```toml
# options-cafe-release.toml
repo = "options-cafe"
branch = "release"

[[tabs]]
name = "deploy"
command = "./scripts/release.sh"
```

## See also

- [Actions Reference](../actions/) — the full action format.
- [Projects](../projects/) — the full project schema.
- [Worktree Auto-Layout](../worktrees/) — auto-deploy a layout on worktree creation.
- [Template Variables](../variables/) — context you can use in commands.
