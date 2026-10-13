---
title: 03 Remove Commit and Message Flag
description: Hapus git commit dari save dan close, dan hapus flag -m/--message dari save.
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
    - lifecycle
    - git
    - cli
related:
    - ../spec-save-mirrors-open-workdir.md
    - ./02-save-mirrors-all-open-topics.md
---
# WO 03 — Remove Commit and Message Flag

## Tujuan

Mengeluarkan git commit dari `save` dan `close`, serta menghapus flag `-m/--message`
dari `save`. Commit diserahkan sepenuhnya ke passthrough `historic git commit`.

## Scope

- `internal/cmd/save.go` — hapus `gitproxy.Open`, `AddAll`, `Commit`, flag `-m/--message`,
  serta tipe `saveOutput`/`saveData` yang lama bila tidak dipakai lagi.
- `internal/lifecycle/storage.go` — pastikan jalur `close` dan `close all` tidak memanggil
  Git sama sekali.
- Perbarui `docs/commands.md` bagian lifecycle/versioning agar tidak lagi menyebut
  `save -m "message"` sebagai pembuat commit.
- Perbarui `.agents/skills/historic/SKILL.md` pada bagian lifecycle agar konsisten.

## Aturan wajib

- Tidak ada panggilan `repository.Commit(...)` dari jalur `save` maupun `close`.
- `gitproxy` tetap dipakai oleh `log`, `diff`, `restore`, dan `historic git` sesuai kebutuhan.
- `internal/cmd/log_diff.go` tidak berubah selain penyesuaian teks dokumentasi bila perlu.
- Perubahan dokumentasi tidak mengubah makna command lain.

## Acceptance criteria

- `historic save -m "x"` gagal dengan error flag tidak dikenal.
- `historic save` sukses tanpa membuat commit baru; `git log` di `.database` menunjukkan
  commit tidak bertambah karena save.
- `historic close` tidak menambah commit.
- Tidak ada referensi `-m` untuk `save` tersisa di `docs/commands.md` dan skill.
