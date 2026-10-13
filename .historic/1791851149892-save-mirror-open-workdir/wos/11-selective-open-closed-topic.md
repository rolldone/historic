---
title: 11 Selective Open Closed Topic
description: "Ubah historic open tanpa ID menjadi picker untuk memilih satu topic closed berdasarkan waktu terbaru, dan pertahankan ID eksplisit."
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
  - lifecycle
  - open
  - cli
  - tui
related:
  - ../spec-save-mirrors-open-workdir.md
  - ./09-interactive-topic-picker.md
---
# WO 11 — Selective Open Closed Topic

## Tujuan

Memungkinkan pengguna memilih topic closed yang akan dibuka kembali dari daftar
interaktif, tanpa perlu mengingat atau menyalin TopicID.

## Perilaku command

```sh
historic open [id] [--json]
```

- `historic open <id>` mempertahankan perilaku saat ini.
- `historic open` tanpa ID membuka picker WO 09 pada scope closed. Urutan terbaru memakai
  `updated_at`, fallback mtime `_meta.yaml`; picker mendukung pencarian teks penuh.
- Enter membuka satu versi closed terpilih; Esc/q membatalkan tanpa mutasi.
- Bila open dan closed mirror memiliki ID sama, item yang dipilih adalah salinan closed;
  operasi open tetap mengikuti resolver/validasi lifecycle yang ada. Bila lifecycle menolak
  karena open copy sudah ada, tampilkan error yang jelas dan jangan overwrite/hapus versi.
- `open --json` tanpa ID gagal dengan error envelope yang menyarankan ID eksplisit; jangan
  memulai TUI dalam mode JSON/non-interaktif.
- `open` tidak mendapat flag `--all` dalam WO ini.

## Scope teknis

- `internal/cmd/root.go` registration/use definition dan `internal/cmd/close.go` atau file
  command storage yang mendaftarkan `open`.
- Reuse picker WO 09, dan panggil service lifecycle hanya setelah user memilih item.
- Hasil pilihan harus membawa path/storage versi closed, tetapi lifecycle tetap melakukan
  validasi authoritative sebelum mutation.

## Status implementasi (2026-10-13)

Selesai: open mendukung picker satu topic dari scope closed, mempertahankan ID eksplisit dan error envelope JSON non-interaktif, serta test command terkait.

## Acceptance criteria

- Picker hanya berisi versi `closed`; open topics tidak muncul.
- Daftar terurut terbaru dahulu dan full-text query dapat mempersempit hasil.
- Memilih satu closed-only topic membuka topic tersebut dan menghapus salinan closed hanya
  sesuai semantik `Open` saat ini.
- Cancel tidak mengubah filesystem atau index.
- `open <id>` tidak regresi; `open --json` tanpa ID menghasilkan error envelope.
- Tidak ada Git commit, remote, atau push.
