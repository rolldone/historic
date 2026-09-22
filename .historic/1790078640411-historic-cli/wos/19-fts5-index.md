---
id: "00001"
title: WO 19 SQLite FTS5 Index
status: complete
created: "2026-09-19"
updated: "2026-09-19"
tags:
    - phase-4
    - search
    - fts5
    - sqlite
---
# WO 19 — SQLite FTS5 Index

## Tujuan
Menambahkan index FTS5 rebuildable untuk mendukung Advanced Search tanpa menjadikan SQLite sebagai source of truth.

## Tasks
- Validasi dukungan FTS5 pada driver SQLite yang digunakan.
- Buat virtual table FTS5 untuk path, filename, title, dan content.
- Tentukan tokenizer dan strategi escaping query.
- Sinkronkan FTS dengan `index_records` pada `historic rebuild`.
- Rebuild FTS dalam transaction yang sama atau mekanisme atomic setara.
- Hapus/recreate FTS saat rebuild agar stale record tidak tersisa.
- Pertahankan index lama bila scan Markdown atau pembuatan FTS gagal.
- Pastikan `.historic/.index.sqlite` tetap rebuildable dari Markdown.

## Dependensi
WO 09 dan SPEC Phase 4.

## Acceptance Criteria
- FTS5 table tersedia setelah `historic rebuild`.
- Record path/title/content dapat ditemukan melalui FTS.
- Rebuild berulang menghasilkan jumlah dan hasil konsisten.
- Markdown invalid tidak mengganti index valid sebelumnya.
- Index dapat dihapus dan dibangun ulang.
- Test memverifikasi driver benar-benar mendukung FTS5, bukan hanya table biasa.

## Output
Schema FTS5, perubahan indexer/rebuild, migration handling, dan tests.
