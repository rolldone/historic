---
id: 00001
title: WO 11 Lifecycle Commands
status: create
created: 2026-09-18
---

# WO 11 — Lifecycle Commands

## Tujuan
Menyediakan perubahan status topic yang tervalidasi.

## Tasks
- Implementasikan command `progress`, `pending`, `review`, dan `blocked`.
- Implementasikan command close `complete`, `failed`, `cancelled`, dan `archived`.
- Definisikan dan implementasikan transition matrix final berdasarkan SPEC.
- Update frontmatter `_meta.md` dan `updated` secara atomic.
- Reindex entry terkait setelah perubahan status.
- Tampilkan previous status dan new status pada output.

## Dependensi
WO 02, WO 03, WO 09, WO 10.

## Acceptance Criteria
- Status invalid dan ID missing ditolak.
- Open status tidak memindahkan folder.
- Close status memicu archive sesuai WO 12.
- Metadata file tetap valid setelah command.
- Test mencakup repeat status, transition tidak valid, dan concurrent conflict sederhana.

## Output
Lifecycle service, command handlers, transition tests.
