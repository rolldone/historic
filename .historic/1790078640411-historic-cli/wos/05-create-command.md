---
id: 00001
title: WO 05 Create Command
status: create
created: 2026-09-18
---

# WO 05 — `historic create`

## Tujuan
Membuat topik baru dengan ID dan metadata konsisten.

## Tasks
- Generate ID berikutnya dari topik aktif dan arsip.
- Dukung `--id` dengan validasi lima digit dan collision check.
- Slug title menjadi nama folder `<id>-<slug>`.
- Buat `_meta.md` status `create`, tanggal hari ini, dan judul.
- Tolak title kosong, ID invalid, dan folder existing tanpa overwrite.
- Tampilkan path dan ID hasil create; dukung `--json` jika sudah tersedia dari kontrak.

## Dependensi
WO 02–04.

## Acceptance Criteria
- Auto ID tidak bertabrakan meskipun ada gap.
- Manual ID `00015` berhasil bila belum digunakan.
- Duplicate ID/folder ditolak tanpa perubahan parsial.
- Hasil dapat dibaca kembali oleh `show` dan `rebuild`.

## Output
Command create dan test collision/slug/ID.
