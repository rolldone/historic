---
title: WO 01 — Kontrak Status review di Paket Domain
description: Mengunci kontrak status file review (valid, open, bukan close) beserta test eksplisit di internal/domain. Tidak ada sentinel error baru.
status: complete
created: "2026-10-25"
updated: "2026-10-27"
tags:
    - domain
    - status
related:
    - "1792889892100"
---
# WO 01 — Kontrak Status `review` di Paket Domain

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §4, §5.4, §7.
`internal/domain/model.go` sudah memiliki `StatusReview` di `validStatuses` dan
`openStatuses`. WO ini memastikan kontrak tersebut terkunci dengan test eksplisit.

> **Revisi 2026-10-28:** penambahan sentinel `ErrTopicReadOnly` **dibatalkan** karena
> aturan larangan file baru pada topic berlalu dicabut.

## Ruang lingkup

1. Tambahkan test eksplisit: `StatusReview.IsValid()` dan `.IsOpen()` true, `.IsClose()`
   false; `ParseStatus("review")` sukses dan `ParseStatus("Review")` gagal.
2. Bila ditemukan ketidaksesuaian, perbaiki daftar status di `internal/domain/model.go`.
3. Tidak menambah sentinel error baru.
4. Tidak menyentuh command, repository, indexer, TUI, atau field frontmatter.

## Langkah implementasi

1. Tinjau `validStatuses`, `openStatuses`, dan `closeStatuses`.
2. Tambahkan test pada `internal/domain/model_test.go`.
3. Jalankan `go test ./internal/domain/...`.

## Kriteria penerimaan

- **AC-1** `review` valid/open dan bukan status close.
- **AC-2** `ParseStatus("review")` sukses; `ParseStatus("Review")` gagal.
- **AC-3** Test domain hijau.
- **AC-4** Tidak ada perubahan perilaku lain pada paket domain.

## Test plan

| Kasus | Input | Ekspektasi |
| --- | --- | --- |
| Valid status review | `StatusReview.IsValid()` | `true` |
| Open status review | `StatusReview.IsOpen()` | `true` |
| Close status review | `StatusReview.IsClose()` | `false` |
| Case sensitivity | `ParseStatus("Review")` | `ErrInvalidStatus` |

## Risiko dan dependensi

Risiko rendah; perubahan bersifat test/verifikasi. Tidak ada dependensi WO lain.
