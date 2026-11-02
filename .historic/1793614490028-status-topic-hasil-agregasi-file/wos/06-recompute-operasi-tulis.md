---
title: WO 06 — Recompute Status Topic pada Semua Operasi Tulis
description: Memastikan status topic diselaraskan pada status, add, forward, delete, save, dan rebuild tanpa mewajibkan rebuild manual.
status: complete
created: "2026-11-02"
updated: "2026-11-02"
tags:
    - lifecycle
    - status
related:
    - "1793614490028"
---
# WO 06 — Recompute Status Topic pada Semua Operasi Tulis

## Konteks

Referensi SPEC: `../spec-status-topic-hasil-agregasi-file.md` §6.5 dan §10 AC-7.
Agregasi status topic hanya boleh dihitung ulang lewat satu jalur. Semua operasi tulis
yang mengubah himpunan file terkelola atau statusnya harus menyelaraskan
`topics.computed_status` tanpa memerlukan `historic rebuild` manual.

## Ruang lingkup

Pastikan operasi berikut menyelaraskan `computed_status` topic terdampak:

| Operasi | Perubahan |
| --- | --- |
| `historic status <path> <status>` | status file target |
| `historic add` | file terkelola baru bertambah |
| `historic forward` | status file menjadi `forwarded` |
| `historic delete` | file terkelola hilang |
| `historic save <id>`, `save --all` | mirror open → closed (salinan closed dihitung sendiri) |
| `historic rebuild` | penyelarasan global |

Ketentuan:

- Operasi pada topic **closed** juga menyelaraskan salinan closed yang bersangkutan.
- Operasi pada topic **open** menyelaraskan workdir open; mirror closed hanya berubah
  saat `save`/`close`.
- Kegagalan penyelarasan index harus dilaporkan sebagai error parsial yang menyebut
  `historic rebuild` sebagai jalur pemulihan, bukan sukses senyap.
- Tidak ada perhitungan ulang status topic yang ditulis ke Markdown.

## Langkah implementasi

1. Identifikasi di setiap operasi apakah `indexer.Rebuild` (atau padanannya) sudah
   dipanggil. Bila sudah, pastikan ia memakai `domain.AggregateStatus` (WO 02) dan
   cakupannya memuat topic terdampak.
2. Bila ada operasi yang hanya menyentuh file tanpa menyegarkan read model topic,
   tambahkan penyelarasan minimal untuk topic tersebut.
3. Pastikan operasi `delete` menghitung ulang topic yang ditinggalkan (bukan hanya
   menghapus record).
4. Hindari menulis `computed_status` ke `_meta.yaml`.

## Unit test

- Setelah `historic status <path> complete` pada file terakhir yang tersisa, topic
  menjadi `complete` tanpa `rebuild`.
- Setelah `historic add`, topic tanpa file sebelumnya kini memiliki `computed_status`.
- Setelah `historic forward`, topic dengan seluruh file `forwarded` menjadi `complete`.
- Setelah `historic delete` file khusus, status topic dihitung ulang.
- Setelah `historic save <id>`, salinan closed memiliki `computed_status` sendiri yang
  sesuai isinya.
- Hasil akhir identik dengan menjalankan `historic rebuild` setelahnya (idempoten).

## Definisi selesai

- Seluruh operasi pada tabel menyelaraskan `computed_status` topic terdampak.
- `go test ./internal/lifecycle/... ./internal/cmd/...` lulus.
