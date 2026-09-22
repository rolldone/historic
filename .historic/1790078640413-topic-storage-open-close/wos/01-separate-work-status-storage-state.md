---
id: 00003
title: WO 01 Separate Work Status and Storage State
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [lifecycle, storage, open, close, find, breaking-change]
related: []
---

# WO 01 — Separate Work Status and Storage State

## Tujuan

Memisahkan status pekerjaan topic dari status penyimpanan topic. `complete`, `progress`, `blocked`, dan status pekerjaan lain tidak lagi menentukan apakah topic berada di workdir atau database.

## Model

### Work status

Status pekerjaan tetap menggunakan nilai existing:

```text
create, pending, progress, review, blocked, complete, failed, cancelled
```

### Storage state

Storage state ditentukan oleh lokasi filesystem:

```text
open   = .historic/<id>-<slug>/
closed = .historic/.database/<id>-<slug>/
```

Keduanya independen dan dapat dikombinasikan:

- `progress` + `open`
- `progress` + `closed`
- `complete` + `open`
- `complete` + `closed`
- `blocked` + `closed`
- `failed` + `open`

## Command baru

```text
historic close <id> [--json]
historic open <id> [--json]
```

### `historic close`

- Memindahkan topic open dari `.historic/` ke `.historic/.database/`.
- Dapat dilakukan dari work status apa pun.
- Tidak mengubah frontmatter work status.
- Tidak mengubah isi file.
- Tidak menghapus topic.
- Hasil topic menjadi storage `closed`.
- Operasi harus atomic atau memiliki recovery yang aman.

### `historic open`

- Memindahkan topic closed dari `.historic/.database/` ke `.historic/`.
- Mempertahankan work status dan seluruh isi topic.
- Tidak membuat topic duplikat.
- Menolak conflict jika destination sudah ada.
- Hasil topic menjadi storage `open`.
- Operasi harus atomic atau memiliki recovery yang aman.

## Perubahan `complete`

`historic complete <id>` hanya mengubah work status menjadi `complete` dan tidak memindahkan topic ke database.

User bebas menjalankan atau tidak menjalankan `close` setelah `complete`:

```text
complete + open   = selesai, tetap berada di workdir
complete + closed = selesai, disimpan di database
```

Perilaku archive otomatis existing dari `complete` harus dihapus atau diubah sesuai model ini.

## Perubahan `find`

`historic find` secara default mencari topic open dan closed.

Flag lama dihapus karena project masih baru:

```text
--active
--archived
```

Flag baru:

```text
--open
--closed
```

Perilaku:

- tanpa filter: cari open dan closed;
- `--open`: hanya topic di workdir;
- `--closed`: hanya topic di database;
- `--open` dan `--closed` bersamaan ditolak karena ambigu.

Setiap hasil wajib memperlihatkan:

- `storage`: `open` atau `closed`;
- `status`: work status;
- path aktual topic.

Human output minimal menampilkan `[OPEN]` atau `[CLOSED]`. JSON menambahkan field `storage` tanpa menghilangkan `status` dan `path`.

## Perubahan command existing

- `list` default menampilkan topic open.
- `list --closed` menampilkan topic closed.
- Jika `list --archived` tetap dipertahankan, harus dihapus atau diubah menjadi `--closed` sesuai keputusan final implementasi; flag lama tidak menjadi kontrak baru.
- `show <id>` harus dapat membaca topic open maupun closed atau membutuhkan opsi scope yang eksplisit, tetapi tidak boleh ambigu jika ID muncul di kedua lokasi.
- `import` dapat dipertahankan sebagai command internal/compatibility atau diarahkan ke semantik `open` setelah implementasi ditinjau.
- Rebuild dan index harus menyimpan storage state berdasarkan root/path.

## Error dan keamanan

- `close` menolak topic yang tidak ditemukan, sudah closed, symlink, atau destination conflict.
- `open` menolak topic yang tidak ditemukan, sudah open, symlink, atau destination conflict.
- ID yang sama di open dan closed harus menghasilkan conflict yang jelas.
- Kegagalan pemindahan tidak boleh menghapus sumber atau meninggalkan topic partial.
- Archive/internal Git tetap lokal dan tidak memiliki remote.
- Markdown tetap source of truth.

## Acceptance Criteria

- Topic dengan status apa pun dapat di-close.
- Topic dengan status apa pun dapat di-open kembali.
- `complete` tidak otomatis close topic.
- `complete` lalu `close` menghasilkan `status: complete` dan `storage: closed`.
- `complete` tanpa close menghasilkan `status: complete` dan `storage: open`.
- `close` tanpa complete mempertahankan work status sebelumnya.
- `open` mengembalikan topic tanpa mengubah isi atau status.
- `find` default menemukan open dan closed.
- `find --open` hanya menemukan open.
- `find --closed` hanya menemukan closed.
- `find` menampilkan storage state pada human dan JSON output.
- `--active` dan `--archived` tidak lagi menjadi flag yang didukung.
- Open/close conflict dan recovery memiliki test.
- Rebuild/index membedakan storage state secara konsisten.
- `PRD.md` tidak diubah.
- Unit, integration, dan isolated CLI smoke test tersedia dan lulus.

## Batasan

- Tidak mengubah isi file topic saat close/open.
- Tidak menambahkan remote sync.
- Tidak menghapus data archived.
- Tidak melakukan push remote.

## Status

Planned. Implementasi belum dimulai.
