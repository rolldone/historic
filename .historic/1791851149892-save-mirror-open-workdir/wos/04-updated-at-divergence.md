---
title: 04 Updated At Divergence
description: Pastikan salinan open dan closed dapat dibedakan lewat updated_at/mtime setelah save.
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
    - index
    - timestamps
    - storage
related:
    - ../spec-save-mirrors-open-workdir.md
    - ./02-save-mirrors-all-open-topics.md
---
# WO 04 — Updated At Divergence

## Tujuan

Memastikan ketika workdir open dan salinan `.database` sama-sama ada, pengguna dapat
membedakan mana yang terbaru berdasarkan `updated_at`/`mtime` yang sudah tersedia di
index, tanpa menambah kolom atau nilai `storage` baru.

## Scope

- Verifikasi pemetaan `topics.updated_at` dari frontmatter `updated` dan
  `files.updated_at` dari frontmatter serta `files.mtime` dari filesystem
  (`internal/indexer/indexer.go`).
- Pastikan mirror tidak menormalkan atau menimpa timestamp frontmatter sehingga
  salinan closed kehilangan jejak waktu.
- Tambahkan dokumentasi singkat pada SPEC/skill tentang cara membaca divergensi.

## Aturan wajib

- Tidak mengubah schema SQLite, tipe `storage`, atau set nilai `open`/`closed`.
- Frontmatter `updated` tidak diubah oleh operasi mirror.
- `created` dan `updated` tetap bersumber dari konten Markdown sebagai source of truth.

## Acceptance criteria

- Setelah mengedit workdir open lalu menjalankan `historic save`, hasil `find --json`
  menunjukkan `updated_at`/`mtime` salinan closed lebih lama dibanding workdir open.
- Salinan closed tetap dapat dibaca dan di-`show` setelah workdir open berubah.
- Tidak ada kolom atau nilai storage baru yang diperkenalkan.
