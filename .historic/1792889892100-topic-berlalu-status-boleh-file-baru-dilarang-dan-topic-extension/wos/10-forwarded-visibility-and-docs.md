---
title: WO 10 — Visibilitas forwarded_to dan Dokumentasi Forward
description: Menampilkan forwarded_to pada historic show, memastikan filter forwarded konsisten di TUI, dan mendokumentasikan alur forward di README, docs/commands.md, serta skill.
status: complete
created: "2026-10-28"
updated: "2026-10-27"
tags:
    - show
    - tui
    - docs
    - forward
related:
    - "1792889892100"
---
# WO 10 — Visibilitas `forwarded_to` dan Dokumentasi Forward

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §5.6, §6, AC-14/AC-16.
Setelah data dan command tersedia (WO-08, WO-09), relasi forward perlu terlihat dan
terdokumentasi.

## Ruang lingkup

1. `internal/cmd/list_show.go`:
   - `historic show <id> --json` menampilkan `forwarded_to` per file bila ada;
   - human output menampilkan tujuan forward secara ringkas.
2. `internal/tui/search.go`:
   - pastikan `forwarded` ada pada siklus filter status;
   - preview dapat menampilkan `forwarded_to` bila tersedia.
3. Dokumentasi:
   - `README.md`: contoh `historic forward` jalur `--to` dan `--create`;
   - `docs/commands.md`: sintaks, JSON, dan aturan validasi strict;
   - `.agents/skills/historic/SKILL.md`: alur forward, klasifikasi terminal, dan
     hubungannya dengan `--extends`;
   - cantumkan `forwarded` pada daftar status file.
4. Tidak mengubah `PRD.md` dan tidak mendokumentasikan fitur yang belum ada.

## Langkah implementasi

1. Tambahkan field ke DTO `show` dan render human output.
2. Verifikasi siklus filter status TUI dan preview.
3. Perbarui tiga dokumen sesuai perilaku aktual.
4. Tambahkan test command show dan test filter TUI/search.

## Kriteria penerimaan

- **AC-1** `show <topic-id> --json` memuat `forwarded_to` per file (AC-16 SPEC).
- **AC-2** Human show menampilkan tujuan forward bila ada, tanpa mengganggu output lama.
- **AC-3** `forwarded` muncul pada filter status TUI dan search (AC-14 SPEC).
- **AC-4** Ketiga dokumen menyebut `forwarded` dan command `historic forward`.
- **AC-5** Tidak ada klaim fitur di luar SPEC (mis. pemindahan konten otomatis).
- **AC-6** `PRD.md` tidak berubah.

## Test plan

| Kasus | Ekspektasi |
| --- | --- |
| Show file forwarded | Field `forwarded_to` muncul di JSON |
| Show file non-forwarded | Field tidak muncul, output lama tetap |
| Filter TUI | `forwarded` dapat dipilih |
| Preview TUI | Menampilkan tujuan bila ada |
| Dokumentasi | Status dan command terdokumentasi konsisten |

## Risiko dan dependensi

Risiko rendah. **Bergantung pada WO-08 dan WO-09.** Kerjakan bersama WO-06 agar
perubahan dokumentasi tidak bertabrakan.
