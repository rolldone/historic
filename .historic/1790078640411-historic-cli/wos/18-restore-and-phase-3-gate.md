---
id: 00001
title: WO 18 Restore and Phase 3 Gate
status: complete
created: 2026-09-18
updated: 2026-09-19
---

# WO 18 — `restore` and Phase 3 Quality Gate

## Tujuan
Memulihkan snapshot secara aman dan menutup Phase 3 dengan verifikasi versioning.

## Tasks
- Finalkan strategi restore: target snapshot, scope topic, dan perilaku terhadap uncommitted changes.
- Implementasikan validasi snapshot ID dan target path.
- Tambahkan dry-run/confirmation untuk operasi yang menimpa perubahan.
- Pastikan restore tidak keluar dari `.database/` dan tidak mengeksekusi hook tidak tepercaya tanpa policy.
- Rebuild index setelah restore berhasil.
- Uji save → modify → diff → restore → rebuild end-to-end.
- Dokumentasikan backup/recovery dan batasan no-push Phase 3.

## Dependensi
WO 14–17.

## Acceptance Criteria
- Snapshot valid dapat dipulihkan sesuai strategi final.
- Uncommitted changes tidak tertimpa diam-diam.
- Invalid snapshot/dirty state menghasilkan error dan state tetap aman.
- Restore dan rebuild menghasilkan index konsisten.
- Phase 3 success metrics PRD terpenuhi: Git versioning dan restore berjalan.

## Output
Restore command, safety checks, end-to-end tests, versioning docs, dan Phase 3 sign-off.
