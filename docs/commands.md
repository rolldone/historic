# Command Reference

## Workspace and topics

```sh
historic init
historic create "Topic Title" [--extends <topic-id>] [--json]
historic add <name> [--id <topic-id>]
historic list [--closed] [--sort updated|created|title|id] [--json]
historic show <topic-id> [--closed] [--json]
historic version [--json]
```

`historic create --extends <topic-id>` records a one-way `extends: "<topic-id>"` relation in the new topic's `_meta.yaml` and returns it as `data.extends` (empty string when absent). The parent topic may be open or closed; it is validated before any directory is created and its own metadata is never modified. Tags, related entries, and files are not copied. The relation is immutable and cannot be changed or removed by a later command. `--extends` may be given only once, and an unknown or malformed parent fails without creating a topic directory.

`historic list` shows open topics by default; `historic list --closed` shows topics stored in `.historic/.database/`, including mirror snapshots created by `historic save`. Open and closed topics are independent listing scopes, so a closed-only listing also includes topics that still have an open copy. `historic show <topic-id>` lists every storage version of the topic (open first) with `updated_at` and `mtime` markers; `--closed` limits the output to closed versions. Both commands display the derived topic status (`-` when the topic has no managed files) and include `computed_status` in `--json`; the value is never written to Markdown.

`historic list` sorts newest-first by default, matching `find` and the TUI:

```text
updated_at DESC (fallback created_at DESC)
  -> ID DESC
  -> open-first (open copy before closed copy for the same TopicID)
  -> path ASC
```

`--sort` accepts `updated` (default), `created` (created_at DESC), `title` (case-insensitive ascending, then the same tie-breakers), and `id` (legacy ID ASC + open-first, kept for scripts that depended on the old order). An unknown value fails with `invalid --sort value "..."; use updated, created, title, or id`. `--sort` works for both the open scope and `--closed`. Sorting happens in memory over filesystem views; the SQLite read model is untouched. `list --json` keeps the same fields and only the element order changes.

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


Work status belongs only to managed member Markdown files. That status is authoritative; a topic has no persisted status field and no topic lifecycle status command. Instead, the derived topic status is computed from managed files: no managed files -> empty; any open file -> `blocked` > `review` > `progress` > `planned` > `pending` > `draft` > `create`; otherwise `failed` wins, then all-`cancelled`, otherwise `complete`. `forwarded` and `archived` count as `complete`; assets are ignored.

- `open`: `.historic/<topic-id>-<slug>/`
- `closed`: `.historic/.database/<topic-id>-<slug>/`

```sh
historic status <member-path> <status> [--json]
historic save [id] [--all] [--json]
historic close [id] [--all] [--json]
historic open [id] [--json]
historic import <id> [--json]
```

`save` and `close` without an ID open an interactive picker for open topics, sorted by newest `updated_at` (falling back to `_meta.yaml` mtime); type to filter, use Up/Down and Enter to select, or Esc/q to cancel. `save <id>` mirrors one topic while retaining its open workdir; `save --all` explicitly mirrors all open topics. `close <id>` mirrors then closes one topic; `close --all` explicitly closes all open topics and reports partial failures. `open` without an ID opens the same picker scoped to closed topics; `open <id>` remains available. JSON mode without an ID returns an error instead of starting an interactive picker. `--all` cannot be combined with an ID and is supported only by save and close.

`close`, `open`, and `import` change storage only and preserve member frontmatter statuses and topic contents. A partial failure in `close --all` continues processing and exits non-zero; `--json` returns one envelope with per-topic results and totals. Save never changes storage. `save` and `close` auto-commit the internal repository (see Auto-commit below); use `historic git commit -m "message"` only when you want an explicitly labelled manual commit.

## File status

```sh
historic status <path> <status> [--json]
```

This command is for any managed Markdown file with valid Historic frontmatter. It updates that member file's `status` and `updated`, then rebuilds the index. The member frontmatter ID may differ from the parent topic ID. It rejects regular assets, Markdown without valid Historic frontmatter, unsafe paths, and symlink targets. Valid statuses include `create`, `draft`, `pending`, `planned`, `progress`, `review`, `blocked`, `complete`, `failed`, `cancelled`, `archived`, and `forwarded`.

Status may be changed for files in closed topics. Use an explicit path such as `.historic/.database/<topic-id>-<slug>/wos/01-task.md`; a bare filename present in both an open and a closed copy is rejected as ambiguous instead of picking one. The same operation also synchronizes the owning topic's `_meta.yaml.files[]` entry (status and type), so `historic rebuild` is no longer required after a status change. If the manifest write fails, the command exits with a partial-application error naming the repair path (`historic rebuild`); the index stays consistent with Markdown.

Setting any status other than `forwarded` clears `forwarded_to` from both the frontmatter and the manifest. Setting `forwarded` through `status` keeps an existing destination but does not create one; use `historic forward` to record a destination.

## Forward

```sh
historic forward <path> --to <topic-id> [--json]
historic forward <path> --create "<title>" [--json]
```

`forward` marks the source file as `status: forwarded` and records `forwarded_to: "<topic-id>"` in its frontmatter and in `_meta.yaml.files[]`. The Markdown body is neither moved nor copied and the file is never deleted. Exactly one of `--to` or `--create` is required.

- `--to` must name an existing topic whose `extends` equals the source file's topic ID. A target that does not extend the source topic is rejected with no filesystem change.
- `--create "<title>"` creates a new topic with `extends` set to the source topic, then forwards the file to it. If topic creation fails, the file status is unchanged; if forwarding fails afterwards, the newly created topic directory is removed.
- The source file must be a managed Markdown file in an open topic. Forwarding a closed copy is rejected with a `historic open` hint.
- Repeating forward is idempotent and updates `forwarded_to` to the new valid destination.
- JSON output returns `data.path`, `data.previous`, `data.current`, `data.forwarded_to`, and `data.created_topic` (empty unless `--create` was used).

## One-shot search

```sh
historic find "keyword"
historic find "" --status-not complete,cancelled,failed,archived
historic find "keyword" --open --json
historic find "keyword" --status draft,progress
historic find "keyword" --status-not complete,cancelled,failed,archived
historic find "keyword" --sort relevance
historic find "keyword" --sort updated
historic find "keyword" --sort created
historic find "keyword" --sort title
historic find "keyword" --page 1 --page-size 20
historic find "keyword" --status-not complete --page-size 10
historic find "" --page 1 --page-size 20
```

`--page` defaults to `1`; `--page-size` defaults to `20` and accepts values from `1` through `100`. Results are filtered, deduped, then deterministically sorted before in-memory pagination. A page beyond the available results succeeds with an empty `items` array. `find` searches open and closed storage by default, but shows at most one result per logical file: duplicates are collapsed by `(TopicID, relative path)` and the open copy always wins over a closed mirror, regardless of mtime or `updated_at`. A file that exists only in closed storage still appears as `[CLOSED]` with its actual path. Explicit `--open` and `--closed` scopes select their requested storage and are not affected by dedupe. Pagination metadata (`has_more`, `next_page`) is computed from the deduped result set. The search uses the shared SQLite FTS5 index. `--status` is a comma-separated inclusion filter that matches either a file's own status or a topic's derived `computed_status`; `--status-not` excludes both. Because `active` is not a lifecycle status, `--status active` is rejected as unknown. Statuses are validated and overlap between the two flags is rejected. Empty keyword (`historic find ""`) browses recent topics consistently with the TUI. Use `--open` for open topics only or `--closed` for closed topics only; combining them is rejected. `--sort` accepts `relevance`, `updated`, `created`, or `title`; keyword defaults to relevance with freshness/path tie-breakers, and empty queries default to updated/created/path order. Results expose work `status`, `storage`, actual `path`, and for files `created_at`, `updated_at`, `mtime`, `hash`, and `size`. In JSON mode, `data` is an object with `items` and `pagination` (`page`, `page_size`, `has_more`, and nullable `next_page`). Human output is multi-line and labeled: a header with `<topic-id> · <title>` and a right-aligned `[OPEN|CLOSED] · <status>`, then `path`, `time`, `reason` (the full description), `match` (highlighted snippet; for multiple occurrences it adds a hit count and best-line hint), `tag`, and `extends →`/`forwarded →` relations when present. Blank lines separate results. The human layout is not a stable scripting contract; automation should use `--json`. Human output ends with either `Page N · showing M results · more results available` plus the next-page command, or `Page N · showing M results · no more results`; empty output remains `No matches found.`. Legacy `--active` and `--archived` flags are not supported. If the index is missing or invalid, run `historic rebuild`. Pagination uses the SQLite read model only and does not hash/stat files during search. Changes between page requests can cause offset drift; this MVP behavior is eventual consistency.

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
- `n`/`right arrow` changes to the next page when available; `p`/`left arrow` changes to the previous page; `r` refreshes the active page.
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
historic save [id] [--json]
historic log <id> [--json]
historic diff <id> [--json]
historic restore <id> --snapshot <commit> [--force] [--json]
```

The explicit Git passthrough runs in `.historic/.database` and forwards arguments and standard streams directly, without a shell:

```sh
historic git status
historic git log -1 --oneline
historic git diff --stat
```

All Git subcommands and flags are passed through as requested, including `push`, `pull`, `commit`, and `init`; the passthrough itself runs nothing automatically. An explicit `historic git init` is passed to Git normally. The proxy does not otherwise initialize the repository or alter Git configuration. With no Git arguments it prints usage. `historic log`, `historic diff`, and `historic save` remain separate Historic commands.

Snapshots are local-only and include both open and closed topic storage under `.historic`. `save` refreshes the closed snapshot while leaving the open topic intact; `close` mirrors then removes the open workdir. Both are filesystem operations plus one automatic internal commit; storage movement never changes work status. The open copy is authoritative for writes. Search/show can display both versions; compare `updated_at` (frontmatter) and file `mtime` to identify which snapshot is newer.

## Auto-commit

Operations that change the closed mirror `.historic/.database` run `git add --all` and `git commit` in the internal repository. Auto-commit is local-only: no push, no remote, no branch/tag/checkout, and no workdir/`.git`-tracked files.

| Trigger | Message |
| --- | --- |
| `save <topic-id>` | `auto: save <topic-id> (n path berubah)` |
| `save --all` | `auto: save --all (n path berubah)` — one aggregate commit |
| `close <topic-id>` | `auto: close <topic-id> (n path berubah)` |
| `close --all` | `auto: close --all (n path berubah)` — one aggregate commit |
| `status`/`delete` on a closed path | `auto: status <topic-id> (1 path berubah)` |
| `rebuild` | `auto: rebuild ? (n path berubah)` |

- `n` is the number of `git status --porcelain` entries counted before staging; `?` is used when there is no single target topic.
- The `auto:` prefix is reserved for Historic. Manual commits via `historic git commit -m` never use it.
- One operation produces at most one commit. `close --all` and `save --all` produce a single aggregate commit, never one per topic.
- Changes in an **open** workdir do not commit; use `historic save` to mirror them first. `open` (reopen) does not trigger a commit.
- No changes: the step is skipped with `commit.reason` = `no-changes` (not an error). No internal repository: skipped with `no-repository`.
- A commit failure (including a missing Git identity) is a non-fatal warning: the main operation still succeeds, exits zero, and recommends running `historic git add --all` and `historic git commit -m` manually. Historic never invents `user.name`/`user.email`.

Commands that trigger auto-commit include a nullable `commit` object on the JSON envelope:

```json
{"committed": true, "hash": "abc1234", "message": "auto: save 1793622992705 (6 path berubah)", "changed_paths": 6}
```

or `{"committed": false, "reason": "no-changes"}`. Human output adds one line such as `auto-commit: abc1234 auto: save 1793622992705 (6 path berubah)`, `auto-commit: skipped (no-changes)`, or a `auto-commit gagal (tidak fatal): ...` warning.

## Rebuild and output

```sh
historic rebuild
historic rebuild --json
```

Every JSON-capable command uses the envelope:

```json
{"command":"...","ok":true,"data":{},"error":null}
```

`historic rebuild` is the unified workflow for all open and closed topics. It recursively scans every topic file, regenerates the `_meta.yaml` files/assets manifest, recomputes the derived topic status using the same deterministic aggregation as `list`/`show`/`find`, rebuilds storage-aware SQLite/FTS5, validates the temporary index, and atomically replaces the prior index. If `_meta.yaml` is missing, it is minimally recovered from the folder ID/slug; recovery is rolled back if rebuild fails.

The SQLite index schema is owned by the installed binary. Rebuild writes the schema of whichever binary is running, so after pulling source changes that alter the schema you must reinstall before validating (`./build-global.sh`, then `historic version` and `historic rebuild --json`). A stale binary keeps the previous `index_schema_version` and cannot create newer columns such as `topics.extends_id` or `files.forwarded_to`; `historic doctor` then reports `rebuild required`.
