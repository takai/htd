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
| `--path` | string | `.` (current directory) | Specify the htd root directory |

Global options may appear before or after the command group.

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
htd capture add --title TEXT [--body TEXT] [--source NAME] [--tag TAG]...
```

| Option | Required | Description |
|--------|----------|-------------|
| `--title` | yes | Short description of the item |
| `--body` | no | Detailed description (Markdown) |
| `--source` | no | Origin of the item (e.g., `email`, `meeting`, `slack`) |
| `--tag` | no | Tag to attach; repeatable for multiple tags |

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
htd clarify update ID [--title TEXT] [--body TEXT]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--title` | no | New title |
| `--body` | no | New body content |

**Behavior:**

1. Look up the item in `items/inbox/`.
2. Update the specified fields.
3. Set `updated_at` to the current timestamp.
4. Write the file back.

At least one of `--title` or `--body` must be provided.

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

Change an item's category (kind). Moves the file to the corresponding directory.

```
htd organize move ID KIND
```

| Argument | Required | Description |
|----------|----------|-------------|
| `ID` | yes | The item ID |
| `KIND` | yes | Target kind: `next_action`, `project`, `waiting_for`, `someday`, `tickler` |

**Behavior:**

1. Find the item file across all `items/<kind>/` directories.
2. Update `kind` in front matter to the new value.
3. Set `updated_at` to the current timestamp.
4. Move the file to `items/<new-kind>/<id>.md`.

**Constraints:**

- Cannot move to `inbox` (items enter inbox only via `capture add`).
- Cannot move archived items (status must be `active`).

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
| `--defer` | no | Defer-until date (item is hidden until this date) |
| `--review` | no | Next review date |

**Behavior:**

1. Find the item and update the corresponding fields (`due_at`, `defer_until`, `review_at`).
2. Set `updated_at` to the current timestamp.

At least one date option must be provided. To clear a date, pass `--due ""`.

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
4. Sort by `due_at` ascending (items without due dates last).

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

### 5.5 `htd reflect done`

List recently completed items.

```
htd reflect done --since DATE
```

| Option | Required | Description |
|--------|----------|-------------|
| `--since` | yes | Show items completed since this date (`YYYY-MM-DD`) |

**Behavior:**

1. Read all files in `archive/items/` with `status: done`.
2. Filter to items where `updated_at >= --since`.
3. Display: `ID`, `TITLE`, `KIND`, `UPDATED_AT`.
4. Sort by `updated_at` descending.

---

## 6. Engage

Choose and complete work.

### 6.1 `htd engage done`

Mark an item as completed.

```
htd engage done ID
```

**Behavior:**

1. Find the item across all `items/<kind>/` directories.
2. Set `status: done`.
3. Set `updated_at` to the current timestamp.
4. Move the file to `archive/items/<id>.md`.

**Constraints:**

- Only active items can be marked as done.

### 6.2 `htd engage cancel`

Cancel an active item that is no longer being pursued.

```
htd engage cancel ID
```

**Behavior:**

1. Find the item across all `items/<kind>/` directories.
2. Set `status: canceled`.
3. Set `updated_at` to the current timestamp.
4. Move the file to `archive/items/<id>.md`.

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
5. Sort by `due_at` ascending (items without due dates last).

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

### 6.5 `htd engage tickler`

List tickler items whose trigger date has arrived.

```
htd engage tickler
```

**Behavior:**

1. Read all files in `items/tickler/` with `status: active`.
2. For each item, take `defer_until` as the trigger; if absent, fall back to `review_at`; if both are absent, skip.
3. Keep items whose trigger is today or in the past.
4. Display: `ID`, `TITLE`, `DEFER_UNTIL`.
5. Sort by trigger ascending (earliest first).

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

## 9. Command Summary

| Command | Description |
|---------|-------------|
| `htd init` | Create the htd directory layout |
| `htd capture add` | Add a new item to the inbox |
| `htd clarify list` | List inbox items |
| `htd clarify show ID` | Show an inbox item |
| `htd clarify update ID` | Update an inbox item |
| `htd clarify discard ID` | Discard an inbox item |
| `htd organize move ID KIND` | Change item category |
| `htd organize link ID --project PID` | Link item to a project |
| `htd organize schedule ID` | Set dates on an item |
| `htd reflect next-actions` | List active next actions |
| `htd reflect projects` | List active projects |
| `htd reflect waiting` | List waiting-for items |
| `htd reflect review` | List items due for review |
| `htd reflect done --since DATE` | List recently completed items |
| `htd engage done ID` | Mark an item as done |
| `htd engage cancel ID` | Cancel an active item |
| `htd engage next-action` | List next actions ready to work on now |
| `htd engage waiting` | List waiting-for items that need follow-up |
| `htd engage tickler` | List tickler items whose trigger date has arrived |
| `htd item get ID` | Get any item by ID |
| `htd item list` | List items with filters |
| `htd item update ID` | Update item fields directly |
| `htd item archive ID` | Archive an item (last resort) |
