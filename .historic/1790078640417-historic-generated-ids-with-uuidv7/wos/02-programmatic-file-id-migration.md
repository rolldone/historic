---
id: "00007"
title: WO 02 Programmatic Legacy FileID Migration
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [migration, uuidv7, file-id, metadata, atomic, idempotent]
related: [../spec.md, ./01-uuidv7-generated-identifiers.md]
---
# WO 02 — Programmatic Legacy FileID Migration

## 1. Tujuan

Menyediakan migrasi resmi di binary Historic untuk menambahkan UUIDv7 ke `files[]` pada `_meta.yaml` legacy. Migrasi harus memperbarui metadata secara programmatic, atomic, aman diulang, dan tidak membutuhkan script manual.

## 2. Batasan perubahan

- ID topic numerik lima digit tidak boleh berubah.
- Isi Markdown child tidak boleh berubah.
- `path`, `type`, dan `status` existing harus dipertahankan.
- Entry `files[]` yang sudah memiliki UUIDv7 tidak boleh digenerate ulang.
- `assets[]` tidak diberi FileID.
- Mapping identity menggunakan `(topic_id, logical_path)`.
- Migrasi tidak melakukan perubahan ke `PRD.md`, remote repository, atau source Markdown.

## 3. CLI contract

Tambahkan command:

```text
historic migrate-file-ids --dry-run
historic migrate-file-ids --json --dry-run
historic migrate-file-ids
historic migrate-file-ids --json
```

Flags:

| Flag | Perilaku |
|---|---|
| `--dry-run` | Scan dan tampilkan rencana tanpa menulis file |
| `--json` | Output envelope JSON standar |
| `--open` | Batasi topic open |
| `--closed` | Batasi topic closed |
| `--id <topic-id>` | Batasi satu topic numerik |
| `--force` | Tidak diperlukan; tolak jika dipakai |

Jika tidak ada entry legacy, command sukses dengan `updated: 0` dan tidak menyentuh metadata.

## 4. Output JSON

Gunakan envelope existing:

```json
{
  "command": "migrate-file-ids",
  "ok": true,
  "data": {
    "dry_run": false,
    "topics_scanned": 8,
    "manifests_updated": 1,
    "files_migrated": 3,
    "files_preserved": 54,
    "mappings": [
      {
        "topic_id": "00002",
        "path": "wos/01-fts-search.md",
        "legacy_id": "00001",
        "file_id": "0192f3b5-1e20-7abc-8def-0123456789ab",
        "action": "assign"
      }
    ]
  },
  "error": null
}
```

Human output harus menampilkan jumlah topic, manifest, migrated, preserved, skipped, dan error secara ringkas.

## 5. Service API

Buat service terpisah dari command layer:

```go
type FileIDMigrationOptions struct {
    DryRun bool
    TopicID *domain.ID
    Storage *domain.Storage
}

type FileIDMigrationReport struct {
    TopicsScanned int
    ManifestsUpdated int
    FilesMigrated int
    FilesPreserved int
    FilesSkipped int
    Mappings []FileIDMapping
}

type FileIDMapping struct {
    TopicID domain.ID
    Path string
    LegacyID string
    FileID domain.FileID
    Action string // assign, preserve, skip
}

func (service Service) MigrateFileIDs(options FileIDMigrationOptions) (FileIDMigrationReport, error)
```

Service harus dapat menerima generator UUIDv7 injectable untuk test. Command hanya melakukan parse flags, resolve workspace, memanggil service, dan serialisasi output.

## 6. Algoritme migration

### Phase A — Scan read-only

1. Discover workspace `.historic`.
2. Enumerate topic open dan closed.
3. Group duplicate storage berdasarkan topic ID.
4. Parse setiap `_meta.yaml`.
5. Validasi topic ID numerik.
6. Validasi path, duplicate path, duplicate existing FileID, dan overlap files/assets.
7. Untuk setiap `files[]`:
   - UUIDv7 valid → `preserve`;
   - ID kosong → `assign`;
   - ID legacy numerik → `assign` dengan `legacy_id` pada report;
   - format invalid → error, jangan menulis apa pun.
8. Jika open dan closed ada bersamaan, pilih canonical storage sesuai policy existing dan pastikan mapping `(topic_id, path)` konsisten.

### Phase B — Plan

1. Generate FileID hanya untuk entry yang membutuhkan.
2. Pastikan semua generated ID unik terhadap seluruh workspace dan mapping dalam run.
3. Susun rencana per metadata file.
4. `--dry-run` berhenti di sini dan tidak menulis file.
5. JSON mapping harus deterministic dalam ordering topic ID lalu path.

### Phase C — Commit atomic

1. Acquire workspace migration lock.
2. Re-scan metadata setelah lock untuk mencegah lost update.
3. Tulis semua manifest baru ke temporary file pada directory yang sama.
4. `fsync` setiap temporary file.
5. Rename temporary files secara atomic.
6. Jika commit multi-manifest gagal, restore backup setiap manifest yang sudah ter-rename.
7. Release lock.
8. Jangan rebuild SQLite secara partial; caller menjalankan rebuild setelah migration sukses.

## 7. Idempotency dan recovery

- Rerun setelah sukses menghasilkan `updated: 0` untuk entry yang sama.
- Rerun setelah crash menggunakan manifest yang sudah committed dan tidak mengganti ID.
- Temporary file dibersihkan saat startup atau akhir command.
- Existing UUIDv7 tidak pernah diganti hanya karena mtime atau content berubah.
- Jika metadata invalid, return error dengan path, field, dan remediation.
- Jika satu topic gagal, default behavior adalah fail-fast tanpa commit seluruh batch.
- Backup/rollback wajib mempertahankan metadata sebelum command.

## 8. Kompatibilitas dengan rebuild

Setelah migrasi sukses:

```text
historic rebuild --json
historic doctor --json
```

Rebuild harus:

- membaca FileID yang baru ditulis;
- tidak generate ulang FileID;
- mempertahankan ID berdasarkan `(topic_id, logical_path)`;
- meng-index `files.id` sebagai TEXT;
- mengklasifikasikan Markdown valid di `wos/` sebagai `historic_file`;
- mempertahankan index lama bila rebuild gagal.

`rebuild` boleh memiliki internal helper yang sama, tetapi command migration tetap diperlukan untuk dry-run dan report mapping.

## 9. Test matrix

### Unit

- empty ID → assign;
- UUIDv7 valid → preserve;
- legacy numeric ID → assign dan report;
- invalid UUID → error;
- duplicate FileID → error;
- duplicate path → error;
- deterministic ordering report;
- injected generator collision.

### Integration

- dry-run tidak mengubah byte metadata;
- migration mengubah hanya `files[].id`;
- Markdown unchanged dengan hash sebelum/sesudah sama;
- topic ID unchanged;
- rerun menghasilkan `updated: 0`;
- crash/rollback tidak meninggalkan partial metadata;
- open/closed duplicate storage memakai mapping yang sama;
- rebuild sukses setelah migration;
- doctor berstatus `compatible`;
- JSON dan human output stabil.

## 10. Acceptance criteria

- Tidak ada lagi kebutuhan script Python/manual untuk migrasi FileID.
- Semua manifest legacy dapat diproses melalui command resmi.
- Dry-run dapat direview sebelum perubahan.
- Commit metadata atomic dan rollback teruji.
- Migrasi idempotent dan crash-safe.
- Topic numerik tetap kompatibel.
- `go test ./...`, `go vet ./...`, build, dan `git diff --check` lulus.

## 11. Urutan implementasi

1. Tambahkan tipe options/report/mapping.
2. Implementasikan scanner dan validator read-only.
3. Implementasikan planning dengan generator injectable.
4. Implementasikan dry-run dan JSON output.
5. Implementasikan atomic commit, lock, backup, dan rollback.
6. Tambahkan filter topic/storage.
7. Tambahkan integration tests dan rebuild verification.
8. Update docs/skill dan command help.

## Status

Complete. Implemented and committed in `1e89ed2` (`feat: add programmatic file ID migration`). Verified with dry-run, idempotent migration, rebuild, doctor compatibility, tests, vet, build, and diff check.
