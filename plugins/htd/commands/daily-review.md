---
name: daily-review
description: Run the daily review — pull fired ticklers into the inbox, clear anything due, and surface what needs attention today. Use first thing in the morning or whenever the user says "let's do a daily review".
---

# Daily review

You are running the user's daily review. Your job is to walk them through a short, predictable sequence that empties the tickler, processes any inbox that results, flags what's due, and surfaces what to work on. Keep it brisk — the user wants to finish and start doing, not sit in meta-work.

Announce the flow ("Running your daily review — 5 quick steps.") and move through the steps in order.

## 1. Pull fired ticklers

Preview first:

```bash
htd reflect tickler --json
```

Parse the JSON. If empty, say so in one line and skip to step 2.

Otherwise show the user the list (ID + title + trigger date). Ask for confirmation, then run:

```bash
htd reflect tickler --pull
```

The pulled items are now in the inbox as unclarified input — they're prompts to re-decide, not automatic next actions.

## 2. Process the inbox

Check what's there (includes anything just pulled):

```bash
htd clarify list --json
```

If the inbox is empty, say so and skip. Otherwise hand off to the clarify flow:

> "You have N inbox items (including M pulled ticklers). Run `/htd:clarify` to walk through them, or I can start it now."

Don't process the inbox inline — that belongs to the `/htd:clarify` subagent. Your job here is to surface the count and hand off cleanly.

## 3. Review queue

```bash
htd reflect review --json
```

Items whose `review_at` is today or past. Summarize in one short block: count, and the top few (ID, title, kind, review date). If empty, say so.

Don't prescribe an action — just surface. The user decides whether to open each one.

## 4. Today's next actions

```bash
htd reflect next-actions --json
```

Summarize: total count, plus the top 3–5 by due date (soonest first, `-` for undated). Keep this to about 6 lines.

## 5. Stale waiting-for

```bash
htd engage waiting --json
```

If nothing is ≥ 7 days old, say "nothing stale" and skip. Otherwise list the oldest 2–3 with age in days. Suggest `/htd:engage` to draft follow-ups if the user wants to chase them now.

## Wrap-up

One-line summary:

```
Daily review: <N> ticklers pulled, <M> inbox items to clarify, <K> review items, <L> next actions ready, <S> stale waiting.
```

Then stop. The user takes it from here.

## Rules

- **Confirm before `--pull`.** It's the one mutation in this flow; show the list and wait for yes before running.
- **All reads use `--json`.** Parse, don't scrape.
- **Don't mark anything done.** Completion belongs to `/htd:engage` and `htd engage done`.
- **Don't clarify inbox items yourself.** Hand off to `/htd:clarify`.
- **Stay tight.** This is a scan, not a deep session. If the user wants to dig into something, point them at the right phase command and stop.
