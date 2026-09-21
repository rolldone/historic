---
id: 00010
title: WO 10 Delete and Purge Lifecycle
status: planned
created: 2026-09-21
updated: 2026-09-21
tags: [delete, purge, lifecycle, safety]
related: [../spec.md, ./07-unified-rebuild-read-model.md]
---
# WO 10 — Delete and Purge Lifecycle

## Tujuan
Menyediakan penghapusan file/topic yang aman dan membedakannya dari close.

## Scope
- `historic delete <path>` untuk file/asset current state.
- `historic delete-topic <id> [--open|--closed]`.
- `historic purge` untuk penghapusan permanen dengan konfirmasi ekstra.
- Closed topic tidak dimodifikasi diam-diam.
- Staging/backup/trash/rollback.
- Rebuild membersihkan read model setelah delete.
- Symlink/traversal/conflict rejection.

## Acceptance Criteria
- Close tetap reversible; delete/purge destructive.
- File delete mereconcile Files/Assets dan FTS.
- Asset delete tidak memengaruhi aggregate.
- Open/closed scope eksplisit saat keduanya ada.
- Internal Git tidak ikut terhapus.
- Failure tidak meninggalkan partial state.
- Human/JSON output menjelaskan target dan konsekuensi.

## Status
Complete. Implemented and validated in commit `72ab755`.
