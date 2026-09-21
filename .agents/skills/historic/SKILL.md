---
name: historic
description: "Use when operating the installed Historic CLI as a user or AI assistant: creating topics and Work Orders, reading and searching Markdown history, using the interactive TUI, managing lifecycle, snapshots, and metadata synchronization."
---

# Historic — User Operations Skill

Use this skill with the installed or compiled `historic` command. The user only needs the binary and this skill; the source repository and Go toolchain are not required.

## What Historic is

Historic is a local-first work-memory CLI. Markdown is the source of truth. SQLite/FTS5 is a rebuildable search cache, and the internal Git repository is local-only.

## Workspace rules

- The canonical workspace directory is `.historic/`.
- `.histories/` is unsupported and has no compatibility layer. Migrate manually with `mv .histories .historic`.
- If `.historic/` and `.histories/` both exist, reject the operation. Never merge or rename automatically.
- The expected structure includes `.historic/.database/`, `.historic/.database/.git/`, and `.historic/.index.sqlite`.
- Never push the internal Git repository and never add a remote to it.
- Do not modify `PRD.md` unless explicitly requested.
- Do not modify source-code files as part of routine lifecycle operations.

## CLI conventions

- Use five-digit topic IDs and `<id>-<slug>` topic folders.
- User-facing paths should be relative to the project root and use `.historic/`.
- Markdown is the source of truth. Run `historic rebuild --json` after recovery or manual index-affecting filesystem changes.
- JSON-capable commands preserve `{ "command", "ok", "data", "error" }`.
- JSON errors must not be duplicated on stderr.

## Getting started

```sh
historic init
historic create "My Topic"
historic list
historic show 00001
```

Use `historic add` to create notes or Work Orders. When multiple topics are active, provide an explicit `--id` for commands that need to select a topic.

## Sync metadata

```sh
historic sync-meta [<id>|<topic-path>] [--json]
```

- With an ID or path, sync only that active topic.
- Without a target, sync every active topic directly under `.historic/`; archived topics under `.historic/.database/` are excluded.
- Batch processing is deterministic and uses continue-on-error. Remaining topics are processed, errors are collected, and the command exits non-zero if any topic fails.
- `Files` and `Assets` are generated sections derived from the current filesystem.
- Every successful sync fully rebuilds both sections; it does not append to or preserve stale links.
- Managed Markdown with valid Historic frontmatter goes to `## Files`.
- Plain Markdown, images, PDFs, office files, archives, binaries, and other non-managed files go to `## Assets`.
- `_meta.md` is excluded from both sections.
- Rename, move, delete, and classification changes are reflected automatically on the next sync.
- Links are relative POSIX paths and sorted deterministically.
- Sections other than `Files` and `Assets` are preserved.
- Writes are atomic and the index is rebuilt after successful sync.
- Reject absolute/traversal paths, missing or ambiguous topics, symlink topics/files, and invalid `_meta.md`.
- A second sync without filesystem changes returns `updated: false`.

## Search and TUI

- `historic find "query" [--json]` is the one-shot interface for AI, scripts, and automation.
- Search includes both open and closed storage by default.
- Use `--open` for `.historic/<id>-<slug>/` only and `--closed` for `.historic/.database/<id>-<slug>/` only. Combining them is rejected.
- Legacy `--active` and `--archived` flags are not supported.
- Every result exposes independent `status`, `storage`, and actual `path` fields. Human output uses `[OPEN]` and `[CLOSED]` markers.
- `historic search` is the read-only interactive terminal UI for human exploration.
- The TUI uses the shared search service and existing FTS5 index; it has no separate index and must not access SQLite directly from the presentation layer.
- The TUI uses Bubble Tea, Bubbles, and Lip Gloss.
- `historic search --json` is rejected; use `historic find --json`.
- Query execution occurs on Enter. Empty query shows recent topics; `r` refreshes.
- MVP supports status, scope (`open`, `closed`, `all`), and type (`all`, `topic`, `work-order`) filters, pagination, keyboard navigation, and read-only previews.
- Layout must respect terminal viewport height, keep the active result visible, and never render results beneath the footer. Long titles/paths and small terminals require safe truncation or compact layout.

## File and topic lifecycle

Update one managed Markdown file:

```sh
historic status <path> progress
historic status <path> complete
```

Update a whole topic's work status without moving its storage location:

```sh
historic progress 00001
historic pending 00001
historic review 00001
historic blocked 00001
historic complete 00001
historic failed 00001
historic cancelled 00001
```

Move a topic between storage states explicitly:

```sh
historic close 00001
historic open 00001
```

- `open` means `.historic/<id>-<slug>/`.
- `closed` means `.historic/.database/<id>-<slug>/`.
- Work status and storage state are independent: `complete + open` and `complete + closed` are both valid.
- Close/open preserve frontmatter status and all topic bytes. They reject missing topics, duplicate IDs, destination conflicts, and symlink topic roots.
- `historic import 00001` is a compatibility alias for opening a closed topic and preserves its work status.

## Schema compatibility and recovery

```sh
historic doctor [--json]
historic upgrade [--json]
historic rebuild [--json]
```

- `doctor` is read-only. It reports binary version, executable path, workspace format version, current and required index schema versions, Markdown validity, compatibility status, and an actionable recommendation.
- Status values include `compatible`, `upgrade required`, `rebuild required`, `binary too old`, and `workspace invalid`.
- `historic rebuild --json` is the unified recovery workflow: it processes open and closed topics, reconciles `## Files` and `## Assets`, preserves other metadata sections and work status, updates storage-aware SQLite/FTS5 records, and atomically replaces the validated index.
- If the index is missing or damaged, run `historic rebuild` to recreate it from Markdown.
- If the index schema is older, run `historic upgrade`. Upgrade validates the workspace, builds a temporary schema, scans Markdown as source of truth, validates the new index, backs up the old index, and atomically replaces it.
- Upgrade and rebuild use temporary files, cleanup, rollback safeguards, and a lock to prevent concurrent index replacement. Markdown is never modified.
- A failed scan or invalid workspace leaves the existing index protected. Do not add SQLite columns manually or delete the index manually.


## JSON output

JSON-capable commands use:

```json
{"command":"...","ok":true,"data":{},"error":null}
```

Successful empty results use `ok: true`. JSON errors use the envelope and a non-zero exit status.

## Recovery and safety

- Keep Markdown as the authoritative record.
- Treat SQLite and internal Git as secondary/rebuildable mechanisms.
- Run `historic rebuild` after manual recovery or index-affecting filesystem changes.
- Do not modify files outside `.historic/` for routine history operations.
- Do not modify `PRD.md` unless the user explicitly asks.
