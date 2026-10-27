---
title: WO 03 — Buat Topic Extension melalui historic create --extends
description: Menambahkan relasi extends satu arah dan immutable ke metadata topic baru, dengan validasi induk sudah ada dan tanpa memodifikasi induk.
status: complete
created: "2026-10-25"
updated: "2026-10-27"
tags:
    - create
    - topic
    - extension
related:
    - "1792889892100"
---
# WO 03 — Buat Topic Extension melalui `historic create --extends`

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §5.3, §7, §8, AC-4/AC-6.
Topic baru harus dapat menunjuk parent yang sudah ada tanpa menulis backlink atau file ke
topic lama. ID anak tetap dialokasikan normal.

## Ruang lingkup

- Tambahkan optional `extends` ke `markdown.TopicMetadata` sebagai TopicID dan YAML key
  `extends` (`omitempty`); metadata legacy tanpa key tetap valid.
- Tambahkan nilai `Extends` ke domain/read model topic bila diperlukan untuk expose.
- Tambahkan flag `--extends <topic-id>` pada `historic create`.
- Validasi ID format dan bahwa parent sudah ada melalui `TopicLocation`; SPEC saat ini
  mengizinkan parent open maupun closed. Tidak ditemukan → `ErrTopicMissing`; invalid →
  error validasi. Validasi dilakukan sebelum membuat folder agar tidak ada artefak.
- Simpan `extends` pada metadata child saja. Jangan ubah metadata atau file induk.
- Tidak copy tags, related, atau files; satu parent; relasi immutable.
- Update JSON create dengan `data.extends` (string kosong bila tidak ada), sesuai SPEC.

## Langkah implementasi

1. Tambahkan field metadata dan validasi format `TopicID` tanpa membuat rebuild metadata
   legacy gagal.
2. Ubah `CreateTopic` secara kompatibel untuk menerima optional parent; audit semua call
   sites/tests.
3. Validasi target sebelum alokasi/folder creation, memastikan gagal atomik.
4. Persist field pada `_meta.yaml` dan kembalikan pada model domain.
5. Tambahkan flag/JSON field di `internal/cmd/create.go`.
6. Tambahkan unit/integration tests untuk absent, invalid, open/closed, tanpa extends,
   dan memastikan parent bytes tidak berubah.

## Kriteria penerimaan

- **AC-1** `create "Lanjutan" --extends <closed-id>` menghasilkan `_meta.yaml` dengan
  `extends: "<closed-id>"`.
- **AC-2** Parent open yang ada juga diterima (kontrak SPEC §5.3 saat ini).
- **AC-3** `--extends` invalid atau absent gagal sebelum folder dibuat.
- **AC-4** ID anak dialokasikan normal dan berbeda dari parent.
- **AC-5** Parent `_meta.yaml`/file tidak berubah; tidak ada copy/backlink.
- **AC-6** Create tanpa `--extends` kompatibel; JSON `data.extends` kosong.
- **AC-7** Legacy metadata tanpa extends tetap parse/rebuild.

## Test plan

| Caso | Input | Ekspektasi |
| --- | --- | --- |
| Parent closed | `create X --extends <closed-id>` | Berhasil, relasi tersimpan pada child |
| Parent open | `create X --extends <open-id>` | Berhasil sesuai kontrak terkini |
| Parent absent | ID valid tak ada | `ErrTopicMissing`, folder anak tidak ada |
| ID invalid | ID non-kanonis | Error validasi, folder anak tidak ada |
| Tanpa extends | `create X` | Perilaku lama; metadata tidak memuat key |
| Parent immutability | hash parent sebelum/sesudah | Identik |
| Legacy metadata | `_meta.yaml` tanpa key | Valid |

## Risiko dan dependensi

Risiko sedang: perubahan signature `CreateTopic` menyentuh beberapa call sites. Periksa
kompatibilitas API internal. Dapat mulai setelah keputusan kontrak SPEC disetujui; tidak
bergantung pada WO-04.
