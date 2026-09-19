---
id: 00001
title: WO 16 Save Command
status: complete
created: 2026-09-18
updated: 2026-09-19
---

# WO 16 — `historic save`

## Tujuan
Membuat commit manual pada Git repository internal melalui bahasa CLI Historic.

## Tasks
- Validasi message wajib dan non-empty.
- Stage perubahan pada root repository yang telah ditetapkan.
- Commit perubahan dengan timestamp/author policy yang jelas.
- Tangani clean tree sebagai hasil yang aman dan informatif.
- Hindari commit secret/config yang tidak diizinkan berdasarkan policy.
- Keluarkan commit ID pada text dan JSON output.

## Dependensi
WO 15.

## Acceptance Criteria
- `save -m` menghasilkan commit yang dapat dibaca melalui log.
- Empty message ditolak.
- Clean tree tidak menghasilkan commit palsu.
- Commit failure tidak menghapus perubahan working tree.
- Tidak ada network operation.

## Output
Save command, Git staging/commit service, dan tests.
