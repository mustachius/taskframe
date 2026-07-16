# TaskFrame

[![CI](https://github.com/mustachius/taskframe/actions/workflows/ci.yml/badge.svg)](https://github.com/mustachius/taskframe/actions/workflows/ci.yml)

A fast, keyboard-driven task manager for the terminal, inspired by
[Taskwarrior](https://taskwarrior.org/) but simpler. You get the same core
(tasks, projects, subtasks, tags, notes, urgency, undo) in three flavors: an
inline REPL, a quick-capture CLI, and a classic two-pane TUI. Built in Go with
a pure-Go SQLite backend (no CGo), so it runs cleanly on Windows.

![taskframe demo](demo.gif)

## Features

- Three interfaces over the same database: an inline REPL (the default), a
  quick-capture CLI, and a classic Norton Commander-style TUI.
- Projects form a dotted hierarchy (`work.api`); subtasks nest to any depth.
- Tag filters (`+tag` / `-tag`), per-task notes, and free-text search.
- Urgency sorting: a weighted Taskwarrior-style score over due date, priority,
  age, active state, and pending subtasks, with configurable coefficients.
- Contexts (saved default filters), start/stop to mark a task in progress, and
  recurring tasks that spawn the next instance when completed.
- Undo and redo for every change, backed by a per-task activity log. Deletes
  are soft until `purge`.
- JSON export/import for backups, and `taskframe sync` to carry the database
  between machines through a private git repo (last-writer-wins, with an
  automatic backup before every overwrite).
- Ten themes and an English/Portuguese interface, both switchable at runtime
  and persisted.

## Installation

### Windows (recommended)

Run the installer to build and put `taskframe` on your user `PATH` (no admin
required):

```powershell
.\install.ps1
```

It compiles to `%LOCALAPPDATA%\Programs\taskframe` and updates the user `PATH`.
Open a new terminal and run `taskframe`.

### From source

Requires Go 1.26 or newer (see `go.mod`).

```sh
git clone https://github.com/mustachius/taskframe.git
cd taskframe
go build -o taskframe.exe ./cmd/taskframe
```

## Quick start

Running `taskframe` with no arguments opens the REPL. With a subcommand it acts
as a one-shot CLI (no interface is drawn), which is handy for scripts and quick
capture:

```sh
taskframe add "buy milk" pro:home.groceries +urgent due:tomorrow prio:H
taskframe list                  # plain table, sorted by urgency
taskframe done 12               # complete (recurring tasks spawn the next one)
taskframe del 12                # soft delete (undo reverts, purge removes)
taskframe note 12 "waiting on the supplier"
taskframe undo                  # reverse the last change
taskframe export > backup.json  # full backup (tasks, notes, history)
taskframe import backup.json    # restore into an empty database
```

The CLI prints plain text with no escape codes, so `taskframe list | ...` is
pipe-friendly.

## Interfaces

### REPL (default)

The default interface is an inline prompt: the logo is printed once, output
scrolls into your terminal's real scrollback, and the prompt stays pinned at the
bottom. Type natural commands (no slash) or app commands (with a slash). History
is on `↑` / `↓`, and `Tab` completes commands, projects, and tags.

| Command | Action |
|---|---|
| `add <title> [tokens]` | create a task |
| `sub <parent> <title>` | create a subtask under `<parent>` |
| `list [tokens]` | open the navigable list overlay |
| `next` · `overdue` · `today` · `week` · `waiting` · `active` | named reports |
| `done <ids>` · `del <ids>` | complete · delete (`1`, `1,5`, `1-3`) |
| `note <id> [text]` | add a note (no text opens an input) |
| `edit <id> <tokens>` | change fields (`+tag` adds, `-tag` removes) |
| `move <id> pro:x sub:N` | change project / parent (`sub:0` = root) |
| `start` / `stop <ids>` | mark in progress / idle |
| `context [name\|none\|list\|define …]` | manage saved default filters |
| `undo` · `redo` | reverse · reapply the last change |
| `/theme [name]` · `/sort [mode]` · `/lang [en\|pt-br]` | preferences |
| `/help` · `/clear` · `/quit` | help · clear · quit (`Ctrl+D`) |

In the list overlay you act on the highlighted task **without typing its id**:
`↑↓` / `jk` move, `←→` fold/expand, `a` adds a subtask, `n` adds a note, `e`
edits it (same tokens as `add`), `enter` opens the detail (notes + history), `d`
completes, `x` deletes, `esc` closes. The detail view also takes `n` and `e`.

### CLI

Every REPL verb has a CLI counterpart: `add`, `list`, `done`, `del`, `note`,
`move`, `context`, `start`, `stop`, `undo`, `redo`, `purge`, `export`, `import`,
`sync`, `lang`, plus the report names. Run `taskframe help` for the full
reference.

### Classic TUI (`taskframe classic`)

A two-pane, full-screen interface: projects and filters on the left, tasks on
the right, and the TaskFrame logo pinned at the top (the full gradient wordmark
on large terminals, a compact brand line on small ones). Every function key has
a letter alias.

| Key | Action |
|---|---|
| `Tab` | switch panels (projects / tasks) |
| `↑↓` / `jk`, `←→` / `hl` | move / collapse / expand subtasks |
| `Enter`, `F3` | detail: notes + full history |
| `F2` / `a` · `s` | new task · new subtask |
| `F4` / `e` · `F5` / `n` | edit · add note |
| `F6` / `m` | move (project / parent) |
| `F9` / `d` / `Space` | complete / reopen |
| `S` | start / stop (mark in progress) |
| `F8` / `x` | delete (with confirmation) |
| `F7` / `/` | text search |
| `o` · `t` · `L` | sort · theme · language |
| `u` · `U` | undo · redo |
| `F10` / `q` | quit |

The sidebar shows projects (each with a done/total progress bar), virtual
filters (**Today**, **Overdue**, **Week**, **Active**, **Next**, **Waiting**),
the tags in use, and your saved contexts — press `Enter` on a context row to
activate it (again to clear); the active one is marked and applied to every
view, like in the REPL and CLI.

## Tokens and dates

Tokens are accepted by `add` and `list`:

| Token | Meaning |
|---|---|
| `pro:work.api` | project (dotted hierarchy) |
| `+tag` / `-tag` | require / exclude a tag (`-tag`: list only) |
| `due:<date>` | due date |
| `prio:H\|M\|L` | priority |
| `wait:<date>` | hide until the date |
| `recur:daily\|weekly\|3d…` | recurrence |
| `sub:<id>` | create as a subtask of `<id>` (add only) |
| `status:pending\|done\|deleted\|all` | filter by status |
| `all` | include completed and deleted (list only) |

Dates accept `today`, `tomorrow`, `3d`, `2w`, weekday names (`fri`, and
Portuguese `sex`), `15/08`, `2026-08-15`, `eow` (end of week), and `eom` (end of
month).

## Language

The interface ships in **English by default**, with a Portuguese (`pt-br`)
translation you can switch to at any time:

```sh
taskframe lang pt-br     # switch and persist
taskframe lang           # show the current language
```

In the REPL, `/lang pt-br` switches live (and `/lang` alone toggles back).
Resolution order: `--lang` flag > `TASKFRAME_LANG` > saved setting > config file
> English. Task input is language-agnostic (weekday aliases like `fri` and `sex`
both work regardless of the interface language).

## Themes

Ten themes, switchable with `/theme` in the REPL or `t` in the classic TUI (or
`/theme <name>`); the choice is saved. They restyle the text, the interface, and
the markdown rendering, but never paint a line background, so your terminal
shows through:

- **dark** (default) — subtle grays, adapts to light/dark terminals
- **borland** — retro navy, Turbo Vision style
- **green** / **amber** — monochrome CRT phosphor
- **dracula**, **catppuccin**, **nord**, **gruvbox**, **solarized**, **tokyonight** — popular truecolor palettes

Resolution order: `--theme` flag > `TASKFRAME_THEME` > saved setting > config
file > dark.

## Configuration

Runtime preferences (theme, sort, language, active context) are stored in a
SQLite `settings` table and always win over the config file. An optional config
file provides defaults:

- **Location:** `%APPDATA%\taskframe\config.json` (or `TASKFRAME_CONFIG`).
- **Fields:** `theme`, `sort`, `lang`, and `urgency` (coefficient overrides).

Environment variables: `TASKFRAME_THEME`, `TASKFRAME_LANG`, `TASKFRAME_DB`,
`TASKFRAME_CONFIG`. Flags: `--theme`, `--lang`, `--db`, `--ascii`.

For terminals without double box-drawing support, run with `--ascii`.
Windows Terminal is recommended.

## Data and storage

Tasks live in a SQLite database at `%APPDATA%\taskframe\taskframe.db`. Override
it with the `TASKFRAME_DB` variable or the `--db` flag (useful for testing
against a throwaway database).

Every mutation is written to an `activity` table in the same transaction, so
each task carries a complete, visible history that also powers `undo` and
`redo`. The default sort is by **urgency**, a weighted score over due date,
priority, the `+next` tag, age, active state, and pending subtasks.

## Sync across machines

Use the same tasks on several machines through a private git repo, instead of
copying the database around by hand. Set it up once per machine:

```sh
taskframe sync init https://github.com/you/taskframe-data.git
```

The machine that already has tasks publishes them; a fresh machine adopts them.
After that, a single command reconciles local and remote:

```sh
taskframe sync          # picks the direction: pull when the remote is ahead,
                        # push when you have local changes
taskframe sync status   # clone, remote, and whether you're up to date
taskframe sync pull     # or force a direction (last-writer-wins tie-breaker)
taskframe sync push
```

Sync is **last-writer-wins**: it does not merge concurrent edits (the data
model has no global ids to reconcile), so it suits sequential use, like adding
at home and reviewing at work. Nothing is overwritten silently. The local
database is backed up first (the five newest backups are kept next to it), and
a bare `sync` aborts if both sides changed since the last sync, asking you to
`pull` or `push` explicitly. Authentication is handled by your own git
(credential manager or SSH key); TaskFrame stores no credentials, so don't
embed a token in the sync URL. Requires `git` on `PATH`.

## Development

```sh
go build -o taskframe.exe ./cmd/taskframe   # build
go test ./...                                # run the test suite
go vet ./... ; gofmt -l .                    # lint and format check
```

The REPL and TUI are driven by [Bubble Tea](https://github.com/charmbracelet/bubbletea)
and cannot be driven through a pipe (they read console events directly); the
smoke tests in `internal/repl` and `internal/tui` exercise the models
synchronously instead.

## License

Released under the [MIT License](LICENSE). Copyright (c) 2026 Victor Sartor.
