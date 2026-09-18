# Recovery Runbook

## Principles

Markdown files remain the source of truth. Archive and import use staging directories and never delete the source before a destination is ready.

## Detecting partial operations

1. Run `historic rebuild --json`.
2. Inspect `.histories/` and `.histories/.database/` for `.staging-*` directories.
3. A staging directory may be removed only after confirming its source and destination are intact.
4. If a topic exists in both active and archive roots, do not overwrite either copy. Resolve the conflict manually, then rebuild.

## Recovery commands

```sh
historic rebuild
historic list --archived
historic show 00014 --archived
```

If an import fails, the archive remains intact and a partial active destination is removed. If an archive move fails, the source remains active unless the final move has completed. Rebuild the index after manual recovery.

## Safety policy

Symlinks in archive trees are rejected for import/archive operations. No operation performs network access or remote Git push.
