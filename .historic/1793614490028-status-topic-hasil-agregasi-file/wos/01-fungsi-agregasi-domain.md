---
title: WO 01 — Fungsi Agregasi Status Topic Tunggal
description: Menetapkan satu fungsi agregasi status topic di domain yang deterministik tanpa aritmetika, beserta unit test tabel contoh.
status: complete
created: "2026-11-02"
updated: "2026-11-02"
tags:
    - domain
    - aggregation
related:
    - "1793614490028"
---
# WO 01 — Fungsi Agregasi Status Topic Tunggal

## Konteks

Referensi SPEC: `../spec-status-topic-hasil-agregasi-file.md` §5 dan §10 AC-1, AC-2.
Saat ini aturan agregasi tersebar dan saling bertentangan: `internal/indexer/indexer.go`
(`aggregateStatus`, urutan scan path) dan `internal/query/query.go` (perhitungan ulang di
SQL dengan label sintetis `active`). WO ini menetapkan satu sumber aturan.

## Ruang lingkup

Tambahkan fungsi agregasi tunggal pada paket `internal/domain` (nama usulan
`AggregateStatus`). Fungsi harus:

- Menerima daftar status file **terkelola** (`historic_file` saja; asset diabaikan oleh
  pemanggil).
- Mengembalikan `domain.Status` atau nilai kosong bila input kosong.
- Menerapkan algoritma Bagian §5.1 tanpa aritmetika:

```text
Langkah 0: input kosong                         -> "" (kosong)
Langkah 1: ada file terbuka                     -> status terbuka berprioritas tertinggi
           blocked > review > progress > planned > pending > draft > create
Langkah 2: semua tertutup:
           ada failed                            -> failed
           semua cancelled                       -> cancelled
           selain itu                            -> complete
```

- Bersifat deterministik dan idempoten (tidak bergantung urutan input).

## Langkah implementasi

1. Definisikan urutan prioritas status terbuka sebagai satu slice eksplisit di `domain`,
   sehingga urutan penambahan status baru mudah diaudit.
2. Implementasikan `AggregateStatus(statuses []Status) Status` mengikuti §5.1.
3. Tambahkan helper internal untuk pemeriksaan "semua sama" bila diperlukan.
4. Jangan mengubah `validStatuses`, `openStatuses`, atau `closeStatuses`.
5. Pastikan fungsi tidak melakukan I/O dan tidak bergantung pada topic lain (tanpa
   pengetahuan soal topic graph).

## Unit test

Tambahkan test yang memverifikasi **seluruh** tabel contoh Bagian §5.2, termasuk:

- topic tanpa file → kosong;
- hanya asset → kosong (di level pemanggil, input kosong);
- `complete` + `cancelled` → `complete`;
- semua `cancelled` → `cancelled`;
- `cancelled` + `failed` → `failed`;
- `forwarded` diperlakukan setara `complete`;
- `archived` diperlakukan setara `complete`;
- prioritas status terbuka (`blocked` mengalahkan `review`, `review` mengalahkan
  `progress`, dst.).

Tambahkan test properti/permutasi: hasil `AggregateStatus` tidak berubah untuk permutasi
input pada beberapa himpunan status.

## Definisi selesai

- Fungsi agregasi tunggal tersedia di `domain` dan tercakup unit test tabel §5.2 penuh.
- Tidak ada duplikasi aturan agregasi lain yang baru diperkenalkan di WO ini.
- `go test ./internal/domain/...` lulus.
