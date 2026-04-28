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
| `reference` | Tool-scoped reference notes |
| `journal` | Time-stamped journals and retros |

### 1.3 Global Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--json` | flag | `false` | Output in JSON format instead of human-readable text |
| `--verbose` / `-v` | flag | `false` | Print per-mutation confirmations on mutating commands (see §1.6) |
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

### 1.6 Verbose Mode on Mutating Commands

Mutating commands (see below) exit silently on success by default, which keeps pipelines script-friendly. Pass `--verbose` / `-v` to surface what happened:

- **Text mode**: one `updated <id>: field=value [field=value ...]` line per item. Only the fields actually changed are shown. Date inputs are echoed back in RFC 3339 form (e.g., `--defer 2026-04-27` is reported as `defer_until=2026-04-27T00:00:00+09:00`).
- **`--json --verbose`**: a single JSON array containing the full, post-mutation item object(s), matching the shape produced by read commands like `item get` / `item list`.

Affected commands:

- `htd clarify update` / `clarify discard`
- `htd organize move` / `organize link` / `organize unlink` / `organize schedule`
- `htd engage done` / `engage cancel`
- `htd item update` / `item archive` / `item restore`
- `htd reference update` / `reference archive` / `reference restore`

Commands that already print on success (`capture add`, `reference add`, `organize promote`, `reflect tickler --pull`, `init`) ignore `--verbose`; their output is unchanged.

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

To unlink, use `htd organize unlink ID`. Passing `--project ""` is still accepted as a legacy alias and will be removed in a future release.

### 4.3 `htd organize unlink`

Clear the project link on an item.

```
htd organize unlink ID
```

| Argument | Required | Description |
|----------|----------|-------------|
| `ID` | yes | The item ID whose `project` field should be cleared |

**Behavior:**

1. Find the item across all `items/<kind>/` directories.
2. Clear the `project` field in the item's front matter.
3. Set `updated_at` to the current timestamp.

Idempotent: unlinking an item that is not currently linked to a project is a silent no-op (aside from bumping `updated_at`).

### 4.4 `htd organize schedule`

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

When a datetime is supplied, it is preserved to the second and `engage next-actions` / `reflect next-actions` sort intra-day by the exact moment. A date-only value is interpreted as midnight in the local timezone.

### 4.5 `htd organize promote`

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

### 6.3 `htd engage next-actions`

List next actions that are ready to work on now.

```
htd engage next-actions [--project PROJECT_ID] [--tag TAG]...
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

All list-returning commands in this CLI use the plural form that matches the list they query. `engage next-actions` and `reflect next-actions` both return the same shape of list; the difference between them is mechanical, not judgment-based:

- `reflect next-actions` is the survey view of the full list.
- `engage next-actions` is the narrowing view for picking work right now, via `--project` and `--tag` (and future mechanical predicates such as parent-project activity).

Priority, context, time, and energy are not applied inside `htd` — those judgments stay with the caller, who can compose a `--query` expression on `item list`, pipe the JSON, or scan the returned list by eye.

The previous singular alias `htd engage next-action` is still accepted but prints a deprecation warning; update scripts to the plural form.

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
htd item list [--kind KIND] [--status STATUS] [--tag TAG] [--project PROJECT_ID] [--query EXPR]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--kind` | no | Filter by kind |
| `--status` | no | Filter by status (default: `active`) |
| `--tag` | no | Filter by tag; repeatable |
| `--project` | no | Filter by project ID |
| `--query` | no | Filter with a query expression (see §7.2.1) |

**Behavior:**

1. Scan the appropriate directories based on filters.
2. If `--status` includes non-active statuses, also scan `archive/items/`.
3. If `--query` is given, parse the expression and keep only items that
   match. The query is AND-combined with the other filters above.
4. Display: `ID`, `TITLE`, `KIND`, `STATUS`, `UPDATED_AT`.

#### 7.2.1 Query expressions

`--query` accepts a small DSL for filtering items. It composes with the
other flags with AND: items must pass both the flag filters and the
query.

**Grammar**

```
query      := orExpr
orExpr     := andExpr { OR andExpr }
andExpr    := unaryExpr { unaryExpr }      // implicit AND by juxtaposition
unaryExpr  := [NOT | "-"] primary
primary    := term | "(" query ")"
term       := [field ":"] value
value      := bareword | "quoted string"
```

- Space between terms means AND. Explicit `AND` is also accepted.
- `OR` chooses either side. `AND` binds tighter than `OR`.
- `NOT` (or a leading `-`) negates the following primary only. To negate a
  compound, wrap it in parentheses: `NOT (a b)`.
- Quoted values allow whitespace and support `\"` and `\\` as the only
  escapes.

**Fields (whitelist)**

`id`, `title`, `body`, `kind`, `status`, `project`, `source`, `tag`, `ref`.

`tag` and `ref` are singular (matching the `--tag` / `--ref` flags) and
map to the `tags` and `refs` arrays — a term matches if any element of
the array contains the needle. Unknown fields are a parse error.

Without a field, the needle is searched across `title`, `body`, `tags`,
`refs`, `source`, `project`, and `id`. `kind` and `status` are not
searched unfielded (use `kind:` or `status:` explicitly).

**Matching**

- Default operator is case-insensitive substring match.
- Date fields (`due_at`, `defer_until`, `review_at`, `created_at`,
  `updated_at`) are not searchable in v1.
- URL values that contain `:` (e.g., `https://…`) must be quoted:
  `ref:"https://github.com/foo/bar"`. Short hostname matches work
  unquoted: `ref:github.com`.

**Composition notes**

- The default `--status active` still applies. To search across
  terminated items, pass `--status ''` or an explicit status (e.g.,
  `--status done`) alongside `--query`. The query itself does not
  expand the scan into the archive.
- `--query ''` (empty string) matches every item — convenient for
  shell scripts that build the query from a variable.
- Invalid expressions exit with code `1` and an error on stderr.

**Examples**

```
# Substring search across all fields
htd item list --query 'panic'

# Fielded match
htd item list --query 'title:"fix panic"'
htd item list --query 'ref:github.com'

# Boolean composition
htd item list --query 'ref:github.com OR ref:notion.so'
htd item list --query '(ref:github.com OR ref:notion.so) tag:bug'

# Negation
htd item list --query '-status:done title:"refactor"'

# Compose with flags (AND)
htd item list --kind next_action --query 'tag:bug'
```

### 7.3 `htd item update`

Update fields on an item. Each argument is a `FIELD=VALUE` pair; multiple pairs are applied in order and written in a single file update.

```
htd item update ID FIELD=VALUE [FIELD=VALUE]...
```

**Supported fields:**

| Field | Format | Notes |
|-------|--------|-------|
| `title` | string | Short description |
| `body` | string | Markdown body (the content after front matter, not a front-matter field itself) |
| `kind` | enum | One of `inbox`, `next_action`, `project`, `waiting_for`, `someday`, `tickler` |
| `status` | enum | One of `active`, `done`, `canceled`, `discarded`, `archived` |
| `project` | string | ID of a project-kind item |
| `source` | string | Origin string (e.g., `manual`, `email`, `slack`) |
| `tags` | list | Comma-separated, optionally bracketed: `foo,bar` or `[foo,bar]`. Pass `tags=` (empty) to clear. |
| `refs` | list | Comma-separated URL list; same syntax as `tags`. |
| `due_at` | date / datetime | `YYYY-MM-DD` or RFC 3339 (e.g., `2026-05-01T14:30:00+09:00`). Pass `due_at=` to clear. |
| `defer_until` | date / datetime | Same format as `due_at`. |
| `review_at` | date | Same format as `due_at`. |

Unknown fields are rejected with an error that lists the supported fields.

**Protected fields** (cannot be changed):

- `id`
- `created_at`

**Behavior:**

1. Find the item by ID.
2. Update each specified front matter field (or body, for `body=`).
3. Set `updated_at` to the current timestamp.
4. If `kind` is changed, move the file to the appropriate directory.
5. If `status` is changed to a non-active status, move to `archive/items/`.

**Cross-references:**

`item update` is the low-level CRUD entry point (see §7 intro). For the normal workflow, prefer the dedicated commands:

- `organize schedule` — set `due_at` / `defer_until` / `review_at`
- `organize link` / `organize unlink` — set or clear `project`
- `organize move` — change `kind`
- `engage done` / `engage cancel` / `item archive` / `item restore` — change `status`

Scheduling, linking, and kind changes are accepted here as well so that scripts and agents can set several fields in one call; humans working through the five-phase workflow should reach for the workflow commands instead.

**Examples:**

```
$ htd item update 20260417-write_the_man_page kind=next_action
$ htd item update 20260417-write_the_man_page tags='[cli,docs,v1]'
$ htd item update 20260417-write_the_man_page refs='[https://github.com/foo/bar/pull/42]'
$ htd item update 20260417-write_the_man_page due_at=2026-05-01 body="Draft outline first."
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

## 8. Reference

The `reference` command group manages tool-scoped reference notes — durable, AI-readable context stored under `reference/<tool>/` (see `docs/datamodel.md §3`). Notes are grouped per tool so multi-assistant repos do not collide; the `--tool` flag selects the namespace and defaults to `claude`.

Each tool directory has an auto-generated `INDEX.md` that lists every active reference grouped by `type:*` tag (`## user`, `## feedback`, `## area_of_focus`, `## project`, `## reference`, with a trailing `## other` for entries that carry no canonical type tag). The index is rewritten on every mutation; see §8.5 for the exact format and the repair verb.

### 8.1 `htd reference add`

Add a new reference.

```
htd reference add --title TEXT [--body TEXT] [--tag TAG]... [--tool TOOL]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--title` | yes | Short description |
| `--body` | no | Body content (Markdown) |
| `--tag` | no | Tag (repeatable). Use `type:user`, `type:feedback`, `type:area_of_focus`, `type:project`, or `type:reference` to drive `INDEX.md` grouping; other tags appear in `## other`. |
| `--tool` | no | Tool namespace (default `claude`). Determines `reference/<tool>/`. |

**Behavior:**

1. Generate a new ID following the same `YYYYMMDD-<slug>` rule used by items, with collision suffixing (`_2`, `_3`, ...) checked across both items and references in every tool — IDs are globally unique per `docs/datamodel.md §5.2`.
2. Set `created_at` and `updated_at` to the current timestamp.
3. Create `reference/<tool>/` if missing (lazy — no separate setup step is required for a new tool).
4. Write the file to `reference/<tool>/<id>.md`.
5. Rewrite `reference/<tool>/INDEX.md`.
6. Print the created ID to stdout.

**Example:**

```
$ htd reference add --title "Branch + PR workflow" --tag type:feedback \
    --body "Every change lands via a feature branch, never directly on main."
20260427-branch_pr_workflow
```

### 8.2 `htd reference get`

Retrieve a single reference by ID.

```
htd reference get ID
```

| Argument | Required | Description |
|----------|----------|-------------|
| `ID` | yes | The reference ID |

**Behavior:**

1. Search across every tool directory under `reference/`, then `archive/reference/`, for the given ID. Active hits win when both exist.
2. Display the front matter and body. When the hit comes from the archive, prepend an `(archived)` line above the metadata block in text mode and set `archived: true` in `--json` mode (the field is omitted entirely for active hits).
3. Exit with code `2` if the reference is not found.

**JSON output:** A single object with the reference fields plus a `tool` field. `archived: true` appears only when the reference came from the archive.

### 8.3 `htd reference list`

List references for a tool.

```
htd reference list [--tool TOOL] [--tag TAG] [--archived]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--tool` | no | Tool namespace (default `claude`) |
| `--tag` | no | Filter to references containing this exact tag |
| `--archived` | no | List archived references under `archive/reference/<tool>/` instead of active ones |

**Behavior:**

1. Read all `*.md` files under `reference/<tool>/` (excluding `INDEX.md`). When `--archived` is set, scan `archive/reference/<tool>/` instead — the active and archived views are mutually exclusive.
2. Apply the `--tag` filter if given.
3. Display columns: `ID`, `TOOL`, `UPDATED_AT`, `TITLE`. Archived rows carry an `(archived)` prefix on the title column in text mode.
4. Sort by `updated_at` descending, with `id` ascending as the deterministic tiebreaker.

**JSON output:** Array of reference objects. `archived: true` is set on every row when `--archived` is used.

### 8.4 `htd reference update`

Update fields on a reference.

```
htd reference update ID FIELD=VALUE [FIELD=VALUE]...
```

**Supported fields:**

| Field | Format | Notes |
|-------|--------|-------|
| `title` | string | Short description |
| `body` | string | Markdown body (the content after front matter) |
| `tags` | list | Comma-separated, optionally bracketed: `foo,bar` or `[foo,bar]`. Pass `tags=` (empty) to clear. |

**Protected fields** (cannot be changed):

- `id`
- `created_at`
- `tool` — to move a reference between tools, archive and re-add.

**Behavior:**

1. Find the reference by ID across every tool directory (active first, archive fallback).
2. Update each specified field in order.
3. Set `updated_at` to the current timestamp.
4. Rewrite the active `INDEX.md` for the owning tool when the reference is active (regrouping when `tags` changes the `type:*` tag). Archived references do not appear in `INDEX.md`, so the file is not rewritten when an archived reference is updated.

### 8.5 `htd reference archive`

Archive a reference. Archival is location-only — references have no `status` field. The file moves from `reference/<tool>/<id>.md` to `archive/reference/<tool>/<id>.md` and the active `INDEX.md` is rewritten to drop the entry.

```
htd reference archive ID
```

| Argument | Required | Description |
|----------|----------|-------------|
| `ID` | yes | The reference ID |

**Behavior:**

1. Locate the reference. Archived references are rejected; archiving is one-way.
2. Update `updated_at` to the current timestamp.
3. Move the file to `archive/reference/<tool>/<id>.md`.
4. Rewrite `reference/<tool>/INDEX.md`. When the last active reference for a tool is archived, INDEX.md falls back to the empty-state stub (the file is still written rather than deleted so archive-then-empty stays diff-clean).

### 8.6 `htd reference restore`

Restore an archived reference to active. Symmetric inverse of `archive`.

```
htd reference restore ID
```

| Argument | Required | Description |
|----------|----------|-------------|
| `ID` | yes | The reference ID |

**Behavior:**

1. Locate the reference. Active references are rejected.
2. Update `updated_at` to the current timestamp.
3. Move the file from `archive/reference/<tool>/<id>.md` back to `reference/<tool>/<id>.md`.
4. Rewrite the active `INDEX.md`.

### 8.7 `INDEX.md` format

`reference/<tool>/INDEX.md` is generated by the `reference` commands and is treated as a scoped exception to `docs/datamodel.md §10` ("no generated index files"). The exception exists so AI sessions can recover context cheaply at startup without scanning the filesystem.

The format is fully deterministic — the same set of references produces a byte-for-byte identical file:

- An H1 line: `# Reference index`.
- One section per non-empty `type:*` group, in fixed order: `## user`, `## feedback`, `## area_of_focus`, `## project`, `## reference`. Entries whose canonical type tag is missing or unrecognized (e.g. `type:misc`) land in a trailing `## other` section.
- Within each section, entries sort by `updated_at` descending, with `id` ascending as the tiebreaker.
- Each entry is one bullet line: `- [title](id.md) — short description`. The "short description" is the first non-blank line of the body with leading `#` stripped, truncated to 80 runes. The em-dash and description are omitted when no usable description line exists.
- When no references are present, the body is the empty-state stub `_No entries._` (the file is still written rather than deleted, so archive-then-empty stays diff-clean).

INDEX.md is active-only. Archived references do not appear; use `htd reference list --archived` to inspect the archive.

### 8.8 `htd reference reindex`

Rewrite `reference/<tool>/INDEX.md` from the current set of references. This is a repair verb only — the index is already kept in sync by every mutation; reach for `reindex` when the file diverged from disk through manual edits or merge conflicts.

```
htd reference reindex [--tool TOOL]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--tool` | no | Tool namespace (default `claude`) |

**Behavior:**

1. Scan `reference/<tool>/` for active references.
2. Render `INDEX.md` per §8.7 and write it atomically.
3. Idempotent: running `reindex` twice produces the same disk state.

There is no `htd reference index` noun-verb — `htd reference list` already covers "show me what's there."

---

## 9. Journal

The `journal` command group manages a time-stamped lane for daily journals, weekly retros, and ad-hoc observation logs. Journals fit neither Items (not actionable) nor References (not durable lookup); they capture the user's own observations and are read on demand. They are stored under `journal/` at the htd root, flat and tool-agnostic — there is no per-tool namespacing because journals belong to the user, not to any AI assistant.

Journal entries are write-once Markdown. There is no `update`, `archive`, or `delete` verb — users edit entries in `$EDITOR` and Git handles archival.

### 9.1 `htd journal new`

Create a new journal entry from a template. The file is not overwritten if it already exists.

```
htd journal new [--type daily|weekly|adhoc] [--date YYYY-MM-DD] [--title TEXT] [--tag TAG]...
```

| Option | Required | Description |
|--------|----------|-------------|
| `--type` | no | Entry type: `daily` (default), `weekly`, or `adhoc` |
| `--date` | no | Date in `YYYY-MM-DD` form. Defaults to today; for `--type=weekly`, snaps to the Monday of the ISO week containing the date. |
| `--title` | yes (adhoc only) | Title for ad-hoc entries; the slug becomes the filename. |
| `--tag` | no | Tag (repeatable). Stored in YAML frontmatter. |

**Filename rules:**

| Type | Filename |
|------|----------|
| `daily` | `YYYY-MM-DD.md` (e.g. `2026-04-28.md`) |
| `weekly` | `YYYY-MM-DD-weekly.md` where the date is the Monday of the ISO week (e.g. `2026-04-27-weekly.md`) |
| `adhoc` | `<slug>.md` derived from `--title` in snake_case (e.g. `postmortem_on_outage.md`) |

**Behavior:**

1. Resolve the date (default today; weekly snaps to Monday of that week).
2. Derive the filename per the rules above. Refuse to overwrite an existing file.
3. Write a small Markdown scaffold with optional YAML frontmatter (`created_at`, `updated_at`, `tags`).
4. Print the resulting name (filename without `.md`) to stdout.

**Templates:**

- Daily: `# YYYY-MM-DD` followed by `## What I did`, `## What I learned`, `## Tomorrow`.
- Weekly: `# Week of YYYY-MM-DD` followed by `## Wins`, `## Misses`, `## Lessons`, `## Focus next week`.
- Ad-hoc: H1 from `--title`, no further scaffold.

**Example:**

```
$ htd journal new
2026-04-28

$ htd journal new --type weekly --date 2026-04-30
2026-04-27-weekly

$ htd journal new --type adhoc --title "Postmortem on outage" --tag retro
postmortem_on_outage
```

### 9.2 `htd journal list`

List journal entries, most recent first.

```
htd journal list [--since YYYY-MM-DD]
```

| Option | Required | Description |
|--------|----------|-------------|
| `--since` | no | Show entries on or after this date. The filename-derived date is the primary key (so the filter matches what users intuit from "since 2026-04-15"); `created_at` is the fallback for ad-hoc entries that carry no date prefix. |

**Behavior:**

1. Read every `*.md` file under `journal/` (subdirectories are ignored).
2. Apply the `--since` filter if given.
3. Display columns: `NAME`, `CREATED_AT`, `TAGS`.
4. Sort by filename descending.

**JSON output:** Array of journal objects (`name`, `created_at`, `updated_at`, `tags`); body is omitted from list output.

### 9.3 `htd journal show`

Show a single journal entry.

```
htd journal show NAME
```

| Argument | Required | Description |
|----------|----------|-------------|
| `NAME` | yes | Filename without `.md` extension (e.g. `2026-04-28`, `2026-04-27-weekly`, `postmortem_on_outage`) |

**Behavior:**

1. Read the file at `journal/<name>.md`.
2. Display front matter (if present) followed by the body.
3. Exit with code `2` if not found.

Hand-edited entries that omit YAML frontmatter are accepted — `show` returns the file as plain Markdown.

**JSON output:** A single object with `name`, optional `created_at`/`updated_at`/`tags`, and `body`.

---

## 10. Init

### 10.1 `htd init`

Create the htd directory layout at the root and print the directory set.

```
htd init
```

**Behavior:**

1. Ensure every directory required by the storage layout (see `docs/datamodel.md §7`) exists under the root specified by `--path`.
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
journal
```

---

## 11. Completion

### 11.1 `htd completion`

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

## 12. Command Summary

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
| `htd organize unlink ID` | Clear the project link on an item |
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
| `htd engage next-actions` | List next actions ready to work on now |
| `htd engage waiting` | List waiting-for items that need follow-up |
| `htd item get ID` | Get any item by ID |
| `htd item list` | List items with filters (supports `--query` DSL) |
| `htd item update ID` | Update item fields directly |
| `htd item archive ID` | Archive an item (last resort) |
| `htd item restore ID` | Restore a terminal item to active status |
| `htd reference add` | Add a reference under `reference/<tool>/` |
| `htd reference get ID` | Get a reference by ID (with `(archived)` marker if applicable) |
| `htd reference list [--archived]` | List active references for a tool, or archived ones |
| `htd reference update ID` | Update reference fields |
| `htd reference archive ID` | Archive a reference |
| `htd reference restore ID` | Restore an archived reference |
| `htd reference reindex` | Rewrite `reference/<tool>/INDEX.md` (repair) |
| `htd journal new` | Create a daily/weekly/ad-hoc journal entry from a template |
| `htd journal list` | List journal entries (most recent first) |
| `htd journal show NAME` | Show a single journal entry |
| `htd completion SHELL` | Emit a shell completion script |
