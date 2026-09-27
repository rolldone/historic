---
title: Fallback open/closed topic untuk show dan find
description: Issue terkait pemilihan salinan open sebagai prioritas dengan fallback ke closed.
status: complete
created: "2026-09-28"
updated: "2026-09-28"
tags:
  - issue
  - search
  - lifecycle
---
# Fallback open/closed topic untuk show dan find

## Latar belakang

Perintah `historic show <id>` gagal saat topic hanya tersedia di storage closed. Sementara itu, hasil default `historic find` dapat menampilkan salinan closed yang sudah tidak menjadi salinan authoritative ketika salinan open dengan ID yang sama tersedia.

## Perubahan

- `show` memilih salinan open jika tersedia; bila tidak, menggunakan salinan closed.
- Pencarian default `find` memilih open per TopicID dan menggunakan closed hanya jika tidak ada versi open.
- Filter `--open` dan `--closed` tetap menjadi pilihan eksplisit dan tidak diubah.
- Resolusi topic pada repository juga menggunakan prioritas open lalu fallback closed.

## Verifikasi

- Tes repository, search, dan CLI lulus: `go test ./internal/repository ./internal/search ./internal/cmd`.
- `git diff --check` lulus.
- Binary global berhasil dibangun dan dipasang dengan `build-global.sh`.

## Status penyelesaian

Complete.
