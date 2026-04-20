---
name: organize
description: Categorize, link, and schedule an htd item. Pass the ID; Claude suggests kind/project/dates with confirmations.
argument-hint: <item-id>
---

# Organize phase

You are helping the user organize a specific item: `$ARGUMENTS`.

If `$ARGUMENTS` is empty, ask which item they want to organize (offer to show `htd item list --json` filtered to recently-updated items if they don't know).

## Steps

1. Fetch the item:

   ```bash
   htd item get "$ARGUMENTS" --json
   ```

   If it's terminal (status ≠ active), tell the user it's already archived and stop.

2. Inspect the current state: `kind`, `project`, `due_at`, `defer_until`, `review_at`, `tags`. Summarize briefly to the user.

3. Propose organization changes based on the title and body. Ask the user about each dimension that could be set, but skip ones that are already reasonable. For each proposal, show the exact `htd` command and wait for confirmation before running.

   - **Kind** — if it's still `inbox` or the wrong kind, suggest one of next_action / project / waiting_for / someday / tickler and run `htd organize move <kind> <id>`. Cannot target `inbox`. If the item clearly needs to become a project with obvious first sub-actions, prefer `htd organize promote <id> --child "<title 1>" [--child "<title 2>"]...` to promote and seed children in one shot.
   - **Project link** — if the item looks related to an existing project, suggest it. To narrow candidates, pick one or two keywords from the item's title/body and search with `--query`: `htd item list --kind project --query '<keyword>' --json` (or combine terms: `--query 'cli OR docs'`). Fall back to `htd item list --kind project --status active --json` only if nothing matches. Run `htd organize link <id> --project <project-id>` to link, or `--project ""` to clear.
   - **Dates** — if the user mentions timing ("next week", "by Friday", "defer until the 15th"), convert to `YYYY-MM-DD` and run `htd organize schedule <id> [--due …] [--defer …] [--review …]`. Pass `""` to clear a date.
   - **Tags** — if the user wants to tag: `htd item update <id> tags='[a,b]'`.

4. When done, show the final state:

   ```bash
   htd item get "$ARGUMENTS"
   ```

## Rules

- **One change at a time, confirm each.** Don't batch multiple mutations into a single approval.
- **Don't propose changes the user didn't ask for or doesn't imply.** If they just want to set a due date, don't also suggest a kind change.
- **Never target `inbox` as a kind.** That's only reachable via `htd capture add`.
- **Never modify terminal items** (done/canceled/discarded/archived). Tell the user to use `htd item update` directly if they need to correct an error.
- If a command fails, show the stderr and ask how to proceed.
