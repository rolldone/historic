---
id: 00006
title: WO 06 Differential Close and Archive Normalization
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [lifecycle, close, diff, hash, archive]
related: [../spec.md, ./04-topic-identity-logical-paths.md, ./05-copy-based-open.md]
---
# WO 06 — Differential `historic close`

## Tujuan
Menutup topic dengan overwrite snapshot terbaru secara aman dan hemat operasi file.

## Scope
- Manifest logical path + SHA-256 content.
- Cari archive berdasarkan ID, bukan nama folder.
- Stage archive baru dengan slug terbaru.
- Copy hanya file baru/berubah.
- Hapus file yang hilang dari staged snapshot.
- Rename archive lama jika slug berubah.
- Atomic replacement lalu hapus workdir.
- No-op content jika manifest dan slug sama.

## Acceptance Criteria
- Content sama tidak disalin ulang.
- File berubah saja yang diperbarui.
- File baru/hilang ditangani.
- Archive lama dengan slug berbeda dinormalisasi.
- Close gagal menjaga archive lama dan workdir.
- Close menghasilkan satu snapshot per topic ID.
- Internal Git tidak membuat diff content untuk file identik.

## Status
Complete. Implemented and validated in commit `3dd9f47`.
