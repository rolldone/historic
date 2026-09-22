# Command Reference

## Workspace and topics

```sh
historic init
historic create "Topic Title"
historic add <name> [--id <topic-id>]
historic list [--closed] [--json]
historic show <topic-id> [--closed] [--json]
historic version [--json]
```


`historic version` prints `VersionName`, `VersionCode`, `WorkspaceFormatVersion`, and `IndexSchemaVersion`. JSON output includes all four values as a stable envelope.

`historic add` canonicalizes the basename before writing it: whitespace, `_`, and repeated `-` separators become a single lowercase `-`, and `.md` is added exactly once. Directory prefixes are preserved. A `wos/` entry receives the next available `NN-` Work Order number; an existing `NN-` prefix is preserved. Absolute paths, traversal, hidden paths, and names without usable characters are rejected. User-facing JSON paths always use `/` separators.

Topic IDs for new topics are 13-digit positive Unix epoch milliseconds. Topic folders use `<topic-id>-<slug>`, for example `1758537600123-search-read-model`. Allocation is monotonic and serialized across processes. Legacy five-digit IDs remain readable during migration and may be resolved through the alias map. Managed file IDs are independent lowercase canonical UUIDv7 values.

Managed file IDs use lowercase canonical UUIDv7 values. Legacy manifests can be migrated with:

```sh
historic migrate-topic-ids --dry-run
historic migrate-topic-ids --json --dry-run
historic migrate-topic-ids --id 00001
historic migrate-topic-ids

historic migrate-file-ids --dry-run
historic migrate-file-ids --json --dry-run
historic migrate-file-ids [--id <topic-id>] [--open|--closed]
```

`migrate-topic-ids` is the focused/admin migration for legacy five-digit topic folders. `migrate-file-ids` assigns UUIDv7 values to legacy manifest entries. Both preserve existing modern identities and support dry-run/idempotent behavior. Normal users should prefer `historic upgrade`.

## Legacy workspace upgrade

```sh
historic upgrade --dry-run
historic upgrade --json --dry-run
historic upgrade
historic upgrade --json
historic upgrade --backup-dir /path/outside/.historic
historic upgrade --no-file-ids --no-topic-ids
```

`upgrade` is the official orchestrator for legacy workspaces. It acquires an upgrade lock, scans and validates topics/manifests, creates an immutable backup outside `.historic`, migrates legacy FileID and TopicID data, writes topic aliases, rebuilds and validates SQLite/FTS, and replaces the index atomically. A successful rerun reports `migration_required: false` and `updated: 0`.

`--no-file-ids` and `--no-topic-ids` are recovery/admin flags only. The command reports warnings and deliberately leaves the selected legacy identity data unchanged. `--dry-run` does not write metadata, folders, aliases, backups, or indexes. `--backup-dir` must be outside `.historic`.

The JSON result uses the standard envelope and includes `migration_required`, `topics_scanned`, `file_ids_migrated`, `topic_ids_migrated`, `index_rebuilt`, `backup_path`, `aliases_written`, `updated`, and `warnings`.


Work status belongs only to managed member Markdown files. Topics have no work-status field or topic lifecycle status command.

- `open`: `.historic/<topic-id>-<slug>/`
- `closed`: `.historic/.database/<topic-id>-<slug>/`

```sh
historic status <member-path> <status> [--json]
historic close <id> [--json]
historic open <id> [--json]
historic import <id> [--json]
```

`close`, `open`, and `import` change storage only and preserve member frontmatter statuses and topic contents.

## File status

```sh
historic status <path> <status> [--json]
```

This command is for any managed Markdown file with valid Historic frontmatter. It updates that member file's `status` and `updated`, then rebuilds the index. The member frontmatter ID may differ from the parent topic ID. It rejects regular assets, Markdown without valid Historic frontmatter, unsafe paths, and symlink targets. Valid statuses include `create`, `draft`, `pending`, `progress`, `review`, `blocked`, `complete`, `failed`, and `cancelled`.

## One-shot search

```sh
historic find "keyword"
historic find "keyword" --open --json
historic find "keyword" --status progress
historic find "keyword" --folder .historic/<topic-id>-<slug>
historic find "keyword" --closed
historic find "keyword" --type task --id <topic-id> --json
```

`find` searches open and closed storage by default and uses the shared SQLite FTS5 index. Use `--open` for open topics only or `--closed` for closed topics only; combining them is rejected. Results expose work `status`, `storage`, and actual `path`. Legacy `--active` and `--archived` flags are not supported. If the index is missing or invalid, run `historic rebuild`.

## Interactive search TUI

```sh
historic search
```

`historic search` is a read-only Bubble Tea interface over the same search service as `find`.

- Empty query shows recent active topics.
- Query runs when Enter is pressed.
- `/` focuses query; `f` focuses filters.
- `↑`/`↓` or `j`/`k` navigates results.
- `Enter` updates the preview.
- `n`/`p` changes page; `r` refreshes.
- `Esc`, `q`, or `Ctrl+C` exits.
- Filters include status, scope, and type.
- `historic search --json` is rejected. Use `historic find --json` for automation.
- Preview is read-only; no lifecycle mutation is performed.

The TUI must respect terminal viewport height. Results must not render beneath the footer, and the active result must remain visible.

## Schema compatibility and recovery

```sh
historic doctor [--json]
historic upgrade [--dry-run] [--json] [--backup-dir <path>] [--no-file-ids] [--no-topic-ids]
historic rebuild [--json]
```

`doctor` performs a read-only compatibility diagnosis and recommends `historic upgrade` when legacy five-digit topic folders are detected. `upgrade` is the official legacy-workspace orchestrator; it creates an immutable backup outside `.historic`, migrates FileID/TopicID identities, writes aliases, rebuilds and validates SQLite/FTS, and is idempotent. Use `rebuild` when the index is missing or damaged. Both commands use temporary files, atomic replacement, rollback safeguards, and locks.

Possible diagnosis statuses are `compatible`, `upgrade required`, `rebuild required`, `binary too old`, and `workspace invalid`.

## Lifecycle and versioning

```sh
historic save -m "message" [--json]
historic log <id> [--json]
historic diff <id> [--json]
historic restore <id> --snapshot <commit> [--force] [--json]
```

Snapshots are local-only and include both open and closed topic storage under `.historic`. Closing and opening topics are filesystem operations with recovery safeguards; they do not automatically change work status.

## Rebuild and output

```sh
historic rebuild
historic rebuild --json
```

Every JSON-capable command uses the envelope:

```json
{"command":"...","ok":true,"data":{},"error":null}
```

`historic rebuild` is the unified workflow for all open and closed topics. It recursively scans every topic file, regenerates the `_meta.yaml` files/assets manifest, computes aggregate status only in SQLite, rebuilds storage-aware SQLite/FTS5, validates the temporary index, and atomically replaces the prior index. If `_meta.yaml` is missing, it is minimally recovered from the folder ID/slug; recovery is rolled back if rebuild fails.
