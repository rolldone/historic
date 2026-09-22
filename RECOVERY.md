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
historic rebuild --json
historic doctor --json
```

- `rebuild` scans both open and closed topics recursively and rebuilds SQLite/FTS5 from Markdown.
- If a valid topic folder is missing `_meta.yaml`, rebuild creates minimal canonical metadata using the folder ID and slug title, then regenerates `files` and `assets`.
- If the scan or index replacement fails, recovered metadata is rolled back and the previous index remains protected.
- Invalid existing `_meta.yaml` is not silently overwritten; fix it or resolve the reported conflict before rebuilding.
- A second rebuild without filesystem changes is idempotent and reports `updated: 0`.


## Safety policy

Symlinks in archive trees are rejected for import/archive operations. No operation performs network access or remote Git push.
