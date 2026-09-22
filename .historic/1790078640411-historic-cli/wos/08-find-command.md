---
id: 00001
title: WO 08 Find Command
status: create
created: 2026-09-18
---

# WO 08 — `historic find`

## Tujuan
Menyediakan pencarian MVP pada nama file dan isi Markdown.

## Tasks
- Scan working directory dan, bila diminta, `.database/`.
- Cari keyword secara case-insensitive pada path, title, dan content.
- Tambahkan filter `--status`, `--folder`, `--active`, dan `--archived` sesuai dukungan fase.
- Hindari hasil duplikat untuk entry yang sama.
- Tambahkan `--json` dengan snippet/path/status.
- Ukur waktu pencarian untuk benchmark lokal.

## Dependensi
WO 03, WO 05–07.

## Acceptance Criteria
- Keyword ditemukan pada filename maupun body.
- Filter menghasilkan scope yang benar.
- Empty result exit sukses dan data kosong.
- Path traversal/keyword input tidak mengeksekusi command shell.
- Benchmark 1.000 file dicatat dan target PRD diverifikasi.

## Output
Search service filesystem, filter parser, benchmark, dan tests.
