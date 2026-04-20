# htd CLI Specification

## 1. General

### 1.1 Binary Name

```
htd
```

### 1.2 Command Structure

```
htd <command-group> <subcommand> [arguments] [options]
```

Command groups map to the five workflow phases plus a low-level `item` group:

| Group | Phase |
|-------|-------|
| `capture` | Capture |
| `clarify` | Clarify |
| `organize` | Organize |
| `reflect` | Reflect |
| `engage` | Engage |
| `item` | Low-level CRUD |

### 1.3 Global Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--json` | flag | `false` | Output in JSON format instead of human-readable text |
| `--path` | string | `$HTD_PATH` or `.` | Specify the htd root directory (overrides `$HTD_PATH`) |

Global options may appear before or after the command group.

**Environment variables**

| Variable | Description |
|----------|-------------|
| `HTD_PATH` | Default value for `--path`. Used only when `--path` is not passed on the command line. An absolute path is recommended so that `htd` operates on the same data directory regardless of the current working directory; relative values are resolved against the cwd, matching the flag's behavior. |

Resolution order for the root directory: `--path` (if given) → `$HTD_PATH` (if non-empty) → `.`.

### 1.4 Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error (invalid arguments, file I/O error, etc.) |
| `2` | Item not found |

### 1.5 Output Conventions

- **Default (human-readable)**: Tab-separated columns for lists, YAML-like key-value for detail views.
- **`--json`**: A single JSON object (for `show`/`get`) or a JSON array (for `list`) written to stdout.
- **Errors**: Written to stderr.

---

## 2. Capture

### 2.1 `htd capture add`

Add a new item to the inbox.

```
htd capture add --title TEXT [--body TEXT] [--source NAME] [--tag TAG]... [--ref URL]... [--done]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--title` | yes | Short description of the item |
| `--body` | no | Detailed description (Markdown) |
| `--source` | no | Origin of the item (e.g., `email`, `meeting`, `slack`) |
| `--tag` | no | Tag to attach; repeatable for multiple tags |
| `--ref` | no | External reference URL (e.g., PR, ticket, doc); repeatable |
| `--done` | no | Capture the item as already completed (see below) |

**Behavior:**

1. Generate a new ID: `YYYYMMDD-<slug>` where slug is derived from the title in snake_case.
2. Set `kind: inbox`, `status: active`.
3. Set `created_at` and `updated_at` to the current timestamp.
4. Write the file to `items/inbox/<id>.md`.
5. Print the created ID to stdout.

**Example:**

```
$ htd capture add --title "Write the man page" --source manual --tag cli --tag docs
20260417-write_the_man_page

$ htd capture add --title "Review PR #42" --ref https://github.com/foo/bar/pull/42
20260421-review_pr_42
```

**`--done` behavior:**

When `--done` is passed, the item is captured as already completed instead of entering the inbox. This is a shortcut for items that were completed on the spot — a single action whose capture and completion collapse into the same step.

1. Generate a new ID (same rule as above).
2. Set `kind: next_action`, `status: done`.
3. Set `created_at` and `updated_at` to the current timestamp.
4. Write the file directly to `archive/items/<id>.md` (no temporary stop in `items/inbox/`).
5. Print the created ID to stdout.

`--body`, `--source`, `--tag`, and `--ref` still apply when `--done` is set, so metadata is preserved on the archived item.

**Example:**

```
$ htd capture add --title "Reply to Alice" --done
20260420-reply_to_alice
```

---

## 3. Clarify

Process inbox items — inspect, refine, or discard.

### 3.1 `htd clarify list`

List all items currently in the inbox.

```
htd clarify list
```

**Behavior:**

1. Read all files in `items/inbox/`.
2. Display a list of items with columns: `ID`, `TITLE`, `CREATED_AT`.
3. Sort by `created_at` ascending (oldest first).

**JSON output:** Array of item objects with all front matter fields.

### 3.2 `htd clarify show`

Display a single inbox item in detail.

```
htd clarify show ID
```

| Argument | Required | Description |
|----------|----------|-------------|
| `ID` | yes | The item ID |

**Behavior:**

1. Look up the item file by ID in `items/inbox/`.
2. Display all front matter fields and the body content.
3. Exit with code `2` if the item is not found in the inbox.

### 3.3 `htd clarify update`

Update the content of an inbox item.

```
htd clarify update ID [--title TEXT] [--body TEXT] [--ref URL]...
```

| Option | Required | Description |
|--------|----------|-------------|
| `--title` | no | New title |
| `--body` | no | New body content |
| `--ref` | no | New reference URL; repeatable. Each invocation of this command replaces the full `refs` list with the supplied values. Pass no `--ref` at all to leave existing refs untouched. |

**Behavior:**

1. Look up the item in `items/inbox/`.
2. Update the specified fields.
3. Set `updated_at` to the current timestamp.
4. Write the file back.

At least one of `--title`, `--body`, or `--ref` must be provided.

### 3.4 `htd clarify discard`

Discard an inbox item that is not actionable and not worth keeping.

```
htd clarify discard ID
```

**Behavior:**

1. Look up the item in `items/inbox/`.
2. Set `status: discarded`.
3. Set `updated_at` to the current timestamp.
4. Move the file to `archive/items/<id>.md`.

**Constraints:**

- Only inbox items can be discarded. Items that have already been moved out of the inbox (i.e., `kind != inbox`) must be canceled via `engage cancel` instead.

---

## 4. Organize

Categorize, link, and schedule items.

### 4.1 `htd organize move`

Change the category (kind) of one or more items. Moves each file to the corresponding directory.

```
htd organize move KIND ID [ID...]
```

| Argument | Required | Description |
|----------|----------|-------------|
| `KIND` | yes | Target kind: `next_action`, `project`, `waiting_for`, `someday`, `tickler` |
| `ID` | yes | One or more item IDs to move to `KIND` |

**Behavior:**

1. For each `ID`, in order:
   1. Find the item file across all `items/<kind>/` directories.
   2. Update `kind` in front matter to `KIND`.
   3. Set `updated_at` to the current timestamp.
   4. Move the file to `items/<new-kind>/<id>.md`.
2. On the first failure (missing ID, terminal status, etc.), stop processing and exit with an error. IDs processed before the failure remain moved.

**Constraints:**

- Cannot move to `inbox` (items enter inbox only via `capture add`).
- Cannot move archived items (status must be `active`).

**Example:**

```
$ htd organize move someday 20260417-read_article 20260417-try_tool 20260417-watch_talk
```

### 4.2 `htd organize link`

Link an item to a project.

```
htd organize link ID --project PROJECT_ID
```

| Option | Required | Description |
|--------|----------|-------------|
| `--project` | yes | The ID of a project-kind item |

**Behavior:**

1. Verify that `PROJECT_ID` exists and has `kind: project`.
2. Set the `project` field in the item's front matter.
3. Set `updated_at` to the current timestamp.

To unlink, pass `--project ""` (empty string).

### 4.3 `htd organize schedule`

Set scheduling-related dates on an item.

```
htd organize schedule ID [--due DATE] [--defer DATE] [--review DATE]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--due` | no | Due date (`YYYY-MM-DD` or `YYYY-MM-DDThh:mm:ss±hh:mm`) |
| `--defer` | no | Defer-until date (`YYYY-MM-DD` or `YYYY-MM-DDThh:mm:ss±hh:mm`); item is hidden until this moment |
| `--review` | no | Next review date (`YYYY-MM-DD`) |

**Behavior:**

1. Find the item and update the corresponding fields (`due_at`, `defer_until`, `review_at`).
2. Set `updated_at` to the current timestamp.

At least one date option must be provided. To clear a date, pass `--due ""`.

When a datetime is supplied, it is preserved to the second and `engage next-action` / `reflect next-actions` sort intra-day by the exact moment. A date-only value is interpreted as midnight in the local timezone.

### 4.4 `htd organize promote`

Promote an item to a project in one shot, creating and linking initial next-action children.

```
htd organize promote ID --child TITLE [--child TITLE]...
```

| Option | Required | Description |
|--------|----------|-------------|
| `--child` | yes | Title of a next-action child to create and link; repeatable (at least one required) |

**Behavior:**

1. Find the parent item across all `items/<kind>/` directories.
2. If the parent's `kind` is not already `project`, set `kind: project`, update `updated_at`, and move the file to `items/project/<id>.md`.
3. For each `--child TITLE`, in order:
   - Generate an ID via the usual `YYYYMMDD-<slug>` rule, with collision suffixing (`_2`, `_3`, ...) so same-titled siblings stay distinct.
   - Create an item with `kind: next_action`, `status: active`, `project: <parent-id>`, and both timestamps set to now.
   - Write it to `items/next_action/<id>.md`.
4. Print the parent ID followed by each child ID, one per line. With `--json`, print a single object of shape `{"parent": "<id>", "children": ["<id>", ...]}`.

**Constraints:**

- The parent must exist and have an active status; terminal items cannot be promoted.
- If the parent is already `kind: project`, the command skips the kind change and still creates/links the requested children (idempotent parent, additive children).
- This command only creates next-action children; to promote a parent without adding children, use `organize move ID project`.

**Example:**

```
$ htd organize promote 20260420-launch_cli \
    --child "Verify on staging" \
    --child "Release to production"
20260420-launch_cli
20260420-verify_on_staging
20260420-release_to_production
```

---

## 5. Reflect

Review and inspect the state of the system.

### 5.1 `htd reflect next-actions`

List all active next actions.

```
htd reflect next-actions
```

**Behavior:**

1. Read all files in `items/next_action/` with `status: active`.
2. Exclude items where `defer_until` is in the future.
3. Display: `ID`, `TITLE`, `PROJECT`, `DUE_AT`.
4. Sort by `due_at` ascending (items without due dates last). Datetimes sort by their exact moment; date-only values sort as midnight local time.

### 5.2 `htd reflect projects`

List all active projects and their status.

```
htd reflect projects [--stalled]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--stalled` | no | Show only projects that have no linked active next actions |

**Behavior:**

1. Read all files in `items/project/` with `status: active`.
2. For each project, count linked active `next_action` items.
3. Display: `ID`, `TITLE`, `NEXT_ACTION_COUNT`.
4. If `--stalled`, filter to projects with `NEXT_ACTION_COUNT == 0`.

### 5.3 `htd reflect waiting`

List all active waiting-for items.

```
htd reflect waiting
```

**Behavior:**

1. Read all files in `items/waiting_for/` with `status: active`.
2. Display: `ID`, `TITLE`, `CREATED_AT`.
3. Sort by `created_at` ascending.

### 5.4 `htd reflect review`

List items that are due for review.

```
htd reflect review
```

**Behavior:**

1. Scan all active items across all kinds.
2. Filter to items where `review_at` is today or in the past.
3. Display: `ID`, `TITLE`, `KIND`, `REVIEW_AT`.
4. Sort by `review_at` ascending.

### 5.5 `htd reflect log`

List recently resolved items — an activity log for daily standups, weekly reviews, and retros.

```
htd reflect log --since DATE [--until DATE] [--kind KIND] [--tag TAG]... [--status STATUS]...
```

| Option | Required | Description |
|--------|----------|-------------|
| `--since` | yes | Show items updated on or after this date (`YYYY-MM-DD`) |
| `--until` | no | Show items updated on or before this date (`YYYY-MM-DD`); inclusive end-of-day |
| `--kind` | no | Filter by kind |
| `--tag` | no | Filter by tag; repeatable (items must match all supplied tags) |
| `--status` | no | Filter by terminal status; repeatable. Values: `done`, `canceled`, `discarded`, `archived`. Defaults to `done` when omitted. |

**Behavior:**

1. Read all files in `archive/items/` matching the status filter.
2. Filter to items where `updated_at >= --since` (and `<= --until` if given).
3. Apply `--kind` and `--tag` filters.
4. Display: `ID`, `KIND`, `STATUS`, `UPDATED_AT`, `TITLE`.
5. Sort by `updated_at` descending.

**Examples:**

```
# What did I finish today?
$ htd reflect log --since 2026-04-20

# Weekly wrap-up, including canceled items
$ htd reflect log --since 2026-04-14 --status done --status canceled

# What docs-tagged next actions closed this month?
$ htd reflect log --since 2026-04-01 --kind next_action --tag docs
```

### 5.6 `htd reflect tickler`

List tickler items whose trigger date has arrived, or pull them into the inbox for re-clarification.

```
htd reflect tickler [--pull]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--pull` | no | Move fired tickler items into `items/inbox/` instead of just listing them |

**Behavior:**

1. Read all files in `items/tickler/` with `status: active`.
2. For each item, take `defer_until` as the trigger; if absent, fall back to `review_at`; if both are absent, skip.
3. Keep items whose trigger is today or in the past.
4. Sort by trigger ascending (earliest first).
5. Without `--pull`: display `ID`, `TITLE`, `DEFER_UNTIL`. No state change.
6. With `--pull`, for each selected item in trigger order:
   - Set `kind: inbox`.
   - Clear `defer_until` (the deferral has served its purpose). `review_at` is preserved.
   - Set `updated_at` to the current timestamp.
   - Move the file to `items/inbox/<id>.md`.
   - Print the moved ID to stdout.

Pulled items land in the inbox as unclarified input, so they flow through the normal `clarify` process — a fired tickler is a prompt to re-decide, not an auto-promotion to `next_action`.

**JSON output:**

- Without `--pull`: array of items.
- With `--pull`: `{"pulled": ["<id>", ...]}`.

**Examples:**

```
# See what fired today
$ htd reflect tickler

# Empty the fired items into the inbox for clarify
$ htd reflect tickler --pull
20260417-quarterly_review_prep
```

---

## 6. Engage

Choose and complete work.

### 6.1 `htd engage done`

Mark one or more items as completed.

```
htd engage done ID [ID...]
```

**Behavior:**

1. For each `ID`, in order:
   1. Find the item across all `items/<kind>/` directories.
   2. Set `status: done`.
   3. Set `updated_at` to the current timestamp.
   4. Move the file to `archive/items/<id>.md`.
2. On the first failure (missing ID, already-terminal item, etc.), stop processing and exit with an error. IDs processed before the failure remain marked done.

**Constraints:**

- Only active items can be marked as done.

### 6.2 `htd engage cancel`

Cancel one or more active items that are no longer being pursued.

```
htd engage cancel ID [ID...]
```

**Behavior:**

1. For each `ID`, in order:
   1. Find the item across all `items/<kind>/` directories.
   2. Set `status: canceled`.
   3. Set `updated_at` to the current timestamp.
   4. Move the file to `archive/items/<id>.md`.
2. On the first failure, stop processing and exit with an error. IDs processed before the failure remain canceled.

**Constraints:**

- Only active items can be canceled.
- `inbox` items can also be canceled via this command, though `clarify discard` is preferred for inbox items that were never actionable.

### 6.3 `htd engage next-action`

List next actions that are ready to work on now.

```
htd engage next-action [--project PROJECT_ID] [--tag TAG]...
```

| Option | Required | Description |
|--------|----------|-------------|
| `--project` | no | Filter by project ID |
| `--tag` | no | Filter by tag; repeatable (items must match all supplied tags) |

**Behavior:**

1. Read all files in `items/next_action/` with `status: active`.
2. Exclude items where `defer_until` is in the future.
3. Apply `--project` and `--tag` filters if provided.
4. Display: `ID`, `TITLE`, `PROJECT`, `DUE_AT`.
5. Sort by `due_at` ascending (items without due dates last). Datetimes sort by their exact moment; date-only values sort as midnight local time.

This command overlaps with `reflect next-actions` in content; the difference is intent (Engage = pick work; Reflect = review system) plus the filter flags above.

### 6.4 `htd engage waiting`

List waiting-for items that need follow-up because they have been untouched for a while.

```
htd engage waiting [--stale-days N]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--stale-days` | no | Stale threshold in days (default `7`). Items whose `updated_at` is older than this are shown. |

**Behavior:**

1. Read all files in `items/waiting_for/` with `status: active`.
2. For each item, compute age as `now - updated_at` (or `created_at` if `updated_at` is absent).
3. Keep items with age ≥ `--stale-days`.
4. Display: `ID`, `TITLE`, `AGE_DAYS`, `UPDATED_AT`.
5. Sort by age descending (oldest first).

**JSON output:** Array of items with an added `age_days` integer field.

---

## 7. Item (Low-Level CRUD)

Direct access to items without workflow constraints. Intended for scripting, automation, and agent integration.

### 7.1 `htd item get`

Retrieve a single item by ID.

```
htd item get ID
```

**Behavior:**

1. Search across all `items/<kind>/` directories and `archive/items/` for the given ID.
2. Display the full item (front matter + body).
3. Exit with code `2` if not found.

### 7.2 `htd item list`

List items with optional filters.

```
htd item list [--kind KIND] [--status STATUS] [--tag TAG] [--project PROJECT_ID]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--kind` | no | Filter by kind |
| `--status` | no | Filter by status (default: `active`) |
| `--tag` | no | Filter by tag; repeatable |
| `--project` | no | Filter by project ID |

**Behavior:**

1. Scan the appropriate directories based on filters.
2. If `--status` includes non-active statuses, also scan `archive/items/`.
3. Display: `ID`, `TITLE`, `KIND`, `STATUS`, `UPDATED_AT`.

### 7.3 `htd item update`

Update arbitrary fields on an item.

```
htd item update ID FIELD=VALUE [FIELD=VALUE]...
```

**Behavior:**

1. Find the item by ID.
2. Update each specified front matter field.
3. Set `updated_at` to the current timestamp.
4. If `kind` is changed, move the file to the appropriate directory.
5. If `status` is changed to a non-active status, move to `archive/items/`.

**Protected fields** (cannot be changed):

- `id`
- `created_at`

**Examples:**

```
$ htd item update 20260417-write_the_man_page kind=next_action
$ htd item update 20260417-write_the_man_page tags='[cli,docs,v1]'
$ htd item update 20260417-write_the_man_page refs='[https://github.com/foo/bar/pull/42]'
```

### 7.4 `htd item archive`

Move an active item to the archive as a last resort (when neither `done` nor `canceled` semantically applies).

```
htd item archive ID
```

**Behavior:**

1. Find the item across all `items/<kind>/` directories.
2. Set `status: archived`.
3. Set `updated_at` to the current timestamp.
4. Move the file to `archive/items/<id>.md`.

**Constraints:**

- Only active items can be archived this way.
- Prefer `engage done` or `engage cancel` for normal workflow completion. Use `item archive` only when an item needs to be removed from active lists without a clear done/canceled semantics (e.g., a project that was superseded rather than finished).

### 7.5 `htd item restore`

Bring a terminal item back to active status. Symmetric to `engage done` / `engage cancel` / `item archive`.

```
htd item restore ID
```

**Behavior:**

1. Find the item in `archive/items/`.
2. Set `status: active`.
3. Set `updated_at` to the current timestamp.
4. Move the file back to `items/<kind>/<id>.md`, based on the item's `kind` front-matter value.

**Constraints:**

- The item must have a terminal status (`done`, `canceled`, `discarded`, or `archived`). Restoring an already-active item fails.
- Restoring a `discarded` inbox item lands it back in `items/inbox/` for re-clarification.
- Use this for error correction when an item was terminated by mistake; prefer `item update` only when field-level edits are truly needed.

---

## 8. Init

### 8.1 `htd init`

Create the htd directory layout at the root and print the directory set.

```
htd init
```

**Behavior:**

1. Ensure every directory required by the storage layout (see `docs/datamodel.md §6`) exists under the root specified by `--path`.
2. Print the full directory set to stdout, one path per line in a stable order.

**Notes:**

- Running other `htd` commands also creates any missing directories as a side effect; `htd init` exists to make the setup explicit and to confirm the layout.
- The command is idempotent — re-running it produces the same output and does not modify existing files.

**JSON output:** A JSON array of directory paths.

**Example:**

```
$ htd init
items/inbox
items/next_action
items/project
items/waiting_for
items/someday
items/tickler
archive/items
archive/reference
reference
```

---

## 9. Completion

### 9.1 `htd completion`

Emit a shell completion script for the given shell to stdout.

```
htd completion SHELL
```

| Argument | Required | Description |
|----------|----------|-------------|
| `SHELL` | yes | One of `bash`, `zsh`, `fish`, `powershell` |

**Behavior:**

1. Write the completion script for `SHELL` to stdout.
2. The script covers all command groups, subcommands, and flags.
3. Does not read or create any files under `--path`; safe to run anywhere.

**Examples:**

```
# Bash (current session)
source <(htd completion bash)

# Zsh (one-time install)
htd completion zsh > "${fpath[1]}/_htd"
```

---

## 10. Command Summary

| Command | Description |
|---------|-------------|
| `htd init` | Create the htd directory layout |
| `htd capture add` | Add a new item to the inbox |
| `htd clarify list` | List inbox items |
| `htd clarify show ID` | Show an inbox item |
| `htd clarify update ID` | Update an inbox item |
| `htd clarify discard ID` | Discard an inbox item |
| `htd organize move KIND ID [ID...]` | Change the category of one or more items |
| `htd organize link ID --project PID` | Link item to a project |
| `htd organize schedule ID` | Set dates on an item |
| `htd organize promote ID --child TITLE...` | Promote to a project with next-action children |
| `htd reflect next-actions` | List active next actions |
| `htd reflect projects` | List active projects |
| `htd reflect waiting` | List waiting-for items |
| `htd reflect review` | List items due for review |
| `htd reflect log --since DATE` | List recently resolved items (activity log) |
| `htd reflect tickler [--pull]` | List fired tickler items, or pull them into the inbox |
| `htd engage done ID [ID...]` | Mark one or more items as done |
| `htd engage cancel ID [ID...]` | Cancel one or more active items |
| `htd engage next-action` | List next actions ready to work on now |
| `htd engage waiting` | List waiting-for items that need follow-up |
| `htd item get ID` | Get any item by ID |
| `htd item list` | List items with filters |
| `htd item update ID` | Update item fields directly |
| `htd item archive ID` | Archive an item (last resort) |
| `htd item restore ID` | Restore a terminal item to active status |
| `htd completion SHELL` | Emit a shell completion script |
