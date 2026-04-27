---
name: htd-workflow
description: Use when the user wants help managing tasks with htd, or storing durable AI context — any mention of htd, inbox, next actions, projects, waiting-for, someday, tickler, capture, clarify, organize, reflect, engage, reference, or "remember this fact / project context / preference". Teaches the five-phase workflow, the reference (tool-scoped memory) surface, and how to pick the right CLI command.
version: 0.2.0
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

## References

A Reference is **non-actionable, durable** information stored for future retrieval — typically by AI assistants recovering context at session start. References are fully separate from items: they cannot be promoted to items or linked via `project`. The data type fits things like:

- User profile (`type:user`) — role, preferences, knowledge.
- Feedback patterns (`type:feedback`) — corrections and validations the user has given about how to work.
- Project context (`type:project`) — non-derivable background on ongoing work.
- External pointers (`type:reference`) — dashboards, trackers, source-of-truth links.

Storage layout is `reference/<tool>/<id>.md` — `<tool>` namespaces references per AI assistant so multi-assistant repos don't collide. The `--tool` flag selects the namespace and defaults to `claude`.

Each tool directory carries an auto-generated `INDEX.md` that lists every active reference grouped by `type:*` tag (`## user`, `## feedback`, `## project`, `## reference`, trailing `## other` for anything else). The index is rewritten on every mutation; AI sessions can load it cheaply at startup. Do not edit `INDEX.md` by hand — run `htd reference reindex` to repair if it ever drifts (e.g., merge conflict).

The body convention is **fact line first** (used as the INDEX description, truncated to 80 runes) optionally followed by a `## How to apply` section. The convention isn't enforced; just follow it.

## Invariants you must respect

1. **Inbox items must be clarified before being ended.** Do not send an inbox item to `done`/`canceled` directly — run it through `htd clarify` first.
2. **`discarded` is inbox-only.** Use `htd clarify discard` for inbox items that were never actionable. Anything that has already been organized (kind ≠ inbox) ends with `htd engage done` or `htd engage cancel`.
3. **`archived` is a last resort.** Reach for `htd item archive` only when neither done nor canceled fits (e.g., a project superseded by another). Default to `done` or `canceled`.
4. **`kind` and directory always agree.** The CLI enforces this; never move files by hand.
5. **Terminal items are nearly immutable.** Only correct them via `htd item update` for fixing genuine errors.
6. **A project should have at least one next action.** If none, the project is stalled — surface it for review.
7. **Never name the underlying methodology.** Refer only to the five-phase workflow in all user-facing output, commits, and docs.
8. **All written artifacts are English.** Item titles, bodies, commits, comments, references — English. The user may converse in Japanese but artifacts stay English.
9. **References are not items.** Don't capture project context as an inbox item; use `htd reference add`. Don't try to mark a reference `done`; archive it via `htd reference archive` when stale.
10. **Never edit `INDEX.md` by hand.** It is regenerated on every reference mutation. If it drifts, run `htd reference reindex`.

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
- `htd engage next-actions [--project ID] [--tag T]...` — what's ready to work on now.
- `htd engage waiting [--stale-days N]` — waiting-for items untouched ≥ N days (default 7). JSON includes `age_days`.
- `htd engage done ID [ID...]` — accepts multiple IDs.
- `htd engage cancel ID [ID...]` — accepts multiple IDs.

**Item (low-level CRUD)** — use for scripting; workflow commands are preferred.
- `htd item get ID`
- `htd item list [--kind KIND] [--status STATUS] [--tag TAG] [--project PID] [--query EXPR]` — `--query` accepts a small DSL (substring by default, `field:value` for targeted match, `AND`/`OR`/`NOT`/parens). Useful for narrowing project candidates, e.g. `htd item list --kind project --query 'cli OR docs' --json`.
- `htd item update ID FIELD=VALUE...` — `id` and `created_at` are protected.
- `htd item archive ID`
- `htd item restore ID` — undo an accidental `engage done`/`cancel`/`discard`/`archive`; brings a terminal item back to `active` and moves it to `items/<kind>/`.

**Reference (tool-scoped durable notes)** — see "References" above for the data type. All reference verbs accept `--tool TOOL` and default to `claude`.
- `htd reference add --title TEXT [--body TEXT] [--tag TAG]... [--tool TOOL]` — tag with `type:user|feedback|project|reference` to drive INDEX.md grouping; other tags fall into `## other`.
- `htd reference get ID` — falls back to the archive automatically. Archived hits are marked `(archived)` in text mode and `archived: true` in JSON.
- `htd reference list [--tool TOOL] [--tag TAG] [--archived]` — default lists the active set; `--archived` flips to the archive view (mutually exclusive).
- `htd reference update ID FIELD=VALUE...` — supported fields: `title`, `body`, `tags`. Protected: `id`, `created_at`, `tool`.
- `htd reference archive ID` — moves the file to `archive/reference/<tool>/`. Refuses already-archived input.
- `htd reference restore ID` — symmetric inverse of `archive`. Refuses active input.
- `htd reference reindex [--tool TOOL]` — repair verb: rewrites `reference/<tool>/INDEX.md` from disk. Idempotent. Reach for this only when the index has drifted (manual edit, merge conflict).

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
| "I marked the wrong item done", undo an accidental terminal transition | `htd item restore ID` |
| "remember that I prefer X", "save this as a fact", "for future sessions you should know Y" | `/htd:reference` or `htd reference add --title "..." --tag type:user --body "..."` |
| "what do you know about my project", "load my context", session-start orientation | Read `reference/<tool>/INDEX.md` directly, then `htd reference get <id>` for entries you need |
| "this fact is stale" / "we don't do X anymore" | `htd reference archive ID` (use `restore` if you regret it) |
| "the index looks wrong" / merge conflict in `INDEX.md` | `htd reference reindex` |

## Interaction principles

1. **Confirm before destructive actions.** `discard`, `cancel`, `done`, `archive`, and any status change: propose first, ask for confirmation, then execute.
2. **Propose, don't impose.** When clarifying or organizing, suggest a kind/project/tag based on the title, but let the user veto.
3. **Use `--json` when you need to parse.** Pipe `htd ... --json` into `jq` or parse in-shell; don't parse the human-readable output.
4. **Stay in the current directory.** The plugin is path-agnostic; always let `htd` default to the CWD unless the user explicitly specifies `--path`.
5. **Never invent item IDs.** IDs are `YYYYMMDD-<slug>` generated by `htd capture add`. Read them from output; don't guess.
