---
id: "00001"
title: Historic CLI Foundation, Lifecycle, and Versioning
status: progress
created: "2026-09-18"
updated: "2026-09-18"
tags:
    - historic
    - cli
    - mvp
    - phase-1
    - phase-2
    - phase-3
---
# Historic CLI Foundation, Lifecycle, and Versioning

## Deskripsi
Implementasi terencana untuk mencapai Phase 1 Foundation, Phase 2 Lifecycle, dan Phase 3 Versioning dari PRD Historic.

## Files
- [Technical SPEC](./spec.md)
- [WO 01: Project Bootstrap](./wos/01-project-bootstrap.md)
- [WO 02: Domain Model](./wos/02-domain-model.md)
- [WO 03: Markdown Frontmatter](./wos/03-markdown-frontmatter.md)
- [WO 04: Init Command](./wos/04-init-command.md)
- [WO 05: Create Command](./wos/05-create-command.md)
- [WO 06: Add Command](./wos/06-add-command.md)
- [WO 07: List and Show](./wos/07-list-and-show.md)
- [WO 08: Find Command](./wos/08-find-command.md)
- [WO 09: Rebuild and SQLite Index](./wos/09-rebuild-and-index.md)
- [WO 10: MVP Quality Gate](./wos/10-mvp-quality-gate.md)
- [WO 11: Lifecycle Commands](./wos/11-lifecycle-commands.md)
- [WO 12: Archive Complete](./wos/12-archive-complete.md)
- [WO 13: Import Archived Topic](./wos/13-import-archived-topic.md)
- [WO 14: Lifecycle Recovery](./wos/14-lifecycle-recovery.md)
- [WO 15: Internal Git Repository](./wos/15-internal-git-repository.md)
- [WO 16: Save Command](./wos/16-save-command.md)
- [WO 17: Log and Diff](./wos/17-log-and-diff.md)
- [WO 18: Restore and Phase 3 Gate](./wos/18-restore-and-phase-3-gate.md)
- [WO 19: SQLite FTS5 Index](./wos/19-fts5-index.md)
- [WO 20: Advanced Query and Filters](./wos/20-advanced-query-filters.md)
- [WO 21: Ranking, Snippet, and Highlight](./wos/21-ranking-snippet-highlight.md)
- [WO 22: Search Benchmark and Quality Gate](./wos/22-search-quality-gate.md)
- [WO 23: Sync Topic Metadata](./wos/23-sync-meta.md)
- [WO 24: Interactive Search TUI](./wos/24-interactive-search-tui.md)
- [WO 25: Batch Sync Topic Metadata](./wos/25-batch-sync-meta.md)
- [Interactive Search TUI SPEC](./spec-interactive-search-tui.md)

## Assets

- [spec-interactive-search-tui.md](./spec-interactive-search-tui.md)
- [24-interactive-search-tui.md](./wos/24-interactive-search-tui.md)
- [25-batch-sync-meta.md](./wos/25-batch-sync-meta.md)

## Progress
- Phase 1 Foundation: WO 01–10 complete
- Phase 2 Lifecycle: WO 11–14 complete
- Phase 3 Versioning: WO 15–18 complete
- Phase 4 Advanced Search: WO 19–22 complete
- Maintenance: WO 23 sync-meta in progress; WO 25 batch sync-meta complete
- Phase 5 Interactive Search TUI: WO 24 in progress
- Handoff: Phase 1–4 ready; maintenance and Phase 5 active

## Decision Notes
- Transition matrix WO 02 dipertahankan sesuai implementasi saat ini.
- Matrix lifecycle yang lebih ketat wajib dikunci sebelum WO 11 Lifecycle Commands.
- Breaking change workspace disetujui: `.historic/` menjadi satu-satunya nama canonical sebelum Phase 4.
- `.histories/` tidak memiliki compatibility layer; rename workspace lama dilakukan manual.

Dokumen ini hanya berisi SPEC dan Work Order koordinasi. Tidak ada source code yang dibuat atau diubah.
