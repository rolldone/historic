---
id: "00008"
title: Historic Version Name, Version Code, and Compatibility
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [spec, versioning, version-name, version-code, sqlite, compatibility, rebuild]
related: [./wos/01-version-metadata-compatibility-and-rebuild.md]
---
# Historic Version Name, Version Code, and Compatibility SPEC

## 1. Tujuan

Mendefinisikan versioning aplikasi seperti Android dengan memisahkan versi untuk manusia dari angka versi untuk compatibility check. Database SQLite adalah read model/cache yang boleh dihapus dan dibangun ulang. Markdown dan `_meta.yaml` tetap source of truth.

## 2. Model versi

### Application version

```text
version_name = "0.3.0"
version_code = 3
```

- `version_name` adalah string semver untuk output manusia dan diagnostik.
- `version_code` adalah integer positif yang monoton naik untuk setiap release.
- `version_code` baru wajib lebih besar daripada code release sebelumnya.
- `version_code` tidak boleh dipakai sendirian untuk menentukan rebuild.

### Workspace/index version

```text
workspace_format_version = 1
index_schema_version = 2
```

- `workspace_format_version` mengatur kompatibilitas `.historic`, Markdown, dan `_meta.yaml`.
- `index_schema_version` mengatur kompatibilitas SQLite read model.
- Perubahan application version dengan schema sama tidak wajib menghapus SQLite.
- Perubahan index schema wajib memicu upgrade/rebuild atau migration yang eksplisit.

## 3. Metadata SQLite

Tambahkan tabel metadata internal:

```sql
CREATE TABLE historic_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

Key minimal:

```text
app_version_name
app_version_code
workspace_format_version
index_schema_version
built_at
binary_commit
```

Nilai versi disimpan sebagai text canonical. Parser wajib menolak nilai integer negatif, kosong, atau tidak valid.

## 4. Compatibility policy

| Kondisi | Tindakan |
|---|---|
| DB tidak ada | Rebuild dari Markdown dan `_meta.yaml` |
| Metadata DB tidak terbaca | Backup lalu rebuild |
| `version_code` sama, schema sama | Gunakan DB |
| `version_code` berubah, schema sama | Update metadata DB tanpa delete |
| `index_schema_version` berbeda | Backup, migrate/replace, lalu rebuild atomic |
| `workspace_format_version` berbeda dan migrasi tersedia | Jalankan migrasi workspace |
| Workspace version lebih baru dari binary | Tolak dengan remediation |
| DB corrupt atau schema invalid | Jangan gunakan; backup lalu rebuild |

Jangan menghapus database hanya karena `version_name` atau `version_code` berubah. Trigger teknis adalah compatibility index/workspace version.

## 5. Startup flow

```text
initialize workspace
  -> locate SQLite
  -> read historic_meta
  -> validate metadata and schema
  -> compare workspace/index versions
  -> compatible: open read model
  -> incompatible/missing/corrupt: backup and rebuild
  -> write current metadata after successful rebuild
```

Backup harus berada di path yang deterministic atau memiliki timestamp, tidak boleh ditimpa tanpa policy retention. Rebuild gagal harus mempertahankan backup dan tidak boleh menganggap database baru compatible.

## 6. `_meta.yaml` policy

`_meta.yaml` tidak dibuat ulang pada setiap update versi.

Reconcile hanya boleh:

- mempertahankan topic `id` numerik;
- mempertahankan `files[].id` UUIDv7;
- mempertahankan metadata manual `description`, `tags`, dan `related`;
- menambahkan file baru dengan UUIDv7;
- menghapus manifest entry untuk file yang benar-benar hilang;
- mempertahankan `path`, `type`, dan `status` selama file masih ada;
- menulis perubahan secara atomic.

Jika `_meta.yaml` tidak ada pada topic valid, buat metadata minimal dan generate manifest. Jika `_meta.yaml` invalid, jangan overwrite otomatis; tampilkan error actionable.

## 7. Release dan downgrade

- `version_code` harus disimpan di build metadata dan database.
- Binary yang lebih baru dapat membaca database lama jika schema compatible.
- Binary yang lebih lama tidak boleh membuka workspace format yang lebih baru tanpa migration support.
- Downgrade hanya boleh jika workspace/index versions compatible.
- Jika downgrade berisiko, backup index dan minta upgrade binary.
- `version_name` boleh berubah tanpa menaikkan schema, tetapi `version_code` tetap harus naik.

## 8. CLI/diagnostic contract

```text
historic version
historic doctor --json
historic rebuild --json
historic upgrade --json
```

`version` menampilkan `version_name` dan `version_code`. `doctor` menampilkan application, workspace, dan index versions serta status compatibility. JSON harus membedakan version name/code dari schema versions.

## 9. Safety and atomicity

- SQLite adalah cache dan selalu dapat direbuild.
- Source Markdown/_meta tidak boleh diganti demi menyesuaikan app version.
- Backup dilakukan sebelum delete/replace index.
- Temporary SQLite dan metadata harus dibersihkan setelah sukses.
- Rebuild memakai temp database lalu atomic replace.
- Lock mencegah dua proses upgrade/rebuild bersamaan.
- Failure injection test wajib memverifikasi rollback.

## 10. Definition of Done

- Version name dan version code tersedia pada binary dan output version/doctor.
- `historic_meta` tersimpan pada SQLite dan divalidasi.
- Perubahan app version dengan schema sama tidak menghapus DB.
- Perubahan index schema memicu policy upgrade/rebuild yang benar.
- DB missing/corrupt dapat dibangun ulang atomic.
- `_meta.yaml` direconcile, bukan dihapus/recreate sembarangan.
- Downgrade/workspace-incompatible menghasilkan error actionable.
- Test mencakup startup, upgrade, rebuild, backup, rollback, dan concurrency lock.
- `go test ./...`, `go vet ./...`, build, dan `git diff --check` lulus.

## Status

Implemented. Scope delivered by WO 01 in commit `a627618` (`feat: add version compatibility metadata`). Version name/code, storage version metadata, compatibility decisions, version/doctor output, and validation are complete. Future startup-path enhancements or broader failure-injection coverage require a new WO.
