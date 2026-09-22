---
id: "1790088125634"
title: Historic Search Pagination and More Results Indicator
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [spec, search, pagination, cli, tui, json, freshness]
related: [./wos/01-search-pagination-and-more-indicator.md]
---
# Historic Search Pagination SPEC

## 1. Tujuan

Memberi informasi yang jelas ketika hasil `historic find` lebih dari satu halaman, tanpa mengubah urutan search yang sudah deterministic. User dan AI harus dapat mengetahui halaman aktif, jumlah item yang ditampilkan, dan apakah masih ada hasil berikutnya.

## 2. CLI contract

```text
historic find "work" --page 1 --page-size 20
historic find "work" --page 2 --page-size 20
historic find "work" --status-not complete --page-size 10
```

Flags:

```text
--page <positive-integer>       default 1
--page-size <positive-integer>  default 20, maximum 100
```

Output policy:

- `--json` selalu mengembalikan machine-readable object `data.items + data.pagination`.
- Tanpa `--json`, output human-readable menampilkan hasil lalu ringkasan pagination.
- Perilaku existing tanpa `--page` dipertahankan melalui default page 1 dan page size 20; perubahan JSON menjadi object dianggap kontrak pagination version baru dan wajib didokumentasikan.
- AI/automation wajib menggunakan `--json`.

Validasi:

- `page >= 1`;
- `page-size >= 1`;
- `page-size <= 100`;
- invalid value menghasilkan error actionable;
- filter dan sorting diterapkan sebelum pagination;
- empty query tetap mendukung pagination.

## 3. More-results information

Human output wajib memberi informasi halaman:

```text
Page 1 · showing 20 results · more results available
Use --page 2 --page-size 20
```

Halaman terakhir:

```text
Page 2 · showing 6 results · no more results
```

Hasil kosong:

```text
No matches found.
```

Jika page lebih besar dari hasil, command tetap sukses dengan empty result dan informasi page yang konsisten.

## 4. JSON contract

```json
{
  "command": "find",
  "ok": true,
  "data": {
    "items": [],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "has_more": true,
      "next_page": 2
    }
  },
  "error": null
}
```

Halaman terakhir memakai:

```json
{
  "page": 2,
  "page_size": 20,
  "has_more": false,
  "next_page": null
}
```

`total` tidak wajib pada tahap awal. `has_more` dapat dihitung dengan mengambil `page_size + 1` record.

## 5. Query ordering

Urutan wajib diterapkan sebelum `LIMIT/OFFSET`:

```text
filters
→ deterministic sort
→ LIMIT page_size + 1
→ OFFSET (page - 1) * page_size
→ trim extra record
```

Sorting mengikuti Search Improvement:

- keyword: relevance DESC, updated_at DESC, mtime DESC, path ASC;
- empty/recent: updated_at DESC NULLS LAST, created_at DESC NULLS LAST, mtime DESC, path ASC;
- title sort: title ASC, path ASC.

## 6. TUI behavior

TUI menggunakan service search yang sama dan menampilkan:

```text
Page 1 · 20 results · More available
```

Keyboard minimum:

```text
n / right  next page
p / left   previous page
r          refresh current page
```

Halaman aktif dan filter harus dipertahankan saat refresh. Tidak boleh ada hasil yang tertutup footer.

## 7. Performance and consistency

- Gunakan `LIMIT/OFFSET` untuk MVP.
- Jangan wajib menjalankan `COUNT(*)` pada FTS query besar.
- Ambil `page_size + 1` untuk menentukan `has_more`.
- Query memakai SQLite read model; tidak menghitung hash filesystem runtime.
- Sorting harus deterministic agar perpindahan page stabil.
- `has_more` adalah indikator resmi; `total` tidak wajib pada MVP.
- Cursor pagination ditunda ke WO lanjutan jika dataset besar atau perubahan realtime menyebabkan offset drift.
- Perubahan data di antara request halaman dapat menyebabkan offset drift; dokumentasikan sebagai eventual consistency MVP.

## 8. Compatibility

- Tanpa `--page`, perilaku existing tetap mengembalikan page 1 dengan default page size.
- JSON envelope existing tetap dipertahankan, dengan `data` berubah menjadi object `items + pagination` secara versioned/backward-compatible policy.
- Filter `status`, `status-not`, tag, type, folder, open/closed, date, dan sort tetap bekerja.

## 9. Definition of Done

- CLI menampilkan informasi more-results yang jelas.
- JSON menyediakan `page`, `page_size`, `has_more`, dan `next_page`.
- Pagination diterapkan setelah filter dan sorting.
- Empty query dan TUI mendukung pagination.
- Page boundary, empty page, invalid options, dan max page size diuji.
- Existing search filters dan deterministic ordering tidak regresi.
- `go test ./...`, `go vet ./...`, build, dan `git diff --check` lulus.

## Status

Draft. Implementasi mengikuti WO 01 setelah SPEC disetujui.
