# Historic CLI

Historic stores work history as portable Markdown files. Markdown is the source of truth; SQLite/FTS5 is a rebuildable cache, and the internal Git repository is local-only.

## Install globally

From a checkout of this repository:

```sh
./build-global.sh
historic version
```

The installer builds the CLI and installs it to `~/.local/bin/historic`. Ensure `~/.local/bin` is on `PATH`.

- [Command reference](docs/commands.md)

## Phase 1 commands

Initialize a workspace:

```sh
historic init
```

Create a topic. IDs are five digits and can be generated automatically or supplied explicitly:

```sh
historic create "Admin Dashboard"
historic create "Admin Dashboard" --id 00014
```

Add Markdown entries and Work Orders:

```sh
historic add prd --id 00014
historic add issue-login-bug --id 00014
historic add wos/scaffold --id 00014
```

Update one file's lifecycle status without archiving its topic:

```sh
historic status 19-fts5-index.md complete --json
historic status .historic/00001-historic-cli/wos/20-advanced-query-filters.md review
```

`historic status` changes only the target Markdown frontmatter, sets `updated`, rejects absolute/traversal paths, and rebuilds the SQLite/FTS index. It accepts `create`, `pending`, `progress`, `review`, `blocked`, `complete`, `failed`, `cancelled`, and `archived`. Use it for individual Work Orders; use topic lifecycle commands only when the entire topic should change or be archived.

Synchronize manually created topic files into `_meta.md`:

```sh
historic sync-meta 00014 --json
historic sync-meta .historic/00014-admin-dashboard --json
historic sync-meta --json
```

`historic sync-meta` fully reconciles the generated `## Files` and `## Assets` sections from the current filesystem. Valid managed Markdown goes to `Files`; plain Markdown, images, PDFs, office files, archives, and binaries go to `Assets`. It excludes `_meta.md`, removes stale and duplicate links, reflects rename/move/delete and classification changes, uses deterministic relative POSIX links, preserves other metadata sections, and rebuilds the index. Without a target, it processes every active topic directly under `.historic/` and excludes `.historic/.database/`. Batch mode continues after per-topic errors and exits non-zero if any topic fails. Assets are not errors; only managed Markdown files are accepted by `historic status`.

Interactive read-only search:

```sh
historic search
```

`historic search` is a Bubble Tea TUI over the same FTS5/search service used by `historic find`. It uses a default limit of 20, empty query shows recent topics, and supports `/` query focus, `f` filter focus, `↑/↓` navigation, `Enter` preview, `n/p` pagination, `r` refresh, `Esc`, and `q`/`Ctrl+C` exit. MVP filters use status, storage scope (`open`, `closed`, `all`), and type (`work-order` maps to `task`). Markdown previews are read-only; assets/binary files are represented by metadata. `historic search --json` is rejected; use `historic find --json` for automation.

## Work status and storage state

Work status and filesystem storage are independent.

- Open topic: `.historic/<id>-<slug>/`
- Closed topic: `.historic/.database/<id>-<slug>/`

A topic may be `complete + open` or `complete + closed`. Completing a topic changes only its work status; it does not move files.

```sh
historic complete 00014 --json
historic close 00014 --json
historic open 00014 --json
```

`close` and `open` preserve frontmatter status and all topic contents. They use safe staged moves, reject duplicate IDs and destination conflicts, and rebuild the index after a successful move. `import <id>` remains as a compatibility alias for opening a closed topic without changing its work status.

Read topics with explicit closed scope when needed:

```sh
historic list
historic list --closed
historic show 00014
historic show 00014 --closed
```

Search both storage states by default:

```sh
historic find "oauth"
historic find "oauth" --open --json
historic find "oauth" --closed --json
```

`--active` and `--archived` are no longer supported. Human search output marks results as `[OPEN]` or `[CLOSED]`; JSON results include `status`, `storage`, and the actual `path`.

Synchronize manually created topic files into `_meta.md`:

FTS results are ranked by relevance with ascending path as a deterministic tie-breaker. Human output includes a contextual snippet with yellow ANSI emphasis (`ESC[1;33m...ESC[0m`) around matched terms; JSON output contains plain data without terminal highlight codes. Empty results are successful JSON responses with `ok: true`. Query terms are treated as literal terms, so punctuation and FTS operators do not execute shell commands or alter the source Markdown.

If a rebuild encounters invalid Markdown, it fails before changing the existing index. Fix the file and run `historic rebuild` again.

Rebuild the SQLite cache from Markdown:

```sh
historic rebuild
historic rebuild --json
```

Every JSON-capable command uses the stable envelope `{ "command", "ok", "data", "error" }`. Successful empty list/find results return `ok: true` with an empty `data` array. Errors return a non-zero exit code and a message on stderr; JSON mode also emits the error envelope on stdout.

## Schema compatibility and recovery

Historic separates binary, workspace-format, and SQLite index-schema versions. Diagnose an existing workspace before recovery:

```sh
historic doctor --json
```

`doctor` is read-only and reports the executable, binary version, workspace format, current/required index schema, Markdown validity, compatibility status, and an actionable recommendation.

```sh
historic upgrade --json
historic rebuild --json
```

- Use `upgrade` when the index schema is older than the binary requirement.
- Use `rebuild` when the index is missing or damaged.
- Both operations scan Markdown as the source of truth and leave Markdown unchanged.
- A temporary SQLite database is validated before atomic replacement.
- The previous index is backed up and protected by rollback handling.
- A lock prevents concurrent upgrade/rebuild replacement.
- Do not manually add SQLite columns or delete `.historic/.index.sqlite`.

## Development and quality gate

```sh
go test ./...
go vet ./...
go build ./...
./build-global.sh
go test ./internal/search -run '^$' -bench BenchmarkFind1000Files -benchtime=1x
```

The Phase 4 FTS5 benchmark suite covers 1,000, 10,000, and 100,000 Markdown files. On the Linux amd64 development machine, a single-run search measured approximately 5.9 ms, 33.7 ms, and 364.5 ms respectively (including the benchmark's indexed result materialization). The 1,000-file result is below the 500 ms baseline target. Run the 100,000-file case separately because fixture creation and indexing are intentionally larger.
