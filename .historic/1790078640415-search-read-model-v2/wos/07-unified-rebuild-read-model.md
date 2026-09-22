---
id: 00007
title: WO 07 Unified Rebuild Read Model
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [rebuild, reconcile, index, aggregate]
related: [../spec.md, ./02-sqlite-topics-files-schema.md, ./03-manual-description-frontmatter.md, ./04-topic-identity-logical-paths.md]
---
# WO 07 — Unified `historic rebuild`

## Tujuan
Menggabungkan reconcile metadata, scan file, aggregate status, dan rebuild index dalam satu workflow.

## Scope
- Scan topic open dan closed.
- Reconcile Files/Assets untuk current copy.
- Classify historic_file/asset.
- Build topics/files read model.
- Read file statuses tanpa menulis status topic ke Markdown.
- Build FTS5 dan storage state.
- Temporary SQLite, validation, atomic replacement, rollback, lock.
- JSON summary topics/updated/errors/records/index.

## Acceptance Criteria
- Satu rebuild melakukan seluruh sinkronisasi.
- Open/closed tidak duplicate.
- Rename/delete/move/classification tercermin.
- Work status file tidak berubah.
- Invalid Markdown menjaga index lama.
- Rebuild kedua no-op/equivalent.
- Open/closed smoke test lulus.

## Status
Complete. Implemented and validated in commit `08d3151`.
