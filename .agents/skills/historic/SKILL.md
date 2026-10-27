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
- Calendar dates (`created`, `updated`) are always UTC `YYYY-MM-DD`, regardless of host timezone. `updated` is never earlier than `created`, including for legacy documents.
- Markdown is the source of truth. Run `historic rebuild --json` after recovery or manual index-affecting filesystem changes.
- JSON-capable commands preserve `{ "command", "ok", "data", "error" }`.
- JSON errors must not be duplicated on stderr.
- The installed binary determines the SQLite index schema it can write. After pulling source changes that alter the schema, reinstall before validating behavior (`./build-global.sh`, then `historic version` and `historic rebuild --json`). A stale binary keeps the old `index_schema_version` and cannot create the newer columns, which `historic doctor` reports as `rebuild required`.

## Getting started

```sh
historic init
historic create "My Topic"
historic list
historic show <timestamp-topic-id>
```

`historic create` allocates a monotonic Unix epoch-millisecond TopicID under an inter-process lock. During migration, legacy five-digit IDs can still be resolved. Use `historic add` to create notes or Work Orders. When multiple topics are active, provide an explicit `--id` for commands that need to select a topic.

`historic create "<title>" --extends <topic-id>` starts a continuation topic. The parent may be open or closed; its metadata is never modified, and no files, tags, or related entries are copied. The relation is stored one-way as `extends: "<topic-id>"` in the child `_meta.yaml`, is immutable, and is shown by `historic show <child-id>` (`data[].extends`). An unknown or malformed parent fails before any directory is created.

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
historic status <path> review
historic status .historic/.database/<topic-id>-<slug>/wos/01-task.md complete
historic status <path> forwarded
```

- Work status belongs only to managed member frontmatter. Valid statuses: `create`, `draft`, `pending`, `planned`, `progress`, `review`, `blocked` (open) and `complete`, `failed`, `cancelled`, `archived`, `forwarded` (terminal).
- Topics have no work-status field or topic status command.
- Status changes work on closed topics too. Pass an explicit `.historic/.database/...` path; a path that matches both an open and a closed copy is rejected as ambiguous.
- Each status change also rewrites `_meta.yaml.files[]` (status and type) in the same operation, so no manual rebuild is needed. A manifest failure is reported as a partial-application error that points to `historic rebuild`.
- Assigning a non-`forwarded` status clears `forwarded_to`; assigning `forwarded` keeps an existing destination but does not create one. Use `historic forward` to record a destination.
- `computed_status` is an SQLite aggregate/cache value only.
- `close` and `open` change storage only and never change member status.
- `historic save [id] [--all] [--json]` mirrors open topics into `.historic/.database/` and leaves open workdirs intact. Without an ID it opens an interactive newest-first picker for exactly one open topic; with a topic ID it mirrors only that topic; `--all` explicitly mirrors every open topic. It does not accept `-m/--message` and creates no Git commit.
- `close` mirrors an open topic to `.database` before removing the open workdir; `close` also creates no Git commit. Use `historic git commit -m "message"` for explicit commits.
- `historic open [id] [--json]` reopens a closed topic. Without an ID it opens an interactive newest-first picker scoped to closed topics. `--json` without an ID returns an error instead of starting a picker; `--all` is not supported for open.
- A topic may have both open and closed copies. Open remains authoritative for writes; `find`, `show`, and TUI display versions separately. Compare frontmatter `updated_at` and file `mtime` to identify the newer copy.

```sh
historic close 00001
historic close --all [--json]
historic open [id] [--json]
historic import 00001
historic save [id] [--all] [--json]
```

- `historic list` shows open topics by default; `historic list --closed` lists topics in `.historic/.database/`, including mirror snapshots that still have an open copy. The `--closed` flag does not mean “include closed.”
- `historic close <id>` closes one topic; `historic close --all` closes all open topics discovered at the start of that command. Without an ID or `--all`, `close` opens the interactive picker and acts on exactly one selected topic.
- Bulk close processes topics deterministically by TopicID then path and reuses the single-topic close lifecycle. Each topic reports an individual result; a failure does not stop later topics, but any partial failure returns a non-zero exit code.
- `historic close --all --json` returns one standard JSON envelope. `data` contains `succeeded`, `failed`, `total`, `closed`, and `failed_count`; `ok` is false when one or more topics fail. An empty target list succeeds with zero counts.
- `historic save --all --json` returns the same per-topic result shape with `saved` instead of `closed`; one failure does not stop later mirrors and makes `ok` false. `historic save <id>` reports a single-target result.
- `save` and `close` without IDs support interactive topic search; JSON mode without an ID returns a validation error instead of starting the TUI. `--all` cannot be combined with an ID.
- Only valid first-level open topic folders are targets. Closed topics, metadata/index files, staging entries, and nested folders are not included.
- `open` means `.historic/<topic-id>-<slug>/`.
- `closed` means `.historic/.database/<topic-id>-<slug>/`.
- Close/open preserve all topic bytes and member frontmatter statuses.
- `import` is a compatibility alias for opening a closed topic.

## Forwarding work to a continuation topic

When a task cannot be finished in its topic, forward it to a topic that extends it:

```sh
historic forward wos/03-analisis-kuota.md --to <child-topic-id>
historic forward wos/03-analisis-kuota.md --create "Lanjutan Proyek"
```

- Exactly one of `--to` or `--create` is required.
- `--to` must name an existing topic whose `extends` equals the source file's topic ID; any other target is rejected without filesystem changes.
- `--create "<title>"` creates the continuation topic with `extends` = source topic first, then forwards the file. If creation fails, file status is unchanged; if forwarding then fails, the new topic directory is removed.
- Only managed Markdown files in an **open** topic can be forwarded. Forward a closed copy after `historic open`.
- Forward never moves, copies, or deletes content: only `status: forwarded` and `forwarded_to: "<topic-id>"` change (frontmatter plus `_meta.yaml.files[]`).
- Repeating forward is idempotent and retargets `forwarded_to`. Any later non-`forwarded` status clears it.
- `historic show <topic-id> --json` exposes `data[].files[].forwarded_to`, and `historic find --status forwarded` filters these files.

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
