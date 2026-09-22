---
id: 00001
title: WO 15 Internal Git Repository
status: complete
created: 2026-09-18
updated: 2026-09-19
---

# WO 15 — Internal Git Repository

## Tujuan
Menyiapkan repository Git internal pada `.database/` sebagai fondasi versioning.

## Tasks
- Inisialisasi Git repository internal secara idempotent.
- Tentukan adapter (`go-git` atau Git executable) dan abstraction interface.
- Pastikan metadata `.git` tidak masuk hasil find sebagai entry Markdown.
- Tetapkan user-facing behavior saat Git tidak tersedia/korup.
- Definisikan policy file yang boleh/versioned dan root repository.
- Tambahkan integration test dengan repository temporary.

## Dependensi
WO 12–14.

## Acceptance Criteria
- `.database/.git` tersedia setelah init/phase setup.
- Repeated initialization tidak merusak history.
- Adapter dapat membaca status/log/diff dasar.
- Tidak ada remote atau push yang dilakukan pada Phase 3.

## Output
Git repository adapter dan initialization tests.
