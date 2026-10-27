---
title: WO 09 — Command historic forward dengan Validasi Strict
description: 'Menambahkan command historic forward untuk memindahkan task ke topic lanjutan: --to topic yang sudah ada atau --create topic baru, dengan validasi extends wajib.'
status: complete
created: "2026-10-28"
updated: "2026-10-27"
tags:
    - cli
    - forward
    - extension
related:
    - "1792889892100"
---
# WO 09 — Command `historic forward` dengan Validasi Strict

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §5.6, §6, §8, AC-11–AC-13.
Command ini menandai file sebagai `forwarded` sekaligus memastikan topic tujuan memang
kelanjutan topic asal.

## Ruang lingkup

### Sintaks

```sh
historic forward <path> --to <topic-id> [--json]
historic forward <path> --create "<title>" [--json]
```

### Perilaku

- `--to` dan `--create` **saling eksklusif**; keduanya wajib salah satu.
- `--to <topic-id>`:
  - topic tujuan harus ada dan `extends`-nya sama dengan topic asal file;
  - jika tidak memenuhi → error validasi, **tanpa perubahan filesystem**.
- `--create "<title>"`:
  - membuat topic baru dengan `extends` = topic asal (memakai jalur WO-03);
  - setelah topic berhasil dibuat, tandai file sebagai `forwarded`;
  - jika pembuatan topic gagal, status file **tidak berubah** (atomik).
- File target harus file terkelola di **topic open**. Forward pada salinan closed
  ditolak dengan arahan `historic open`.
- Hasil: `status: forwarded` + `forwarded_to: "<topic-id>"`, `updated` diperbarui,
  manifest tersinkron, index di-rebuild.
- Forward ulang idempoten: `forwarded_to` diganti ke tujuan baru yang valid.
- Tidak menghapus dan tidak menyalin isi file.

### JSON

```json
{"command":"forward","ok":true,"data":{"path":"...","forwarded_to":"...","created_topic":"..."},"error":null}
```

- `created_topic` hanya terisi bila memakai `--create`.

## Langkah implementasi

1. Buat `internal/cmd/forward.go`; daftarkan di root command.
2. Tambahkan helper resolusi topic asal dari path file (pakai yang ada di lifecycle).
3. Implementasikan jalur `--to`: validasi keberadaan tujuan + kecocokan `extends`.
4. Implementasikan jalur `--create`: buat topic dengan `extends`, lalu forward; pastikan
   rollback status jika penulisan gagal.
5. Tambahkan test command: sukses, target tidak terkait, dua opsi bersamaan, closed file,
   forward ulang, dan `--create` gagal.

## Kriteria penerimaan

- **AC-1** `forward --to <child-id>` sukses dan mengisi `forwarded_to` (AC-11 SPEC).
- **AC-2** Target yang tidak meng-extend topic asal ditolak tanpa perubahan (AC-12).
- **AC-3** `--create` membuat topic ber-`extends` lalu menandai file (AC-13).
- **AC-4** `--to` + `--create` bersamaan → error validasi.
- **AC-5** Forward dari salinan closed ditolak dengan arahan `historic open`.
- **AC-6** Tidak ada penghapusan/penyalinan konten file.
- **AC-7** JSON envelope konsisten dan `error` kosong saat sukses.

## Test plan

| Kasus | Ekspektasi |
| --- | --- |
| Forward ke child valid | Sukses, `forwarded_to` terisi |
| Forward ke topic tak terkait | Error validasi, filesystem tidak berubah |
| Forward dengan `--create` | Topic baru `extends` asal, file forwarded |
| Dua opsi bersamaan | Error validasi |
| Tanpa opsi | Error validasi |
| File closed | Ditolak, arahkan ke `open` |
| Forward ulang | `forwarded_to` diperbarui |
| Create gagal | Status file tidak berubah |

## Risiko dan dependensi

Risiko sedang: atomisitas jalur `--create` harus dijaga. **Bergantung pada WO-03
(`--extends`) dan WO-08 (status + field).**
