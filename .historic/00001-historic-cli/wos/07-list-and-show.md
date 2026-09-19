---
id: 00001
title: WO 07 List and Show
status: create
created: 2026-09-18
---

# WO 07 — `list` and `show`

## Tujuan
Memberikan pembacaan topik yang cepat untuk manusia dan AI.

## Tasks
- Implementasikan `historic list` dengan grouping status/kanban sederhana.
- Implementasikan `historic show <id>` dengan metadata, file list, dan progress.
- Tetapkan scope default aktif dan opsi archived sesuai keputusan SPEC.
- Implementasikan schema JSON `command`, `ok`, `data`, `error`.
- Pastikan output tidak bergantung pada urutan filesystem.

## Dependensi
WO 05–06.

## Acceptance Criteria
- List menampilkan topik dengan ID, title, status, dan updated.
- Show menampilkan semua file yang relevan tanpa membaca path di luar root.
- Output JSON valid dan stabil untuk empty result maupun error.
- Test mencakup topik aktif, arsip, dan ID tidak ditemukan.

## Output
Command list/show, serializer JSON, dan tests.
