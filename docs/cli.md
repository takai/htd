# htd CLI Specification

## 1. General

### 1.1 Binary & Structure

```
htd <command-group> <subcommand> [arguments] [options]
```

Groups: `capture`, `clarify`, `organize`, `reflect`, `engage`, `item`, `reference`, `journal`, plus `init` and `completion`.

### 1.2 Global Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--json` | flag | `false` | JSON output instead of text |
| `--verbose` / `-v` | flag | `false` | Per-mutation confirmation (see §1.5) |
| `--path` | string | `$HTD_PATH` or `.` | htd root directory (overrides env) |

May appear before or after the command group. Root resolution: `--path` → `$HTD_PATH` (non-empty) → `.`. Absolute `$HTD_PATH` is recommended; relative values resolve against cwd.

### 1.3 Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error (invalid args, I/O, etc.) |
| `2` | Item not found |

### 1.4 Output Conventions

- Default: tab-separated columns for lists, YAML-like key-value for detail views.
- `--json`: a single object (`get`/`show`) or array (`list`) on stdout.
- Errors go to stderr.

### 1.5 Verbose Mode

Mutating commands are silent on success by default. With `--verbose`:

- Text: one `updated <id>: field=value [...]` line per item; only fields actually changed. Dates echo back as RFC 3339 (e.g., `--defer 2026-04-27` → `defer_until=2026-04-27T00:00:00+09:00`).
- `--json --verbose`: array of full post-mutation objects (same shape as `item get` / `item list`).

Applies to: `clarify update`/`discard`, `organize move`/`link`/`unlink`/`schedule`, `engage done`/`cancel`, `item update`/`archive`/`restore`, `reference update`/`archive`/`restore`.

Commands that already print on success (`capture add`, `reference add`, `organize promote`, `reflect tickler --pull`, `init`) ignore `--verbose`.

### 1.6 Common Behavior

For all mutating commands:

- `updated_at` is set to the current timestamp on every change.
- When `kind` changes, the file moves to `items/<new-kind>/<id>.md`.
- When `status` becomes terminal, the file moves to `archive/items/<id>.md`.
- ID generation follows `YYYYMMDD-<slug>` (snake_case from title) with collision suffix `_2`, `_3`, … (see `docs/datamodel.md §5`).

---

## 2. Capture

### 2.1 `htd capture add`

```
htd capture add --title TEXT [--body TEXT] [--source NAME] [--tag TAG]... [--ref URL]... [--kind KIND] [--child TITLE]... [--done]
```

| Option | Req | Description |
|--------|-----|-------------|
| `--title` | yes | Short description |
| `--body` | no | Markdown body |
| `--source` | no | Origin (e.g., `email`, `slack`) |
| `--tag` | no | Tag; repeatable |
| `--ref` | no | External URL; repeatable |
| `--kind` | no | Land directly as this kind instead of `inbox`. Accepts `next_action`, `project`, `waiting_for`, `someday`, `tickler`. `inbox` is rejected as redundant. |
| `--child` | no | Child next-action title to create and link; repeatable. Requires `--kind project`. |
| `--done` | no | Capture as already completed (see below). Mutually exclusive with `--kind` and `--child`. |

**Behavior:** Generate ID, set `kind: inbox` (or the `--kind` value), `status: active`, write to `items/<kind>/<id>.md`, print the ID.

**`--kind`:** Skips the inbox and lands the item directly in `items/<kind>/`. Equivalent to `capture add` + `organize move <kind>` collapsed into one step.

**`--child` (with `--kind project`):** After writing the parent project, creates one `next_action` per `--child TITLE` with `project: <parent-id>`. Output then matches `organize promote`: parent ID followed by each child ID, one per line (or `{"parent": "...", "children": [...]}` with `--json`). Empty `--child` titles are rejected.

**`--done`:** Writes directly to `archive/items/<id>.md` with `kind: next_action`, `status: done` (no inbox stop). Metadata flags (`--body`/`--source`/`--tag`/`--ref`) still apply.

Examples:
```
$ htd capture add --title "Reply to Alice" --done
20260420-reply_to_alice

$ htd capture add --kind project --title "Launch v2" \
    --child "Verify staging" --child "Release to prod"
20260518-launch_v2
20260518-verify_staging
20260518-release_to_prod
```

---

## 3. Clarify

Process inbox items.

### 3.1 `htd clarify list`

List `items/inbox/`. Columns: `ID`, `TITLE`, `CREATED_AT`. Sort by `created_at` ascending.

### 3.2 `htd clarify show ID`

Show one inbox item (front matter + body). Exit `2` if not found in inbox.

### 3.3 `htd clarify update`

```
htd clarify update ID [--title TEXT] [--body TEXT] [--ref URL]...
```

At least one of `--title`/`--body`/`--ref` is required. Each invocation with `--ref` replaces the full `refs` list; omit `--ref` to leave existing refs alone.

### 3.4 `htd clarify discard ID`

Inbox-only. Sets `status: discarded`, moves to `archive/items/<id>.md`. For non-inbox items, use `engage cancel`.

---

## 4. Organize

### 4.1 `htd organize move KIND ID [ID...]`

`KIND`: `next_action` | `project` | `waiting_for` | `someday` | `tickler` (not `inbox`).

Processes IDs in order; stops on first failure (missing ID, terminal status). IDs processed before the failure remain moved. Only active items can be moved.

### 4.2 `htd organize link ID PROJECT_ID`

`PROJECT_ID` must exist with `kind: project`. Sets `project` field. Empty positional is rejected — use `organize unlink` to clear.

The legacy flag form `--project <pid>` (including `--project ""` for unlink) still works but is deprecated and prints a warning; it will be removed in a future release.

### 4.3 `htd organize unlink ID`

Clears `project` field. Idempotent (no-op aside from bumping `updated_at` when already unlinked).

### 4.4 `htd organize schedule`

```
htd organize schedule ID [--due DATE] [--defer DATE] [--review DATE]
```

Sets `due_at`, `defer_until`, `review_at`. At least one is required. Format: `YYYY-MM-DD` or RFC 3339 (`YYYY-MM-DDThh:mm:ss±hh:mm`). Pass `--due ""` (etc.) to clear. Date-only values are interpreted as midnight local; datetimes preserve to the second and sort by exact moment in `engage`/`reflect next-actions`.

### 4.5 `htd organize promote`

```
htd organize promote ID --child TITLE [--child TITLE]...
```

In one shot: promote parent to project (if not already) and create+link next-action children.

**Behavior:**

1. Find parent; reject if terminal.
2. If parent's `kind` is not `project`, set it and move the file.
3. For each `--child TITLE`: create a `kind: next_action`, `status: active`, `project: <parent-id>` item with the usual ID rule and collision suffixing.
4. Print parent ID then each child ID, one per line. `--json`: `{"parent": "<id>", "children": ["<id>", ...]}`.

If the parent is already a project, kind change is skipped (idempotent parent, additive children). For promotion without children, use `organize move ID project`.

---

## 5. Reflect

### 5.1 `htd reflect next-actions`

`items/next_action/`, `status: active`, exclude future `defer_until`. Columns: `ID`, `TITLE`, `PROJECT`, `DUE_AT`. Sort by `due_at` ascending (nil last); datetimes by exact moment.

### 5.2 `htd reflect projects [--stalled]`

`items/project/`, `status: active`. Columns: `ID`, `TITLE`, `NEXT_ACTION_COUNT`. `--stalled` keeps only projects with `NEXT_ACTION_COUNT == 0`.

### 5.3 `htd reflect waiting`

`items/waiting_for/`, `status: active`. Columns: `ID`, `TITLE`, `CREATED_AT`. Sort by `created_at` ascending.

### 5.4 `htd reflect review`

All active items where `review_at` is today or past. Columns: `ID`, `TITLE`, `KIND`, `REVIEW_AT`. Sort by `review_at` ascending.

### 5.5 `htd reflect log`

```
htd reflect log [--since DATE] [--until DATE] [--kind KIND] [--tag TAG]... [--status STATUS]...
```

| Option | Req | Description |
|--------|-----|-------------|
| `--since` | no | Updated on/after this date (`YYYY-MM-DD`). Default: 30 days ago. Pass `--since ""` to show all. |
| `--until` | no | Updated on/before (inclusive end-of-day) |
| `--kind` | no | Filter by kind |
| `--tag` | no | Filter by tag; repeatable (AND) |
| `--status` | no | Terminal status filter; repeatable. Values: `done`, `canceled`, `discarded`, `archived`. Default: `done`. |

Reads `archive/items/`. Columns: `ID`, `KIND`, `STATUS`, `UPDATED_AT`, `TITLE`. Sort by `updated_at` descending. JSON output is always a valid array — empty results render as `[]`.

### 5.6 `htd reflect tickler [--pull] [--all | --pending]`

`items/tickler/`, `status: active`. Trigger = `defer_until` else `review_at`; skip if both absent. Sort by trigger ascending.

Visibility flags (mutually exclusive):

- Default: fired only — items whose trigger is today or past.
- `--pending`: pending only — items whose trigger is in the future.
- `--all`: both fired and pending in one list (`--all` is the weekly-review view: "what just fired plus what's coming back").

- Without `--pull`: print `ID`, `TITLE`, `DEFER_UNTIL`. No state change.
- With `--pull`: for each fired item in order, set `kind: inbox`, clear `defer_until` (preserve `review_at`), move to `items/inbox/<id>.md`, print the ID. `--pull` rejects `--all`/`--pending` since pending items have nothing to pull.

Pulled items flow through normal clarify; a fired tickler is a re-decide prompt, not auto-promotion.

JSON: without pull → array; with pull → `{"pulled": ["<id>", ...]}`.

### 5.7 `htd reflect project ID [--since DATE]`

Show a project's metadata, body, and children in one call.

| Arg/Option | Req | Description |
|------------|-----|-------------|
| `ID` | yes | Project ID |
| `--since` | no | Cutoff for archived children (`YYYY-MM-DD`). Default: 30 days ago. `--since ""` shows all. |

**Behavior:**

1. Look up by `ID`; exit `2` if missing or `kind != project`.
2. Display project metadata + body (same shape as `clarify show` / `item get`).
3. List active children (`project: ID`) in three sections:
   - **next actions** — `kind: next_action`, sort `updated_at` desc. Columns: `ID`, `TITLE`, `TAGS`, `UPDATED_AT`.
   - **waiting for** — `kind: waiting_for`, sort `created_at` asc. Columns: `ID`, `TITLE`, `CREATED_AT`.
   - **ticklers** — `kind: tickler`, sort `defer_until` asc (nil last). Columns: `ID`, `TITLE`, `DEFER_UNTIL`.
4. **archived** — terminal children with `updated_at >= cutoff`, sort `updated_at` desc. Columns: `ID`, `KIND`, `STATUS`, `UPDATED_AT`, `TITLE`.
5. Empty sections render the header followed by `(none)`.

`someday` children are not surfaced (rare in practice).

JSON output:
```json
{
  "project":      { "id": "...", "title": "...", "body": "...", ... },
  "next_actions": [ ... ],
  "waiting_for":  [ ... ],
  "ticklers":     [ ... ],
  "archived":     [ ... ]
}
```
Project entry includes the body; child entries omit it. Empty sections render `[]`, never `null`.

---

## 6. Engage

### 6.1 `htd engage done ID [ID...]`

Processes in order; stops on first failure (missing ID, already terminal). Sets `status: done`, moves to archive. Only active items.

### 6.2 `htd engage cancel ID [ID...]`

Same shape as `done` but sets `status: canceled`. Inbox items can be canceled here, but `clarify discard` is preferred for never-actionable inbox items.

### 6.3 `htd engage next-actions`

```
htd engage next-actions [--project PROJECT_ID] [--tag TAG]...
```

`items/next_action/`, `status: active`, excluding future `defer_until`. Apply `--project`/`--tag` (multi-tag AND). Columns and sort identical to `reflect next-actions`.

`engage` is the narrowing view for picking work now; `reflect` is the survey view. Priority/context/time/energy judgments are not applied inside htd — compose via `item list --query` or pipe JSON.

Singular alias `engage next-action` still works but prints a deprecation warning.

### 6.4 `htd engage waiting`

```
htd engage waiting [--stale-days N]
```

`items/waiting_for/`, `status: active`. Age = `now - updated_at` (fallback `created_at`). Keep age ≥ `--stale-days` (default `7`). Columns: `ID`, `TITLE`, `AGE_DAYS`, `UPDATED_AT`. Sort by age descending. JSON adds `age_days`.

---

## 7. Item (Low-Level CRUD)

Direct access without workflow constraints.

### 7.1 `htd item get ID`

Search `items/<kind>/` and `archive/items/`. Display full item. Exit `2` if not found.

### 7.2 `htd item list`

```
htd item list [--kind KIND] [--status STATUS] [--tag TAG] [--project PROJECT_ID] [--query EXPR]
```

| Option | Req | Description |
|--------|-----|-------------|
| `--kind` | no | Filter by kind |
| `--status` | no | Filter by status (default `active`) |
| `--tag` | no | Filter by tag; repeatable |
| `--project` | no | Filter by project ID |
| `--query` | no | Query DSL (see §7.2.1); AND-combined with other filters |

Non-active `--status` also scans `archive/items/`. Columns: `ID`, `TITLE`, `KIND`, `STATUS`, `UPDATED_AT`.

#### 7.2.1 Query DSL

```
query      := orExpr
orExpr     := andExpr { OR andExpr }
andExpr    := unaryExpr { unaryExpr }      // implicit AND
unaryExpr  := [NOT | "-"] primary
primary    := term | "(" query ")"
term       := [field ":"] value
value      := bareword | "quoted string"
```

- Space = AND; explicit `AND` also accepted.
- `AND` binds tighter than `OR`.
- `NOT`/leading `-` negates the next primary only; wrap compounds with parens.
- Quoted values support `\"` and `\\` only.

Whitelisted fields: `id`, `title`, `body`, `kind`, `status`, `project`, `source`, `tag`, `ref`. `tag`/`ref` match any element of `tags`/`refs`. Unknown fields error.

Unfielded needles search `title`, `body`, `tags`, `refs`, `source`, `project`, `id` (not `kind`/`status`).

Matching: case-insensitive substring. Date fields not searchable in v1. URLs with `:` must be quoted (`ref:"https://github.com/foo/bar"`); short hostnames work unquoted (`ref:github.com`).

Composition: default `--status active` still applies — pass `--status ''` or explicit status to scan archive. `--query ''` matches all. Invalid expressions exit `1`.

Examples:
```
htd item list --query 'panic'
htd item list --query 'title:"fix panic"'
htd item list --query 'ref:github.com OR ref:notion.so'
htd item list --query '(ref:github.com OR ref:notion.so) tag:bug'
htd item list --query '-status:done title:"refactor"'
htd item list --kind next_action --query 'tag:bug'
```

### 7.3 `htd item update`

```
htd item update ID FIELD=VALUE [FIELD=VALUE]...
```

Pairs applied in order, one file write.

| Field | Format | Notes |
|-------|--------|-------|
| `title` | string | |
| `body` | string | Markdown body (content after frontmatter, not a frontmatter field) |
| `kind` | enum | `inbox`/`next_action`/`project`/`waiting_for`/`someday`/`tickler` |
| `status` | enum | `active`/`done`/`canceled`/`discarded`/`archived` |
| `project` | string | Project-kind item ID |
| `source` | string | |
| `tags` | list | Comma-separated, optionally bracketed: `foo,bar` or `[foo,bar]`. Empty (`tags=`) clears. |
| `refs` | list | Same syntax as `tags` |
| `due_at` | date/dt | `YYYY-MM-DD` or RFC 3339. Empty clears. |
| `defer_until` | date/dt | Same |
| `review_at` | date | Same |

Protected (cannot change): `id`, `created_at`. Unknown fields are rejected.

**Cross-references** — prefer workflow commands when possible:
- `organize schedule` — dates
- `organize link`/`unlink` — project
- `organize move` — kind
- `engage done`/`cancel`, `item archive`/`restore` — status

### 7.4 `htd item archive ID`

Last-resort terminal transition when neither `done` nor `canceled` semantically applies. Sets `status: archived`, moves to archive. Active items only.

### 7.5 `htd item restore ID`

Bring a terminal item back to active. Locate in `archive/items/`, set `status: active`, move to `items/<kind>/<id>.md` based on recorded `kind`. Restoring an active item fails. Restoring a `discarded` inbox item lands it back in `items/inbox/`.

---

## 8. Reference

Tool-scoped reference notes under `reference/<tool>/` (see `docs/datamodel.md §3`). `--tool` defaults to `claude`. Each tool directory carries an auto-generated `INDEX.md` (see §8.5).

### 8.1 `htd reference add`

```
htd reference add --title TEXT [--body TEXT] [--tag TAG]... [--tool TOOL]
```

Tag conventions for INDEX grouping: `type:user`, `type:feedback`, `type:area_of_focus`, `type:project`, `type:reference`. Other tags go to `## other`.

**Behavior:** generate ID (globally unique across items + references in all tools), set timestamps, create `reference/<tool>/` lazily, write `<id>.md`, rewrite `INDEX.md`, print ID.

### 8.2 `htd reference get ID`

Search every `reference/<tool>/` then `archive/reference/<tool>/`. Active hits win. Display front matter + body. Exit `2` if missing.

Archived hits: text mode prepends `(archived)` above the metadata block; JSON sets `archived: true` (omitted for active hits). JSON also includes a `tool` field.

### 8.3 `htd reference list`

```
htd reference list [--tool TOOL] [--tag TAG] [--archived]
```

Read `reference/<tool>/` (or `archive/reference/<tool>/` with `--archived`); the two views are mutually exclusive. Exclude `INDEX.md`. Columns: `ID`, `TOOL`, `UPDATED_AT`, `TITLE`. Archived rows have `(archived)` prefix on `TITLE`. Sort by `updated_at` desc, `id` asc tiebreaker.

JSON: array; `archived: true` on every row when `--archived`.

### 8.4 `htd reference update`

```
htd reference update ID FIELD=VALUE [FIELD=VALUE]...
```

Fields: `title`, `body`, `tags` (same syntax as item `tags`). Protected: `id`, `created_at`, `tool` (to move tools, archive and re-add).

Locates across tools, active first then archive. Rewrites the active `INDEX.md` for active references (regroups when `type:*` changes). Archived references do not appear in INDEX, so it is not rewritten.

### 8.5 `htd reference archive ID`

Move `reference/<tool>/<id>.md` → `archive/reference/<tool>/<id>.md`. Update `updated_at`. Rewrite `INDEX.md` to drop the entry. Already-archived rejected (one-way). When the last active reference for a tool is archived, INDEX.md falls back to the empty-state stub (kept rather than deleted for diff cleanliness).

### 8.6 `htd reference restore ID`

Inverse of `archive`. Active references rejected. Move back, rewrite INDEX.

### 8.7 INDEX.md format

`reference/<tool>/INDEX.md` is generated by `reference` commands. Treated as a scoped exception to "no generated index files". Deterministic — same input → byte-for-byte identical file.

- H1: `# Reference index`.
- One section per non-empty `type:*` group, fixed order: `## user`, `## feedback`, `## area_of_focus`, `## project`, `## reference`. Missing/unknown type tags → trailing `## other`.
- Within each section: sort `updated_at` desc, `id` asc tiebreaker.
- Each entry: `- [title](id.md) — short description`. Description = first non-blank body line with leading `#` stripped, truncated to 80 runes. Omit em-dash + description when no usable line exists.
- Empty file: body is `_No entries._` stub (file kept rather than deleted).

INDEX.md is active-only; use `htd reference list --archived` for archive.

### 8.8 `htd reference reindex [--tool TOOL]`

Repair verb. The index is normally kept in sync by every mutation — reach for `reindex` after manual edits or merge conflicts. Scans `reference/<tool>/`, rewrites `INDEX.md` atomically per §8.7. Idempotent.

There is no `reference index` noun-verb — `reference list` already covers "show me what's there."

---

## 9. Journal

Time-stamped journals/retros/observations under `journal/` at the htd root. Flat, tool-agnostic, write-once — no `update`/`archive`/`delete` verbs.

### 9.1 `htd journal new`

```
htd journal new [--type daily|weekly|adhoc] [--date YYYY-MM-DD] [--title TEXT] [--tag TAG]...
```

| Option | Req | Description |
|--------|-----|-------------|
| `--type` | no | `daily` (default), `weekly`, or `adhoc` |
| `--date` | no | Date; default today. `weekly` snaps to Monday of that ISO week. |
| `--title` | adhoc only | Title; slug becomes filename |
| `--tag` | no | Tag (repeatable); stored in frontmatter |

Filenames:
- `daily` → `YYYY-MM-DD.md`
- `weekly` → `YYYY-MM-DD-weekly.md` (date = Monday of ISO week)
- `adhoc` → `<slug>.md` (snake_case from `--title`)

**Behavior:** resolve date, derive filename, refuse to overwrite, write a Markdown scaffold with optional YAML frontmatter (`created_at`, `updated_at`, `tags`), print the name (filename without `.md`).

Templates:
- Daily: `# YYYY-MM-DD` + `## What I did`, `## What I learned`, `## Tomorrow`.
- Weekly: `# Week of YYYY-MM-DD` + `## Wins`, `## Misses`, `## Lessons`, `## Focus next week`.
- Ad-hoc: H1 from `--title`, no further scaffold.

### 9.2 `htd journal list [--since YYYY-MM-DD]`

Read every `*.md` under `journal/` (no subdirs). `--since` filters by filename-derived date for dated entries, `created_at` fallback for ad-hoc. Columns: `NAME`, `CREATED_AT`, `TAGS`. Sort by filename descending. JSON: array of `name`/`created_at`/`updated_at`/`tags` (no body).

### 9.3 `htd journal show NAME`

Read `journal/<NAME>.md`. Display frontmatter (if present) then body. Exit `2` if missing. Hand-edited entries without frontmatter are accepted. JSON: single object with `name`, optional `created_at`/`updated_at`/`tags`, and `body`.

---

## 10. Init

### 10.1 `htd init`

Create the directory layout (see `docs/datamodel.md §7`) under `--path` and print the set, one path per line, stable order. Idempotent. Other commands also create missing directories as a side effect; `init` makes setup explicit. JSON: array of directory paths.

Example output:
```
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

### 11.1 `htd completion SHELL`

Emit completion script for `bash`, `zsh`, `fish`, or `powershell` to stdout. Covers all groups, subcommands, flags. Does not touch `--path`; safe anywhere.

```
source <(htd completion bash)
htd completion zsh > "${fpath[1]}/_htd"
```

---

## 12. Command Summary

| Command | Description |
|---------|-------------|
| `htd init` | Create the htd directory layout |
| `htd capture add` | Add to inbox (`--kind` skips inbox; `--kind project --child ...` seeds children; `--done` archives immediately) |
| `htd clarify list` / `show ID` / `update ID` / `discard ID` | Process inbox |
| `htd organize move KIND ID...` | Change kind |
| `htd organize link ID PROJECT_ID` / `unlink ID` | Manage project link |
| `htd organize schedule ID` | Set dates |
| `htd organize promote ID --child TITLE...` | Promote to project with children |
| `htd reflect next-actions` / `projects [--stalled]` / `waiting` / `review` | List views |
| `htd reflect log [--since DATE]` | Activity log of resolved items (default: last 30 days) |
| `htd reflect tickler [--pull \| --all \| --pending]` | Fired (default), pending, or both; `--pull` empties fired into the inbox |
| `htd reflect project ID` | Project with active + archived children |
| `htd engage done ID...` / `cancel ID...` | Terminal transitions |
| `htd engage next-actions` / `waiting` | Pick work now |
| `htd item get ID` / `list` / `update ID` / `archive ID` / `restore ID` | Low-level CRUD |
| `htd reference add` / `get` / `list` / `update` / `archive` / `restore` / `reindex` | Tool-scoped notes |
| `htd journal new` / `list` / `show NAME` | Journal entries |
| `htd completion SHELL` | Shell completion |
