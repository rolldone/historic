---
title: 01 Workspace Validation Contract
description: Mendefinisikan state, urutan pemeriksaan, dan kontrak typed error untuk readiness workspace Historic.
status: complete
created: "2026-09-23"
updated: "2026-09-23"
tags:
    - workspace
    - validation
    - contract
    - errors
related:
    - ../spec.md
---
# WO 01 — Workspace Validation Contract

## Tujuan

Definisikan kontrak terpusat untuk menentukan apakah workspace Historic siap dipakai command lain. Work Order ini berfokus pada model domain dan aturan pemeriksaan, bukan integrasi seluruh command.

## Scope

- Identifikasi root workspace berdasarkan konfigurasi existing.
- Definisikan state readiness: `ready`, `not_historic_workspace`, `legacy_conflict`, `missing_index`, `invalid_index`, dan `invalid_structure`.
- Definisikan typed error atau error code yang dapat dipakai output human-readable dan JSON.
- Tentukan urutan pemeriksaan agar `.historic/` diverifikasi sebelum database dibuka.
- Reuse helper versi/schema yang sudah ada; jangan membuat sumber kebenaran kompatibilitas baru.

## Aturan wajib

- Jika `.historic/` tidak ada, return error `not_historic_workspace`.
- Jika `.historic/` dan `.histories/` ada bersama, return `legacy_conflict`.
- Jika index SQLite tidak ada, return `missing_index`.
- Jika index tidak dapat dibuka atau schema tidak kompatibel, return `invalid_index`.
- Tidak boleh membuat folder, file, index, metadata, atau backup.
- Tidak boleh memanggil `rebuild`.

## Acceptance criteria

- Kontrak readiness dapat dipakai tanpa membuka query database terlebih dahulu.
- Setiap state memiliki error code stabil dan remediation message.
- Pemeriksaan tidak mengubah filesystem pada semua jalur sukses maupun gagal.
- Unit test mencakup setiap state dan urutan pemeriksaan.
- Implementasi tidak menduplikasi aturan `.historic/`/`.histories/` yang sudah ditetapkan.

## Catatan implementasi

Periksa package `config`, `indexer`, dan command root sebelum menentukan lokasi helper. Jangan mengubah source code di luar scope implementasi Work Order ini.
