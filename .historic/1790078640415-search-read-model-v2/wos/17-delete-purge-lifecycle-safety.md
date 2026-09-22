---
id: 00017
title: WO 17 Delete and Purge Lifecycle Safety
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [delete, purge, lifecycle, safety, recovery]
related: [../spec.md]
---
# WO 17 — Delete and Purge Lifecycle Safety

## Tujuan
Menyediakan delete file/topic dan purge permanen yang aman, eksplisit, dan tidak mengganggu snapshot internal.

## Scope
- `historic delete <path>`.
- `historic delete-topic <id> [--open|--closed]`.
- `historic purge <id> [--open|--closed]`.
- Confirmation, scope protection, staging, backup/trash, rollback.
- Closed topic tidak dimodifikasi diam-diam.
- Rebuild cleanup read model/FTS.
- Symlink/path/conflict safety.

## Acceptance Criteria
- Close reversible; delete/purge destructive.
- File/asset deletion tepat scope.
- Internal Git tidak ikut terhapus.
- Failure no partial state.
- Open/closed scope eksplisit bila diperlukan.
- Human/JSON output menjelaskan konsekuensi.

## Status
Complete. Implemented and validated in commit `d2ce3ec`.
