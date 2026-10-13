---
title: 'SPEC: historic save mirrors open workdir'
description: Ubah save menjadi operasi mirror (salin konten ke .database) tanpa menutup topic dan tanpa git commit; commit dikeluarkan dari save dan close.
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
    - lifecycle
    - save
    - close
    - storage
    - snapshot
related:
    - "1790078640413"
    - "1790518993269"
---
# SPEC: `historic save` sebagai mirror open workdir

## Latar belakang

Saat ini `historic save` hanya menjalankan `git add --all` + `git commit` di dalam
`.historic/.database`, sehingga isi topic yang masih **open** (`.historic/<id>-<slug>/`)
tidak pernah ikut tersimpan. Konten baru masuk `.database` hanya lewat `historic close`.

Akibatnya pengguna harus menutup topic agar bisa menyimpan progress, padahal menutup
topic bukan hal yang diinginkan saat pekerjaan masih berjalan.

## Keputusan

`save` diubah menjadi operasi **mirror**: menyalin konten topic open ke `.database`
dengan mekanisme yang sama seperti `close`, tetapi **workdir open tetap dipertahankan**
dan topic tetap berstatus open.

```
save  = mirror ke .database, topic tetap open
close = mirror ke .database, lalu workdir open dihapus (topic closed)
```

`close` adalah `save` + penutupan. Keduanya memakai satu operasi mirror yang sama.

## Perubahan perilaku

1. `historic save` (tanpa argumen) mem-mirror **semua topic open** ke `.database`.
2. Workdir open **tidak dihapus** dan status storage tetap `open`.
3. Salinan di `.database` adalah snapshot titik-waktu (point-in-time); ia tidak ikut
   berubah ketika workdir open diedit lagi.
4. `save` **tidak** melakukan git commit.
5. `close` **tidak** melakukan git commit.
6. Flag `-m/--message` **dihapus** dari `save`; `save` tidak lagi menerima argumen apa pun.
7. Commit sepenuhnya diserahkan ke pengguna melalui passthrough `historic git commit`.

## Koeksistensi open dan closed

Setelah `save`, satu TopicID dapat memiliki dua salinan: open dan closed. Ini sah dan
sudah didukung resolver yang ada (`TopicLocation`): bila keduanya ada, salinan **open
bersifat authoritative**.

Pembedanya adalah waktu:

- `topics.updated_at` bersumber dari frontmatter `updated`
- `files.updated_at` bersumber dari frontmatter `updated`
- `files.mtime` bersumber dari mtime filesystem

Dengan pembeda berbasis `updated_at`/`mtime`, pengguna dapat mengenali salinan mana yang
terbaru. Tidak ada nilai `storage` baru; `storage` tetap `open` atau `closed`.

## Visibilitas kedua versi

Karena koeksistensi menjadi keadaan normal, **penekanan duplikat (dedup) dihapus**.
Sebelumnya `internal/search/search.go` menyembunyikan salinan `closed` bila ada salinan
`open` dengan ID sama, sehingga snapshot hasil `save` tidak terlihat.

Perilaku baru:

- `historic find` menampilkan **kedua versi** sebagai hasil terpisah, selalu (tanpa flag).
- `historic show <id>` menampilkan **kedua versi** dalam blok terpisah.
- `historic search` (TUI) menampilkan **kedua versi** sebagai entri terpisah.
- Penanda `[OPEN]`/`[CLOSED]` disertai `updated_at` (dan `mtime` untuk file) menjadi pembeda.
- `--open` dan `--closed` tetap berfungsi sebagai *filter scope*, bukan dedup.
- Pagination dihitung setelah dedup dilepas; `has_more`/`next_page` tetap akurat.

## Kriteria penerimaan

- `historic save` menyalin konten seluruh topic open ke `.database` tanpa menghapus workdir open.
- Setelah `save`, `historic list` tetap menampilkan topic tersebut sebagai open.
- Setelah `save`, isi `.database/<id>-<slug>/` identik dengan workdir open pada saat save.
- `historic save` dan `historic close` tidak membuat commit Git baru.
- `historic save --json` memakai envelope standar dan melaporkan hasil per topic.
- `historic save -m "..."` ditolak sebagai flag tidak dikenal.
- `historic close` tetap berperilaku seperti sebelumnya, kecuali tidak commit.
- `historic close all` tetap non-zero bila ada kegagalan parsial.
- Setelah `save`, `find`/`show`/TUI menampilkan dua versi (open dan closed) dengan penanda
  dan `updated_at` sebagai pembeda.
- `historic git commit -m "..."` tetap tersedia sebagai jalur commit eksplisit pengguna.

## Non-goals

- Tidak menambah subcommand atau state storage baru.
- `_meta.yaml` dan nilai `storage` (`open`/`closed`) tidak berubah. Read-model SQLite
  **diizinkan** berubah bila perlu untuk menampung dua versi: schema `topics`/`files`
  bersifat rebuildable dan dinomori ulang melalui index schema version 4 → 5,
  sedangkan FTS5 dipopulasi ulang penuh pada setiap rebuild.
- Tidak menambah remote atau push otomatis.
- Tidak mengubah `historic log`, `historic diff`, dan `historic restore` semantiknya.
