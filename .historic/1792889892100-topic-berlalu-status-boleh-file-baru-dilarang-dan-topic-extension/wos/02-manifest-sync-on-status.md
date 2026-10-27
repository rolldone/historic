---
title: WO 02 — Sinkronisasi _meta.yaml.files pada Perubahan Status
description: Memperbaiki historic status agar menyinkronkan _meta.yaml.files (status dan type) dalam satu operasi, tanpa mewajibkan historic rebuild manual.
status: complete
created: "2026-10-25"
updated: "2026-10-27"
tags:
    - lifecycle
    - status
    - manifest
related:
    - "1792889892100"
---
# WO 02 — Sinkronisasi `_meta.yaml.files` pada Perubahan Status

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §5.5 dan AC-11.
`internal/lifecycle/file_status.go` memperbarui frontmatter dan index, tetapi
`_meta.yaml.files[].status` serta `files[].type` baru sinkron setelah `historic rebuild`.

## Ruang lingkup

- Dalam `ChangeFileStatus`, sinkronkan manifest topic pemilik file:
  - status file target;
  - type file berdasarkan klasifikasi indexer (`file` → `spec`/`task`/`note` bila
    relevan).
- Berlaku pada topic open dan closed.
- Pertahankan `assets` dan field metadata lain (`id`, `title`, `created`, `tags`,
  `related`, `extends`).
- Operasi idempoten.
- Kegagalan menulis manifest harus dikembalikan sebagai error non-silent yang menjelaskan
  kegagalan parsial dan jalur pemulihan `historic rebuild`.
- Tidak mengubah resolusi path ambigu; tetap `ErrConflict`.

## Langkah implementasi

1. Petakan file ke folder topic dan path relatif manifest.
2. Baca metadata dengan helper `markdown` yang ada.
3. Perbarui entri manifest sasaran; gunakan klasifikasi type bersama dengan indexer,
   ekstrak helper bersama bila perlu dan hindari duplikasi aturan.
4. Tulis via `markdown.WriteTopicMetadataManifest`, pertahankan assets.
5. Rebuild index setelah sinkronisasi; pastikan kegagalan dilaporkan jelas.
6. Tambahkan test lifecycle/status untuk open, closed, idempotensi, assets dan error write.

## Kriteria penerimaan

- **AC-1** Setelah `historic status <path> draft`, manifest langsung berstatus `draft`.
- **AC-2** `files[].type` sesuai klasifikasi file yang sama dengan indexer.
- **AC-3** Assets dan field metadata lain tidak berubah.
- **AC-4** Perubahan berulang idempoten.
- **AC-5** Berfungsi pada salinan open dan closed.
- **AC-6** Kegagalan manifest menghasilkan error non-silent; index tetap konsisten dengan
  sumber Markdown.
- **AC-7** `go test ./...` hijau.

## Test plan

| Kasus | Ekspektasi |
| --- | --- |
| Ubah status file open | Manifest status langsung sama dengan frontmatter |
| Ubah status file closed | Manifest salinan closed langsung sama |
| Klasifikasi type | Type termanifest sama dengan indexer |
| Status yang sama dua kali | Tidak ada perubahan manifest kedua |
| Topic punya assets | Daftar assets identik sebelum/sesudah |
| Simulasi write error | Error parsial non-silent dan dapat dipulihkan |

## Risiko dan dependensi

Risiko sedang: klasifikasi type harus identik dengan indexer. Tidak ada dependensi WO;
dapat dikerjakan paralel dengan WO-01.
