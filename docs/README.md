# Historic Documentation

Historic is a local-first work-memory CLI. Markdown files are the source of truth; SQLite/FTS5 and internal Git are secondary, rebuildable/local mechanisms.

## Start here

- [Command reference](commands.md)

## Quick start

```sh
historic init
historic create "My Topic"
historic list
historic find "keyword"
historic search
```

Use `historic find "keyword" --json` for AI agents, scripts, and automation. Use `historic find ""` or `historic search` for recent interactive browsing. Search supports comma-separated `--status` inclusion, `--status-not` exclusion, and deterministic `--sort relevance|updated|created|title`. File results include SQLite-backed freshness metadata (`created_at`, `updated_at`, `mtime`, `hash`, `size`).


## Workspace and identity

The canonical workspace is `.historic/`. The legacy `.histories/` directory is unsupported and must be migrated manually. New topics use 13-digit Unix epoch-millisecond TopicIDs and folders named `<topic-id>-<slug>`. Managed file IDs remain lowercase UUIDv7 values and are never interchangeable with TopicIDs. Legacy five-digit topic folders remain readable until an explicit upgrade.

## Upgrade and safety

```sh
historic doctor --json
historic upgrade --dry-run --json
historic upgrade --json
```

`doctor` is read-only and recommends `historic upgrade` for legacy topic folders. `upgrade` is the single supported legacy-workspace path: it backs up data outside `.historic`, migrates TopicID/FileID identities, writes aliases, rebuilds SQLite/FTS, and is designed to be idempotent. Use `--no-file-ids` or `--no-topic-ids` only for recovery/admin workflows; warnings are included in the result. `--dry-run` must not change workspace bytes.

Historic lifecycle operations are designed to preserve data, but always inspect `git status` before committing. Workspace migration may appear as many deleted and untracked paths because legacy five-digit folders are renamed to modern timestamp folders; review rename detection before staging. Never push the internal Historic Git repository or modify `PRD.md` unless explicitly requested.
