---
id: 00001
title: WO 25 Batch Sync Topic Metadata
status: complete
created: 2026-09-19
updated: 2026-09-19
tags: [maintenance, metadata, sync-meta, batch]
related: [./23-sync-meta.md, ../spec.md]
---

# WO 25 — Batch `historic sync-meta`

## Tujuan

Memperluas command `historic sync-meta` agar dapat menyelaraskan metadata seluruh topic aktif di workdir ketika command dipanggil tanpa target.

## Latar Belakang

Saat ini `sync-meta` memerlukan target berupa ID atau path topic. User membutuhkan mode batch untuk menyelaraskan semua topic yang berada di working directory tanpa menjalankan command satu per satu.

## Kontrak Command

```text
historic sync-meta [<id>|<topic-path>] [--json]
```

Perilaku:

- Dengan `<id>`, hanya topic aktif yang sesuai yang diproses.
- Dengan `<topic-path>`, hanya topic pada path tersebut yang diproses.
- Tanpa target, semua topic aktif di `.historic/` diproses.
- Topic di `.historic/.database/` tidak ikut diproses karena merupakan archive.
- File di luar folder topic tidak ikut diproses.
- Aturan klasifikasi managed Markdown, `Files`, `Assets`, `_meta.md`, atomic write, dan rebuild tetap mengikuti WO 23.
- Tanpa topic aktif, command berhasil dengan hasil kosong.

## Output Batch

Output manusia harus menampilkan ringkasan per topic, minimal:

- path atau ID topic;
- jumlah managed files;
- jumlah assets;
- apakah metadata berubah atau tetap unchanged;
- error bila topic gagal diproses.

`--json` tetap menggunakan envelope standar:

```json
{
  "command": "sync-meta",
  "ok": true,
  "data": {
    "topics": [],
    "total": 0,
    "updated": 0,
    "errors": 0
  },
  "error": null
}
```

Schema final boleh menambahkan field, tetapi harus tetap stabil dan membedakan hasil setiap topic.

## Error Policy

Mode batch menggunakan **continue-on-error**:

- Jika satu topic gagal, topic lain tetap diproses.
- Semua error dikumpulkan pada hasil batch.
- Exit code non-zero jika terdapat satu atau lebih topic yang gagal.
- Error pada satu topic tidak boleh membuat metadata topic lain menjadi partial.
- Kegagalan harus tetap mematuhi atomic write dan tidak meninggalkan temporary file.

## Tasks

- Izinkan command `sync-meta` dipanggil tanpa positional target.
- Enumerasi hanya topic aktif yang valid di workdir.
- Jangan memasukkan `.database`, `.index.sqlite`, atau file non-topic sebagai target.
- Reuse service sync-meta existing untuk setiap topic.
- Tentukan urutan pemrosesan secara deterministik berdasarkan path atau ID ascending.
- Tambahkan output human batch per topic dan ringkasan total.
- Tambahkan schema JSON batch dengan hasil per topic dan error teragregasi.
- Pertahankan perilaku command targeted dan kontrak JSON existing.
- Tambahkan unit test enumerasi topic aktif dan continue-on-error.
- Tambahkan integration test untuk zero, single, dan multiple active topics.
- Tambahkan test bahwa archived topics tidak diproses.
- Tambahkan test idempotensi pada mode batch.
- Dokumentasikan penggunaan `historic sync-meta` tanpa target.

## Acceptance Criteria

- `historic sync-meta` tanpa target memproses semua topic aktif di workdir.
- Target ID dan target path tetap hanya memproses satu topic.
- Topic archived tidak berubah dan tidak masuk hasil batch.
- `.index.sqlite`, `.database`, dan file non-topic tidak diproses.
- Semua topic diproses dalam urutan deterministik.
- Satu topic gagal tidak menghentikan pemrosesan topic berikutnya.
- Exit code non-zero jika terdapat error batch.
- JSON success/error tetap memiliki envelope standar.
- JSON batch membedakan hasil setiap topic dan error per topic.
- Batch pada workspace tanpa topic aktif berhasil dengan hasil kosong.
- Menjalankan batch dua kali tidak menggandakan link dan run kedua menunjukkan unchanged.
- Atomic write dan rebuild tetap berjalan per topic sesuai WO 23.
- Tidak ada perubahan pada topic atau file yang tidak menjadi target.
- Unit dan integration test tersedia serta lulus.
- Dokumentasi command diperbarui.

## Dependensi

- WO 23 — `historic sync-meta`.
- Struktur topic aktif dan archive pada workspace `.historic/`.
- JSON envelope command existing.

## Batasan

- Tidak mengubah klasifikasi `Files` dan `Assets`.
- Tidak menambahkan aksi lifecycle.
- Tidak memproses topic archived secara default.
- Tidak mengubah source of truth Markdown.
- Tidak mengubah `PRD.md`.
- Tidak melakukan push ke remote repository.

## Status

Complete. Batch `historic sync-meta` telah diimplementasikan, divalidasi, dan di-commit pada `7cead84`.
