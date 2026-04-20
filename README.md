# htd — Headless To-Do

A headless task management CLI for humans and AI agents. Data is stored as plain Markdown files with YAML front matter, designed to be Git-friendly and scriptable.

## Installation

```
go install github.com/takai/htd/cmd/htd@latest
```

## Workflow

htd organises work around five phases:

| Phase | Command group | Purpose |
|-------|--------------|---------|
| Capture | `htd capture` | Collect anything into the inbox |
| Clarify | `htd clarify` | Process inbox items |
| Organize | `htd organize` | Categorise, link, and schedule |
| Reflect | `htd reflect` | Review lists and progress |
| Engage | `htd engage` | Mark work done or cancelled |

## Commands

### Init

```
htd init
```

Creates the full directory layout under the htd root and prints each directory path. Idempotent — safe to run repeatedly. Other commands also create missing directories on demand, so running `htd init` is optional.

### Capture

```
htd capture add --title TEXT [--body TEXT] [--source NAME] [--tag TAG]...
```

### Clarify

```
htd clarify list
htd clarify show ID
htd clarify update ID [--title TEXT] [--body TEXT]
htd clarify discard ID
```

### Organize

```
htd organize move KIND ID [ID...]
htd organize link ID --project PROJECT_ID
htd organize schedule ID [--due DATE] [--defer DATE] [--review DATE]
```

`KIND` is one of: `next_action`, `project`, `waiting_for`, `someday`, `tickler`

Dates accept `YYYY-MM-DD` or RFC 3339. Pass an empty string to clear a date.

### Reflect

```
htd reflect next-actions
htd reflect projects [--stalled]
htd reflect waiting
htd reflect review
htd reflect done --since DATE
```

### Engage

```
htd engage next-action [--project PROJECT_ID] [--tag TAG]...
htd engage waiting [--stale-days N]
htd engage tickler
htd engage done ID [ID...]
htd engage cancel ID [ID...]
```

The list commands surface what demands action now: next actions ready to work on, waiting-for items untouched for `--stale-days` or more (default 7), and ticklers whose trigger date has arrived.

### Low-level item access

```
htd item get ID
htd item list [--kind KIND] [--status STATUS] [--tag TAG] [--project PROJECT_ID]
htd item update ID FIELD=VALUE [FIELD=VALUE]...
htd item archive ID
htd item restore ID
```

## Global options

| Option | Default | Description |
|--------|---------|-------------|
| `--path DIR` | `.` | htd root directory |
| `--json` | off | Machine-readable JSON output |

## Data layout

```
<root>/
├── items/
│   ├── inbox/          ← new items land here
│   ├── next_action/
│   ├── project/
│   ├── waiting_for/
│   ├── someday/
│   └── tickler/
├── reference/
└── archive/
    ├── items/          ← done / cancelled / discarded items
    └── reference/      ← archived reference materials
```

All files are Markdown with YAML front matter and can be read, edited, or committed to Git directly.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Item not found |

## Claude Code plugin

This repository also ships a Claude Code plugin that drives the five-phase workflow from inside Claude Code. It exposes one slash command per phase (`/htd:capture`, `/htd:clarify`, `/htd:organize`, `/htd:reflect`, `/htd:engage`), plus two review rituals — `/htd:daily-review` for a fast morning check-in and `/htd:weekly-review` for the full weekly sweep — a workflow skill, and two subagents for the longer clarify and reflect flows. The plugin wraps the `htd` CLI, so the binary must be installed and on `PATH`.

Install the CLI and create a working directory as above, then inside Claude Code:

```
/plugin marketplace add takai/htd
/plugin install htd@htd
```

Launch Claude Code from whichever directory you run `htd init` in — the plugin operates on the current working directory. To point at a local clone instead of the remote repository, use `/plugin marketplace add /path/to/htd`.
