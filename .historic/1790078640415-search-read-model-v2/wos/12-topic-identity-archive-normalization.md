---
id: 00012
title: WO 12 Topic Identity Rename and Archive Normalization
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [identity, rename, slug, archive]
related: [../spec.md]
---
# WO 12 — Topic Identity Rename and Archive Normalization

## Tujuan
Menjadikan ID sebagai identitas topic dan menangani rename slug/folder tanpa duplicate logical topic.

## Scope
- Detect topic by ID.
- Relative logical file paths.
- Rename folder read model update.
- Duplicate ID conflict detection.
- Archive slug normalization at close.
- Open/archive copy deduplication.

## Acceptance Criteria
- Rename open folder diperbarui pada rebuild.
- Archive lama dengan slug berbeda dinormalisasi saat close.
- Tidak ada duplicate logical result.
- Conflict tidak menghapus data.
- Path traversal/symlink ditolak.

## Status
Complete. Implemented and validated in commit `2113672`.
