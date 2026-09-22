---
id: 00001
title: WO 23 Sync Topic Metadata
status: progress
created: 2026-09-19
updated: 2026-09-19
tags: [maintenance, metadata, sync-meta, cli]
---

# WO 23 — `historic sync-meta`

## Tujuan
Menyediakan command eksplisit untuk menyelaraskan section `Files` dan `Assets` pada `_meta.md` setelah user membuat file secara manual.

## Latar Belakang
`historic rebuild` hanya membangun ulang SQLite/FTS index dan tidak memodifikasi Markdown. Karena topic adalah wadah bebas, file managed dan asset perlu dicatat terpisah tanpa memaksa seluruh file memiliki frontmatter Historic.

## Kontrak Command

```text
historic sync-meta <id> [--json]
historic sync-meta <topic-path> [--json]
```

## Tasks
- Tambahkan command Cobra `sync-meta`.
- Resolve topic aktif/arsip berdasarkan ID atau path yang aman.
- Jika ID tanpa `--archived` ambigu antara active/archive, tolak dengan error jelas.
- Scan seluruh file dalam topic, kecuali `_meta.md`.
- Klasifikasikan Markdown dengan frontmatter Historic valid sebagai managed file.
- Klasifikasikan Markdown biasa, gambar, PDF, Word, spreadsheet, archive, binary, dan file lain sebagai asset.
- Tambahkan link managed file yang belum tercantum ke section `## Files`.
- Tambahkan link asset yang belum tercantum ke section `## Assets`.
- Asset bukan error, tidak diberi warning, dan tidak diubah isinya.
- Pertahankan link existing, urutan existing, dan link stale; jangan hapus otomatis.
- Gunakan label link stabil dan path POSIX relatif terhadap topic.
- Tulis `_meta.md` secara atomic.
- Jalankan `historic rebuild` setelah update berhasil.
- Dukung `--json` dengan envelope `command`, `ok`, `data`, `error`.
- Tolak absolute path, traversal, topic missing, topic ambiguous, symlink, dan `_meta.md` invalid.
- Pastikan command tidak mengubah status topic, memindahkan folder, atau mengubah file non-target.
- Tambahkan unit test untuk klasifikasi, link generation, duplicate detection, dan integration test CLI.
- Dokumentasikan command dan section `Files`/`Assets` di README.

## Dependensi
WO 03, WO 07, WO 09, dan file-level status command.

## Acceptance Criteria
- Markdown managed yang dibuat manual muncul pada `Files` setelah `sync-meta`.
- Gambar/PDF/Word/file biasa muncul pada `Assets` setelah `sync-meta`.
- Menjalankan command dua kali tidak menggandakan link.
- Link existing dan stale tetap dipertahankan.
- `_meta.md` sendiri tidak masuk section apa pun.
- Asset tanpa frontmatter tidak menyebabkan command gagal.
- `_meta.md` tetap valid dan topic tetap aktif.
- Index berhasil direbuild setelah sync.
- Failure tidak meninggalkan file temporary atau partial metadata.
- JSON success/error valid; JSON error tidak dicetak ulang ke stderr.
- `historic status` menolak asset dan Markdown biasa karena tidak memiliki managed frontmatter.
- Path output relatif terhadap root `.historic`.
- Test membuktikan topic lain dan file lain tidak berubah.

## Output
Command `sync-meta`, service metadata synchronization, tests, README update, dan report validasi.
