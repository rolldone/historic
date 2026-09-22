---
id: "00008"
title: WO 01 Version Metadata, Compatibility, and Rebuild
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [versioning, version-code, sqlite, compatibility, rebuild, atomic, migration]
related: [../spec.md]
---
# WO 01 — Version Metadata, Compatibility, and Rebuild

## 1. Keputusan final

Implementasikan dua versi aplikasi dan dua versi storage:

| Field | Type | Tujuan |
|---|---|---|
| `VersionName` | string | Versi manusia, contoh `0.3.0` |
| `VersionCode` | positive integer | Urutan release, wajib naik |
| `WorkspaceFormatVersion` | integer | Kompatibilitas `.historic` dan `_meta.yaml` |
| `IndexSchemaVersion` | integer | Kompatibilitas SQLite |

`VersionCode` berubah pada setiap release. Rebuild tidak dipicu oleh perubahan `VersionName`/`VersionCode` saja; trigger teknis adalah schema/workspace incompatibility, database missing, atau database corruption.

## 2. Konstanta dan build metadata

Tambahkan konfigurasi terpusat:

```go
const (
    VersionName = "0.3.0"
    VersionCode = 3
    WorkspaceFormatVersion = 1
    IndexSchemaVersion = 2
)
```

Jika commit/build override diperlukan, gunakan `-ldflags` tanpa mengubah fallback compile-time. Semua output command harus menggunakan sumber versi yang sama.

Validasi startup:

- VersionName tidak kosong dan format semver valid.
- VersionCode > 0.
- WorkspaceFormatVersion > 0.
- IndexSchemaVersion > 0.

## 3. Schema SQLite

Tambahkan tabel ke schema read model:

```sql
CREATE TABLE historic_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

Insert/update key wajib dilakukan dalam transaction yang sama dengan index build:

```text
BEGIN
  create topics/files/fts tables
  create historic_meta
  insert current metadata
  validate counts and versions
COMMIT
```

Required keys:

```text
app_version_name
app_version_code
workspace_format_version
index_schema_version
built_at
binary_commit
```

`files.id` tetap `TEXT` untuk UUIDv7. `historic_meta` bukan source of truth dan harus dapat direcreate dari binary + build process.

## 4. Compatibility service

Buat service yang mengembalikan keputusan eksplisit:

```go
type CompatibilityAction string

const (
    ActionUseExisting CompatibilityAction = "use_existing"
    ActionUpdateAppMeta CompatibilityAction = "update_app_meta"
    ActionRebuild CompatibilityAction = "rebuild"
    ActionMigrateWorkspace CompatibilityAction = "migrate_workspace"
    ActionReject CompatibilityAction = "reject"
)

type IndexMetadata struct {
    AppVersionName string
    AppVersionCode int
    WorkspaceFormatVersion int
    IndexSchemaVersion int
    BuiltAt time.Time
    BinaryCommit string
}

type CompatibilityDecision struct {
    Action CompatibilityAction
    Reason string
    BackupRequired bool
}

func CheckCompatibility(current AppVersion, stored IndexMetadata) CompatibilityDecision
```

Decision table implementation:

- missing/corrupt metadata → `rebuild`, backup required;
- index schema mismatch → `rebuild`, backup required;
- workspace version greater than supported → `reject`;
- workspace version lower and migration available → `migrate_workspace`;
- schema/workspace equal but app version differs → `update_app_meta`;
- all equal → `use_existing`.

Do not compare version strings lexicographically for compatibility decisions.

## 5. Startup integration

Pada workspace initialization/query path:

1. Discover `.historic`.
2. Acquire index lock.
3. Open SQLite read-only jika metadata tersedia.
4. Parse and validate `historic_meta`.
5. Call `CheckCompatibility`.
6. `use_existing`: release lock and continue.
7. `update_app_meta`: transaction update app fields, release lock.
8. `rebuild`: create backup, build temporary SQLite from Markdown/metadata, validate, atomic replace.
9. `migrate_workspace`: call explicit migration service, then rebuild.
10. `reject`: leave source and index untouched, return actionable error.

Avoid deleting the only index before the replacement index is valid. “Delete and rebuild” is implemented as backup plus atomic replacement.

## 6. Backup and rollback

- Backup existing index before replacement using non-conflicting path.
- Store schema/version metadata with backup report.
- Build to `.index.sqlite.tmp-*` in the same directory.
- Validate SQLite integrity, schema version, row counts, and FTS availability.
- Rename current index to backup only immediately before final replacement.
- Rename temp index to `.index.sqlite` atomically.
- On failure, restore backup and remove temp files.
- Never delete `_meta.yaml` as part of index version handling.

## 7. `_meta.yaml` reconcile contract

Expose one reconcile service used by rebuild/migration:

```go
type ReconcileOptions struct {
    GenerateMissingFileIDs bool
    PreserveManualMetadata bool
    DryRun bool
}

func ReconcileTopicMetadata(root string, options ReconcileOptions) (Report, error)
```

Rules:

- Preserve numeric topic ID.
- Preserve existing valid FileID by `(topic_id, logical_path)`.
- Generate UUIDv7 only when enabled and FileID missing.
- Preserve Markdown bytes and manual metadata.
- Atomic YAML write and rollback.
- Invalid YAML fails without overwrite.
- Missing metadata may be recovered only for a valid topic folder.

## 8. CLI changes

`historic version` output:

```text
Historic 0.3.0 (version code 3)
workspace format 1, index schema 2
```

`historic version --json`:

```json
{
  "version_name": "0.3.0",
  "version_code": 3,
  "workspace_format_version": 1,
  "index_schema_version": 2
}
```

`historic doctor --json` harus menampilkan stored/current values dan compatibility action. Error harus menyebut command remediation, misalnya `historic rebuild` atau upgrade binary.

## 9. Test plan

### Unit

- semver validation;
- positive/monotonic version code;
- metadata parse and invalid values;
- every compatibility decision;
- no lexicographic version comparison;
- reconcile preserve/generate/dry-run.

### Integration

- first build writes `historic_meta`;
- same app/schema reuses DB;
- app version change with same schema updates metadata without deleting DB;
- schema change backs up and rebuilds;
- missing DB rebuilds;
- corrupt DB backup/rebuilds;
- workspace newer rejects without source changes;
- failed rebuild restores backup;
- concurrent startup has one rebuild owner;
- `_meta.yaml` FileIDs remain stable.

### Quality gate

```text
go test ./...
go vet ./...
go build .
git diff --check
```

## 10. Implementation order

1. Centralize version constants/build metadata.
2. Add `historic_meta` schema and transaction writes.
3. Implement metadata parser and compatibility decision service.
4. Integrate startup/doctor/version output.
5. Integrate backup/temp/atomic rebuild.
6. Extract `_meta.yaml` reconcile service.
7. Add migration hooks and failure recovery.
8. Add unit/integration/concurrency tests.
9. Update docs and Historic skill.

## 11. Acceptance criteria

- App version, code, workspace version, and schema version are visible and distinct.
- Version code comparison is numeric and policy-driven.
- App version update does not unnecessarily delete SQLite.
- Schema/workspace incompatibility triggers safe rebuild/migration.
- No source metadata is deleted/recreated merely due to app update.
- Backup and rollback work under injected failure.
- Doctor accurately reports action and remediation.
- Full quality gate passes.

## Status

Complete. Implemented and committed in `a627618` (`feat: add version compatibility metadata`). Verified with version output, JSON version output, doctor compatibility, tests, vet, build, and diff check.
