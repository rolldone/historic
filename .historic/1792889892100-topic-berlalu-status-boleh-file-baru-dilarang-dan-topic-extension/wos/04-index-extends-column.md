---
title: 'WO 04 — Index dan Migrasi untuk Relasi Topic Extension'
description: "Membuat read model SQLite menyimpan extends_id yang diturunkan dari metadata Markdown dan memigrasikan index lama secara aman."
status: create
created: "2026-10-25"
updated: "2026-10-28"
tags:
    - index
    - migration
    - extension
related:
    - "1792889892100"
---
# WO 04 — Index dan Migrasi untuk Relasi Topic Extension

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §7, AC-8.
Metadata Markdown adalah source of truth; SQLite adalah read model yang dapat dibangun
ulang. Extension perlu diindeks untuk pembacaan topic tanpa mengubah sumber.

## Ruang lingkup

- Tambahkan `extends_id` nullable pada tabel topic yang sesuai di `internal/indexer/schema.go`.
- Tambahkan migrasi schema/version untuk index lama: kolom lama diisi `NULL`.
- Rebuild index mengambil `extends` dari `_meta.yaml`, menyimpan parent ID bila ada.
- Rebuild idempoten dan mempertahankan safeguards atomic replacement/rollback.
- Dry-run, bila didukung oleh flow upgrade/index migration terkait, tidak menulis schema
  atau metadata.
- Tidak membuat daftar children/backlink pada parent; itu non-goal.

## Langkah implementasi

1. Tinjau schema versioning dan migrasi yang ada sebelum menetapkan versi baru.
2. Tambahkan `extends_id` nullable; pastikan index lama termigrasi otomatis tanpa
   kehilangan data.
3. Perbarui scanner/index records dan query insert/update.
4. Baca relasi dari metadata authoritative, bukan disimpulkan dari nama folder.
5. Tambahkan test migrasi dari schema lama, rebuild parent/child, `NULL` untuk metadata
   legacy, idempotensi, dan rollback bila penggantian index gagal.

## Kriteria penerimaan

- **AC-1** Topic tanpa `extends` memiliki `extends_id = NULL`.
- **AC-2** Child menyimpan ID parent yang benar setelah rebuild.
- **AC-3** Index schema lama termigrasi otomatis tanpa kehilangan data.
- **AC-4** Rebuild berulang tidak menghasilkan perubahan data yang tidak perlu.
- **AC-5** Parent tidak mendapat backlink dan metadata parent tetap byte-identical.
- **AC-6** Proteksi atomic/rollback index tidak regresi.
- **AC-7** `historic rebuild --json` melaporkan 0 error di fixture extension.

## Test plan

| Kasus | Ekspektasi |
| --- | --- |
| Metadata no extends | `extends_id` NULL |
| Child metadata extends parent | `extends_id` sama dengan ID parent |
| Workspace lama | Migrasi schema sukses, rows lama tetap ada |
| Rebuild berulang | Hasil sama dan idempoten |
| Parent tidak dimodifikasi | Metadata parent tidak berubah |
| Index replacement failure | Index lama tetap tersedia |

## Risiko dan dependensi

Risiko tinggi-menengah: schema migration dan atomic replace sensitif. **WO-03** harus
menentukan format metadata `extends` sebelum implementasi indeks.
