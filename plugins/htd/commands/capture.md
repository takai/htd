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
- **Tags** (optional, comma-separated). Before introducing a fresh tag, run `htd tag list --similar <candidate>` to surface existing variants and avoid drift (e.g., `ivry_job_scheduler` vs `ivry-job-scheduler`).

Then build a single `htd capture add` invocation:

```bash
htd capture add --title "<title>" [--body "<body>"] [--source <source>] [--tag <tag>]...
```

Run it, print the resulting ID, and stop. Do not chain into clarify/organize — the user will do that later with `/htd:clarify` or `/htd:organize`.

## Already-done shortcut

If the user reports that they just finished something quick ("I just did X", "done: X", "already handled X"), pass `--done` to capture it as completed in one step:

```bash
htd capture add --title "<title>" --done
```

This lands the item directly in `archive/items/` with `kind: next_action`, `status: done` — no stop in the inbox, no follow-up `engage done` needed. `--body`, `--source`, and `--tag` still apply. Prefer this over `capture add` + `engage done <id>` for anything the user has already completed.

## Skip-inbox shortcut (`--kind`)

If the disposition is already clear at capture time ("this is a project called X with sub-tasks A and B", "this is a tickler for next Monday"), pass `--kind` to land directly in the right list and skip clarify. With `--kind project`, repeat `--child "<title>"` to seed linked next-action children in the same call:

```bash
# Project + first sub-actions in one step
htd capture add --kind project --title "<project title>" \
  --child "<first sub-action>" [--child "<more>"]...

# Skip-inbox for non-project kinds
htd capture add --kind next_action --title "<title>"
htd capture add --kind tickler --title "<title>"
```

Only use this when the user is unambiguous about disposition; when in doubt, stay on the default `capture add` (lands in inbox) and let `/htd:clarify` decide. `--kind inbox` is rejected (redundant with the default); `--child` requires `--kind project`; `--kind` and `--done` are mutually exclusive.

## Notes

- Titles stay in English per project convention.
- Keep titles concise — aim for roughly 6–8 words and under 50 characters. The ID is derived from the title as `YYYYMMDD-<slug>`, and long IDs clutter list output and shell history. Trim filler words, drop redundant context (repo names, ticket numbers beyond the primary one), and save details for `--body`.
- The ID is auto-generated from the title. Don't try to set it yourself.
- Without `--done`, the item lands in `items/inbox/` with `kind: inbox`, `status: active`. With `--done`, it lands in `archive/items/` with `kind: next_action`, `status: done`.
- If `$ARGUMENTS` would produce an empty title (whitespace only), fall through to the "arguments empty" path.
