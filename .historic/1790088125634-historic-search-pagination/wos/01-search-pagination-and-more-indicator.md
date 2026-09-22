---
id: "1790088125634"
title: WO 01 Search Pagination and More Results Indicator
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [search, pagination, cli, tui, json, sqlite, testing, implementation]
related: [../spec.md]
---
# WO 01 — Search Pagination and More Results Indicator

## 1. Tujuan teknis

Tambahkan pagination pada shared search service sehingga CLI `find`, JSON output, dan TUI dapat menyatakan apakah hasil lebih dari satu halaman secara konsisten. Implementasi tidak boleh mengubah source Markdown atau menghitung freshness dari filesystem saat query.

## 2. Scope dan file target

Area utama yang perlu diperiksa/diperbarui:

```text
internal/cmd/find.go
internal/cmd/find_test.go
internal/search/search.go
internal/search/search_test.go
internal/search/search_benchmark_test.go
internal/query/query.go
internal/query/query_test.go
internal/tui/search.go
internal/tui/search_test.go
internal/indexer/*_test.go
internal/config/* bila perlu untuk output contract/version
README.md
docs/commands.md
.agents/skills/historic/SKILL.md
```

Jangan mengubah `PRD.md`. Gunakan shared search service; jangan membuat query SQLite baru langsung dari layer TUI.

## 3. API contract

Tambahkan/ubah tipe pada package search:

```go
type Options struct {
    Keyword       string
    Status        domain.Status
    StatusNot     []domain.Status
    Tags          []string
    Folder        string
    Type          string
    ID            domain.TopicID
    OpenOnly      bool
    ClosedOnly    bool
    CreatedAfter  string
    CreatedBefore string
    UpdatedAfter  string
    UpdatedBefore string
    Sort          SortMode
    Page          int
    PageSize      int
}

type SortMode string

const (
    SortRelevance SortMode = "relevance"
    SortUpdated   SortMode = "updated"
    SortCreated   SortMode = "created"
    SortTitle     SortMode = "title"
)

type Pagination struct {
    Page     int  `json:"page"`
    PageSize int  `json:"page_size"`
    HasMore  bool `json:"has_more"`
    NextPage *int `json:"next_page"`
}

type SearchPage struct {
    Items      []Result   `json:"items"`
    Pagination Pagination `json:"pagination"`
}
```

Jika project sudah memiliki `StatusNot` atau `SortMode`, gunakan tipe existing dan hindari duplikasi. `Page=0` dan `PageSize=0` dari caller di-normalisasi menjadi default `1` dan `20`; nilai negatif ditolak. Maximum `PageSize=100`.

## 4. Query service implementation

### 4.1 Validation sebelum database

Buat helper teruji, misalnya:

```go
func NormalizePagination(page, pageSize int) (Pagination, error)
```

Validasi:

- page minimum 1;
- page size default 20;
- page size minimum 1;
- page size maksimum 100;
- cegah overflow pada `(page-1)*pageSize`;
- reject sort tidak dikenal;
- reject conflict status/status-not sebelum query.

Error harus actionable dan tidak membuka database jika options invalid.

### 4.2 Predicate dan sorting

Urutan query wajib:

```text
parse keyword/FTS
→ build SQL predicates + args
→ apply status/status-not/open/closed/tag/type/date filters
→ apply deterministic ORDER BY
→ apply LIMIT/OFFSET
```

Jangan interpolasi nilai user langsung ke SQL. Gunakan placeholder args.

ORDER BY:

```text
relevance: rank ASC atau score DESC sesuai konvensi FTS existing,
           lalu updated_at DESC NULLS LAST,
           mtime DESC,
           path ASC
updated:   updated_at DESC NULLS LAST,
           created_at DESC NULLS LAST,
           mtime DESC,
           path ASC
created:   created_at DESC NULLS LAST,
           updated_at DESC NULLS LAST,
           mtime DESC,
           path ASC
title:     lower(title) ASC,
           path ASC
```

Gunakan konvensi score existing dan pastikan arah relevance tidak terbalik. Tie-breaker `path ASC` wajib final.

### 4.3 LIMIT/OFFSET

- Hitung `offset = (page - 1) * pageSize` setelah overflow check.
- Bind `LIMIT pageSize + 1`.
- Bind `OFFSET offset`.
- Baca maksimal pageSize+1.
- Jika row ekstra ada, set `HasMore=true`, buang row ekstra dari Items.
- `NextPage = page + 1` hanya jika HasMore; selain itu `nil`.
- Jangan menjalankan `COUNT(*)` sebagai requirement MVP.
- Empty query memakai jalur recent query dan pagination yang sama.

Jika tidak ada item pada page > 1, return sukses dengan `items: []`, `has_more: false`, `next_page: null`; jangan error.

### 4.4 Freshness result

Pertahankan field read model existing pada `Result`/file result:

```go
CreatedAt string `json:"created_at"`
UpdatedAt string `json:"updated_at,omitempty"`
Mtime     string `json:"mtime"`
Hash      string `json:"hash"`
Size      int64  `json:"size"`
```

Nilai dibaca dari SQLite. Tidak boleh `os.Stat`, hash, atau parse Markdown pada runtime search.

## 5. CLI implementation

Update `internal/cmd/find.go`:

```text
--status string
--status-not string
--sort string
--page int
--page-size int
--json
```

Gunakan parser comma-separated yang sama untuk `status` dan `status-not`. Parse status dengan `domain.ParseStatus`. Jika status sama masuk inclusion dan exclusion, return error.

JSON envelope wajib:

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

Human output:

```text
<existing result lines>
Page 1 · showing 20 results · more results available
Use --page 2 --page-size 20
```

Halaman terakhir:

```text
Page 2 · showing 6 results · no more results
```

Saat tidak ada hasil, pertahankan `No matches found.` lalu tampilkan ringkasan page jika sesuai contract. AI/automation wajib memakai `--json`.

## 6. Backward compatibility JSON

JSON `data` berubah dari array result menjadi object `items + pagination`. Karena ini perubahan contract:

- dokumentasikan breaking/contract version pada `docs/commands.md` dan skill;
- update semua test consumer internal;
- jika ada consumer legacy yang harus dipertahankan, sediakan explicit legacy mode atau command version, jangan mengubah shape secara diam-diam;
- error tetap memakai envelope `{command, ok, data, error}`;
- output JSON harus deterministic dan tidak mencampur log ke stdout.

## 7. TUI implementation

`internal/tui/search.go` hanya memanggil shared search service.

State minimum:

```go
type searchModel struct {
    query    string
    page     int
    pageSize int
    pageData search.SearchPage
}
```

Keyboard:

```text
n / right arrow → next page jika HasMore
p / left arrow  → previous page jika page > 1
r               → refresh page aktif
```

Requirements:

- query, filters, sort, dan page dipertahankan saat refresh;
- next/previous tidak keluar batas;
- footer menampilkan `Page N · showing M results · More available/no more results`;
- loading/error state tidak merender stale page sebagai page baru;
- viewport/footer safety existing tetap dipertahankan;
- tidak mengakses SQLite langsung dari presentation layer.

## 8. Test matrix

### Unit search

- default page/page size;
- page 1 dan page 2;
- exact multiple page size;
- one extra row menghasilkan HasMore;
- last page HasMore false;
- empty page valid;
- next_page nil/present;
- page zero/negative;
- page size zero/negative/above 100;
- offset overflow;
- invalid sort;
- status/status-not combination;
- deterministic tie-breaker.

### CLI

- help menampilkan page flags;
- human more-results line;
- human last-page line;
- JSON object shape;
- JSON error envelope;
- empty keyword;
- filters plus pagination;
- no logs on stdout.

### Integration/read model

- keyword FTS pagination;
- empty/recent pagination;
- topic/file result pagination;
- open/closed pagination;
- status/status-not pagination;
- freshness sort pagination;
- FileID/freshness fields preserved;
- deleted file absent after rebuild;
- FTS and SQLite results consistent.

### TUI

- next/previous page;
- refresh current page;
- footer metadata;
- no rendering under footer;
- error/loading state.

### Performance

- benchmark page 1 and deep page;
- verify no filesystem hash/stat in query path;
- verify no mandatory COUNT query.

## 9. Acceptance criteria

- `historic find "work" --page 1 --page-size 20` menghasilkan output valid.
- Page dengan row ekstra menampilkan `has_more=true` dan `next_page`.
- Page terakhir menampilkan `has_more=false` dan `next_page=null`.
- Human output menyatakan apakah masih ada hasil.
- JSON AI contract menggunakan `data.items` dan `data.pagination`.
- Filter/sort diterapkan sebelum pagination.
- Empty query dan TUI memakai pagination yang sama.
- Existing FileID, timestamp, status, open/closed, dan FTS tidak regresi.
- Offset drift didokumentasikan sebagai eventual consistency MVP.
- Quality gate lulus:

```text
go test ./...
go vet ./...
go build .
git diff --check
```

## 10. Non-goals

- Cursor pagination belum diperlukan pada MVP.
- Total count tidak wajib.
- Tidak ada remote search backend.
- Tidak mengubah Markdown source.
- Tidak mengubah `PRD.md`.

## Status

Complete. Implementasi dan quality gate telah selesai.

Commit: 87cfe42
