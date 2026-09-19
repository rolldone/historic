---
id: 00001
title: WO 09 Rebuild and SQLite Index
status: create
created: 2026-09-18
---

# WO 09 — `historic rebuild` and SQLite Index

## Tujuan
Membangun ulang index dari Markdown tanpa menjadikan SQLite sebagai sumber kebenaran.

## Tasks
- Implementasikan scanner untuk working directory dan `.database/`.
- Parse frontmatter dan infer type dari path: meta, prd, spec, issue, note, decision, task, file.
- Hitung path fields, word_count, mtime, dan hash.
- Buat schema entries dan SQL indexes sesuai SPEC.
- Jalankan rebuild dalam transaction; pertahankan index lama bila scan gagal.
- Sediakan progress summary dan JSON result bila kontrak mengizinkan.
- Pastikan index dapat dihapus lalu direbuild penuh.

## Dependensi
WO 01–03 dan WO 08.

## Acceptance Criteria
- Semua Markdown valid terindeks.
- Satu file invalid dilaporkan dengan path dan tidak menghasilkan index parsial yang dianggap sukses.
- Type inference mengikuti tabel PRD.
- Rebuild kedua menghasilkan hasil konsisten.
- Query list/show/find dapat menggunakan index tanpa mengubah Markdown.

## Output
SQLite repository, indexer, migration/schema, rebuild command, dan tests.
