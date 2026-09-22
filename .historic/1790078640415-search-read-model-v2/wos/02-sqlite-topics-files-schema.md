---
id: 00002
title: WO 02 SQLite Topics and Files Schema
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [sqlite, schema, topics, files, read-model]
related: [../spec.md, ./01-two-table-search-read-model.md]
---
# WO 02 — SQLite `topics` and `files` Schema

## Tujuan
Mendefinisikan dan mengimplementasikan dua tabel read model utama tanpa mengubah perilaku CLI lain.

## Scope
- Tabel `topics` untuk identitas, slug, storage, description, tags, related, dan timestamps.
- Tabel `files` untuk logical relative path, type `historic_file|asset`, metadata, content, dan hash.
- Foreign key/relasi `files.topic_id` ke `topics.id`.
- Schema metadata/version yang kompatibel dengan schema calibration.
- Migration/rebuild test dari Markdown fixture.

## Batasan
- Markdown tetap source of truth.
- Tidak mengubah open/close.
- Tidak menambahkan Vector DB.

## Acceptance Criteria
- Schema dapat dibuat ulang dari empty database.
- Topic dan file/asset memiliki record terpisah.
- Asset `status` nullable.
- Logical path tidak mengandung root workspace atau archive path.
- Schema version terdeteksi sebelum query.
- Test unit/integration lulus.

## Status
Complete. Implemented and validated in commit `2ac8d4c`.
