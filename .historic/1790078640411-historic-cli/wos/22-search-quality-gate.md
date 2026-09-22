---
id: "00001"
title: WO 22 Search Benchmark and Quality Gate
status: complete
created: "2026-09-19"
updated: "2026-09-19"
tags:
    - phase-4
    - search
    - benchmark
    - quality-gate
---
# WO 22 — Search Benchmark and Quality Gate

## Tujuan
Menutup Phase 4 dengan bukti performa, recovery, kompatibilitas output, dan dokumentasi penggunaan.

## Tasks
- Benchmark query pada 1.000, 10.000, dan 100.000 file.
- Benchmark rebuild index dan FTS.
- Uji index missing, FTS missing/corrupt, Markdown invalid, dan recovery rebuild.
- Uji empty query, invalid FTS syntax, Unicode, punctuation, dan special characters.
- Uji semua kombinasi filter penting dan JSON error behavior.
- Bandingkan hasil FTS dengan baseline filesystem untuk correctness.
- Dokumentasikan target, hasil benchmark, batasan, dan cara menjalankan `historic rebuild`.
- Update README dan catat temuan dogfooding yang relevan.

## Dependensi
WO 19–21.

## Acceptance Criteria
- Semua test Phase 4 lulus.
- Index lama tetap aman saat rebuild gagal.
- Hasil FTS benar terhadap fixture baseline.
- Benchmark memenuhi target yang disetujui atau deviasi dicatat dengan keputusan.
- README memuat query/filter Phase 4 dan recovery index.
- SPEC Phase 4 ditandai ready/complete sesuai hasil review.

## Output
Benchmark report, test report, README Phase 4, recovery notes, dan Phase 4 sign-off.
