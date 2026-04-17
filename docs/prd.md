# Headless To-Do (htd) — Product Requirements Document

## 1. Overview

htd is a headless task management system that enables both AI agents and CLI users to operate a structured workflow. The system uses the local file system for storage, stores data in Markdown with YAML front matter, and assumes Git-based version control.

### 1.1 Goals

- Provide a CLI tool that maps directly to a five-phase workflow: **Capture → Clarify → Organize → Reflect → Engage**
- Enable AI agents to read and write task data directly via the file system
- Keep all data human-readable, diffable, and Git-friendly
- Remain headless — no UI dependency

### 1.2 Target Users

| User Type | Usage Pattern |
|-----------|---------------|
| Human (CLI) | Invokes `htd` commands from the terminal |
| AI Agent | Reads/writes Markdown files directly, or invokes `htd` commands |

### 1.3 Implementation Language

Go — single binary distribution with no runtime dependencies.

---

## 2. Core Concepts

### 2.1 Five-Phase Workflow

The system structures all task management around five sequential phases:

| Phase | Purpose | CLI Command Group |
|-------|---------|-------------------|
| **Capture** | Collect inputs into a single inbox | `htd capture` |
| **Clarify** | Process inbox items — define, update, or discard | `htd clarify` |
| **Organize** | Categorize, link to projects, and schedule | `htd organize` |
| **Reflect** | Review lists, detect stalled projects, check progress | `htd reflect` |
| **Engage** | Choose and complete work | `htd engage` |

### 2.2 Data Types

The system uses exactly two data types:

- **Item** — An actionable or incomplete piece of work (tasks, projects, waiting items, etc.)
- **Reference** — Non-actionable information stored for future retrieval

These are fully separate; a Reference is never promoted to an Item or vice versa.

### 2.3 Item Categories (Kind)

Items are classified into one of six categories, each represented as a directory:

| Kind | Description |
|------|-------------|
| `inbox` | Unclarified input — the entry point for all new items |
| `next_action` | A concrete, actionable task ready to be worked on |
| `project` | A multi-step outcome that requires more than one action |
| `waiting_for` | An action delegated to someone else |
| `someday` | Items deferred for future consideration |
| `tickler` | Time-triggered reminders |

### 2.4 Item Lifecycle (Status)

| Status | Description | Location |
|--------|-------------|----------|
| `active` | Currently live | `items/<kind>/` |
| `done` | Completed successfully | `archive/items/` |
| `canceled` | Intentionally abandoned | `archive/items/` |
| `discarded` | Removed during clarification (not actionable, not worth keeping) | `archive/items/` |
| `archived` | Manually archived for reference | `archive/items/` |

Non-active items (`done`, `canceled`, `discarded`, `archived`) are moved to `archive/items/` (flat structure).

---

## 3. Design Principles

### 3.1 Headless First

The system has no GUI or TUI dependency. All interactions occur through CLI commands or direct file manipulation. This enables integration with any frontend, editor plugin, or AI agent.

### 3.2 File-Based Storage

All data is stored as Markdown files with YAML front matter on the local file system. There is no database. This makes the system trivially portable, inspectable, and recoverable.

### 3.3 Git Friendly

The file format is designed for version control:

- Plain text files that diff cleanly
- Deterministic file paths derived from IDs
- No binary blobs or opaque state files
- History, branching, and merging are supported natively by Git

### 3.4 Agent Native

The file format and directory layout are optimized for AI agent access:

- Structured YAML front matter for programmatic field access
- Markdown body for natural language content
- Predictable file paths (no need for an index or database query)
- `--json` output flag for machine-readable CLI output

---

## 4. State Transition Rules

1. **Inbox items must go through Clarify** — An item in `inbox` cannot be moved directly to `done`. It must first be processed via `clarify` commands (updated, moved to another kind, or discarded).
2. **`discarded` is inbox-only** — `clarify discard` applies exclusively to inbox items that were never actionable (e.g., noise, junk). Once an item has been moved out of the inbox into any list, it must be ended via `engage done` or `engage cancel` instead.
3. **`done` or `canceled` for list items** — Items that have left the inbox have a clear lifecycle: either they are completed (`done`) or abandoned (`canceled`). There is no ambiguity about which terminal state to use.
4. **`archived` is a last resort** — `item archive` exists for edge cases where neither `done` nor `canceled` semantically applies (e.g., a project superseded by another). Normal workflow should always end with `done` or `canceled`.
5. **A project must have at least one next action** — Stalled projects (those with no linked `next_action` items) are flagged during `reflect projects --stalled`.
6. **Terminal items are nearly immutable** — Items with status `done`, `canceled`, `discarded`, or `archived` should not be modified except for correcting errors via `htd item update`.

---

## 5. Non-Goals

The following are explicitly out of scope:

- **Real-time synchronization** — The system is single-user, file-based. Sync is delegated to Git.
- **Multi-user conflict resolution** — No locking, merging of concurrent edits, or access control.
- **Advanced search indexing** — Search is file-system-based (e.g., `grep`). No full-text index.
- **GUI / TUI** — The system is headless by design. UI layers are external consumers.

---

## 6. Low-Level Item Access

In addition to the workflow-oriented command groups, a low-level `htd item` command group provides direct CRUD access to items. This is primarily intended for scripting, automation, and agent use where the workflow abstraction is unnecessary.

---

## 7. Future Considerations

The following features are planned but not included in the initial release:

- **Agent-specific commands** — Commands like `capture fetch` (automated inbox population), `clarify suggest` (AI-powered triage), and `clarify apply` (apply suggestions after user approval) may be added in future versions.
- **Reference management commands** — CRUD operations for reference data.
- **Tagging and filtering** — Advanced filtering by tags across commands.
