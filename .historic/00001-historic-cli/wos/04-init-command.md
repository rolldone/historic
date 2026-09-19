---
id: 00001
title: WO 04 Init Command
status: progress
created: 2026-09-18
updated: 2026-09-18
---

# WO 04 — `historic init`

## Tujuan
Membuat workspace Historic yang siap digunakan.

## Tasks
- Deteksi root workspace dari current directory.
- Buat `.historic/` dan `.database/` jika belum ada.
- Buat konfigurasi/index awal sesuai keputusan final nama index.
- Bersikap idempotent: init kedua tidak merusak data.
- Validasi kondisi path berupa file, permission gagal, dan partial initialization.
- Tolak workspace yang hanya memiliki `.histories/`; user harus rename manual ke `.historic/`.
- Tolak kondisi ketika `.historic/` dan `.histories/` sama-sama ada.

## Dependensi
WO 01–03.

## Acceptance Criteria
- `historic init` menghasilkan struktur minimal yang terdokumentasi.
- Menjalankan init dua kali tidak menghapus atau mengubah isi topik.
- Error filesystem memiliki exit code non-zero dan pesan actionable.
- Integration test memakai temporary directory.

## Output
Command init, filesystem abstraction, dan integration tests.
