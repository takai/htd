---
name: engage
description: Engage phase — show what demands action now (ready next actions, stale waiting) and help drill into one.
---

# Engage phase

The user wants to act on their system. Give them an overview of what needs attention, then let them choose how to drill in.

## Overview

Run these three in parallel and parse the JSON:

```bash
htd engage next-action --json
htd engage waiting --json
htd reflect tickler --json
```

Summarize in a short block — counts plus the top couple of items per category. For example:

```
Ready next actions: 7 (top: "Write man page", "Review PR #42")
Stale waiting-for: 2 (oldest: "Client sign-off", 14 days)
Fired ticklers: 1 (run /htd:daily-review to process)
```

If the fired-tickler count is > 0, point the user at `/htd:daily-review` — tickler processing is a Reflect-phase concern (empty into inbox, re-clarify) and doesn't belong in Engage. Don't drill into it here.

If everything is empty, tell the user their plate is clear and stop.

## Drill-down

Ask which category (or "done", "none") they want to dive into:

### a. Pick a next action

Ask briefly about context and time: "How long do you have, and what kind of work fits right now (deep focus, quick wins, specific project)?" Use the answers to narrow:

- For a specific project: `htd engage next-action --project <id> --json`.
- For a tag/context: `htd engage next-action --tag <t> --json`.
- No narrowing: use the existing list.

Propose 1–3 candidates. Don't pick for the user. Once they choose, step back — they'll go work on it, and mark it done later with `htd engage done <id>`.

### b. Chase a waiting-for item

For the stale items, help the user nudge the person they're waiting on:

1. Ask which item they want to follow up on.
2. Ask the medium (email, Slack, chat, in-person) and the recipient's name if not obvious from the title.
3. Draft a concise follow-up message in English. Show it to the user to copy and send manually. **Do not send anything — the plugin has no channel access.**
4. Offer to update the item after they send it: `htd clarify update <id> --body "<note about the follow-up>"` or `htd item update <id>`. This refreshes `updated_at` and drops the item out of the stale list.

## Rules

- **Don't mark anything done in this command.** Completion happens via `htd engage done <id>` as a separate step once the user finishes the work.
- **Always confirm mutating commands.** Show the exact `htd` call and wait.
- **Don't send messages to anyone.** You draft follow-ups; the user sends them.
- **Don't process ticklers inline.** Hand off to `/htd:daily-review`, which pulls them into the inbox for clarify.
- Use `--json` for every read and parse the output. Don't scrape the human-readable format.
