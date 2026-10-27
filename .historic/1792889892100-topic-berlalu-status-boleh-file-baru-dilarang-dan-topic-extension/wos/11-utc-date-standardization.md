---
title: WO 11 — Standarkan Tanggal created/updated ke UTC
description: Menghilangkan inkonsistensi timezone antara created (waktu lokal) dan updated (UTC) yang dapat menghasilkan updated lebih awal dari created.
status: complete
created: "2026-10-28"
updated: "2026-10-28"
tags:
    - lifecycle
    - timezone
    - date
related:
    - "1792889892100"
---
# WO 11 — Standarkan Tanggal `created`/`updated` ke UTC

## Konteks

Temuan WO-07 (quality gate) saat verifikasi manual:

- `created` pada `historic create`/`historic add` dihitung dari waktu **lokal**
  (`internal/repository/topics.go`: `dateToday()` → `timeNow()` tanpa UTC).
- `updated` pada perubahan status dihitung dari waktu **UTC**
  (`internal/lifecycle/file_status.go`: `time.Now().UTC().Format("2006-01-02")`).
- `internal/lifecycle/sync_meta.go` (`recoverTopicMetadata`) sudah UTC.

Akibatnya, pada zona waktu di depan UTC (mis. WIB, UTC+7), file yang dibuat dan diubah
pada hari yang sama bisa memiliki `updated` **lebih awal** dari `created`, misalnya
`created: "2026-10-28"` dan `updated: "2026-10-27"`. Ini membingungkan pembaca dan
mengganggu perbandingan "mana yang lebih baru" pada dokumen dual-copy.

## Ruang lingkup

- Tetapkan **UTC** sebagai satu-satunya basis tanggal `created` dan `updated` untuk
  seluruh penulisan Historic.
- Perbaiki `dateToday()` di `internal/repository/topics.go` agar memakai UTC.
- Audit seluruh pemakaian `time.Now()` yang menghasilkan tanggal/`updated` dan
  samakan ke UTC: minimal `internal/repository/topics.go`,
  `internal/lifecycle/file_status.go`, `internal/lifecycle/sync_meta.go`.
- Pastikan `updated` selalu `>= created` untuk penulisan baru; pertimbangkan guard yang
  tidak menurunkan `updated` di bawah `created` bila sumbernya berbeda.
- Tambahkan test eksplisit yang mengunci UTC (mis. dengan clock injection bila tersedia),
  termasuk kasus `updated >= created`.

## Bukan ruang lingkup

- Tidak ada migrasi retroaktif data lama yang sudah tertulis; dokumen lama dibiarkan.
- Tidak ada perubahan format (`YYYY-MM-DD`) atau perubahan `mtime`.
- Tidak menyentuh logika status/extension/forward selain tanggal.
- Tidak mengubah `PRD.md`.

## Langkah implementasi

1. Ubah basis tanggal penulis menjadi UTC (helper bersama bila perlu, hindari duplikasi).
2. Audit dan perbarui semua call site yang menulis `created`/`updated`.
3. Tambahkan test: `created` dan `updated` UTC, urutan `updated >= created`, dan
   konsistensi dengan `sync_meta` recovery.
4. Jalankan `go test ./...`.

## Kriteria penerimaan

- **AC-1** `created` dan `updated` memakai UTC pada semua jalur penulis.
- **AC-2** Dokumen baru tidak pernah memiliki `updated` lebih awal dari `created`.
- **AC-3** Test mengunci perilaku UTC dan lulus pada zona waktu non-UTC.
- **AC-4** Tidak ada perubahan format tanggal atau perilaku lain.
- **AC-5** `go test ./...` lulus.

## Test plan

| Kasus | Ekspektasi |
| --- | --- |
| Create topic di zona non-UTC | `created` = tanggal UTC |
| Add entri di zona non-UTC | `created` = tanggal UTC |
| Status change | `updated` = tanggal UTC, `updated >= created` |
| Recovery metadata | `created`/`updated` UTC dan konsisten |
| Dokumen legacy | Tidak diubah |

## Risiko dan dependensi

Risiko rendah. Bergantung pada perilaku tanggal WO-02 (sinkronisasi manifest) dan
WO-08 (status `forwarded`) karena keduanya ikut menulis `updated`.

## Hasil pelaksanaan (2026-10-28)

- `internal/repository/topics.go`: `dateToday()` kini `timeNow().UTC().Format("2006-01-02")`
  — satu-satunya penulis `created` (dipakai `CreateTopic` dan `AddEntry`).
- `internal/lifecycle/file_status.go`: ditambah `fileStatusNow` (injectable) dan helper
  `laterDate(created, updated)`; `updated` = `max(created, UTC today)` sehingga tidak
  pernah mundur melewati `created` pada dokumen legacy.
- `internal/lifecycle/sync_meta.go`: `recoverTopicMetadata` memakai `laterDate` yang sama
  bila `updated` tidak kosong.
- Audit penulis tanggal: hanya `dateToday()` (created) dan `file_status.go` (updated)
  yang menulis tanggal; `sync_meta.go` sudah UTC sebelumnya. Tidak ada penulis lain
  (mirror/close/save/upgrade mempertahankan bytes).

Test baru:
- `internal/repository/utc_date_test.go` — `created` UTC pada `CreateTopic` dan `AddEntry`
  dengan clock UTC+7.
- `internal/lifecycle/utc_date_test.go` — `updated` UTC, guard `updated >= created`
  (clock lebih awal dari `created` legacy), dan guard pada recovery metadata.

Verifikasi: `gofmt` bersih, `go vet ./...` bersih, `go test ./...` lulus. Smoke test
zona `Pacific/Kiritimati` (UTC+14, tanggal lokal 2026-10-28) menghasilkan
`created: "2026-10-27"` dan `updated: "2026-10-27"` (UTC), sesuai harapan.
