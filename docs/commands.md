# Command Reference

## Workspace and topics

```sh
historic init
historic create "Topic Title" [--id 00014]
historic add <name> [--id 00014]
historic list [--closed] [--json]
historic show <id> [--closed] [--json]
historic version [--json]
```

`historic version` prints `VersionName`, `VersionCode`, `WorkspaceFormatVersion`, and `IndexSchemaVersion`. JSON output includes all four values as a stable envelope.

`historic add` canonicalizes the basename before writing it: whitespace, `_`, and repeated `-` separators become a single lowercase `-`, and `.md` is added exactly once. Directory prefixes are preserved. A `wos/` entry receives the next available `NN-` Work Order number; an existing `NN-` prefix is preserved. Absolute paths, traversal, hidden paths, and names without usable characters are rejected. User-facing JSON paths always use `/` separators.

Topic IDs use five digits and topic folders use `<id>-<slug>`.

Managed file IDs use lowercase canonical UUIDv7 values. Legacy manifests can be migrated with:

```sh
historic migrate-file-ids --dry-run
historic migrate-file-ids --json --dry-run
historic migrate-file-ids [--id 00014] [--open|--closed]
```

The migration changes only `_meta.yaml` `files[].id` values, preserves Markdown bytes and existing UUIDv7 IDs, and is atomic and idempotent. `--force` is rejected.

## Work status and storage state

Work status belongs only to managed member Markdown files. Topics have no work-status field or topic lifecycle status command.

- `open`: `.historic/<id>-<slug>/`
- `closed`: `.historic/.database/<id>-<slug>/`

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
historic find "keyword" --folder .historic/00014-topic
historic find "keyword" --closed
historic find "keyword" --type task --id 00014 --json
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
historic upgrade [--json]
historic rebuild [--json]
```

`doctor` performs a read-only compatibility diagnosis. It reports binary version, executable path, workspace format version, current and required index schema versions, stored application version/code, compatibility action, Markdown validity, status, and recovery recommendation.

Use `upgrade` for an older index schema and `rebuild` for a missing or damaged index. Both commands scan Markdown without modifying it, build and validate a temporary SQLite database, protect the old index with a backup and rollback path, replace atomically, clean temporary files, and use a lock against concurrent replacement. Legacy workspaces do not require manual SQL changes.

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
