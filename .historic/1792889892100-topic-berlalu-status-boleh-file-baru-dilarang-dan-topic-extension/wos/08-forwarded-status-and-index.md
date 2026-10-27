---
title: WO 08 — Status forwarded pada Domain, Frontmatter, Manifest, dan Index
description: Menambahkan status file terminal forwarded beserta field forwarded_to pada frontmatter, manifest, dan read model SQLite sebagai fondasi fitur forward.
status: complete
created: "2026-10-28"
updated: "2026-10-27"
tags:
    - domain
    - status
    - manifest
    - index
related:
    - "1792889892100"
---
# WO 08 — Status `forwarded` pada Domain, Frontmatter, Manifest, dan Index

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §4.6, §5.6, §7, AC-11/AC-15.
WO ini menyiapkan seluruh lapisan data sebelum command `historic forward` (WO-09)
diimplementasikan.

## Ruang lingkup

1. `internal/domain/model.go`:
   - tambah `StatusForwarded Status = "forwarded"`;
   - masukkan ke `validStatuses` dan `closeStatuses`;
   - **tidak** masukkan ke `openStatuses`;
   - tambah `ForwardedTo TopicID` pada `Frontmatter` dengan
     `yaml:"forwarded_to,omitempty"`.
2. `internal/markdown/metadata.go`:
   - tambah `ForwardedTo domain.TopicID` (`yaml:"forwarded_to,omitempty"`) pada
     `ManifestFile`;
   - pastikan metadata legacy tanpa field ini tetap valid.
3. `internal/lifecycle/file_status.go`:
   - saat status berubah ke non-`forwarded`, hapus `ForwardedTo` dari frontmatter;
   - sinkronkan perubahan tersebut ke `_meta.yaml.files[]` (mengikuti kontrak WO-02).
4. `internal/indexer/schema.go` + migrasi:
   - tambah kolom `files.forwarded_to` (nullable);
   - bump versi index; data lama diisi `NULL`;
   - rebuild membaca `forwarded_to` dari frontmatter Markdown sebagai source of truth.
5. `internal/search/search.go`:
   - `forwarded` masuk daftar status yang diterima `--status`/`--status-not`;
   - pesan validasi status menyebut `forwarded`.

Bukan ruang lingkup: command `historic forward` (WO-09), tampilan `show` (WO-10).

## Langkah implementasi

1. Tambah konstanta dan klasifikasi status, lalu sesuaikan test domain.
2. Tambah field frontmatter dan manifest; pastikan round-trip YAML tetap stabil.
3. Sesuaikan `ChangeFileStatus` agar membersihkan `ForwardedTo` pada status non-forwarded
   dan menyinkronkan manifest.
4. Tambah kolom index + migrasi + pembacaan saat rebuild.
5. Perbarui validasi status pada search dan test-nya.
6. Jalankan `go test ./...`.

## Kriteria penerimaan

- **AC-1** `forwarded` valid dan tergolong close, bukan open.
- **AC-2** Frontmatter mendukung `forwarded_to` dan tetap valid bila kosong.
- **AC-3** `_meta.yaml.files[]` mendukung `forwarded_to`.
- **AC-4** `historic status <path> progress` pada file `forwarded` menghapus
  `forwarded_to` di frontmatter dan manifest (AC-15 SPEC).
- **AC-5** Index menyimpan `files.forwarded_to`; migrasi index lama berjalan tanpa error.
- **AC-6** `historic find --status forwarded` dan `--status-not forwarded` berfungsi.
- **AC-7** Metadata legacy dan workspace lama tetap rebuild tanpa error.

## Test plan

| Kasus | Ekspektasi |
| --- | --- |
| Klasifikasi status | `forwarded` valid, `IsClose()` true, `IsOpen()` false |
| Round-trip frontmatter | `forwarded_to` tersimpan dan terbaca kembali |
| Manifest tanpa field | Legacy tetap valid |
| Clear forwarded_to | Status ke `progress` menghapus field |
| Migrasi index | Schema lama naik versi, rows utuh |
| Filter search | `--status forwarded` hanya file forwarded |
| Rebuild | Idempoten, `errors: 0` |

## Risiko dan dependensi

Risiko sedang: bump schema index perlu hati-hati agar rollback/atomic replacement tetap
aman. **WO-09 bergantung pada WO ini**. Kontrak sinkronisasi manifest mengikuti WO-02.
