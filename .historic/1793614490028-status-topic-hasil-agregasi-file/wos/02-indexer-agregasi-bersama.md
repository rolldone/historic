---
title: WO 02 — Indexer Memakai Agregasi Bersama dan Memperbaiki Klasifikasi forwarded
description: Mengganti aggregateStatus lokal di indexer dengan fungsi domain bersama dan memperbaiki forwarded agar diperlakukan sebagai status terminal.
status: complete
created: "2026-11-02"
updated: "2026-11-02"
tags:
    - indexer
    - status
related:
    - "1793614490028"
---
# WO 02 — Indexer Memakai Agregasi Bersama dan Memperbaiki Klasifikasi `forwarded`

## Konteks

Referensi SPEC: `../spec-status-topic-hasil-agregasi-file.md` §6.1, §6.5, dan §10 AC-1, AC-6.
`internal/indexer/aggregateStatus` mengembalikan **status non-terminal pertama** menurut
urutan scan path. Akibatnya hasil tidak deterministik dan `forwarded` — yang
`StatusForwarded.IsClose() == true` — diperlakukan seolah status terbuka.

## Ruang lingkup

- Hapus `aggregateStatus` dan `allStatus` lokal di `internal/indexer/indexer.go`.
- Panggil `domain.AggregateStatus` saat menulis `topic.ComputedStatus`.
- Pastikan hanya status dari file **terkelola** (`historic_file`) yang dikirim ke fungsi
  agregasi; asset tidak boleh ikut.
- Nilai yang ditulis ke `topics.computed_status` adalah status domain konkret sesuai
  aturan §5.1 (bukan label sintetis).
- Operasi ini dipakai oleh `historic rebuild` dan seluruh jalur yang memanggil
  `indexer.Rebuild`.

## Langkah implementasi

1. Ganti pemanggilan `aggregateStatus(statuses)` di `scanTopic`/`buildTopicReadModel`
   dengan `domain.AggregateStatus(statuses)`.
2. Pastikan slice `statuses` hanya berisi status file terkelola (bukan asset); bila
   sebelumnya asset sempat masuk, perbaiki agar konsisten.
3. Hapus fungsi dan helper yang menjadi tidak terpakai.
4. Periksa test indexer yang mengasumsikan perilaku lama dan perbarui.

## Unit test

- Topic dengan `forwarded` + `complete` → `complete`.
- Topic dengan semua `forwarded` → `complete`.
- Topic dengan `blocked` di antara file tertutup → `blocked`.
- Topic dengan semua `cancelled` → `cancelled`.
- Topic tanpa file terkelola → `computed_status` kosong/NULL.
- Hasil rebuild identik antara dua kali rebuild berturut-turut (idempoten).

## Definisi selesai

- Tidak ada lagi fungsi agregasi duplikat di `internal/indexer`.
- `internal/indexer/indexer_test.go` (dan test terkait) diperbarui dan lulus.
- `go test ./internal/indexer/...` lulus.
