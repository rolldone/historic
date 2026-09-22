---
id: 00008
title: WO 08 Aggregate Topic File Queries
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [query, aggregate, status, sqlite]
related: [../spec.md, ./02-sqlite-topics-files-schema.md, ./07-unified-rebuild-read-model.md]
---
# WO 08 — Aggregate Topic/File Queries

## Tujuan
Menyediakan query SQL lintas `topics` dan `files` untuk summary AI.

## Scope
- Join topic/file.
- COUNT/SUM/MAX/GROUP BY/HAVING.
- total, active, resolved, complete, cancelled.
- computed status tanpa mengubah source Markdown.
- Filter status, tags, storage, type, topic, folder, dates.
- JSON context topic untuk hasil file.

## Acceptance Criteria
- Complete/cancelled dihitung resolved.
- Asset tidak memengaruhi status aggregate.
- Query topic tanpa file tidak gagal.
- Filter dapat dikombinasikan.
- Aggregate deterministic dan testable.

## Status
Complete. Implemented and validated in commit `600b5f4`.
