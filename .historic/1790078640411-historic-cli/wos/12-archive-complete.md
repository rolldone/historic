---
id: 00001
title: WO 12 Archive Complete
status: create
created: 2026-09-18
---

# WO 12 — Archive Topic

## Tujuan
Memindahkan topic close dari working directory ke `.database/` tanpa kehilangan data.

## Tasks
- Implementasikan archive move setelah status close tervalidasi.
- Pertahankan folder name, seluruh file, metadata, dan relative links.
- Validasi destination conflict sebelum operasi.
- Gunakan staging/temp path atau operasi atomic yang sesuai platform.
- Update index hanya setelah move berhasil.
- Tangani failure dan sediakan pesan recovery tanpa menghapus source secara prematur.

## Dependensi
WO 04, WO 09, WO 11.

## Acceptance Criteria
- `complete`, `failed`, `cancelled`, dan `archived` berada di `.database/`.
- Topic tidak lagi muncul sebagai aktif setelah sukses.
- Arsip muncul pada scope archived dan dapat di-show.
- Conflict destination ditolak tanpa merusak source.
- Symlink/path traversal tidak dapat keluar dari root Historic.

## Output
Archive service dan integration tests untuk normal/conflict/failure.
