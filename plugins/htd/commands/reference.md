---
name: reference
description: Save or look up a tool-scoped reference note — durable AI context that survives across sessions. Pass text as the title for a quick add, or run bare to get prompted.
argument-hint: [title text...]
---

# Reference notes

You are managing **References** — durable, AI-readable notes stored under `reference/<tool>/<id>.md`. References are not tasks; they are facts, preferences, project context, or pointers that future sessions should be able to load cheaply. The tool defaults to `claude`; respect that unless the user explicitly asks for another tool.

## If arguments are present

The user's arguments `$ARGUMENTS` are the title for a new reference. Quickly clarify with the user (in one short turn) two things you cannot infer from the title alone:

1. **Type tag** — pick one of `type:user`, `type:feedback`, `type:area_of_focus`, `type:project`, `type:reference`, or "other" (skip the type tag entirely). Suggest a reasonable default based on the title and confirm.
2. **Body** — at minimum the one-line fact (used as the INDEX.md description). Optionally a `## How to apply` section. If the user only gave you a title and you can paraphrase a clear fact line, propose it; otherwise ask.

Then run:

```bash
htd reference add --title "$ARGUMENTS" [--tag type:<x>] --body "<body>"
```

Print the resulting ID. Do not chain into anything else.

## If arguments are empty

Ask the user what they want to do:

- **Save a new reference** → ask for title (required), type tag (one of the five canonical, or skip), and body (fact line + optional "How to apply"). Then run `htd reference add ...` as above.
- **Look up an existing one** → run `htd reference list --json` and surface IDs/titles. If the user names one, run `htd reference get <id>`.
- **Archive a stale fact** → confirm the ID, then `htd reference archive <id>`.
- **Restore something** → confirm the ID, then `htd reference restore <id>`.

Pick the path the user describes; don't run a menu unless they're unsure.

## Type tag guide

| Tag | When to use |
|-----|-------------|
| `type:user` | Anything about the user themselves — role, preferences, knowledge level, working style. |
| `type:feedback` | Corrections or validations the user has given about how to work. Capture *why* alongside the rule. |
| `type:area_of_focus` | An area of standing attention without a defined outcome — an ongoing responsibility, role, or domain. Re-tag as `type:project` once a deliverable and deadline appear. |
| `type:project` | Non-derivable context on a project — motivations, deadlines, stakeholder asks. |
| `type:reference` | Pointers to external sources of truth (dashboards, trackers, doc URLs). |
| (no tag) | Falls into `## other` in INDEX.md. Use only when none of the five fit. |

## Notes

- Titles stay in English per project convention. Bodies too.
- Keep titles concise — they become the INDEX line label. Aim for under 50 characters.
- The first non-blank line of `--body` becomes the INDEX description (truncated to 80 runes). Lead with the fact, then `## How to apply` for the application context.
- `--tool` defaults to `claude`. Pass `--tool other` only when the user explicitly works in a different assistant's namespace.
- IDs are auto-generated as `YYYYMMDD-<slug>` with collision suffixing across both items and references; do not try to set them yourself.
- Every mutation (add/update/archive/restore) rewrites `reference/<tool>/INDEX.md` automatically. Do not edit `INDEX.md` directly. If it ever drifts (merge conflict, manual edit), run `htd reference reindex`.

## Rules

- **Confirm before destructive actions.** `archive` and `restore` move files; show the exact command and wait for yes.
- **Don't write tasks here.** If the user is recording an action they need to take, send them to `/htd:capture` instead — references are for context, not work.
- **Don't invent IDs.** Always read them from `htd reference list --json` or the output of `htd reference add`.
- **If `$ARGUMENTS` is whitespace only**, fall through to the empty-arguments path.
