---
name: htd-workflow
description: Use when the user wants help managing tasks with htd, storing durable AI context, or keeping a journal/retro — any mention of htd, inbox, next actions, projects, waiting-for, someday, tickler, capture, clarify, organize, reflect, engage, reference, journal, retro, daily note, "remember this fact / project context / preference", or "let's do a retro". Teaches the five-phase workflow, the reference (tool-scoped memory) surface, the journal lane, and how to pick the right CLI command.
version: 0.3.1
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
- Areas of focus (`type:area_of_focus`) — standing attention without a defined outcome (e.g. an ongoing responsibility or role). Promote to `type:project` when a deliverable and deadline appear.
- Project context (`type:project`) — non-derivable background on ongoing work.
- External pointers (`type:reference`) — dashboards, trackers, source-of-truth links.

Storage layout is `reference/<tool>/<id>.md` — `<tool>` namespaces references per AI assistant so multi-assistant repos don't collide. The `--tool` flag selects the namespace and defaults to `claude`.

Each tool directory carries an auto-generated `INDEX.md` that lists every active reference grouped by `type:*` tag (`## user`, `## feedback`, `## area_of_focus`, `## project`, `## reference`, trailing `## other` for anything else). The index is rewritten on every mutation; AI sessions can load it cheaply at startup. Do not edit `INDEX.md` by hand — run `htd reference reindex` to repair if it ever drifts (e.g., merge conflict).

The body convention is **fact line first** (used as the INDEX description, truncated to 80 runes) optionally followed by a `## How to apply` section. The convention isn't enforced; just follow it.

## Journals

A Journal entry is a **time-stamped observation** that fits neither items (not actionable) nor references (not durable lookup). The lane covers:

- **Daily journals** — what got done, what was learned, plans for tomorrow.
- **Weekly retros** — wins, misses, lessons, focus for next week.
- **Ad-hoc logs** — postmortems, decision memos, observation notes pinned to a topic rather than a date.

Storage is `journal/` at the htd root, **flat and tool-agnostic** (no per-tool subdir). Journals belong to the user, not to any AI assistant — they don't need the per-tool isolation that references have.

Filename forms:

| Type | Filename |
|------|----------|
| Daily | `YYYY-MM-DD.md` (e.g. `2026-04-28.md`) |
| Weekly | `YYYY-MM-DD-weekly.md` where the date is the **Monday** of the ISO week (e.g. `2026-04-27-weekly.md`) |
| Ad-hoc | `<slug>.md` derived from a `--title` (e.g. `postmortem_on_outage.md`) |

The filename is the identifier — there is no `id` field. YAML frontmatter (`created_at`, `updated_at`, `tags`) is optional; `htd journal new` writes it, hand-edited files may omit it.

Journals are **write-once** in htd's view: there is no `update`, `archive`, or `restore` verb. Users edit entries in `$EDITOR`; Git is the audit log; deletion is direct (`rm`). Listings show every file under `journal/`, most-recent-first by filename.

Journals **do not** appear in any INDEX.md and are **not** loaded at AI session start. They are consulted on demand ("what did I learn the week of 2026-04-20?", "show me the postmortem").

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
11. **Journals are write-once.** Don't try to update a journal entry through htd — there is no verb. Edit the file directly in `$EDITOR` if you need to change it; `git diff` is the audit trail.
12. **Pick the right bucket.** A new fact about the user → reference (`type:user`). A retro → journal weekly. A task to do → item (capture). When unsure, ask the user before placing it.

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
- `htd reference add --title TEXT [--body TEXT] [--tag TAG]... [--tool TOOL]` — tag with `type:user|feedback|area_of_focus|project|reference` to drive INDEX.md grouping; other tags fall into `## other`.
- `htd reference get ID` — falls back to the archive automatically. Archived hits are marked `(archived)` in text mode and `archived: true` in JSON.
- `htd reference list [--tool TOOL] [--tag TAG] [--archived]` — default lists the active set; `--archived` flips to the archive view (mutually exclusive).
- `htd reference update ID FIELD=VALUE...` — supported fields: `title`, `body`, `tags`. Protected: `id`, `created_at`, `tool`.
- `htd reference archive ID` — moves the file to `archive/reference/<tool>/`. Refuses already-archived input.
- `htd reference restore ID` — symmetric inverse of `archive`. Refuses active input.
- `htd reference reindex [--tool TOOL]` — repair verb: rewrites `reference/<tool>/INDEX.md` from disk. Idempotent. Reach for this only when the index has drifted (manual edit, merge conflict).

**Journal (time-stamped observations)** — see "Journals" above. All journal entries live under `journal/` at the htd root.
- `htd journal new [--type daily|weekly|adhoc] [--date YYYY-MM-DD] [--title TEXT] [--tag TAG]...` — creates a templated file. `daily` defaults to today; `weekly` snaps to Monday of the ISO week containing `--date`; `adhoc` requires `--title` and derives a slug. Refuses to clobber existing files.
- `htd journal list [--since YYYY-MM-DD]` — chronological, most recent first. `--since` filters by filename-derived date for dated entries (`created_at` fallback for ad-hoc slugs).
- `htd journal show NAME` — `NAME` is the filename without `.md` (e.g. `2026-04-28`, `2026-04-27-weekly`, `postmortem_on_outage`). Hand-edited entries with no frontmatter are still readable.
- No `update`/`archive`/`restore` — edit in `$EDITOR`; remove with `rm` if needed.

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
| "I'm taking on X as an ongoing responsibility", "I'm now stewarding Y but no concrete deliverable yet" | `htd reference add --title "..." --tag type:area_of_focus --body "..."` (promote to `type:project` once a deliverable + deadline appear) |
| "what do you know about my project", "load my context", session-start orientation | Read `reference/<tool>/INDEX.md` directly, then `htd reference get <id>` for entries you need |
| "this fact is stale" / "we don't do X anymore" | `htd reference archive ID` (use `restore` if you regret it) |
| "the index looks wrong" / merge conflict in `INDEX.md` | `htd reference reindex` |
| "let's do a daily journal", "I want to write today's notes" | `/htd:journal` or `htd journal new` |
| "weekly retro", "let's reflect on last week" | `htd journal new --type weekly` (date snaps to Monday of the ISO week) |
| "postmortem on the outage", "decision memo on X" | `htd journal new --type adhoc --title "..."` |
| "what did I learn last week", "pull up the journal entries since X" | `htd journal list --since YYYY-MM-DD` then `htd journal show <name>` |

## Interaction principles

1. **Confirm before destructive actions.** `discard`, `cancel`, `done`, `archive`, and any status change: propose first, ask for confirmation, then execute.
2. **Propose, don't impose.** When clarifying or organizing, suggest a kind/project/tag based on the title, but let the user veto.
3. **Use `--json` when you need to parse.** Pipe `htd ... --json` into `jq` or parse in-shell; don't parse the human-readable output.
4. **Stay in the current directory.** The plugin is path-agnostic; always let `htd` default to the CWD unless the user explicitly specifies `--path`.
5. **Never invent item IDs.** IDs are `YYYYMMDD-<slug>` generated by `htd capture add`. Read them from output; don't guess.
