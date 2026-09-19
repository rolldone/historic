---
id: 00001
title: WO 17 Log and Diff
status: complete
created: 2026-09-19
updated: 2026-09-19
---

# WO 17 — `log` and `diff`

## Tujuan
Menyediakan riwayat dan perbandingan perubahan topic tanpa meminta user memakai Git langsung.

## Tasks
- Implementasikan `historic log <id>` dengan filter path/topic.
- Implementasikan `historic diff <id>` untuk perubahan yang belum tersimpan dan/atau antar snapshot sesuai keputusan final.
- Tetapkan format output text dan JSON.
- Tangani topic missing, commit missing, dan repository corrupt.
- Pastikan output tidak membocorkan path absolut atau credential.
- Tambahkan fixtures commit untuk multi-topic dan multi-file.

## Dependensi
WO 15–16.

## Acceptance Criteria
- Log menampilkan commit ID, waktu, message, dan scope topic.
- Diff membedakan added/modified/deleted dengan benar.
- Empty diff berhasil dengan output kosong terstruktur.
- Tests mencakup perubahan pada lebih dari satu topic.

## Output
Log/diff commands, serializers, dan integration tests.
