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
htd organize move ID KIND
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
htd engage done ID
htd engage cancel ID
```

### Low-level item access

```
htd item get ID
htd item list [--kind KIND] [--status STATUS] [--tag TAG] [--project PROJECT_ID]
htd item update ID FIELD=VALUE [FIELD=VALUE]...
htd item archive ID
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
