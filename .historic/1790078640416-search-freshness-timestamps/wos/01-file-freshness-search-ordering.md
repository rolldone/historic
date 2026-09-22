---
title: WO 01 File Freshness and Search Timestamp Ordering
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [search, timestamps, freshness, sorting, rebuild, ai]
related: []
---
# WO 01 — File Freshness and Search Timestamp Ordering

## Tujuan

Memperjelas bagaimana `rebuild` mendeteksi perubahan file dan bagaimana `find`/TUI mengurutkan hasil berdasarkan relevansi serta waktu terbaru.

## Timestamp read model

Managed `historic_file` harus menyimpan:

```text
created_at    dari frontmatter created
updated_at    dari frontmatter updated jika tersedia
mtime         dari filesystem
hash          SHA-256 content
```

Asset minimal menyimpan:

```text
mtime
hash
size
```

`rebuild` tidak boleh menulis `updated` otomatis ke frontmatter hanya karena content berubah. Perubahan content diperbarui di read model melalui hash/mtime.

## Deteksi perubahan

Perubahan file ditentukan melalui kombinasi:

- logical relative path;
- content hash;
- frontmatter metadata;
- mtime dan size sebagai metadata filesystem.

Perilaku:

- hash berubah → refresh record dan FTS;
- logical path berubah → remove old logical record dan add new path;
- frontmatter `updated` berubah → update `updated_at`;
- mtime berubah tetapi hash sama → content tetap unchanged, metadata filesystem boleh diperbarui;
- file baru → insert;
- file hilang → remove record dan reconcile manifest.

## Search ordering

Keyword search:

```text
relevance score DESC
updated_at DESC
mtime DESC
path ASC
```

Empty query/recent browser:

```text
updated_at DESC
created_at DESC
mtime DESC
path ASC
```

Aggregate topic result:

```text
last_file_updated_at DESC
updated_at DESC
path ASC
```

## Status filtering and recent browsing

`historic find` accepts comma-separated status inclusion/exclusion filters:

```text
--status <status[,status...]>
--status-not <status[,status...]>
```

Unknown statuses and overlap between the two flags are rejected. An empty keyword (`historic find ""`) uses the same recent-topic behavior as the TUI.


```text
--sort relevance
--sort updated
--sort created
--sort title
```

Default:

- keyword ada → `relevance`;
- keyword kosong → `updated`.

Status filtering wajib mendukung:

```text
--status <status[,status...]>
--status-not <status[,status...]>
```

Implementasi:

- parse comma-separated values;
- validasi setiap value dengan `domain.ParseStatus`;
- `--status` menjadi predicate inclusion;
- `--status-not` menjadi predicate exclusion;
- reject jika status yang sama muncul di kedua predicate;
- reject unknown status dengan nama flag dan nilai yang bermasalah;
- gunakan predicate yang sama untuk human output, JSON, TUI, dan aggregate query;
- kombinasi dengan `--open`, `--closed`, `--tag`, `--type`, folder, ID, dan sorting tetap didukung.

Use case utama:

```text
historic find "" --status-not complete,cancelled,failed,archived
historic find "migration" --status-not complete
```

Jika command saat ini mewajibkan keyword non-empty, dukung empty/recent query terlebih dahulu atau gunakan keyword netral yang terdokumentasi; jangan menghasilkan perilaku status filter yang berbeda.

Structured filters tetap dapat digabungkan dengan sorting.

## JSON output

Hasil file wajib dapat menampilkan:

```json
{
  "created_at": "2026-09-21",
  "updated_at": "2026-09-22",
  "mtime": "2026-09-22T08:12:33Z",
  "hash": "..."
}
```

Hasil topic dapat menampilkan:

```json
{
  "updated_at": "2026-09-22",
  "last_file_updated_at": "2026-09-22"
}
```

## Acceptance Criteria

- Rebuild mendeteksi file baru, berubah, dipindahkan, dan dihapus.
- Hash berubah memperbarui read model dan FTS.
- Mtime berubah dengan hash sama tidak dianggap content change.
- Frontmatter `created`/`updated` tersimpan akurat.
- Rebuild tidak mengubah frontmatter timestamp otomatis.
- `find` mengembalikan field timestamp yang dibutuhkan AI.
- Default keyword search mengutamakan relevance lalu freshness.
- Empty query mengutamakan file terbaru.
- Aggregate menyediakan `last_file_updated_at`.
- Sort deterministik dengan path sebagai tie-breaker.
- Open/closed filtering tetap bekerja.
- `--status` dan `--status-not` mendukung satu atau beberapa status comma-separated.
- Konflik status inclusion/exclusion ditolak dengan error actionable.
- Status filter bekerja konsisten pada CLI, JSON, TUI, dan aggregate result.
- Query backlog dapat mengecualikan `complete`, `cancelled`, `failed`, dan `archived`.
- Unit, integration, benchmark, dan isolated smoke test tersedia.
- `PRD.md` tidak diubah.

## Batasan

- Tidak menggunakan Vector DB.
- Tidak mengubah Markdown hanya untuk memperbarui timestamp.
- Tidak menghapus source file.
- SQLite tetap rebuildable read model.

## Status

Complete. Implemented and validated in the current workspace. `--status-not`, freshness fields, deterministic sorting, empty/recent browsing, and rebuild deletion reconciliation are implemented and tested.
