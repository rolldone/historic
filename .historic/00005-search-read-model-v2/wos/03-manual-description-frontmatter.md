---
id: 00003
title: WO 03 Manual Description Frontmatter
status: planned
created: 2026-09-21
updated: 2026-09-21
tags: [frontmatter, description, metadata]
related: [../spec.md, ./02-sqlite-topics-files-schema.md]
---
# WO 03 — Manual `description` Frontmatter

## Tujuan
Menambahkan dukungan description manual pada topic metadata dan managed Markdown tanpa prompt atau flag wajib.

## Scope
- Parser menerima `description` nullable/empty.
- `create` membuat description kosong.
- `add` membuat description kosong.
- Rebuild membaca description topic/file ke read model.
- FTS dapat mengindeks description.
- Existing frontmatter tanpa description tetap valid.

## Acceptance Criteria
- Create/add tidak meminta input tambahan.
- User dapat mengisi description manual.
- Rebuild mempertahankan description.
- Description asset nullable.
- Invalid description menghasilkan error parser yang actionable.

## Status
Complete. Implemented and validated in commit `0e0f62d`.
