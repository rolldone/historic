---
id: 00004
title: WO 02 Fix closed-only topic listing
status: complete
created: 2026-09-27
updated: 2026-09-27
tags: [lifecycle, storage, list, closed, regression]
related: [./01-separate-work-status-storage-state.md]
---

# WO 02 — Fix `historic list --closed` scope

## Konteks

Implementasi awal `historic list --closed` menambahkan topic dari closed storage ke hasil list, tetapi tidak mengecualikan topic open. Akibatnya perintah menampilkan kedua storage state meskipun flag bernama `--closed` dan kontrak WO 01 menetapkan scope closed saja.

## Perubahan

- Ubah pemfilteran `TopicStore.ListTopics` agar memilih tepat satu storage scope: default/open jika flag false, closed jika flag true.
- Perjelas help text flag menjadi `list only closed topics`.
- Tambahkan regression test repository untuk memastikan closed listing tidak mengembalikan topic open dan memilih salinan open yang otoritatif saat ID ganda.
- Tambahkan test CLI dengan satu topic open dan satu closed untuk memastikan `historic list --closed` hanya menampilkan yang closed.
- Perjelas docs dan skill bahwa `--closed` adalah filter eksklusif.

## Validasi

- `go test ./internal/cmd ./internal/repository`
- `go test ./...`
- `./build-global.sh && historic list --closed`: berhasil; hanya topic closed yang tampil.

## Commit

`377f4c1 Fix closed-only topic listing`
