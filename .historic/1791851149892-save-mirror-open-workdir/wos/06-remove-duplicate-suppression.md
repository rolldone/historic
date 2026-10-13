---
title: 06 Remove Duplicate Suppression
description: Lepas dedup open/closed pada layer pencarian dan tampilkan updated_at/mtime agar kedua versi terlihat.
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
    - search
    - find
    - storage
related:
    - ../spec-save-mirrors-open-workdir.md
    - ./02-save-mirrors-all-open-topics.md
    - ./04-updated-at-divergence.md
---
# WO 06 — Remove Duplicate Suppression

## Tujuan

Melepas penekanan duplikat pada layer pencarian sehingga satu TopicID yang punya
salinan open dan closed tampil sebagai dua hasil terpisah.

## Kondisi saat ini

Pada `internal/search/search.go` terdapat predikat:

```sql
(t.storage = 'open' OR NOT EXISTS (
   SELECT 1 FROM topics AS open_topic
   WHERE open_topic.id = t.id AND open_topic.storage = 'open'))
```

Predikat ini menyembunyikan salinan closed bila open dengan ID sama ada. Setelah
`save` menjadi mirror, kondisi ini adalah keadaan normal dan tidak boleh disembunyikan.

## Scope

- `internal/search/search.go` — hapus predikat dedup untuk mode default (tanpa scope eksplisit).
- Pertahankan makna `--open`/`--closed` sebagai filter scope eksplisit, bukan dedup.
- Pastikan hasil menyertakan `storage` dan `updated_at`/`mtime` untuk tiap versi.
- `internal/cmd/find.go` — tampilkan penanda `[OPEN]`/`[CLOSED]` dan `updated_at` pada
  output human-readable.
- Urutan hasil harus deterministik; bila dua versi punya ID sama, urutkan konsisten
  (mis. open sebelum closed) dan tetap memakai tie-breaker path.

## Aturan wajib

- Tidak mengubah schema FTS5 maupun SQLite.
- Tidak mengubah `--open`/`--closed` menjadi kombinasi yang diizinkan secara bersamaan.
- Pagination tetap dihitung setelah dedup dilepas; `has_more`/`next_page` tetap akurat.
- `find --json` tetap memakai envelope `{command, ok, data, error}` dengan `items` dan `pagination`.

## Acceptance criteria

- Setelah `save`, `historic find "<judul>"` menampilkan dua baris: satu `[OPEN]`, satu `[CLOSED]`.
- `find --open` hanya menampilkan versi open; `find --closed` hanya versi closed.
- Setiap item JSON memuat `storage` dan `updated_at` (serta `mtime` untuk file).
- Topic tanpa salinan closed tetap tampil satu kali.

## Status implementasi (2026-10-13) — selesai

Blocker awal (indexer hanya mengindeks satu versi per TopicID karena `topics.id TEXT
PRIMARY KEY`) telah diperbaiki bersama WO 08/U0:

- `internal/schema/schema.go` — PK `topics` menjadi `(id, storage)`, PK `files` menjadi
  `(id, storage)` dengan FK `(topic_id, storage)`; `path` UNIQUE tetap.
- `internal/indexer/indexer.go` — `scan()` mengindeks semua versi (open urut sebelum
  closed); kolom `storage` ditambahkan ke read model file dan ke `historic_fts`
  (`UNINDEXED`) agar join tetap presisi.
- `internal/search/search.go` — join FTS memakai `f.storage/t.storage = historic_fts.storage`;
  tie-breaker `open` > `closed` ditambahkan ke `resultOrder`/`recentOrder`.
- `internal/query/query.go` — join file↔topic memakai `(topic_id, storage)`;
  `FileResult` memuat `storage`; order memakai storage rank.
- `historic list --closed` kini menampilkan salinan `.database` (termasuk mirror).
- Index schema version 4 → 5; index lama otomatis diwajibkan `rebuild`.

Verifikasi di workspace nyata (`umah_loka_stage`): `find ""` → 10 `[OPEN]` + 10 `[CLOSED]`,
`find --open` → 10, `find --closed` → 11, `list --closed` → 11, `show` → dua blok.
