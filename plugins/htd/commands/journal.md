---
name: journal
description: Create or look up a journal entry — daily journal, weekly retro, or ad-hoc observation log. Pass arguments to bypass prompts; run bare to be guided.
argument-hint: [daily|weekly|adhoc] [title text...]
---

# Journal

You are managing **Journal** entries — time-stamped observations stored under `journal/<name>.md`. Journals are not tasks (use `/htd:capture` instead) and not durable facts (use `/htd:reference` instead). They are the third lane: daily notes, weekly retros, and ad-hoc observation logs.

## Routing the request

Inspect `$ARGUMENTS`. The first token (case-insensitive) routes the action:

- `daily` (or empty) → create today's daily journal (or the date the user names).
- `weekly` → create a weekly retro for the ISO week containing the date (defaults to this week). The CLI snaps `--date` to Monday automatically.
- `adhoc` → create an ad-hoc entry whose remaining tokens are the title.
- `list` / `show` → look up existing entries.

When the user's intent is ambiguous (no leading keyword, just a phrase), prefer the daily flow if they said "journal" / "today's notes" and the weekly flow if they said "retro" / "weekly".

## Creating an entry

### Daily

```bash
htd journal new --type daily [--date YYYY-MM-DD]
```

Defaults to today. If the user wants to backfill yesterday or an earlier day, pass `--date`. Tags optional via `--tag`.

The CLI writes a scaffold (`## What I did`, `## What I learned`, `## Tomorrow`). Print the resulting name and stop — the user fills in the body in their editor.

### Weekly retro

```bash
htd journal new --type weekly [--date YYYY-MM-DD]
```

`--date` is any day in the target week; the CLI snaps to Monday of that ISO week. Defaults to the current week.

Scaffold: `## Wins`, `## Misses`, `## Lessons`, `## Focus next week`. If the user wants you to pre-fill any section from recent activity, you can offer to run `htd reflect log --since <one-week-ago>` and quote the closed items in `## Wins`. **Do not write the retro for them** — surface the data and let the user reflect.

### Ad-hoc

```bash
htd journal new --type adhoc --title "<title>"
```

Slug is derived from `--title`. The scaffold is just an H1 from the title. Use this for postmortems, decision memos, or anything that's date-loose but worth a dedicated file.

## Listing and reading

- `htd journal list [--since YYYY-MM-DD]` — most recent first. The `--since` filter compares filename-derived dates for dated entries; ad-hoc slugs are kept based on `created_at`.
- `htd journal show NAME` — `NAME` is the filename without `.md` (e.g. `2026-04-28`, `2026-04-27-weekly`, `postmortem_on_outage`).

If the user asks "what did I learn last week", run `list --since` then propose two or three entries to `show`.

## Notes

- **Journals are write-once via htd.** There is no `update`, `archive`, or `restore` verb. If the user wants to edit, point them to `$EDITOR` (`open journal/2026-04-28.md` in macOS, etc.). Git is the audit log; deletion is `rm`.
- **All written artifacts are English** per project convention — including journal bodies. The user may converse in Japanese, but the file content stays English.
- The CLI refuses to overwrite an existing file. If the user re-runs `new` for the same date, surface the conflict and ask whether they meant to open the existing entry instead.
- `journal/` is flat and tool-agnostic. Do **not** pass a `--tool` flag — that does not exist for journals (it would be a confusing carry-over from `htd reference`).
- Don't auto-write content into the body. The point of the lane is the user's own reflection; you scaffold and propose, they write.

## Rules

- **Confirm before destructive actions** (e.g. if you propose `rm` of an entry the user wants to discard). Show the exact command and wait for yes.
- **Don't conflate buckets.** If the user is recording an action they need to do, send them to `/htd:capture`. If they're noting a durable fact about themselves or a project, send them to `/htd:reference`. Journals are for time-stamped reflections only.
- **Use `--json` when parsing.** `htd journal list --json` for any programmatic flow.
