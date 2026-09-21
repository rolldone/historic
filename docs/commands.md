# Command Reference

## Workspace and topics

```sh
historic init
historic create "Topic Title" [--id 00014]
historic add <name> [--id 00014]
historic list [--closed] [--json]
historic show <id> [--closed] [--json]
```

Topic IDs use five digits and topic folders use `<id>-<slug>`.

## Work status and storage state

Work status is independent from storage location:

- `open`: `.historic/<id>-<slug>/`
- `closed`: `.historic/.database/<id>-<slug>/`

```sh
historic progress <id> [--json]
historic pending <id> [--json]
historic review <id> [--json]
historic blocked <id> [--json]
historic complete <id> [--json]
historic failed <id> [--json]
historic cancelled <id> [--json]
historic close <id> [--json]
historic open <id> [--json]
historic import <id> [--json]
```

Lifecycle commands update work status only. `close` and `open` are the explicit storage moves; they preserve frontmatter status and topic contents. `import` is a compatibility alias for `open`. Duplicate IDs, destination conflicts, missing topics, and symlink topic roots are rejected.

## File status

```sh
historic status <path> <status> [--json]
```

This command is only for managed Markdown with valid Historic frontmatter whose ID matches the topic. It updates that file's `status` and `updated`, then rebuilds the index. It rejects regular assets and Markdown without valid managed frontmatter.

## Synchronize metadata

```sh
# One active topic
historic sync-meta 00014
historic sync-meta .historic/00014-topic

# Every active topic in the workdir
historic sync-meta
historic sync-meta --json
```

`sync-meta` fully reconciles the generated `## Files` and `## Assets` sections from the current topic filesystem:

- valid managed Markdown goes to `Files`;
- plain Markdown and other files go to `Assets`;
- `_meta.md` is excluded;
- rename, move, delete, and classification changes are reflected automatically;
- stale and duplicate links are removed;
- links are relative POSIX paths sorted deterministically;
- all other `_meta.md` sections are preserved.

Targeted mode processes one active topic. No-target mode processes all active topics directly under `.historic/`, excluding `.historic/.database/`. Batch mode continues after per-topic errors, reports them, and exits non-zero if any topic fails. A second run without filesystem changes reports `updated: false`.

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

`doctor` performs a read-only compatibility diagnosis. It reports binary version, executable path, workspace format version, current and required index schema versions, Markdown validity, status, and recovery recommendation.

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

Markdown remains authoritative; rebuildable indexes must never be treated as the source of truth.
