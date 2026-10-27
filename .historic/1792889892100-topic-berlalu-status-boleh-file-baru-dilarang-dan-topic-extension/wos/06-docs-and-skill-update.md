---
title: WO 06 — Dokumentasi Topic Berlalu, Extension, dan Status review
description: Menyinkronkan README, docs/commands.md, dan Historic skill dengan perilaku add pada closed, create --extends, status closed, dan review.
status: complete
created: "2026-10-25"
updated: "2026-10-27"
tags:
    - docs
    - skill
    - status
    - extension
related:
    - "1792889892100"
---
# WO 06 — Dokumentasi Topic Berlalu, Extension, dan Status `review`

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §5, §6, §9, AC-7.
Dokumentasi perlu menyatakan dengan tegas bahwa status adalah milik file; status file
pada topic closed dapat diubah; kelanjutan pekerjaan menggunakan topic baru dengan
`--extends`.

> **Revisi 2026-10-28:** dokumentasi **tidak boleh** lagi menyebut larangan menambah file
> pada topic berlalu. Perilaku `add` pada topic closed tetap seperti sebelumnya.

## Ruang lingkup

Perbarui sesuai implementasi final:

- `README.md`: contoh `historic create ... --extends` dan `historic status <path> review`
  untuk path closed.
- `docs/commands.md`: sintaks dan error/JSON `create`, `status`, `show`.
- `.agents/skills/historic/SKILL.md`: aturan lifecycle/status, definisi closed, dan
  extension.
- Daftar status dan contoh harus memuat `review` secara konsisten.
- Jangan menambahkan klaim larangan file baru pada topic berlalu.
- Jangan ubah `PRD.md`; jangan mendokumentasikan fitur yang belum diimplementasikan.

## Langkah implementasi

1. Cocokkan aktual CLI/output dengan hasil WO-01–05.
2. Ubah bagian lifecycle terkait saja; pertahankan aturan lain yang tidak berubah.
3. Pastikan contoh path/ID mencerminkan format canonical saat ini.
4. Periksa link relatif dan command examples.

## Kriteria penerimaan

- **AC-1** Ketiga dokumen menyebut status `review` untuk file.
- **AC-2** Aturan status pada topic closed dan extension sesuai perilaku aktual.
- **AC-3** Tidak ada klaim larangan menambah file pada topic berlalu.
- **AC-4** Tidak ada perubahan pada `PRD.md`.
- **AC-5** Pemeriksaan dokumentasi/link/contoh berhasil.

## Test plan

| Pemeriksaan | Ekspektasi |
| --- | --- |
| Pencarian daftar status | `review` ada di semua dokumen yang relevan |
| Contoh create | Sintaks `--extends` cocok dengan CLI |
| Status pada closed | Dijelaskan sebagai operasi yang didukung |
| Skill lifecycle | Menjelaskan status file vs storage topic |
| Larangan file baru | Tidak ada klaim seperti itu |
| PRD | Tidak disentuh |

## Risiko dan dependensi

Risiko rendah, namun dokumentasikan hanya perilaku yang sudah diterapkan. Kerjakan setelah
WO-03 dan WO-05 agar contoh CLI akurat.
