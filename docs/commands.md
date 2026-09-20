# Command Reference

## Workspace and topics

```sh
historic init
historic create "Topic Title" [--id 00014]
historic add <name> [--id 00014]
historic list [--archived] [--json]
historic show <id> [--archived] [--json]
```

Topic IDs use five digits and topic folders use `<id>-<slug>`.

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
historic find "keyword" --active --json
historic find "keyword" --status progress
historic find "keyword" --folder .historic/00014-topic
historic find "keyword" --archived
historic find "keyword" --type task --id 00014 --json
```

`find` uses the shared SQLite FTS5 index and supports filters for status, folder, active/archived scope, type, and topic ID. If the index is missing or invalid, run `historic rebuild`.

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

## Lifecycle and versioning

```sh
historic progress <id> [--json]
historic pending <id> [--json]
historic review <id> [--json]
historic blocked <id> [--json]
historic complete <id> [--json]
historic failed <id> [--json]
historic cancelled <id> [--json]
historic import <id> [--json]
historic save -m "message" [--json]
historic log <id> [--json]
historic diff <id> [--json]
historic restore <id> --snapshot <commit> [--force] [--json]
```

Closing lifecycle operations may archive topics under `.historic/.database/`. Import and restore have conflict and recovery safeguards.

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
