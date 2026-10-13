---
title: 05 Save Mirror Test Matrix
description: Matriks pengujian untuk save-as-mirror, koeksistensi open/closed, dan penghapusan commit.
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
    - testing
    - lifecycle
    - save
related:
    - ../spec-save-mirrors-open-workdir.md
    - ./01-extract-mirror-operation.md
    - ./02-save-mirrors-all-open-topics.md
    - ./03-remove-commit-and-message-flag.md
    - ./04-updated-at-divergence.md
---
# WO 05 — Save Mirror Test Matrix

## Tujuan

Menyediakan matriks pengujian yang membuktikan perilaku baru `save` dan `close` sesuai SPEC.

## Scope

- Perbarui `internal/cmd/save_test.go` yang saat ini mengasumsikan `save` menghasilkan commit.
- Tambah tes pada `internal/lifecycle/` untuk operasi mirror.
- Putuskan apakah tes memerlukan helper `gitproxy` baru; hindari helper yang hanya dipakai tes commit lama.

## Kasus uji minimal

1. `save` pada workspace tanpa topic open: sukses, hitungan nol, tidak error.
2. `save` dengan satu topic open: `.database` terisi, workdir open tetap ada, `list` tetap open.
3. `save` dua kali: idempotent, isi tidak berubah.
4. `save` setelah workdir open diedit: salinan `.database` diperbarui.
5. `save -m "x"`: ditolak sebagai flag tidak dikenal.
6. `save` tidak menambah commit Git; `close` juga tidak.
7. `close` tetap menghapus workdir open dan menghasilkan `storage: closed`.
8. `close all` dengan satu kegagalan: melanjutkan, exit code non-zero, JSON memuat `failed`.
9. Setelah `save`, `find`/`show`/TUI menampilkan dua versi (open dan closed), bukan satu.
10. Salinan closed dapat dibedakan lewat `updated_at`/`mtime` setelah workdir open berubah.
11. `find --open` hanya versi open; `find --closed` hanya versi closed; pagination tetap akurat.
12. Operasi tulis (`close`, `status`, `add`) tetap memakai salinan open sebagai authoritative.

## Acceptance criteria

- Semua kasus di atas punya tes otomatis yang lulus.
- `go test ./...` lulus.
- Tes lama yang mengharapkan commit dari `save` dihapus atau disesuaikan ekspektasinya.
- Tidak ada tes yang bergantung pada remote Git atau push.
