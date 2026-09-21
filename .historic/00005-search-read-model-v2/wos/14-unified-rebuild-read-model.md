---
id: 00014
title: WO 14 Unified Rebuild Topic File Read Model
status: planned
created: 2026-09-21
updated: 2026-09-21
tags: [rebuild, topics, files, aggregate, metadata]
related: [../spec.md]
---
# WO 14 — Unified Rebuild Topic/File Read Model

## Tujuan
Menjadikan `rebuild` workflow tunggal untuk metadata reconciliation, scan open/closed, aggregate status, SQLite two-table read model, dan FTS preparation.

## Scope
- Reconcile Files/Assets current copy.
- Build `topics` dan `files` records.
- Classify historic_file/asset.
- Read file status dan description.
- Compute aggregate tanpa menulis status topic.
- Storage state dan logical paths.
- Temporary DB, validation, atomic replacement, rollback, lock.
- Idempotence dan invalid Markdown safety.

## Acceptance Criteria
- Satu rebuild mencakup seluruh topic open/closed.
- Tidak duplicate logical topic/file.
- Work status authoritative tetap di file.
- Asset tidak memengaruhi aggregate.
- Rebuild kedua no-op/equivalent.
- Failure menjaga index lama.

## Status
Complete. Implemented and validated in commit `c84540a`.
