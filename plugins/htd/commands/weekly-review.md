---
name: weekly-review
description: Run the weekly review — collect loose material, empty the inbox, walk every active list, review the calendar backward and forward, unblock stalled projects, check areas of focus, and capture new ideas. Use when the user says "let's do the weekly review", "weekly check-in", or wants the full system sweep.
---

# Weekly review

You are running the user's weekly review — the heavyweight ritual that keeps every htd list current and trustworthy. The daily review is a quick glance; this is the deliberate walk, and skipping steps defeats the point.

Announce the flow ("Running your weekly review — full walk in three phases: Get Clear, Get Current, Get Creative. Aim for 30–60 minutes.") and move through the steps in order.

Compute `<one-week-ago>` once at the start: `date -v-7d +%Y-%m-%d` on macOS, `date -d '7 days ago' +%Y-%m-%d` on Linux. Reuse it later.

---

## Phase A — Get Clear

### 1. Collect loose material

Prompt:

> "Before we touch htd, gather everything that hasn't been processed yet: paper notes, sticky notes, whiteboard photos, business cards, receipts, open browser tabs, screenshots, message drafts — physical and digital. Put it all in one pile."

Wait for the user to confirm they are ready. For each item they surface, capture it:

```bash
htd capture add --title "<text>" --source <origin>
```

Pick a source value that fits (`manual`, `email`, `slack`, `meeting`, `calendar`, …).

### 2. Empty the inbox

```bash
htd clarify list --json
```

If non-empty, hand off to `/htd:clarify` — it walks items one at a time. Don't process inline.

If empty, say so and move on.

### 3. Empty your head (mind sweep)

Prompt:

> "Anything still on your mind that isn't captured? Commitments, ideas, nags, decisions you've been putting off, things you keep remembering at inconvenient times. Dump it all now — any order."

Capture each item with `htd capture add`. When the user stops, loop back to step 2 to process the new inbox entries. Repeat steps 2–3 until the inbox is empty *and* the user says nothing else is on their mind.

### 4. Empty fired ticklers

Preview:

```bash
htd reflect tickler --json
```

If non-empty, show the list, confirm, then:

```bash
htd reflect tickler --pull
```

Hand off to `/htd:clarify` to process the pulled items.

---

## Phase B — Get Current

The goal: every list reflects reality, and nothing has been quietly forgotten.

### 5. Review active next actions

```bash
htd reflect next-actions --json
```

Walk the list with the user. For each item, offer:

- Still the right next step → leave it.
- Already done → `htd engage done <id>`.
- No longer relevant → `htd engage cancel <id>`.
- Should wait → `htd organize move someday <id>`, or defer via `htd organize schedule <id> --defer <date>`.
- Belongs elsewhere → `htd organize move <kind> <id>`.

Confirm any mutation before running.

After the audit, surface what closed this week:

```bash
htd reflect log --since <one-week-ago> --json
```

This is recognition, not action. Ask whether the user wants to drop a retro note (`htd capture add --title "Retro: ..." --tag retro`).

### 6. Previous week's calendar

Prompt:

> "Open your calendar and scan the last seven days. Look for meetings that generated an action you never captured, promised follow-ups, commitments made in conversation, notes you took and forgot to process."

Capture each item via `htd capture add`. Loop back through `/htd:clarify` on any new entries.

htd has no calendar — this step is a user-facing prompt only.

### 7. Upcoming week's calendar

Prompt:

> "Now scan the next seven days. Which meetings need prep? Decisions that need to be made ahead of time? Material to bring or read?"

Capture prep actions. For date-sensitive prep, offer `htd organize schedule <id> --defer <date>` so the action stays hidden until it's relevant.

### 8. Chase waiting-for items

```bash
htd reflect waiting --json
```

For each item, ask:

- Chase now → `htd item update <id>` with a note so `updated_at` resets; if the user wants help drafting the chase, hand off to `/htd:engage`.
- Still reasonable to wait → leave.
- No longer waiting → `htd engage cancel <id>`.
- It arrived → `htd engage done <id>`.

Confirm before any mutation.

### 9. Unblock stalled projects

```bash
htd reflect projects --stalled --json
```

Every active project must have at least one next action. For each stalled project, ask: "What's the next concrete step?"

- User names a next action → `htd organize promote <project-id> --child "<title>"` (adds and links the child in one shot).
- User can't name one → `htd engage cancel <id>`, `htd organize move someday <id>`, or `htd item archive <id>`.

### 10. Review queue

```bash
htd reflect review --json
```

For each item due for review:

- Re-set the review date → `htd organize schedule <id> --review <new-date>`.
- Update the item → `htd item update <id> ...`.
- Move or close via the right phase command.

### 11. Areas of focus

Prompt:

> "Think about the different areas and roles you're responsible for — work role, side projects, health, relationships, finances, home, learning. Going area by area: is anything obviously neglected? A commitment you hold somewhere that has zero representation in the active lists?"

For each gap, capture via `htd capture add`, tagging the area where useful. Loop back through `/htd:clarify` on new entries.

This step is external — htd has no first-class areas-of-focus list. If the user keeps one elsewhere, offer to walk through it with them.

### 12. Revisit someday

```bash
htd item list --kind someday --json
```

For each item:

- Activate → `htd organize move next_action <id>` or `htd organize move project <id>`.
- Keep → leave.
- Let it go → `htd engage cancel <id>`.

Nudge the user on items that have sat for many months unchanged — someday is where ambitions rest, not rot.

---

## Phase C — Get Creative

### 13. Capture new ideas

Prompt:

> "Looking at the next week or two — any new projects, experiments, outcomes, or bets you want to put into the system? What do you actually want to be working on?"

Capture each via `htd capture add`. If any clearly belong in a specific kind right away, offer to clarify and move them on the spot. Otherwise leave them in the inbox for the next daily review.

---

## Wrap-up

One-line summary:

```
Weekly review: <L> loose items collected, <I> inbox processed, <P>/<U> past/upcoming calendar captures, <S> stalled projects unblocked, <W> waiting chased, <C> cancelled, <A> areas-of-focus gaps captured, <N> new ideas.
```

Then stop.

## Rules

- **All reads use `--json`.** Parse, don't scrape.
- **Confirm before any mutation.** The user owns the decisions.
- **Hand off, don't do inline.** Clarify loops through `/htd:clarify`; chase drafting through `/htd:engage`. Inline is fine for single-command mutations (capture, move, schedule, promote, cancel, done).
- **Don't skip steps.** If a section is empty, say so in one line and move on — but move on, don't delete the step.
- **Stay terse.** No pep talk, no emoji. The point is the system is clear and current, not that the user did a great job.
- Stay in English throughout.
