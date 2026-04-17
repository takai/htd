---
name: capture
description: Add a new item to the htd inbox. Pass text as the title, or run bare to be prompted for details.
argument-hint: [title text...]
---

# Capture to inbox

You are in the **Capture** phase. The user has something to record. Get it into the inbox with minimum friction and move on — do not try to clarify, categorize, or schedule it here. That's for later phases.

## If arguments are present

The user's arguments `$ARGUMENTS` are the title. Run:

```bash
htd capture add --title "$ARGUMENTS"
```

Print the resulting ID on one line. Do not ask follow-up questions.

## If arguments are empty

Ask the user in one turn for:

- **Title** (required, short — one line).
- **Body** (optional, free-form Markdown). Skip if they don't offer one.
- **Source** (optional, e.g., `email`, `meeting`, `slack`).
- **Tags** (optional, comma-separated).

Then build a single `htd capture add` invocation:

```bash
htd capture add --title "<title>" [--body "<body>"] [--source <source>] [--tag <tag>]...
```

Run it, print the resulting ID, and stop. Do not chain into clarify/organize — the user will do that later with `/htd:clarify` or `/htd:organize`.

## Notes

- Titles stay in English per project convention.
- The ID is auto-generated from the title as `YYYYMMDD-<slug>`. Don't try to set it yourself.
- The item lands in `items/inbox/` with `kind: inbox`, `status: active`.
- If `$ARGUMENTS` would produce an empty title (whitespace only), fall through to the "arguments empty" path.
