---
title: WO 08 — Quality Gate dan Regresi Agregasi Status Topic
description: Menjalankan gate kualitas menyeluruh untuk agregasi status topic mencakup determinisme, idempotensi rebuild, konsistensi lintas perintah, dan doctor.
status: complete
created: "2026-11-02"
updated: "2026-11-02"
tags:
    - quality-gate
    - regression
related:
    - "1793614490028"
---
# WO 08 — Quality Gate dan Regresi Agregasi Status Topic

## Konteks

Referensi SPEC: `../spec-status-topic-hasil-agregasi-file.md` §10 (seluruh AC).
WO ini adalah gerbang akhir sebelum topic dianggap selesai: memastikan seluruh aturan
agregasi berlaku konsisten pada semua jalur baca dan tulis, serta tidak menimbulkan
inkonsistensi baru pada `doctor`.

## Ruang lingkup

1. Bangun ulang binary global (`./build-global.sh`) lalu verifikasi versi dan rebuild:

```sh
./build-global.sh
historic version --json
historic rebuild --json
historic doctor --json
```

2. Verifikasi konsistensi lintas perintah pada workspace contoh. Untuk setiap topic,
   nilai status dari ketiga jalur harus identik:

```sh
historic list --json
historic show <topic-id> --json
historic find "" --json
```

3. Verifikasi seluruh baris tabel §5.2 lewat test otomatis (bukan hanya manual), idealnya
   sebagai test integrasi yang membangun workspace sementara berisi topic untuk tiap baris
   tabel.

4. Verifikasi perubahan perilaku filter:

```sh
historic find --status complete   # mencakup topic agregat complete
historic find --status active     # harus gagal: status tidak dikenal
```

5. Verifikasi determinisme:

- Rebuild dua kali berturut-turut menghasilkan `computed_status` identik.
- Permutasi urutan file di dalam topic tidak mengubah `computed_status`.

6. Verifikasi bucket counter: topic dengan seluruh file `forwarded` memiliki
   `active_files = 0` dan `computed_status = complete`.

7. Jalankan seluruh test suite: `go test ./...` dan `go vet ./...`.

## Regression checklist

- [ ] Satu fungsi agregasi dipakai indexer, `list`/`show`, dan `search` (AC-1).
- [ ] Tabel §5.2 terpenuhi seluruhnya (AC-2).
- [ ] Status topic tampil pada `list` dan `show`, human dan `--json` (AC-3).
- [ ] `find --status` mencakup file dan topic; `active` ditolak (AC-4).
- [ ] `forwarded` tidak terhitung sebagai file aktif (AC-5).
- [ ] `rebuild` menghasilkan `computed_status` sesuai aturan (AC-6).
- [ ] Operasi tulis menyelaraskan status tanpa `rebuild` manual (AC-7).
- [ ] `doctor --json` tidak melaporkan inkonsistensi baru (AC-8).
- [ ] Tidak ada penulisan status topic ke `_meta.yaml` atau frontmatter.
- [ ] Tidak ada sisa referensi label `active` pada kode/dokumentasi yang relevan.
- [ ] Git internal tidak menerima commit/push otomatis.

## Definisi selesai

- Seluruh checklist tercentang dengan bukti keluaran perintah atau nama test.
- `go test ./...` dan `go vet ./...` lulus.
- Temuan/penyimpangan dicatat pada catatan hasil WO, termasuk pertanyaan terbuka §9 SPEC
  yang belum diputuskan.
