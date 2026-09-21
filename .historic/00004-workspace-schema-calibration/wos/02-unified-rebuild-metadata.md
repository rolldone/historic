---
id: 00002
title: WO 02 Unified Rebuild and Metadata Reconciliation
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [rebuild, sync-meta, metadata, index, calibration]
related: [./01-schema-calibration-upgrade.md]
---
# WO 02 — Unified `historic rebuild`

## Tujuan

Menggabungkan sinkronisasi metadata topic dan rebuild index ke dalam satu workflow `historic rebuild`, sehingga user tidak perlu menjalankan `sync-meta` dan `rebuild` secara terpisah.

## Kontrak Baru

```text
historic rebuild [--json]
```

`historic rebuild` harus memproses seluruh topic open dan closed, lalu:

1. mereconcile section `## Files` dan `## Assets` pada `_meta.md` berdasarkan filesystem terbaru;
2. membangun ulang index SQLite/FTS5 dengan schema terbaru;
3. memvalidasi hasil build;
4. mengganti index lama secara atomic.

Topic open berada di `.historic/<id>-<slug>/`. Topic closed berada di `.historic/.database/<id>-<slug>/`. Keduanya wajib diproses dan tetap searchable.

## Alur Operasi

```text
Open + closed topic files
          │
          ▼
Reconcile _meta.md Files/Assets
          │
          ▼
Scan Markdown dan storage state
          │
          ▼
Build temporary SQLite/FTS5
          │
          ▼
Validate metadata and index
          │
          ▼
Atomic replace index
```

## Aturan Metadata

- `Files` dan `Assets` direbuild penuh seperti kontrak reconcile WO sebelumnya.
- Rename, move, delete, perubahan klasifikasi, stale link, dan duplicate link harus tercermin.
- `_meta.md` tidak dimasukkan ke `Files` atau `Assets`.
- Section lain pada `_meta.md` dipertahankan.
- Work status topic tidak boleh berubah akibat `rebuild`.
- Storage state (`open`/`closed`) harus diperbarui berdasarkan root filesystem.
- Rebuild harus dapat memproses topic closed tanpa membukanya.

## Hubungan dengan `sync-meta`

`historic sync-meta` tetap boleh dipertahankan sebagai command khusus untuk sinkronisasi metadata topic tertentu atau batch metadata saja.

Namun workflow user utama menjadi:

```text
historic rebuild
```

`rebuild` tidak boleh memanggil command melalui subprocess. Gunakan service/library bersama agar perilaku dan error handling konsisten.

## Schema dan Recovery

- Gunakan schema calibration dari WO 01.
- Baca schema/version metadata sebelum query kolom yang mungkin belum tersedia.
- Build index pada database temporary.
- Backup index lama sebelum replacement jika diperlukan oleh recovery policy.
- Atomic replacement hanya setelah temporary index tervalidasi.
- Jika metadata sync atau index build gagal, source Markdown dan index lama harus tetap aman.
- Temporary database, lock, dan staging leftovers harus dibersihkan.
- Concurrent rebuild harus ditolak atau diserialisasi secara aman.
- Error harus actionable dan menggunakan JSON envelope standar.

## Scope

### Termasuk

- Unified metadata reconciliation dan index rebuild.
- Pemrosesan semua topic open dan closed.
- Reuse service reconcile `Files`/`Assets`.
- Reuse schema-aware atomic index builder.
- Storage state pada index.
- JSON summary untuk jumlah topic, metadata update, index records, dan error.
- Isolated test untuk topic open dan closed.
- Test bahwa work status tidak berubah.
- Test rename/delete/move/classification sebelum rebuild.
- Test idempotence pada rebuild kedua.
- Test rollback dan invalid Markdown.
- Dokumentasi command baru.

### Tidak termasuk

- Otomatis mengubah topic `complete` menjadi `progress`.
- Otomatis melakukan `open` atau `close`.
- Mengubah source Markdown selain generated `Files`/`Assets` metadata.
- Filesystem watcher.
- Remote sync atau push.
- Perubahan `PRD.md`.

## Acceptance Criteria

- Satu `historic rebuild` mereconcile metadata dan rebuild index.
- Topic open dan closed sama-sama diproses.
- `find` dapat menemukan topic open dan closed setelah rebuild.
- Storage state pada index konsisten dengan lokasi topic.
- Rename/delete/move/classification tercermin pada `_meta.md` dan index.
- Work status tidak berubah hanya karena rebuild.
- Rebuild kedua tanpa perubahan menghasilkan metadata/index unchanged atau equivalent no-op.
- `sync-meta` targeted tetap dapat digunakan tanpa rebuild penuh bila dibutuhkan.
- Index lama tidak rusak jika metadata sync atau build temporary gagal.
- Invalid Markdown menghasilkan error actionable dan tidak mengganti index lama.
- Atomic replacement, lock, backup, rollback, dan cleanup tervalidasi.
- `historic rebuild --json` memakai envelope standar.
- Isolated smoke test membuktikan workflow satu-command.
- Tidak ada SQL manual yang diperlukan user.

## Status

Planned. Implementasi belum dimulai.
