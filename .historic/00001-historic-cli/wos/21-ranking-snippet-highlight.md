---
id: "00001"
title: WO 21 Ranking Snippet and Highlight
status: complete
created: "2026-09-19"
updated: "2026-09-19"
tags:
    - phase-4
    - search
    - ranking
    - snippet
---
# WO 21 — Ranking, Snippet, and Highlight

## Tujuan
Meningkatkan kualitas hasil Advanced Search untuk manusia dan AI tanpa mengubah kontrak JSON yang sudah ada.

## Tasks
- Tambahkan ranking relevansi berbasis FTS5 BM25 atau strategi yang disetujui.
- Gunakan path ascending sebagai tie-breaker deterministik.
- Buat snippet kontekstual dari field yang match.
- Tambahkan highlight pada output manusia dengan format aman dan terdokumentasi.
- Jangan memasukkan ANSI/control formatting ke JSON.
- Pastikan hasil tidak membocorkan absolute path.
- Uji match pada filename, title, dan body secara terpisah serta simultan.
- Uji Unicode, multi-match, long line, dan no-snippet fallback.

## Dependensi
WO 20.

## Acceptance Criteria
- Hasil paling relevan muncul lebih dulu.
- Hasil dengan relevance sama stabil berdasarkan path.
- Snippet berisi konteks match dan tidak memotong invalid UTF-8.
- JSON tetap plain data tanpa highlight terminal.
- Test human output dan JSON output tersedia.

## Output
Ranking query, snippet/highlight formatter, deterministic sorting, dan tests.
