---
id: "00010"
title: Historic Timestamp Topic Identity and Serialized Allocation
status: draft
created: "2026-09-22"
updated: "2026-09-22"
tags: [spec, topic-id, timestamp, millisecond, serialization, migration]
related: [./wos/01-millisecond-topic-id-serialization.md]
---
# Historic Timestamp Topic Identity SPEC

## 1. Tujuan

Mengganti ID topic numerik berurutan dengan ID berbasis Unix epoch millisecond yang monotonic, sortable, dan aman terhadap dua request `create` pada millisecond yang sama. Managed file tetap menggunakan UUIDv7 sesuai kebijakan topic sebelumnya.

## 2. Model ID

Topic baru menggunakan format canonical:

```text
<unix-millisecond>-<slug>
```

Contoh:

```text
1758537600123-search-read-model
```

ID topic adalah bagian numeric sebelum `-` dan harus berupa integer positif 13 digit pada masa sekarang. Slug hanya untuk keterbacaan dan tidak menjadi identity.

Managed file tetap:

```text
UUIDv7
```

Perbedaan domain wajib dipertahankan: `TopicID` timestamp dan `FileID` UUIDv7 tidak boleh memakai parser atau kolom yang sama.

## 3. Allocation policy

Semua pembuatan topic melewati allocator serialized:

```text
acquire inter-process lock
  -> read last allocated topic ID from filesystem/index state
  -> now_ms = UTC Unix millisecond
  -> allocated_ms = max(now_ms, last_id + 1)
  -> verify no topic folder uses allocated_ms
  -> create topic folder and _meta.yaml atomically
  -> release lock
```

Jika dua request masuk pada millisecond yang sama, request berikutnya mendapat `last_id + 1`. Tidak perlu menunggu clock berubah.

## 4. Uniqueness and clock safety

- Lock berlaku antar-process, bukan hanya in-process mutex.
- Lock file berada di `.historic/.topic-id.lock` atau mekanisme equivalent.
- Setelah lock diperoleh, filesystem harus di-scan ulang.
- Clock rollback tidak boleh menghasilkan ID lebih kecil dari ID terakhir.
- Folder existing selalu memicu increment dan retry.
- Crash sebelum commit boleh meninggalkan gap; gap tidak masalah.
- ID yang sudah committed tidak boleh dipakai ulang.
- Atomic directory/file creation mencegah partial topic.
- Generator harus injectable untuk test.

## 5. Canonical metadata

Topic `_meta.yaml` menyimpan timestamp ID sebagai string:

```yaml
id: "1758537600123"
title: Search Read Model
created: "2026-09-22"
files: []
assets: []
```

`files[].id` tetap UUIDv7. Rebuild tidak boleh mengubah TopicID hanya karena app version, mtime, atau slug berubah.

## 6. Migration

Legacy topic `00001-topic` dipetakan ke timestamp TopicID baru melalui migration map:

```text
legacy_topic_id + canonical_slug/path -> timestamp_topic_id
```

Aturan:

- migration harus dry-run, atomic, idempotent, dan rollback-safe;
- isi topic dan `_meta.yaml` manual dipertahankan;
- referensi topic lama dapat di-resolve melalui alias map;
- dua storage open/closed untuk topic yang sama memakai TopicID yang sama;
- duplicate identity atau invalid timestamp menghentikan migration;
- tidak ada silent overwrite folder.

## 7. Compatibility

Naikkan `workspace_format_version` karena parser dan folder identity berubah. `index_schema_version` naik jika kolom SQLite atau type TopicID berubah. Binary lama harus menolak workspace format baru dengan remediation, bukan mengubah folder secara diam-diam.

## 8. CLI contract

```text
historic create "Topic Title"
historic show <timestamp-topic-id>
historic list
historic rebuild --json
historic doctor --json
historic migrate-topic-ids --dry-run
```

Output `create` harus mengembalikan TopicID timestamp yang dibuat. `--id` manual tidak tersedia pada alur normal kecuali migration/admin mode eksplisit.

## 9. Definition of Done

- TopicID timestamp millisecond tervalidasi terpisah dari FileID UUIDv7.
- Concurrent create menghasilkan ID unik dan monotonic.
- Lock inter-process, retry, collision, rollback, dan crash recovery teruji.
- Rebuild mempertahankan TopicID dan manifest FileID.
- Legacy topic dapat dimigrasikan dry-run dan idempotent.
- List/show/open/close/delete/search/find mendukung TopicID baru.
- Version/workspace compatibility mencegah binary lama merusak workspace baru.
- `go test ./...`, `go vet ./...`, build, dan `git diff --check` lulus.

## Status

Draft. Implementasi mengikuti WO 01 setelah SPEC disetujui.
