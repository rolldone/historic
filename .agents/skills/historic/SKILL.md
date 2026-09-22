---
name: historic
description: "Use when operating the installed Historic CLI as a user or AI assistant: creating topics and Work Orders, reading and searching Markdown history, using the interactive TUI, managing lifecycle, snapshots, and metadata synchronization."
---

# Historic — User Operations Skill

Use this skill with the installed or compiled `historic` command. Markdown is the source of truth; SQLite/FTS5 is a rebuildable read model/cache and the internal Git repository is local-only.

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

Update one managed Markdown member file:

```sh
historic status <path> progress
historic status <path> draft
historic status <path> complete
```

- Work status belongs only to managed member frontmatter.
- Topics have no work-status field or topic status command.
- `computed_status` is an SQLite aggregate/cache value only.
- `close` and `open` change storage only and never change member status.

Move a topic between storage states:

```sh
historic close 00001
historic open 00001
historic import 00001
```

- `open` means `.historic/<id>-<slug>/`.
- `closed` means `.historic/.database/<id>-<slug>/`.
- Close/open preserve all topic bytes and member frontmatter statuses.
- `import` is a compatibility alias for opening a closed topic.

## Schema compatibility and recovery

```sh
historic doctor [--json]
historic upgrade [--json]
historic rebuild [--json]
historic migrate-file-ids [--dry-run] [--json] [--id <topic-id>] [--open|--closed]
```

`migrate-file-ids` assigns UUIDv7 IDs to legacy `_meta.yaml.files[]` entries without changing Markdown, topic IDs, paths, types, statuses, or assets. It supports dry-run, deterministic JSON mappings, atomic rollback, and idempotent reruns. `--force` is not supported.

- `doctor` is read-only. It reports binary version, executable path, workspace format version, current and required index schema versions, Markdown validity, compatibility status, and an actionable recommendation.
- Status values include `compatible`, `upgrade required`, `rebuild required`, `binary too old`, and `workspace invalid`.
- `historic rebuild --json` processes open and closed topics, recursively scans every topic file, regenerates `_meta.yaml.files` and `_meta.yaml.assets`, computes aggregate status in SQLite, and atomically replaces the validated index.
- If `_meta.yaml` is missing from a valid topic folder, rebuild creates minimal metadata atomically using the folder ID and folder slug title, then regenerates the manifest.
- Recovered metadata is rolled back if scan/index replacement fails; the previous index remains protected.
- If an existing `_meta.yaml` is invalid, rebuild does not silently overwrite it.
- If the index is missing or damaged, run `historic rebuild` to recreate it from Markdown.
- If the index schema is older, run `historic upgrade`. Upgrade validates the workspace, builds a temporary schema, scans Markdown as source of truth, validates the new index, backs up the old index, and atomically replaces it.
- Upgrade and rebuild use temporary files, cleanup, rollback safeguards, and a lock to prevent concurrent index replacement.


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
