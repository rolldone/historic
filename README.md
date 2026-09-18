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
historic find "oauth" --folder .histories/00014-admin-dashboard
historic find "oauth" --archived
```

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

The Phase 1 filesystem search benchmark is below the PRD target of 500 ms for 1,000 Markdown files on the development baseline.
