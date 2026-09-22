---
id: "00007"
title: Historic Generated IDs with UUIDv7
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [spec, uuidv7, identifiers, metadata, work-order, migration]
related: [./wos/01-uuidv7-generated-identifiers.md]
---
# Historic Generated IDs with UUIDv7 SPEC

## 1. Tujuan

Mempertahankan ID numerik lima digit untuk topic, sekaligus memberikan UUIDv7 sebagai identifier canonical untuk setiap managed Historic file dan Work Order. User, AI, dan file template tidak boleh menulis ID file secara manual. UUIDv7 dipilih karena membawa komponen waktu dan tetap dapat diurutkan secara leksikografis berdasarkan waktu pembuatan.

## 2. Prinsip sumber ID

- `historic create` tetap menghasilkan ID numerik lima digit untuk topic.
- `historic add` menghasilkan UUIDv7 untuk setiap managed file atau Work Order.
- Caller hanya mengirimkan title/path; tidak ada ID file manual pada alur normal.
- ID topic tetap disimpan pada `_meta.yaml` sebagai identitas folder dan relasi.
- ID file hasil generator ditulis oleh aplikasi ke `_meta.yaml` sebagai metadata canonical.
- Markdown child tidak perlu memiliki field `id` yang diketik user.
- SQLite/FTS hanya menyimpan salinan rebuildable dari ID canonical.
- ID file tidak boleh dibuat dari nama file, nomor urut lokal, timestamp string, atau slug.

## 3. Bentuk identifier dan storage

Topic tetap menggunakan format numerik lima digit yang kompatibel dengan workspace saat ini:

```text
00001-topic-title
```

Managed Historic file menggunakan UUIDv7 canonical lowercase:

```text
xxxxxxxx-xxxx-7xxx-8xxx-xxxxxxxxxxxx
```

Slug topic hanya untuk keterbacaan. Rename slug tidak mengubah ID topic. UUIDv7 file tidak mengubah path logical file. Jika storage open dan closed memiliki topic dengan ID numerik yang sama, keduanya adalah dua representasi storage dari topic yang sama.

## 4. Canonical `_meta.yaml`

`_meta.yaml` menyimpan ID numerik topic dan ID UUIDv7 setiap managed member:

```yaml
id: "00007"
title: Historic Generated IDs with UUIDv7
created: "2026-09-22"
files:
  - id: 0192f3b5-1e20-7abc-8def-0123456789ab
    path: wos/01-uuidv7-generated-identifiers.md
    type: task
    status: planned
assets: []
```

`files[].id` adalah identitas Work Order/file, bukan ID topic. Path adalah logical path dan dapat berubah melalui lifecycle rename yang eksplisit.

## 5. Frontmatter Markdown

Frontmatter managed Markdown berisi metadata kerja seperti `title`, `status`, `created`, `updated`, `tags`, dan `related`. Field `id` tidak wajib diisi oleh user. Saat membaca workspace lama, ID numerik pada frontmatter diperlakukan sebagai format legacy file dan dimigrasikan secara deterministik ke UUIDv7 canonical manifest tanpa mengubah ID topic.

File Markdown tanpa frontmatter tetap asset. File dengan frontmatter valid di bawah topic dapat menjadi managed file berdasarkan manifest/path; kecocokan ID child dengan ID topic tidak boleh menjadi syarat klasifikasi.

## 6. Generator UUIDv7

- Gunakan generator UUIDv7 yang sesuai RFC 9562 hanya untuk managed Historic files.
- Topic tidak memakai UUIDv7 dan tetap menggunakan generator numerik yang sudah ada.
- Timestamp UUIDv7 berasal dari waktu UTC saat request diterima.
- Randomness harus cryptographically secure.
- Collision harus ditangani sebagai error dan tidak boleh menimpa file.
- Generator harus injectable untuk test deterministik.
- Clock rollback tidak boleh menghasilkan duplicate ID.
- Semua command yang membuat managed file memakai service generator yang sama.

## 7. Migration dan kompatibilitas

- Workspace dengan topic ID lima digit tetap menjadi format canonical topic.
- `rebuild` tidak boleh membuat ID baru untuk record file yang sudah memiliki ID canonical.
- Migrasi hanya memetakan ID managed file lama ke UUIDv7; ID topic tidak diubah.
- Migrasi harus memiliki dry-run, report mapping old-to-new, backup/rollback, dan idempotency.
- Rename folder topic berdasarkan slug tidak mengubah ID numerik topic.
- Referensi `related` harus dapat dipetakan dari ID file legacy ke UUIDv7.
- Invalid atau duplicate ID file harus menghentikan atomic rebuild dengan error actionable.

## 8. CLI contract

```text
historic create "Topic Title"
historic add "wos/work-order-title" --id 00007
historic rebuild --json
historic show 00007
```

Output JSON untuk `create` mengembalikan ID topic numerik. Output JSON untuk `add` mengembalikan UUIDv7 file yang dibuat aplikasi. Opsi ID file manual tidak tersedia pada alur normal.

## 9. Definition of Done

- Topic baru tetap mendapatkan ID numerik lima digit.
- Work Order/file baru mendapatkan UUIDv7 dari generator aplikasi.
- ID topic dan ID member disimpan pada `_meta.yaml` dengan domain masing-masing.
- User tidak perlu menulis ID file pada template atau command normal.
- Child Work Order dengan ID file berbeda dari topic tetap diklasifikasikan sebagai managed file.
- Rebuild, open, close, delete, search, dan FTS mempertahankan ID topic serta ID file.
- Legacy file ID dapat dimigrasikan secara atomic dan idempotent tanpa mengubah ID topic.
- Test mencakup format, ordering, collision, clock rollback, restart process, migration, dan crash recovery.
- `go test ./...`, `go vet ./...`, build, serta `git diff --check` lulus.

## 10. Status

Implemented — Follow-up Required.

Core UUIDv7 managed-file identifiers are implemented and committed in `8f59db2`. Remaining Definition of Done items are legacy migration (dry-run and idempotency), CLI documentation, and Historic skill/command contract updates.
