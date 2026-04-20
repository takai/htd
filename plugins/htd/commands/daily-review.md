---
name: daily-review
description: Run the daily review — pull fired ticklers into the inbox, clear anything due, and surface what needs attention today. Use first thing in the morning or whenever the user says "let's do a daily review".
---

# Daily review

You are running the user's daily review. Your job is to walk them through a short, predictable sequence that empties the tickler, processes any inbox that results, flags what's due, and surfaces what to work on. Keep it brisk — the user wants to finish and start doing, not sit in meta-work.

Announce the flow ("Running your daily review — 6 quick steps.") and move through the steps in order.

## 1. Calendar check

The calendar is the hard landscape of the day — time-specific commitments that shape what's realistic. Prompt the user:

> "Open your calendar and scan today. Anything time-specific worth flagging before we look at tasks? Meetings that need prep, blocks that change which next actions are realistic, commitments I should know about?"

If the user surfaces anything actionable, capture it:

```bash
htd capture add --title "<text>" --source calendar
```

htd itself has no calendar — this step is a user-facing prompt. If the user has no external calendar, or has already checked it, acknowledge in one line and move on. Any items captured here will flow into step 3 (inbox processing).

## 2. Pull fired ticklers

Preview first:

```bash
htd reflect tickler --json
```

Parse the JSON. If empty, say so in one line and skip to step 3.

Otherwise show the user the list (ID + title + trigger date). Ask for confirmation, then run:

```bash
htd reflect tickler --pull
```

The pulled items are now in the inbox as unclarified input — they're prompts to re-decide, not automatic next actions.

## 3. Process the inbox

Check what's there (includes anything just pulled):

```bash
htd clarify list --json
```

If the inbox is empty, say so and skip. Otherwise hand off to the clarify flow:

> "You have N inbox items (including M pulled ticklers). Run `/htd:clarify` to walk through them, or I can start it now."

Don't process the inbox inline — that belongs to the `/htd:clarify` subagent. Your job here is to surface the count and hand off cleanly.

## 4. Review queue

```bash
htd reflect review --json
```

Items whose `review_at` is today or past. Summarize in one short block: count, and the top few (ID, title, kind, review date). If empty, say so.

Don't prescribe an action — just surface. The user decides whether to open each one.

## 5. Today's next actions

```bash
htd reflect next-actions --json
```

Summarize: total count, plus the top 3–5 by due date (soonest first, `-` for undated). Keep this to about 6 lines.

## 6. Stale waiting-for

```bash
htd engage waiting --json
```

If nothing is ≥ 7 days old, say "nothing stale" and skip. Otherwise list the oldest 2–3 with age in days. Suggest `/htd:engage` to draft follow-ups if the user wants to chase them now.

## Wrap-up

One-line summary:

```
Daily review: <C> calendar items flagged, <N> ticklers pulled, <M> inbox items to clarify, <K> review items, <L> next actions ready, <S> stale waiting.
```

Then stop. The user takes it from here.

## Rules

- **Confirm before `--pull`.** It's the one mutation in this flow; show the list and wait for yes before running.
- **All reads use `--json`.** Parse, don't scrape.
- **Don't mark anything done.** Completion belongs to `/htd:engage` and `htd engage done`.
- **Don't clarify inbox items yourself.** Hand off to `/htd:clarify`.
- **Stay tight.** This is a scan, not a deep session. If the user wants to dig into something, point them at the right phase command and stop.
