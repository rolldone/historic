---
title: 02 Reject Only Middleware
description: Mengimplementasikan preflight readiness yang hanya menolak workspace tidak siap dan tidak pernah melakukan rebuild otomatis.
status: complete
created: "2026-09-23"
updated: "2026-09-23"
tags:
    - middleware
    - reject-only
    - rebuild
    - preflight
related:
    - ../spec.md
    - ./01-workspace-validation-contract.md
---
# WO 02 — Reject-Only Middleware

## Tujuan

Tambahkan satu jalur preflight yang dapat dipanggil command workspace-dependent sebelum handler mengakses storage atau index.

## Scope

- Implementasikan middleware/helper readiness berdasarkan kontrak WO 01.
- Pastikan jalur validasi sukses tidak mengubah data.
- Pastikan jalur gagal berhenti sebelum query dan mutasi lifecycle.
- Pisahkan preflight `rebuild` dari readiness middleware agar tidak terjadi rekursi.
- Pertahankan operasi `doctor` sebagai inspeksi read-only sesuai kontrak existing.

## Perilaku yang dilarang

- Tidak menjalankan `historic rebuild` sebagai side effect.
- Tidak memanggil subprocess CLI untuk memperbaiki workspace.
- Tidak membuat `.historic/`, `.database/`, `.index.sqlite`, atau metadata.
- Tidak menghapus, memindahkan, menggabungkan, atau mengganti nama `.histories/`.
- Tidak melanjutkan ke handler setelah error readiness.

## Acceptance criteria

- Workspace tanpa `.historic/` selalu ditolak.
- Workspace dengan konflik `.historic/` dan `.histories/` selalu ditolak.
- Workspace tanpa atau dengan index invalid ditolak dan menyarankan `historic rebuild`.
- Workspace siap diteruskan ke handler tepat satu kali.
- Tidak ada pemanggilan rebuild otomatis yang dapat diamati dari test spy atau mock.
- Jalur error tidak membuka koneksi/query database command.

## Batasan

Perubahan hanya boleh dilakukan pada source code yang diperlukan untuk middleware dan test-nya. Jangan mengubah perilaku command bootstrap di luar kebutuhan readiness.
