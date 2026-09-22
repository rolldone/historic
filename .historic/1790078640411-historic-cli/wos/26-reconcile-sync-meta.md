---
id: 00001
title: WO 26 Reconcile Sync Topic Metadata
status: complete
created: 2026-09-19
updated: 2026-09-19
tags: [maintenance, metadata, sync-meta, reconcile, rename]
related: [./23-sync-meta.md, ./25-batch-sync-meta.md, ../spec.md]
---

# WO 26 — Reconcile `historic sync-meta`

## Tujuan

Mengubah perilaku sinkronisasi metadata agar section `Files` dan `Assets` pada `_meta.md` selalu dibangun ulang berdasarkan kondisi filesystem topic saat ini.

Fitur ini menangani rename, move, delete, dan perubahan klasifikasi file tanpa meninggalkan stale link atau duplicate link.

## Latar Belakang

Perilaku saat ini bersifat append-only: link yang sudah ada dipertahankan dan file baru ditambahkan. Jika user mengganti nama file secara manual, link lama tetap berada di `_meta.md` dan link baru ditambahkan.

Perilaku yang diinginkan adalah reconciliation penuh untuk section yang merupakan derived metadata dari filesystem.

## Keputusan Desain

- `## Files` dan `## Assets` dianggap generated sections.
- Setiap eksekusi `sync-meta` mengganti isi kedua section berdasarkan hasil scan filesystem terbaru.
- Section lain pada `_meta.md` dipertahankan.
- `_meta.md` tidak pernah dimasukkan ke `Files` atau `Assets`.
- Link stale dihapus karena target file sudah tidak ada atau berubah klasifikasi.
- Link manual di dalam `Files`/`Assets` yang tidak merepresentasikan file aktual tidak dipertahankan.
- Urutan link ditentukan secara deterministik berdasarkan relative POSIX path ascending.
- Rename tidak perlu dideteksi sebagai event khusus; hasil akhir mengikuti nama file aktual.

## Kontrak Command

```text
historic sync-meta [<id>|<topic-path>] [--json]
historic sync-meta [--json]
```

Perilaku targeted dan batch tetap mengikuti WO 23/25. Perbedaannya adalah setiap topic yang berhasil diproses menggunakan reconciliation penuh pada section `Files` dan `Assets`.

Contoh:

```text
sebelum:  note.md
rename:  note.md -> renamed-note.md
sesudah: Files hanya berisi renamed-note.md
```

## Algoritma

1. Resolve satu topic atau enumerate seluruh topic aktif sesuai mode command.
2. Parse `_meta.md` dan pertahankan frontmatter serta section selain `Files` dan `Assets`.
3. Scan seluruh file topic dengan aturan klasifikasi existing.
4. Buat daftar managed Markdown untuk `Files`.
5. Buat daftar file lain untuk `Assets`.
6. Sort kedua daftar dengan relative POSIX path ascending.
7. Replace isi `## Files` dengan hasil scan terbaru.
8. Replace isi `## Assets` dengan hasil scan terbaru.
9. Tulis `_meta.md` secara atomic hanya jika hasil berubah.
10. Rebuild index setelah update berhasil.

## Scope

### Termasuk

- Rebuild penuh section `Files`.
- Rebuild penuh section `Assets`.
- Penghapusan stale link.
- Penghapusan duplicate link.
- Pemindahan otomatis antar-section ketika klasifikasi berubah.
- Dukungan rename, move, dan delete file melalui hasil scan terbaru.
- Deterministic ordering.
- Idempotence.
- Targeted mode dan batch mode.
- JSON per-topic dan summary existing.
- Regression test untuk rename, delete, move, klasifikasi, dan manual stale links.

### Tidak termasuk

- Mengubah section selain `Files` dan `Assets`.
- Mengubah isi file managed atau asset.
- Mengubah status topic atau lifecycle.
- Menambahkan filesystem watcher.
- Menyimpan histori rename sebagai event terpisah.
- Memproses topic archived secara default.
- Mengubah `PRD.md`.
- Push ke remote repository.

## Keamanan dan atomicity

- Invalid `_meta.md`, symlink, absolute path, traversal, dan topic invalid tetap ditolak sesuai kontrak existing.
- Kegagalan scan atau parse tidak boleh meninggalkan partial `_meta.md`.
- Atomic write wajib digunakan.
- Jika rebuild gagal, perilaku rollback harus mengikuti safety contract existing.
- Mode batch tetap menggunakan continue-on-error dan tidak boleh merusak topic lain.

## Acceptance Criteria

- Rename file manual lalu `historic sync-meta` hanya menghasilkan link dengan nama baru.
- File yang dihapus tidak meninggalkan stale link.
- File yang dipindahkan tidak meninggalkan link pada path lama.
- File yang berubah dari managed Markdown menjadi asset berpindah dari `Files` ke `Assets`.
- File yang berubah dari asset menjadi managed Markdown berpindah dari `Assets` ke `Files`.
- Duplicate link pada section target dihapus melalui rebuild.
- Link manual yang targetnya tidak ada di filesystem dihapus dari `Files`/`Assets`.
- Section selain `Files` dan `Assets` tidak berubah.
- `_meta.md` tetap tidak muncul pada kedua section.
- Urutan link deterministik berdasarkan relative POSIX path.
- Eksekusi kedua tanpa perubahan filesystem menghasilkan `updated: false`.
- Targeted mode tetap hanya mengubah topic target.
- Batch mode tetap memproses semua topic aktif dan mengecualikan archive.
- Continue-on-error, JSON envelope, atomic write, dan rebuild tetap bekerja.
- Unit dan integration test lulus.
- Isolated smoke test membuktikan rename, delete, move, klasifikasi, idempotence, dan non-target protection.
- Dokumentasi WO 23/25 dan README diperbarui bila kontrak existing berubah.

## Dependensi

- WO 23 — `historic sync-meta`.
- WO 25 — batch `historic sync-meta`.
- Existing managed Markdown/asset classifier.
- Existing index rebuild dan JSON envelope.

## Status

Complete. Reconcile penuh section `Files` dan `Assets` telah diimplementasikan, divalidasi, dan di-commit pada `a3e4f6a`.
