# Recovery Runbook

## Principles

Markdown files remain the source of truth. Archive and import use staging directories and never delete the source before a destination is ready.

## Detecting partial operations

1. Run `historic rebuild --json`.
2. Inspect `.historic/` and `.historic/.database/` for `.staging-*` directories.
3. A staging directory may be removed only after confirming its source and destination are intact.
4. If a topic exists in both active and archive roots, do not overwrite either copy. Resolve the conflict manually, then rebuild.
5. If search reports that the FTS5 index is unavailable, run `historic rebuild`. The rebuild scans Markdown first and replaces `index_records` and `historic_fts` transactionally.
6. If rebuild reports invalid Markdown, the previous SQLite/FTS index remains available. Fix the reported file, then retry rebuild.

## Recovery commands

```sh
historic rebuild
historic list --archived
historic show 00014 --archived
```

- If a rebuild reports invalid Markdown, do not delete `.historic/.index.sqlite`; the previous index is preserved by the transaction. Correct the source file and retry.
- Search uses SQLite FTS5 only. If `historic find` reports `FTS5 index unavailable`, run `historic rebuild`; there is no silent filesystem fallback.

## Safety policy

Symlinks in archive trees are rejected for import/archive operations. No operation performs network access or remote Git push.
