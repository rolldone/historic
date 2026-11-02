---
title: WO 04 — Tampilkan Status Topic pada list dan show
description: Menambahkan tampilan status topic hasil agregasi pada historic list dan historic show baik human maupun JSON.
status: complete
created: "2026-11-02"
updated: "2026-11-02"
tags:
    - cli
    - list
    - show
related:
    - "1793614490028"
---
# WO 04 — Tampilkan Status Topic pada `list` dan `show`

## Konteks

Referensi SPEC: `../spec-status-topic-hasil-agregasi-file.md` §6.3 dan §10 AC-3.
`historic list` dan `historic show` memakai `repository.TopicView` (walk filesystem) yang
tidak memiliki field status topic, sehingga pengguna tidak dapat melihat status topic.

## Ruang lingkup

- Tambahkan field status topic pada `repository.TopicView` (nama usulan
  `computed_status`, `omitempty`) dan pada output human.
- Hitung nilai memakai `domain.AggregateStatus` dari frontmatter file terkelola yang
  memang sudah dibaca saat membangun view (tanpa I/O tambahan dan tanpa ketergantungan
  pada SQLite).
- Asset tidak ikut agregasi.
- `historic list` (human) menampilkan status topic sebagai kolom/penanda ringkas.
- `historic list --json` menyertakan `computed_status`.
- `historic show` (human) menampilkan status topic pada header/ringkasan topic.
- `historic show --json` menyertakan `computed_status`.
- Bila topic belum memiliki file terkelola → tampilkan kosong/`-` dan `omitempty` pada
  JSON.

## Langkah implementasi

1. Tambahkan field pada `TopicView` di `internal/repository/views.go`.
2. Kumpulkan status file terkelola di `readTopicView` dan set `ComputedStatus`.
3. Perbarui format human di `internal/cmd/list_show.go` agar status topic terlihat
   tanpa merusak tata letak lebar terminal.
4. Jaga stabilitas urutan dan bentuk JSON yang sudah ada; tambahkan field, jangan ubah
   field lama.

## Unit test

Perbarui `internal/cmd/list_show_test.go` dan `internal/repository/views_test.go`:

- `list` menampilkan status topic untuk topic dengan campuran status.
- `list --json` dan `show --json` menyertakan `computed_status` sesuai §5.1.
- Topic tanpa file terkelola tidak menampilkan status.
- Salinan open dan closed dihitung independen sesuai file masing-masing.

## Definisi selesai

- `historic list` dan `historic show` menampilkan status topic pada human dan `--json`.
- Nilai konsisten dengan aturan §5.1 dan identik dengan hasil `find --json`.
- `go test ./internal/cmd/... ./internal/repository/...` lulus.
