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

Use `historic find "keyword" --json` for AI agents, scripts, and automation. Use `historic search` for interactive human exploration in the terminal.

## Workspace

The canonical workspace is `.historic/`. The legacy `.histories/` directory is unsupported and must be migrated manually. Markdown is the source of truth. SQLite/FTS5 is a rebuildable index, and `.historic/.database/.git` is an internal Git repository with no remote.

## Safety

Historic lifecycle operations are designed to preserve data, but always inspect `git status` before committing. Never push the internal Historic Git repository or modify `PRD.md` unless explicitly requested.
