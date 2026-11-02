---
title: WO 05 — Filter find --status Mencakup Status Topic Agregat
description: Membuat find --status mencocokkan file dan computed_status topic serta menolak status sintetis active yang tidak dikenal.
status: complete
created: "2026-11-02"
updated: "2026-11-02"
tags:
    - search
    - filter
related:
    - "1793614490028"
---
# WO 05 — Filter `find --status` Mencakup Status Topic Agregat

## Konteks

Referensi SPEC: `../spec-status-topic-hasil-agregasi-file.md` §6.4 dan §10 AC-4.
`internal/search/search.go` sudah memiliki klausa filter yang memisahkan
`historic_fts.entity_type = 'topic'` (memakai `t.computed_status`) dan `'file'` (memakai
`f.status`). Setelah WO 02 dan WO 03, `computed_status` memakai vocabulary status domain.
Yang perlu dipastikan: perilaku filter konsisten, `active` ditolak, dan hasil tidak
ambigu.

## Ruang lingkup

- `find --status <s>` mengembalikan:
  - file dengan `files.status = <s>`, **atau**
  - topic dengan `topics.computed_status = <s>`.
- `find --status-not <s>` mengecualikan keduanya.
- Input status divalidasi memakai `domain.ParseStatus`/`validStatuses` yang berlaku.
  Karena `active` bukan status domain lagi, `--status active` **ditolak** sebagai status
  tidak dikenal, dengan pesan validasi yang menyebut daftar status yang sah.
- Pesan validasi (`internal/search/search.go` `validStatusHint`) tetap akurat dan
  memuat seluruh status domain.
- Setiap hasil tetap mengekspos `status`, `storage`, dan `path` secara independen
  (perilaku lama dipertahankan).

## Langkah implementasi

1. Audit seluruh klausa `computed_status`/`f.status` pada `internal/search/search.go`.
2. Pastikan daftar status valid berasal dari satu sumber (`domain`), bukan literal
   terpisah, agar `active` otomatis tidak diterima.
3. Pastikan `--status` dan `--status-not` tidak tumpang tindih dan penolakan status tak
   dikenal tetap berjalan sebelum eksekusi query.
4. Perbarui test yang sekarang memakai `active`.

## Unit test

- `find --status complete` mengembalikan file `complete` dan topic dengan
  `computed_status = complete`.
- `find --status forwarded` mengembalikan file `forwarded` (topic mungkin `complete`).
- `find --status blocked` mengembalikan topic agregat `blocked`.
- `find --status active` gagal dengan `ErrInvalidStatus` dan pesan menyebut status sah.
- `find --status-not complete` mengecualikan file `complete` dan topic `complete`.
- `find --status complete --status-not cancelled` tetap divalidasi sebagai sekarang.

## Definisi selesai

- Filter status mencocokkan file dan topic sesuai §6.4.
- `active` ditolak sebagai status tidak dikenal.
- `go test ./internal/search/...` dan test CLI terkait lulus.
