---
title: 04 Validation Test Matrix
description: Memastikan seluruh state readiness, command exception, output error, dan larangan auto-rebuild tervalidasi lewat test.
status: complete
created: "2026-09-23"
updated: "2026-09-23"
tags:
    - testing
    - integration-test
    - regression
    - validation
related:
    - ../spec.md
    - ./01-workspace-validation-contract.md
    - ./02-reject-only-middleware.md
    - ./03-command-integration-and-errors.md
---
# WO 04 — Validation Test Matrix

## Tujuan

Membuat test matrix yang membuktikan middleware reject-only aman, deterministic, dan tidak melakukan side effect tersembunyi.

## Matrix readiness

| Kondisi | Expected state | Expected action |
|---|---|---|
| `.historic/` tidak ada | `not_historic_workspace` | reject, sarankan `historic init` |
| `.historic/` ada, `.histories/` tidak ada | lanjut validasi | lanjut |
| `.historic/` dan `.histories/` sama-sama ada | `legacy_conflict` | reject, tidak merge/rename |
| index tidak ada | `missing_index` | reject, sarankan `historic rebuild` |
| index rusak | `invalid_index` | reject, sarankan `historic rebuild` |
| schema index tidak kompatibel | `invalid_index` | reject, sarankan `historic rebuild` |
| struktur valid dan index valid | `ready` | handler berjalan |

## Test tambahan

- Pastikan middleware tidak membuat file atau direktori pada semua skenario reject.
- Pastikan middleware tidak menjalankan rebuild otomatis.
- Pastikan handler tidak dipanggil ketika readiness gagal.
- Pastikan command bootstrap tetap dapat berjalan pada workspace belum siap.
- Pastikan `rebuild` tidak terblokir oleh readiness middleware.
- Pastikan error human-readable actionable.
- Pastikan JSON error memiliki envelope existing, code stabil, dan tidak duplikat di stderr.
- Pastikan validasi dilakukan sebelum query database atau mutasi lifecycle.
- Pastikan test tidak bergantung pada urutan test dan aman terhadap temporary workspace.

## Acceptance criteria

- Semua baris matrix memiliki test otomatis.
- Terdapat regression test khusus kasus awal: `find` pada workspace sebelum rebuild.
- Test membuktikan hasilnya reject dengan instruksi `historic rebuild`, bukan rebuild otomatis.
- Test command valid membuktikan handler tetap dapat berjalan setelah readiness sukses.
- Test lulus tanpa mengubah source Markdown sebagai efek samping.
