---
id: 00001
title: WO 01 Two-Table Search Read Model
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [search, sqlite, fts5, aggregate, ai, read-model]
related: []
---
# WO 01 — Two-Table Search Read Model

## Tujuan

Membangun read model SQLite yang mudah digunakan AI dengan dua tabel utama: `topics` dan `files`. Markdown tetap source of truth; SQLite hanya cache/index yang dapat dibangun ulang.

## Model Data

### `topics`

Minimal field:

```text
id
num_padded
title
description
path
storage                 open | closed
created_at
updated_at
tags
related
computed_status        derived/cache, bukan authoritative
```

### `files`

Minimal field:

```text
id
topic_id
type                    historic_file | asset
path
filename
title
description
status                  nullable untuk asset
tags
related
content
asset_kind
created_at
updated_at
mtime
hash
size
word_count
```

Relasi:

```text
topics.id 1 ──── * files.topic_id
```

Managed Markdown dengan frontmatter valid menjadi `type=historic_file` dan memiliki `status`. File lain menjadi `type=asset` dan tidak memengaruhi status pekerjaan.

## Source of Truth

- Keberadaan dan isi file berasal dari filesystem Markdown/asset.
- Status file Historic berasal dari frontmatter Markdown.
- Description diisi manual oleh user setelah `create`/`add`; command tidak wajib menerima flag description.
- SQLite menyimpan salinan read model untuk query cepat.
- `rebuild` harus dapat menghasilkan kembali kedua tabel dari filesystem.
- `computed_status` topic hanya hasil agregasi dan tidak ditulis sebagai status authoritative ke `_meta.md`.

## Structured Query

Query harus mendukung kombinasi filter dari `topics` dan `files`:

```text
status
 tags
title
description
storage
type
topic_id
path/folder
created_at
updated_at
```

Contoh kemampuan:

- mencari keyword pada title/description/content topic dan file;
- filter `files.status`;
- filter tags topic/file;
- filter storage open/closed;
- filter type `historic_file`/`asset`;
- filter topic ID/folder/tanggal;
- aggregate total, active, complete, cancelled, dan unresolved files per topic.

## FTS5

Gunakan FTS5 untuk pencarian teks, minimal meliputi:

```text
topic title
topic description
file filename
file path
file title
file description
file content
tags
```

Ranking harus memberi bobot lebih tinggi pada title, description, dan tags dibanding content biasa. Structured filters tetap dieksekusi sebagai SQL, bukan dipaksakan menjadi query FTS.

Vector DB tidak termasuk WO ini. Description manual dan FTS5 menjadi fondasi semantic hint yang ringan. Semantic/vector search dapat menjadi fase berikutnya.

## Aggregate dan hasil AI

Hasil query topic harus dapat menyertakan:

```text
total_files
active_files
resolved_files
complete_files
cancelled_files
computed_status
```

`complete` dan `cancelled` dianggap resolved untuk aggregate, tetapi nilai file tetap dipertahankan apa adanya.

Hasil file minimal memuat:

```text
type
topic_id
title
description
status
storage
tags
path
score
matched_in
snippet
```

## Kontrak CLI awal

Existing `historic find` tetap menjadi interface one-shot. Desain filter baru harus mendukung bentuk yang jelas, misalnya:

```text
historic find "schema" --status progress --tag compatibility --open
historic find "schema" --type historic_file --topic 00005
historic find "schema" --updated-after 2026-09-01
```

Perubahan flag dan JSON schema harus ditentukan secara eksplisit sebelum implementasi. Jangan merusak pemisahan `historic find` untuk AI dan `historic search` untuk TUI.

## Acceptance Criteria

- SQLite memiliki tabel `topics` dan `files` dengan relasi topic-to-files.
- Rebuild menghasilkan record topic, managed file, dan asset.
- Asset memiliki `status` null dan tidak memengaruhi aggregate status.
- Description manual dari frontmatter tersimpan di read model.
- Query dapat menggabungkan filter topic dan file.
- Query dapat mengaggregate status file per topic.
- `complete` dan `cancelled` dihitung resolved tanpa mengubah nilai aslinya.
- Open/closed storage tersedia pada hasil topic dan file.
- FTS5 mencakup title, description, tags, path, filename, dan content sesuai bobot.
- JSON hasil menyertakan konteks topic, storage, status, score, matched fields, dan snippet.
- Rebuild/idempotence/invalid Markdown/rollback memiliki test.
- `PRD.md` tidak diubah.
- Vector DB tidak diperlukan untuk acceptance WO ini.

## Batasan

- Tidak menambahkan Vector DB atau embedding runtime.
- Tidak mengubah Markdown secara otomatis selain generated Files/Assets pada kontrak rebuild.
- Tidak menjadikan `computed_status` topic sebagai source of truth.
- Tidak menghapus full-text content dari managed Markdown.
- Tidak melakukan remote sync atau push.

## Status

Complete. Cakupan desain two-table read model telah diimplementasikan dan divalidasi melalui WO 02–17.
