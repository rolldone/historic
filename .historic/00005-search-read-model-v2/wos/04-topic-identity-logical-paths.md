---
id: 00004
title: WO 04 Topic Identity and Logical Paths
status: planned
created: 2026-09-21
updated: 2026-09-21
tags: [identity, slug, path, storage]
related: [../spec.md, ./02-sqlite-topics-files-schema.md]
---
# WO 04 — Topic Identity and Logical Paths

## Tujuan
Memastikan topic diidentifikasi oleh ID, sedangkan slug/folder dan physical root dapat berubah.

## Scope
- Resolve topic berdasarkan ID.
- Simpan `files.path` relatif POSIX terhadap topic.
- Bedakan storage `open` dan `closed` dari lokasi root.
- Rebuild menangani rename folder tanpa duplicate logical topic.
- Deteksi duplicate ID dalam root yang sama.
- Output dapat menambahkan physical path sebagai field turunan.

## Acceptance Criteria
- Rename folder memperbarui slug/read model saat rebuild.
- `files.path` tetap `wos/task.md`, bukan absolute path.
- Open/archive copy tidak menjadi duplicate search result.
- Duplicate ID conflict jelas dan aman.
- Test rename, duplicate, dan path traversal lulus.

## Status
Complete. Implemented and validated in commit `a0f8b63`.
