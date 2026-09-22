---
id: 00001
title: WO 06 Add Command
status: create
created: 2026-09-18
---

# WO 06 — `historic add`

## Tujuan
Menambahkan dokumen dan Work Order ke topik aktif.

## Tasks
- Terima nama file deskriptif tanpa extension dan tambahkan `.md`.
- Dukung path `wos/<name>` dan buat subfolder yang diperlukan.
- Tentukan topik aktif berdasarkan opsi ID atau aturan current workspace.
- Untuk path di `wos/`, sediakan nomor urut `NN-` bila diperlukan sesuai kontrak.
- Tulis frontmatter minimal yang mewarisi ID topik dan status awal.
- Update daftar Files di `_meta.md` secara aman.
- Tolak traversal path, absolute path, dan overwrite tanpa flag eksplisit.

## Dependensi
WO 03–05.

## Acceptance Criteria
- `add prd`, `add issue-login-bug`, dan `add wos/01-scaffold` menghasilkan file valid.
- Path traversal ditolak.
- Existing file tidak tertimpa default.
- `_meta.md` tetap valid setelah penambahan.

## Output
Command add, path sanitizer, numbering helper, dan tests.
