---
id: 00005
title: WO 05 Copy Based Open
status: planned
created: 2026-09-21
updated: 2026-09-21
tags: [lifecycle, open, copy, snapshot]
related: [../spec.md, ./04-topic-identity-logical-paths.md]
---
# WO 05 — Copy-Based `historic open`

## Tujuan
Mengubah `open` dari move menjadi copy snapshot `.database` ke workdir.

## Scope
- Copy closed topic ke `.historic/` melalui staging.
- Snapshot `.database` tetap ada.
- Menolak destination conflict dan symlink.
- Mempertahankan isi, frontmatter, slug, dan relative paths.
- Update storage state melalui rebuild/read model.
- Rollback jika staging/copy gagal.

## Acceptance Criteria
- Open tidak menghapus snapshot `.database`.
- Working copy tersedia setelah open sukses.
- Search menghasilkan satu logical topic dengan storage open.
- File checksum sama dengan snapshot.
- Failure tidak merusak snapshot atau membuat partial workdir.

## Status
Complete. Implemented and validated in commit `871fcde`.
