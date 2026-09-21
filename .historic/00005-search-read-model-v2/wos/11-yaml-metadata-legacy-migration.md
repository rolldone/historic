---
id: 00011
title: WO 11 YAML Metadata and Legacy Migration
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [yaml, metadata, migration, legacy]
related: [../spec.md]
---
# WO 11 — YAML Metadata and Legacy `_meta.md` Migration

## Tujuan
Menjadikan `_meta.yaml` format metadata topic canonical dan memigrasikan workspace lama yang memakai `_meta.md` secara aman.

## Scope
- Parser/validator `_meta.yaml`.
- Workspace baru membuat `_meta.yaml`.
- Rebuild/upgrade mendeteksi `_meta.md` legacy.
- Konversi field metadata dan description.
- Generated Files/Assets tidak dipindahkan sebagai metadata manual.
- Atomic write, validation, idempotence, conflict, rollback.

## Acceptance Criteria
- User lama tidak perlu menulis ulang metadata manual.
- `_meta.md` tidak dihapus sebelum `_meta.yaml` tervalidasi.
- Invalid legacy metadata menjaga file lama dan memberi error actionable.
- `_meta.yaml` menjadi satu-satunya write target setelah migrasi.
- Existing Markdown/asset/index terlindungi saat failure.
- Workspace baru tidak memerlukan `_meta.md`.

## Status
Complete. Implemented and validated in commit `fe42e6a`.
