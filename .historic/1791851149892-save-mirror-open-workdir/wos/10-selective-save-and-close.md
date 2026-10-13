---
title: 10 Selective Save and Close
description: "Ubah save dan close tanpa ID menjadi pemilih satu topic, serta sediakan flag --all untuk operasi bulk secara eksplisit."
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
  - lifecycle
  - save
  - close
  - cli
related:
  - ../spec-save-mirrors-open-workdir.md
  - ./09-interactive-topic-picker.md
---
# WO 10 — Selective Save and Close

## Tujuan

Mengubah default `save`/`close` dari bulk implisit menjadi pemilihan satu topic open,
dan menjadikan operasi massal eksplisit melalui `--all`.

## Perilaku command

```sh
historic save [id] [--all] [--json]
historic close [id] [--all] [--json]
```

- `historic save <id>` tetap mirror satu topic open.
- `historic save` tanpa ID membuka picker WO 09 pada scope open; Enter hanya mirror
  pilihan itu, Esc/q batal tanpa mutasi.
- `historic save --all` mirror semua topic open. `--all` tidak boleh digabung dengan ID.
- `historic close <id>` tetap mirror lalu close satu topic.
- `historic close` tanpa ID membuka picker WO 09 pada scope open; Enter hanya close
  pilihan itu, Esc/q batal tanpa mutasi.
- `historic close --all` mirror lalu close semua target open yang ditemukan pada snapshot
  awal, deterministik berdasarkan ID/path; kegagalan parsial tetap melanjutkan item lain
  dan menghasilkan exit non-zero.
- `--json` tanpa ID tidak membuka picker (JSON adalah mode non-interaktif): gagal validasi
  dengan pesan bahwa ID atau `--all` diperlukan. Untuk satu ID atau `--all`, pertahankan
  envelope dan result per target yang ada.
- Flag `--all` hanya berlaku pada `save` dan `close`; `historic open --all` di luar scope WO.

## Scope teknis

- `internal/cmd/save.go`, `internal/cmd/close.go` dan tes command terkait.
- Pakai picker bersama WO 09; UI memilih target, service lifecycle tetap menjalankan
  mutation dan rollback/index rebuild.
- Tanpa argumen, picker hanya memilih satu topic; jangan melakukan implicit loop.
- Jangan membuat commit Git, remote, push, atau mengubah arti `storage`.

## Status implementasi (2026-10-13)

Selesai: save/close mendukung picker satu topic, operasi bulk eksplisit `--all`, validasi mode JSON non-interaktif, serta test command terkait.

## Acceptance criteria

- Picker `save` menampilkan open topics terbaru dahulu; memilih satu topic hanya memperbarui
  snapshot topic itu, workdir tetap open dan topic lain tidak termirror.
- Picker `close` menutup hanya satu topic terpilih; cancel tidak mengubah filesystem/index.
- `save --all` memirror semua target open, tanpa menghapus workdir.
- `close --all` menutup semua target open dengan semantik partial failure yang sekarang.
- `save <id>` / `close <id>` tetap bekerja; kombinasi `--all <id>` ditolak.
- `save --json` dan `close --json` tanpa ID menghasilkan error envelope, tanpa mencoba
  membaca stdin atau memulai TUI.
- Tidak ada command `save` atau `close` yang membuat Git commit.
