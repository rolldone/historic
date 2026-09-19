# Historic CLI

Historic stores work history as portable Markdown files. Markdown is the source of truth; SQLite is a rebuildable cache.

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
```

`historic sync-meta` adds valid Historic Markdown to `## Files` and all other files (plain Markdown, images, PDFs, office files, archives, and binaries) to `## Assets`. It excludes `_meta.md`, preserves existing and stale links, uses relative POSIX links, avoids duplicates on repeat runs, and rebuilds the index. Assets are not errors; only managed Markdown files are accepted by `historic status`.

Read topics:

```sh
historic list
historic list --archived
historic show 00014
historic show 00014 --json
```

Search Markdown filename, title, and body:

```sh
historic find "oauth"
historic find "oauth" --active --json
historic find "oauth" --status progress
historic find "oauth" --folder .historic/00014-admin-dashboard
historic find "oauth" --archived
historic find "phase 4" --type task --id 00014 --json
```

`historic find` uses the SQLite FTS5 index built by `historic rebuild`. Search covers path, filename, title, and Markdown body. Filters can be combined with `--status`, `--folder`, `--active`, `--archived`, `--type`, and `--id`. If the index is missing or stale, run `historic rebuild`; the command does not silently fall back to a filesystem scan.

FTS results are ranked by relevance with ascending path as a deterministic tie-breaker. Human output includes a contextual snippet with yellow ANSI emphasis (`ESC[1;33m...ESC[0m`) around matched terms; JSON output contains plain data without terminal highlight codes. Empty results are successful JSON responses with `ok: true`. Query terms are treated as literal terms, so punctuation and FTS operators do not execute shell commands or alter the source Markdown.

If a rebuild encounters invalid Markdown, it fails before changing the existing index. Fix the file and run `historic rebuild` again.

Rebuild the SQLite cache from Markdown:

```sh
historic rebuild
historic rebuild --json
```

Every JSON-capable command uses the stable envelope `{ "command", "ok", "data", "error" }`. Successful empty list/find results return `ok: true` with an empty `data` array. Errors return a non-zero exit code and a message on stderr; JSON mode also emits the error envelope on stdout.

## Development and quality gate

```sh
go test ./...
go vet ./...
go build -o historic .
go test ./internal/search -run '^$' -bench BenchmarkFind1000Files -benchtime=1x
```

The Phase 4 FTS5 benchmark suite covers 1,000, 10,000, and 100,000 Markdown files. On the Linux amd64 development machine, a single-run search measured approximately 5.9 ms, 33.7 ms, and 364.5 ms respectively (including the benchmark's indexed result materialization). The 1,000-file result is below the 500 ms baseline target. Run the 100,000-file case separately because fixture creation and indexing are intentionally larger.
