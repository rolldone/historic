---
title: 07 Show Dual Version
description: Buat `historic show <id>` menampilkan versi open dan closed sekaligus bila keduanya ada.
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
    - cli
    - show
    - storage
related:
    - ../spec-save-mirrors-open-workdir.md
    - ./04-updated-at-divergence.md
    - ./06-remove-duplicate-suppression.md
---
# WO 07 — Show Dual Version

## Tujuan

`historic show <id>` menampilkan kedua versi (open dan closed) ketika satu TopicID
punya salinan di dua storage, beserta penanda `[OPEN]`/`[CLOSED]` dan `updated_at`.

## Kondisi saat ini

`show` memakai satu resolusi lokasi (`TopicLocation`) yang memilih open sebagai
authoritative dan jatuh ke closed bila open tidak ada, sehingga hanya satu versi tampil.

## Scope

- `internal/cmd/list_show.go` — resolusi lokasi untuk `show` menjadi enumerasi semua
  lokasi yang ada, bukan satu lokasi terpilih.
- Output human-readable menampilkan blok per versi dengan penanda storage dan `updated_at`.
- `--json` menyajikan daftar versi (array) alih-alih objek tunggal.
- Flag `--closed` tetap diterima untuk kompatibilitas dan membatasi output ke versi closed.

## Aturan wajib

- Bila hanya satu versi ada, output tetap satu blok seperti sekarang.
- Daftar isi file tiap versi tetap ditampilkan dan tidak dicampur antar versi.
- Urutan versi deterministik: open sebelum closed.
- Tidak mengubah `internal/lifecycle/location.go` sebagai resolver authoritative untuk
  operasi tulis; perubahan hanya pada penyajian `show`.

## Acceptance criteria

- Setelah `save`, `historic show <id>` menampilkan dua blok: open dan closed.
- Masing-masing blok mencantumkan `updated_at` sehingga versi terbaru teridentifikasi.
- Topic yang hanya ada di satu storage tetap tampil satu blok.
- `show --json` tetap memakai envelope standar.
