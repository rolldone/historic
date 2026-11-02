---
title: WO 03 — Query Membaca computed_status dan Memperbaiki Bucket Counter
description: Menghentikan perhitungan ulang agregasi di SQL, membaca computed_status dari index, dan memperbaiki active_files agar tidak menghitung forwarded.
status: complete
created: "2026-11-02"
updated: "2026-11-02"
tags:
    - query
    - read-model
related:
    - "1793614490028"
---
# WO 03 — Query Membaca `computed_status` dan Memperbaiki Bucket Counter

## Konteks

Referensi SPEC: `../spec-status-topic-hasil-agregasi-file.md` §6.1, §6.2, dan §10 AC-1, AC-5, AC-7.
`internal/query/query.go` (`QueryTopicsPage`) menghitung ulang status agregat di SQL dan
mengembalikan label sintetis `active`/`complete`/`cancelled`. Selain itu
`active_files` memakai `status NOT IN ('complete', 'cancelled')` sehingga `forwarded`
(close-like) terhitung sebagai pekerjaan aktif.

## Ruang lingkup

- `computed_status` pada hasil query **dibaca langsung** dari kolom
  `topics.computed_status` yang telah ditulis indexer. Tidak ada `CASE`/perhitungan
  ulang status agregat di SQL.
- Perbaiki bucket counter:
  - `active_files` = jumlah file terkelola dengan status **terbuka** (`IsOpen()`:
    `create`, `draft`, `pending`, `planned`, `progress`, `review`, `blocked`).
  - `resolved_files` = jumlah file terkelola dengan status **tertutup** (`IsClose()`).
  - Pertahankan `complete_files`; `cancelled_files` tetap; pastikan `forwarded` **tidak**
    masuk `active_files`.
- Hapus label sintetis `active` dari hasil `computed_status`.
- `WHERE`/`HAVING` filter topic yang memakai `computed_status` tetap berfungsi
  (`internal/query/query.go` bagian `topicPredicates`).

## Langkah implementasi

1. Ganti ekspresi `CASE WHEN ... THEN 'active' ... END AS computed_status` menjadi
   `t.computed_status`.
2. Definisikan daftar status terbuka/tertutup untuk klausa SQL dari sumber yang sama
   dengan `domain` (hindari duplikasi literal yang mudah lepas sinkron). Bila literal SQL
   tetap dipakai, tambahkan test yang gagal bila daftar status berubah.
3. Perbarui agregasi bucket pada CTE `grouped`.
4. Perbarui dokumentasi struktur JSON pada komentar paket bila menyebut `active`.

## Unit test

Perbarui `internal/query/query_test.go`:

- Ekspektasi JSON stabil (`TestAggregateQuerySmokeHasStableJSONAndExcludesAssets`) harus
  memakai vocabulary status domain, bukan `complete`/`active` sintetis, kecuali nilainya
  memang sama.
- Topic dengan hanya `forwarded` → `computed_status = complete`, `active_files = 0`.
- Topic `progress` + `forwarded` → `computed_status = progress`, `active_files = 1`,
  `resolved_files = 1`.
- Topic tanpa file terkelola → `computed_status` kosong, `total_files = 0`.
- Asset tidak dihitung pada `total_files`/bucket.

## Definisi selesai

- Tidak ada perhitungan ulang status agregat di `internal/query/query.go`.
- Bucket counter konsisten dengan `domain.IsOpen()`/`IsClose()`.
- `go test ./internal/query/...` lulus.
