---
id: 00013
title: WO 13 Copy Based Open and Differential Close
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [open, close, copy, diff, hash, lifecycle]
related: [../spec.md]
---
# WO 13 — Copy-Based Open and Differential Close

## Tujuan
Menjamin open/close berbasis copy dengan snapshot database tetap ada saat open dan overwrite efisien saat close.

## Scope
- `open`: database copy → workdir melalui staging.
- `close`: manifest diff → staged archive → atomic replacement → remove workdir.
- SHA-256 logical manifest.
- Copy hanya file baru/berubah.
- Hapus file yang hilang dari snapshot.
- Rename slug archive.
- No-op untuk tree identik.
- Rollback dan conflict/symlink safety.

## Acceptance Criteria
- Open tidak menghapus snapshot.
- Close menimpa snapshot dengan content terbaru.
- File identik tidak ditulis ulang.
- Rename archive mengikuti slug terbaru.
- Failure mempertahankan archive lama dan workdir.
- Satu snapshot per ID.

## Status
Complete. Implemented and validated in commit `5721309`.
