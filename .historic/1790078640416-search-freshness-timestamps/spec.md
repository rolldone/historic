---
id: 00006
title: Search Freshness and Timestamp Ordering SPEC
status: draft
created: 2026-09-22
updated: 2026-09-22
tags: [spec, search, timestamps, freshness, sorting, rebuild, ai]
related: [./wos/01-file-freshness-search-ordering.md]
---
# Search Freshness and Timestamp Ordering SPEC

## 1. Tujuan

Mendefinisikan kontrak teknis untuk freshness read model dan urutan hasil search agar AI dapat membedakan file baru, file yang baru diperbarui, dan file lama secara deterministic.

## 2. Sumber timestamp

### Managed historic file

```text
created_at = frontmatter.created
updated_at = frontmatter.updated jika ada
mtime      = filesystem modification time UTC RFC3339Nano
hash       = SHA-256 content
size       = filesystem size
```

### Asset

```text
created_at = null
updated_at = null
mtime      = filesystem modification time UTC RFC3339Nano
hash       = SHA-256 content
size       = filesystem size
```

`rebuild` tidak boleh mengubah frontmatter hanya karena content atau `mtime` berubah. Timestamp manual tetap source of truth; mtime hanya metadata teknis read model.

## 3. Change detection

Logical identity file:

```text
topic_id + relative POSIX path
```

Perubahan:

| Kondisi | Read model |
|---|---|
| File baru | insert `files` dan FTS |
| Hash berubah | update content, hash, size, mtime, FTS |
| Frontmatter berubah | update title/description/status/tags/dates |
| Path berubah | remove old logical record, insert new path |
| Mtime berubah, hash sama | update mtime saja; content unchanged |
| File hilang | remove record, manifest, dan FTS |
| Asset berubah | update asset metadata; tidak memengaruhi status aggregate |

Rebuild membangun ulang secara atomic dari filesystem sehingga hasil akhir tidak bergantung pada state index lama.

## 4. Search ordering

### Keyword query

```text
relevance score DESC
updated_at DESC
mtime DESC
path ASC
```

### Empty query/recent browser

```text
updated_at DESC NULLS LAST
created_at DESC NULLS LAST
mtime DESC
path ASC
```

### Aggregate topic

```text
last_file_updated_at DESC NULLS LAST
updated_at DESC NULLS LAST
path ASC
```

Path ascending adalah tie-breaker final agar output deterministic.

## 5. CLI contract

Existing query tetap tersedia:

```text
historic find "keyword"
historic search
```

Optional sort:

```text
--sort relevance
--sort updated
--sort created
--sort title
```

Default:

- keyword ada: `relevance`;
- keyword kosong: `updated`;
- invalid sort ditolak dengan error actionable;
- filters open/closed/status/tags tetap dapat dikombinasikan.

Status filters:

```text
--status <status[,status...]>
--status-not <status[,status...]>
```

- `--status` memilih status yang diizinkan.
- `--status-not` mengecualikan satu atau beberapa status.
- Nilai dipisahkan koma dan setiap nilai divalidasi terhadap daftar status resmi.
- Kedua filter boleh digabungkan selama tidak memiliki status yang sama.
- Status yang muncul di `--status` dan `--status-not` harus ditolak dengan error actionable agar query tidak ambigu.
- Tanpa filter status, semua status dikembalikan.
- Contoh backlog:

```text
historic find "work" --status-not complete,cancelled,failed,archived
historic find "migration" --status draft,progress
```

Filter negatif harus diterapkan pada managed file dan topic result secara konsisten. Field `status` tetap dikembalikan pada human dan JSON output.

## 6. JSON contract

File result minimal:

```json
{
  "type": "historic_file",
  "path": "wos/task.md",
  "created_at": "2026-09-21",
  "updated_at": "2026-09-22",
  "mtime": "2026-09-22T08:12:33.000000000Z",
  "hash": "sha256..."
}
```

Topic aggregate minimal:

```json
{
  "topic_id": "00006",
  "updated_at": "2026-09-22",
  "last_file_updated_at": "2026-09-22",
  "computed_status": "progress"
}
```

JSON harus membedakan timestamp manual (`created_at`, `updated_at`) dan timestamp teknis (`mtime`).

## 7. Performance

- Rebuild boleh membaca hash penuh karena hasilnya source-of-truth accurate.
- Query tidak boleh menghitung hash filesystem saat runtime.
- SQLite index menyediakan index SQL pada `created_at`, `updated_at`, `mtime`, `type`, `status`, dan `storage` jika diperlukan.
- Pagination/limit diterapkan setelah ordering deterministic.
- Benchmark mencakup empty query, keyword query, filter tanggal, dan aggregate.

## 8. Safety

- Tidak ada automatic frontmatter timestamp mutation.
- Invalid timestamp tetap menghasilkan error actionable sesuai parser policy.
- Invalid file tidak boleh mengganti index lama secara partial.
- Rebuild temporary/atomic tetap digunakan.
- `PRD.md` tidak diubah.

## 9. Definition of Done

- Timestamp field tersedia pada `files` dan hasil query.
- Freshness ordering sesuai kontrak.
- `--sort` tervalidasi dan diuji.
- Change detection hash/mtime/path memiliki regression test.
- Rebuild kedua idempotent.
- Open/closed search tetap bekerja.
- Isolated smoke test membuktikan file baru dan file yang berubah muncul lebih dahulu.
- `go test ./...`, `go vet ./...`, build, dan `git diff --check` lulus.

## 10. Status

Draft. Implementasi harus mengikuti WO 01 setelah SPEC disetujui.
