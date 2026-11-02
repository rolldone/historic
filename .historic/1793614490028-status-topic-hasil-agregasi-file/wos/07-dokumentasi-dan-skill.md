---
title: WO 07 — Dokumentasi dan Skill untuk Status Topic
description: Memperbarui README, docs/commands.md, dan skill Historic agar mendokumentasikan status topic hasil agregasi dan penghapusan label active.
status: complete
created: "2026-11-02"
updated: "2026-11-02"
tags:
    - docs
    - skill
related:
    - "1793614490028"
---
# WO 07 — Dokumentasi dan Skill untuk Status Topic

## Konteks

Referensi SPEC: `../spec-status-topic-hasil-agregasi-file.md` §6 dan §8.
Perubahan perilaku (status topic terlihat, `active` dihapus, `forwarded` tidak lagi
"aktif") harus tercermin pada dokumentasi pengguna dan skill agar AI/agent tidak memakai
informasi lama.

## Ruang lingkup

Perbarui:

- `README.md` — jelaskan bahwa status topic adalah nilai turunan dari status file, dan
  sebutkan aturan agregasi secara ringkas.
- `docs/commands.md` — dokumentasikan:
  - status topic pada keluaran `list` dan `show`;
  - perilaku `find --status` yang mencakup file dan topic;
  - penolakan `--status active` sebagai status tidak dikenal.
- `.agents/skills/historic/SKILL.md` — perbarui bagian yang menyebut
  `computed_status`:
  - tegaskan `computed_status` adalah nilai derived/cache dan **bukan** status
    authoritative di Markdown;
  - cantumkan aturan agregasi (§5.1) secara ringkas dan deterministik;
  - nyatakan `forwarded` setara `complete` untuk agregasi;
  - nyatakan `cancelled` hanya menang bila seluruh topic cancelled;
  - hapus/tegaskan bahwa `active` bukan nilai `computed_status`.

## Batasan

- Tidak mengubah `PRD.md` kecuali diminta eksplisit.
- Dokumentasi harus konsisten dengan perilaku yang benar-benar diimplementasikan pada
  WO 01–WO 06; jangan mendokumentasikan perilaku yang belum ada.
- Tidak ada perubahan source code pada WO ini.

## Definisi selesai

- Tidak ada lagi referensi yang menyebut `computed_status` bernilai `active`.
- Aturan agregasi terdokumentasi di README dan skill.
- Contoh perintah pada dokumentasi dapat dieksekusi tanpa error validasi.
