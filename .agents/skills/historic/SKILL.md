---
name: historic
description: "Use when working on the Historic CLI, its SPEC/Work Orders, Markdown history workspace, lifecycle/archive/import, internal Git snapshots, dogfooding, Phase 4 search, Phase 5 Interactive Search TUI, or sync-meta asset metadata, batch synchronization, and reconciliation. Keywords: historic, .historic, SPEC, Work Order, rebuild, archive, import, restore, save, log, diff, sync-meta, Assets, Files, batch, reconcile, rename, TUI, search."
---

# Historic CLI Skill

## Scope

Use this skill when developing, reviewing, testing, or dogfooding Historic. Historic stores work history as Markdown files. Markdown is the source of truth; SQLite/FTS5 is a rebuildable index/cache, and internal Git is local-only.

## Workspace rules

- The canonical workspace directory is `.historic/`.
- `.histories/` is unsupported and has no compatibility layer. Migrate manually with `mv .histories .historic`.
- If `.historic/` and `.histories/` both exist, reject the operation. Never merge or rename automatically.
- The expected structure includes `.historic/.database/`, `.historic/.database/.git/`, and `.historic/.index.sqlite`.
- Never push the internal Git repository and never add a remote to it.
- Do not modify `PRD.md` unless explicitly requested.
- Do not modify source-code files as part of routine lifecycle operations.

## CLI conventions

- Use five-digit topic IDs and `<id>-<slug>` topic folders.
- User-facing paths should be relative to the project root and use `.historic/`.
- Markdown is the source of truth. Run `historic rebuild --json` after recovery or manual index-affecting filesystem changes.
- JSON-capable commands preserve `{ "command", "ok", "data", "error" }`.
- JSON errors must not be duplicated on stderr.

## Sync metadata

```sh
historic sync-meta [<id>|<topic-path>] [--json]
```

- With an ID or path, sync only that active topic.
- Without a target, sync every active topic directly under `.historic/`; archived topics under `.historic/.database/` are excluded.
- Batch processing is deterministic and uses continue-on-error. Remaining topics are processed, errors are collected, and the command exits non-zero if any topic fails.
- `Files` and `Assets` are generated sections derived from the current filesystem.
- Every successful sync fully rebuilds both sections; it does not append to or preserve stale links.
- Managed Markdown with valid Historic frontmatter goes to `## Files`.
- Plain Markdown, images, PDFs, office files, archives, binaries, and other non-managed files go to `## Assets`.
- `_meta.md` is excluded from both sections.
- Rename, move, delete, and classification changes are reflected automatically on the next sync.
- Links are relative POSIX paths and sorted deterministically.
- Sections other than `Files` and `Assets` are preserved.
- Writes are atomic and the index is rebuilt after successful sync.
- Reject absolute/traversal paths, missing or ambiguous topics, symlink topics/files, and invalid `_meta.md`.
- A second sync without filesystem changes returns `updated: false`.

## Search and TUI

- `historic find "query" [--json]` is the one-shot interface for AI, scripts, and automation.
- `historic search` is the read-only interactive terminal UI for human exploration.
- The TUI uses the shared search service and existing FTS5 index; it has no separate index and must not access SQLite directly from the presentation layer.
- The TUI uses Bubble Tea, Bubbles, and Lip Gloss.
- `historic search --json` is rejected; use `historic find --json`.
- Query execution occurs on Enter. Empty query shows recent active topics; `r` refreshes.
- MVP supports status, scope (`active`, `archived`, `all`), and type (`all`, `topic`, `work-order`) filters, pagination, keyboard navigation, and read-only previews.
- Layout must respect terminal viewport height, keep the active result visible, and never render results beneath the footer. Long titles/paths and small terminals require safe truncation or compact layout.

## Validation

Run the relevant checks:

```sh
gofmt -d <changed-go-files>
go test ./...
go vet ./...
go build -o /tmp/historic .
git diff --check
```

Use isolated temporary directories for CLI filesystem tests. For `sync-meta`, cover managed Markdown, plain Markdown, binary/assets, rename, delete, move, classification changes, stale-link cleanup, deterministic ordering, idempotence, archived exclusion, and batch continue-on-error.

## Coordination

For Historic tasks, inspect the relevant SPEC, `_meta.md`, and assigned Work Order first. Preserve Work Order numbering. Record dogfooding blockers as Markdown coordination notes. Commit only task-related files and never push unless explicitly requested.
