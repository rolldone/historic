---
id: 00001
title: WO 13 Import Archived Topic
status: complete
created: 2026-09-18
updated: 2026-09-19
---

# WO 13 — `historic import`

## Tujuan
Mengambil salinan topic dari `.database/` ke working directory untuk dilanjutkan.

## Tasks
- Temukan source topic berdasarkan ID di archive.
- Copy directory recursively dengan preservasi isi Markdown dan file pendukung yang diizinkan.
- Default adalah copy; jangan menghapus source.
- Validasi destination conflict dan opsi overwrite secara eksplisit.
- Update status import sesuai aturan: status close tetap tercatat atau diubah ke `progress` berdasarkan keputusan final.
- Rebuild/update index setelah copy berhasil.

## Dependensi
WO 09, WO 12.

## Acceptance Criteria
- Import ID valid menghasilkan topic aktif yang dapat di-show dan dicari.
- Archive asli tetap utuh pada mode default.
- Conflict menghasilkan error tanpa partial destination.
- Frontmatter `id` dan folder ID konsisten.
- Test mencakup archive missing, destination existing, dan malformed archive.

## Output
Import service, command, conflict handling, dan tests.
