---
title: 09 Interactive Topic Picker
description: "Bangun picker interaktif bersama untuk memilih topic open atau closed, diurutkan berdasarkan waktu terbaru dan mendukung pencarian teks."
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
  - tui
  - lifecycle
  - picker
related:
  - ../spec-save-mirrors-open-workdir.md
  - ./08-tui-dual-version.md
---
# WO 09 — Interactive Topic Picker

## Tujuan

Menyediakan komponen pilihan topic interaktif yang dipakai command `save`, `close`, dan
`open`, agar pengguna dapat memilih target tanpa menyalin ID secara manual.

## Keputusan UX

- Picker berjalan dalam terminal interaktif; daftar item adalah satu versi per topic
  dalam scope yang diminta (open atau closed), tidak mencampur storage.
- Urutkan `updated_at` terbaru dahulu. Jika frontmatter `updated_at` kosong, gunakan
  mtime topic metadata (`_meta.yaml`); tie-breaker deterministik: TopicID, lalu path.
- Navigasi `↑`/`↓`; `Enter` memilih tepat satu item; `Esc`/`q` membatalkan tanpa mutasi.
- Pencarian teks penuh di dalam picker: `/` atau mulai mengetik memfokuskan query;
  filter mencakup TopicID, title, description, tags, path, serta teks terindeks dari
  member/file. `Enter` saat fokus query menerapkan pencarian; `Esc` kembali ke daftar.
- Query/search dan daftar memakai service read-only yang ada; presentation TUI tidak
  boleh mengakses SQLite langsung.
- Terminal non-interaktif tanpa ID harus gagal dengan pesan yang menyarankan ID eksplisit
  atau `--all` untuk bulk command yang mendukungnya.
- Picker kosong menampilkan pesan yang jelas; jangan mengubah workspace.

## Scope

- Buat model/presentation reusable di `internal/tui` atau package interaksi bersama.
- Terima scope storage, item list/context, dan hasil pilihan/cancel; tidak melakukan
  lifecycle mutation sendiri.
- Hormati viewport, clipping judul/path panjang, preview read-only, dan footer.
- Tidak mengubah semantics `historic search` yang sudah ada.

## Status implementasi (2026-10-13)

Selesai: picker bersama untuk scope open/closed, pengurutan updated_at/mtime, filtering teks, navigasi, pembatalan, dan pengujian model UI telah diimplementasikan.

## Acceptance criteria

- Unit test untuk sort newest-first dengan fallback mtime dan tie-break deterministik.
- Unit test navigasi, memilih dengan Enter, membatalkan dengan Esc/q, dan filtering query.
- Hasil pilihan mempertahankan ID dan storage dari versi yang dipilih.
- Picker bekerja pada terminal kecil tanpa menggambar daftar di bawah footer.
- Tidak ada akses SQLite langsung dari layer presentation dan tidak ada filesystem mutation.
