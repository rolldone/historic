---
title: 02 Save Mirrors All Open Topics
description: Ubah `historic save` menjadi mirror semua topic open ke .database tanpa menutup topic.
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
    - lifecycle
    - save
    - cli
related:
    - ../spec-save-mirrors-open-workdir.md
    - ./01-extract-mirror-operation.md
---
# WO 02 — Save Mirrors All Open Topics

## Tujuan

Mengubah `historic save` dari commit-only menjadi operasi mirror seluruh topic open,
tanpa menghapus workdir open dan tanpa mengubah status storage menjadi closed.

## Scope

- `internal/cmd/save.go` — ganti body command agar memanggil operasi mirror untuk semua topic open.
- Enumerasi target memakai mekanisme yang sama dengan `close all`
  (`openTopicTargets`), deterministik berdasarkan TopicID lalu path.
- Command tidak lagi menerima argumen.
- `--json` tetap memakai envelope standar dengan hasil per topic.

## Aturan wajib

- Workdir open tidak dihapus; `historic list` tetap menampilkan topic sebagai `open`.
- Topic yang sudah punya salinan `.database` diperbarui agar identik dengan workdir open.
- Kegagalan pada satu topic tidak menghentikan topic lain; exit code non-zero bila ada kegagalan.
- Hanya folder topic open tingkat pertama yang valid menjadi target; metadata, index,
  staging, dan `_meta.yaml` tidak dianggap topic.
- Tidak memanggil Git dan tidak membuat commit.

## Data JSON yang diharapkan

`data` memuat `succeeded`, `failed`, `total`, `saved`, dan `failed_count`, konsisten
dengan bentuk `close all`.

## Acceptance criteria

- `historic save` pada workspace tanpa topic open sukses dengan hitungan nol.
- Setelah `historic save`, `.database/<id>-<slug>/` ada untuk setiap topic open.
- Workdir open tetap ada dan `historic show <id>` tetap leading ke salinan open.
- Jalur `close all` tidak berubah perilakunya selain tidak commit.

## Status implementasi (2026-10-13) — selesai

`internal/cmd/save.go` memanggil `lifecycle.Service.MirrorAllOpen()` (enumerasi
`openTopicTargets`, deterministik per TopicID/path), menjaga workdir open, tanpa Git,
dan melaporkan ringkasan per topic (`saved`/`failed`). `--json` memakai envelope standar.

Tambahan keputusan 2026-10-13: `historic save <id>` didukung sebagai mirror satu topic
open via `lifecycle.Service.MirrorTopic`; ringkasan satu target dan gagal per-topic bila
ID invalid/tidak ada.
