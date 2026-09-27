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
- Do not push the internal Git repository or add a remote unless the user explicitly requests that action.
- Do not modify `PRD.md` unless explicitly requested.
- Do not modify source-code files as part of routine lifecycle operations.

## CLI conventions

- New topic IDs are 13-digit Unix epoch-millisecond values, rendered as `<unix-millisecond>-<slug>`.
- Legacy five-digit topic IDs remain readable during compatibility and migration.
- Managed file IDs remain lowercase canonical UUIDv7 values; do not mix TopicID and FileID.
- User-facing paths should be relative to the project root and use `.historic/`.
- Markdown is the source of truth. Run `historic rebuild --json` after recovery or manual index-affecting filesystem changes.
- JSON-capable commands preserve `{ "command", "ok", "data", "error" }`.
- JSON errors must not be duplicated on stderr.

## Getting started

```sh
historic init
historic create "My Topic"
historic list
historic show <timestamp-topic-id>
```

`historic create` allocates a monotonic Unix epoch-millisecond TopicID under an inter-process lock. During migration, legacy five-digit IDs can still be resolved. Use `historic add` to create notes or Work Orders. When multiple topics are active, provide an explicit `--id` for commands that need to select a topic.

## Internal Git passthrough

Use `historic git <git-args...>` to run Git with `.historic/.database` as the working directory:

```sh
historic git status
historic git log -1 --oneline
historic git diff --stat
```

- Arguments are passed directly to the Git executable without shell interpretation; stdin, stdout, stderr, and Git's exit status are forwarded.
- The proxy does not run Git actions automatically. It forwards only the subcommand the user explicitly supplies, including `push`, `pull`, `commit`, and `init`; never append or infer such actions.
- Do not add a remote, change Git configuration, or push unless explicitly requested by the user.
- With no Git arguments, the command prints usage. This proxy is distinct from Historic's `historic log`, `historic diff`, `historic save`, and `historic restore` commands.

## Search and TUI

- `historic find "query" [--json]` is the one-shot interface for AI, scripts, and automation; `historic find ""` browses recent topics. `--page` defaults to 1 and `--page-size` defaults to 20 with a maximum of 100; JSON uses `data.items` and `data.pagination` (`page`, `page_size`, `has_more`, nullable `next_page`).
- `--status` accepts comma-separated inclusion statuses and `--status-not` accepts comma-separated exclusions. Overlapping statuses and unknown statuses are rejected.
- `--sort` accepts `relevance`, `updated`, `created`, or `title`; results use deterministic path tie-breakers.
- File results expose SQLite-backed `created_at`, `updated_at`, `mtime`, `hash`, and `size` fields.
- Search includes both open and closed storage by default.
- Use `--open` for `.historic/<id>-<slug>/` only and `--closed` for `.historic/.database/<id>-<slug>/` only. Combining them is rejected.
- Legacy `--active` and `--archived` flags are not supported.
- Every result exposes independent `status`, `storage`, and actual `path` fields. Human output uses `[OPEN]` and `[CLOSED]` markers.
- `historic search` is the read-only interactive terminal UI for human exploration.
- The TUI uses the shared search service and existing FTS5 index; it has no separate index and must not access SQLite directly from the presentation layer.
- The TUI uses Bubble Tea, Bubbles, and Lip Gloss.
- `historic search --json` is rejected; use `historic find --json`.
- Query execution occurs on Enter. Empty query shows recent topics; `r` refreshes the active page. `n`/right arrow advances when `has_more`; `p`/left arrow goes back. Pagination uses the SQLite read model and does not hash/stat the filesystem. Offset drift between requests is eventual consistency in the MVP.
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
historic close all [--json]
historic open 00001
historic import 00001
```

- `historic list` shows open topics by default; `historic list --closed` filters to closed topics only. The `--closed` flag does not mean “include closed.”
- `historic close <id>` closes one topic; `historic close all` closes the open topics discovered at the start of that command.
- Bulk close processes topics deterministically by TopicID then path and reuses the single-topic close lifecycle. Each topic reports an individual result; a failure does not stop later topics, but any partial failure returns a non-zero exit code.
- `historic close all --json` returns one standard JSON envelope. `data` contains `succeeded`, `failed`, `total`, `closed`, and `failed_count`; `ok` is false when one or more topics fail. An empty target list succeeds with zero counts.
- Only valid first-level open topic folders are targets. Closed topics, metadata/index files, staging entries, and nested folders are not included.
- `open` means `.historic/<topic-id>-<slug>/`.
- `closed` means `.historic/.database/<topic-id>-<slug>/`.
- Close/open preserve all topic bytes and member frontmatter statuses.
- `import` is a compatibility alias for opening a closed topic.

## Schema compatibility, migration, and recovery

```sh
historic doctor [--json]
historic upgrade [--dry-run] [--json] [--backup-dir <path>] [--no-file-ids] [--no-topic-ids]
historic migrate-topic-ids [--dry-run] [--json] [--id <legacy-id>]
historic migrate-file-ids [--dry-run] [--json] [--id <topic-id>] [--open|--closed]
historic rebuild [--json]
historic version [--json]
```

`historic upgrade` is the supported one-command legacy workspace orchestrator. It scans legacy five-digit topic folders and manifests, creates an immutable backup outside `.historic`, migrates FileID and TopicID data, writes aliases, rebuilds SQLite/FTS, and reports an idempotent result. Use `migrate-topic-ids` or `migrate-file-ids` for focused admin migrations.

- `doctor` is read-only and never renames legacy folders. It reports binary/version compatibility and recommends `historic upgrade` when legacy topic folders are detected.
- `--no-file-ids` and `--no-topic-ids` are admin/recovery flags; their warnings explain which migration was skipped.
- `--dry-run` must not write topic folders, metadata, aliases, backups, or indexes. `--backup-dir` must be outside `.historic`.
- Topic aliases are retained in `.historic/.topic-id-aliases.yaml` during compatibility migration.
- `historic rebuild --json` processes open and closed topics, recursively scans every topic file, regenerates `_meta.yaml.files` and `_meta.yaml.assets`, computes aggregate status in SQLite, and atomically replaces the validated index.
- If `_meta.yaml` is missing from a valid topic folder, rebuild creates minimal metadata atomically using the folder ID and folder slug title, then regenerates the manifest.
- Recovered metadata is rolled back if scan/index replacement fails; the previous index remains protected.
- If an existing `_meta.yaml` is invalid, rebuild does not silently overwrite it.
- If the index is missing or damaged, run `historic rebuild` to recreate it from Markdown.
- Upgrade and rebuild use temporary files, cleanup, rollback safeguards, and locks to prevent concurrent index replacement.


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
