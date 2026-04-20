---
name: htd-workflow
description: Use when the user wants help managing tasks with htd — any mention of htd, inbox, next actions, projects, waiting-for, someday, tickler, capture, clarify, organize, reflect, or engage. Teaches the five-phase workflow, the htd CLI surface, and how to pick the right command for the user's situation.
version: 0.1.0
---

# htd workflow

htd is a headless task-management CLI. Tasks are Markdown files with YAML front matter stored under an `htd init`-ed directory. The CLI is the stable contract — always prefer calling `htd` over reading/writing item files directly.

## The five phases

| Phase | Intent | CLI group |
|-------|--------|-----------|
| Capture | Collect inputs into the inbox | `htd capture` |
| Clarify | Turn inbox items into defined outcomes, or discard them | `htd clarify` |
| Organize | Categorize, link to projects, schedule | `htd organize` |
| Reflect | Review the state of the system | `htd reflect` |
| Engage | Surface what needs action now, then do the work | `htd engage` |

Capture is friction-free on purpose — never interrupt the user to clarify during capture. Clarify is where the real thinking happens.

## Items

An Item is any actionable or incomplete work. Every item has a `kind`:

- `inbox` — unclarified input, entry point for everything new.
- `next_action` — a concrete, single action ready to work on.
- `project` — a multi-step outcome requiring more than one action. Must have at least one linked `next_action` to avoid being stalled.
- `waiting_for` — delegated to someone else; we're waiting on them.
- `someday` — deferred for future consideration; not committed.
- `tickler` — time-triggered reminder; surfaces on `defer_until`.

And a `status`: `active` (live), or one of the terminal statuses `done`, `canceled`, `discarded`, `archived`. Terminal items live in `archive/items/`.

## Invariants you must respect

1. **Inbox items must be clarified before being ended.** Do not send an inbox item to `done`/`canceled` directly — run it through `htd clarify` first.
2. **`discarded` is inbox-only.** Use `htd clarify discard` for inbox items that were never actionable. Anything that has already been organized (kind ≠ inbox) ends with `htd engage done` or `htd engage cancel`.
3. **`archived` is a last resort.** Reach for `htd item archive` only when neither done nor canceled fits (e.g., a project superseded by another). Default to `done` or `canceled`.
4. **`kind` and directory always agree.** The CLI enforces this; never move files by hand.
5. **Terminal items are nearly immutable.** Only correct them via `htd item update` for fixing genuine errors.
6. **A project should have at least one next action.** If none, the project is stalled — surface it for review.
7. **Never name the underlying methodology.** Refer only to the five-phase workflow in all user-facing output, commits, and docs.
8. **All written artifacts are English.** Item titles, bodies, commits, comments — English. The user may converse in Japanese but items stay English.

## CLI cheat sheet

All commands accept `--json` for machine-readable output and `--path` to target a specific htd root. Omit `--path` to use the current directory.

**Capture**
- `htd capture add --title TEXT [--body TEXT] [--source NAME] [--tag TAG]...`
- `htd capture add --title TEXT --done [--body TEXT] [--source NAME] [--tag TAG]...` — capture an already-completed item. Bypasses the inbox; the item lands directly in `archive/items/` with `kind: next_action`, `status: done`. Prefer this over `capture add` followed by `engage done <id>` when the user has already finished the task.

**Clarify** (inbox only)
- `htd clarify list`
- `htd clarify show ID`
- `htd clarify update ID [--title TEXT] [--body TEXT]`
- `htd clarify discard ID`

**Organize**
- `htd organize move KIND ID [ID...]` — KIND ∈ {next_action, project, waiting_for, someday, tickler}; cannot target `inbox`. Accepts multiple IDs for a shared disposition in one shot.
- `htd organize link ID --project PROJECT_ID` — empty string to unlink.
- `htd organize schedule ID [--due DATE] [--defer DATE] [--review DATE]` — empty string to clear. Dates accept `YYYY-MM-DD` or RFC 3339.
- `htd organize promote ID --child TITLE [--child TITLE]...` — one-shot promote an item to a project and create linked next-action children. Prefer this over the move+capture+link chain when clarifying an item that clearly needs sub-actions.

**Reflect** (review the system)
- `htd reflect next-actions` — all active next actions, deferred hidden.
- `htd reflect projects [--stalled]`
- `htd reflect waiting`
- `htd reflect review` — items whose `review_at` is due.
- `htd reflect log --since YYYY-MM-DD [--until DATE] [--kind KIND] [--tag TAG]... [--status STATUS]...` — recently resolved items (activity log). Defaults to `--status done`.
- `htd reflect tickler [--pull]` — ticklers whose `defer_until` (or `review_at` fallback) is today or past. With `--pull`, move them into the inbox for re-clarification (clears `defer_until`, keeps `review_at`).

**Engage** (act on the system)
- `htd engage next-action [--project ID] [--tag T]...` — what's ready to work on now.
- `htd engage waiting [--stale-days N]` — waiting-for items untouched ≥ N days (default 7). JSON includes `age_days`.
- `htd engage done ID [ID...]` — accepts multiple IDs.
- `htd engage cancel ID [ID...]` — accepts multiple IDs.

**Item (low-level CRUD)** — use for scripting; workflow commands are preferred.
- `htd item get ID`
- `htd item list [--kind KIND] [--status STATUS] [--tag TAG] [--project PID]`
- `htd item update ID FIELD=VALUE...` — `id` and `created_at` are protected.
- `htd item archive ID`

## Choosing a command

| User says / situation | Suggest |
|-----------------------|---------|
| "remember to X", "I just thought of X", a random idea | `htd capture add` or `/htd:capture` |
| "I just did X", "already handled X", a small task already completed | `htd capture add --title "X" --done` |
| "my inbox is full", "process my inbox" | `/htd:clarify` (walks item-by-item) |
| "categorize this", "link this to project Y", "set a due date" | `/htd:organize` |
| "weekly review", "how's my system looking", "what's stalled" | `/htd:reflect` |
| "daily review", "morning routine", "let's start the day" | `/htd:daily-review` |
| "what should I work on now", "what's on my plate today" | `/htd:engage` |
| Chasing a delegated task | `/htd:engage` → drill into waiting |
| Tickler for date X fires | `/htd:daily-review` (pulls fired ticklers into the inbox, then clarify decides) |
| Completing a task | `htd engage done ID` (direct call is fine) |

## Interaction principles

1. **Confirm before destructive actions.** `discard`, `cancel`, `done`, `archive`, and any status change: propose first, ask for confirmation, then execute.
2. **Propose, don't impose.** When clarifying or organizing, suggest a kind/project/tag based on the title, but let the user veto.
3. **Use `--json` when you need to parse.** Pipe `htd ... --json` into `jq` or parse in-shell; don't parse the human-readable output.
4. **Stay in the current directory.** The plugin is path-agnostic; always let `htd` default to the CWD unless the user explicitly specifies `--path`.
5. **Never invent item IDs.** IDs are `YYYYMMDD-<slug>` generated by `htd capture add`. Read them from output; don't guess.
