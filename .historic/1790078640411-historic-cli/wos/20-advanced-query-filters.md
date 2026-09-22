---
id: "00001"
title: WO 20 Advanced Query and Filters
status: complete
created: "2026-09-19"
updated: "2026-09-19"
tags:
    - phase-4
    - search
    - filters
    - cli
---
# WO 20 — Advanced Query and Filters

## Tujuan
Memindahkan `historic find` ke query FTS5 dan menyediakan filter Phase 4 yang dapat dikombinasikan secara aman.

## Tasks
- Gunakan FTS5 sebagai backend pencarian default Phase 4.
- Pertahankan pencarian pada path, filename, title, dan content.
- Tambahkan filter `--type` dan `--id`.
- Pertahankan `--status`, `--folder`, `--active`, dan `--archived`.
- Validasi konflik filter active + archived dan input folder/path.
- Tetapkan perilaku index belum tersedia: error actionable menyarankan `historic rebuild`.
- Tangani query kosong, Unicode, punctuation, operator FTS, dan input invalid tanpa shell execution.
- Pastikan JSON envelope dan exit code stabil.

## Dependensi
WO 19 dan WO 08.

## Acceptance Criteria
- Query dan kombinasi filter menghasilkan scope yang benar.
- Active/archive tidak tertukar.
- `--type` mengikuti type inference index.
- Invalid query menghasilkan error jelas dan tidak mengubah index.
- Empty result sukses dengan `ok: true` dan data kosong.
- Integration test mencakup Markdown aktif dan arsip.

## Output
Search service FTS5, flag CLI, validation, serializer, dan integration tests.
