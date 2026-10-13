---
title: 08 TUI Dual Version
description: Tampilkan kedua versi (open dan closed) pada interface pencarian Bubble Tea.
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
    - tui
    - search
    - storage
related:
    - ../spec-save-mirrors-open-workdir.md
    - ./06-remove-duplicate-suppression.md
    - ./07-show-dual-version.md
---
# WO 08 — TUI Dual Version

## Tujuan

`historic search` (TUI) menampilkan kedua versi open dan closed sebagai hasil terpisah,
konsisten dengan `find`.

## Scope

- `internal/tui/` — daftar hasil memakai search service yang sudah melepas dedup,
  sehingga tidak memerlukan query duplikat terpisah.
- Baris hasil menampilkan penanda `[OPEN]`/`[CLOSED]` dan `updated_at` agar mudah dibedakan.
- Preview tetap read-only dan menampilkan versi yang sedang disorot saja.
- Filter scope (`open`, `closed`, `all`) tetap berperilaku sebagai filter, bukan dedup.

## Aturan wajib

- TUI tidak mengakses SQLite langsung dari layer presentasi.
- Layout tetap menghormati tinggi viewport; hasil tidak boleh render di bawah footer.
- Judul/path panjang tetap dipotong aman pada terminal kecil.
- `historic search --json` tetap ditolak.
- Tidak ada lifecycle mutation dari TUI.

## Acceptance criteria

- Setelah `save`, daftar TUI menampilkan dua entri untuk TopicID yang punya dua versi.
- Filter `open` hanya menampilkan versi open; `closed` hanya versi closed; `all` menampilkan keduanya.
- Penanda storage dan `updated_at` terlihat pada baris hasil.
- Navigasi, pagination (`n`/`p`), dan refresh (`r`) tetap berfungsi.

## Status implementasi (2026-10-13) — selesai

Baris hasil TUI menampilkan `[STORAGE] id title · updated_at/mtime`, scope default
diubah ke `all` (all → active → archived), dan preview menampilkan `storage`,
`updated_at`, dan `mtime` dari hasil tersorot. TUI memakai search service yang kini
sudah mengindeks dua versi, sehingga kedua entri tampil sebagai baris terpisah.

Blocker schema (PK `topics.id`) telah diatasi bersama WO 06; lihat catatan WO 06.
