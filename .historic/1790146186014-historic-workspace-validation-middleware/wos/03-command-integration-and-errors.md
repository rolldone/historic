---
title: 03 Command Integration and Errors
description: Mengintegrasikan readiness middleware ke command yang membutuhkan workspace dan menyamakan pesan error human-readable serta JSON.
status: complete
created: "2026-09-23"
updated: "2026-09-23"
tags:
    - cli
    - integration
    - json
    - errors
related:
    - ../spec.md
    - ./01-workspace-validation-contract.md
    - ./02-reject-only-middleware.md
---
# WO 03 — Command Integration and Errors

## Tujuan

Pasang preflight readiness secara konsisten pada command workspace-dependent tanpa mengubah command bootstrap dan tanpa memicu rebuild tersembunyi.

## Scope command

Audit dan integrasikan command yang membaca atau mengubah topic, minimal:

- `find`, `search`, `list`, `show`;
- `create`, `add`, `status`;
- `close`, `open`, `import`, `restore`;
- command lifecycle lain yang mengakses workspace.

`init`, `doctor`, `version`, `help`, dan `rebuild` harus diperlakukan sesuai pengecualian pada SPEC.

## Kontrak output

Human-readable output harus memberi tindakan berikutnya yang spesifik. JSON harus memakai envelope existing:

```json
{"command":"find","ok":false,"data":null,"error":{"code":"missing_index","message":"Jalankan historic rebuild terlebih dahulu."}}
```

Struktur error boleh mengikuti kontrak existing jika sudah tersedia, tetapi `code` harus stabil dan pesan tidak boleh berbeda antar-command untuk state yang sama.

## Acceptance criteria

- Setiap command dalam scope menjalankan readiness sebelum akses storage/index.
- Error `.historic/` hilang menyarankan `historic init`.
- Error index hilang/rusak menyarankan `historic rebuild`.
- Konflik `.historic/` dan `.histories/` tidak diperbaiki otomatis.
- JSON error hanya dicetak sekali dan exit status tetap non-zero.
- `rebuild` dapat dijalankan pada kondisi index hilang/rusak tanpa terkena middleware yang menghalanginya.
- Test command mencakup output human-readable dan JSON.

## Risiko yang harus diaudit

- Command yang memilih topic sebelum middleware berjalan.
- Command yang membuka database secara global saat root command diinisialisasi.
- Perbedaan error handling antar package command.
- Perubahan pada command `init` yang dapat menyebabkan bootstrap tidak mungkin dilakukan.
