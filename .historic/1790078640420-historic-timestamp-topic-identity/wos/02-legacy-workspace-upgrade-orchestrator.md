---
id: "1790078640420"
title: WO 02 Legacy Workspace Upgrade Orchestrator
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [legacy, upgrade, migration, topic-id, file-id, compatibility, rollback]
related: [../spec.md, ./01-millisecond-topic-id-serialization.md]
---
# WO 02 — Legacy Workspace Upgrade Orchestrator

## 1. Tujuan

Menyediakan satu command upgrade resmi agar user dengan workspace legacy tetap dapat naik ke format modern tanpa menjalankan migrasi FileID dan TopicID secara manual. Upgrade harus aman, dapat direview melalui dry-run, atomic, idempotent, dan rollback-safe.

## 2. Masalah yang diselesaikan

Workspace legacy dapat memiliki:

```text
.historic/00001-topic/
.historic/00002-topic/
.historic/.database/00002-topic/
```

Workspace modern menggunakan:

```text
.historic/<unix-millisecond>-topic/
.historic/.database/<unix-millisecond>-topic/
```

Orchestrator harus menjaga semua Markdown, `_meta.yaml`, FileID UUIDv7, open/closed storage, alias ID, dan internal history.

## 3. CLI contract

Tambahkan command:

```text
historic upgrade --dry-run
historic upgrade --json --dry-run
historic upgrade
historic upgrade --json
```

Flags:

```text
--dry-run       scan dan tampilkan plan tanpa menulis
--json          output envelope JSON
--backup-dir    lokasi backup eksplisit opsional
--no-file-ids   jangan menjalankan migration FileID otomatis
--no-topic-ids  jangan menjalankan migration TopicID otomatis
```

Default upgrade menjalankan seluruh pipeline. Flag `--no-file-ids` dan `--no-topic-ids` hanya untuk admin/recovery dan harus dijelaskan pada output.

## 4. Upgrade detection

Pada startup/doctor/upgrade:

1. Baca `workspace_format_version` dari metadata/index jika tersedia.
2. Scan folder topic untuk mendeteksi format legacy lima digit.
3. Scan `_meta.yaml` untuk `files[].id` kosong/legacy.
4. Tentukan action:
   - `compatible` jika semua modern;
   - `migration_required` jika ada legacy;
   - `rebuild_required` jika index schema mismatch;
   - `reject` jika workspace lebih baru dari binary.
5. Jangan rename folder atau menulis metadata secara diam-diam hanya karena startup.

## 5. Pipeline upgrade

Urutan wajib:

```text
acquire workspace upgrade lock
  -> create immutable backup/report
  -> scan and validate all topics
  -> migrate FileID legacy
  -> allocate/migrate TopicID legacy
  -> update topic references and alias map
  -> update workspace format metadata
  -> rebuild SQLite temporary index
  -> validate index and FTS
  -> atomic replace index
  -> commit upgrade marker
  -> release lock
```

Jika salah satu tahap gagal, rollback seluruh filesystem/metadata/index ke backup sebelum command.

## 6. Backup contract

Backup minimal berisi:

```text
backup/
  manifest.json
  topics/
  database/
  aliases.yaml
  sqlite/
  checksums.sha256
```

`manifest.json` wajib mencatat:

- timestamp UTC;
- app version name/code;
- source workspace format/index schema;
- target workspace format/index schema;
- setiap old/new topic ID;
- setiap old/new FileID;
- path rename;
- checksum sebelum upgrade.

Backup tidak boleh ditempatkan di dalam `.historic` agar tidak ikut discan.

## 7. Alias and reference policy

Simpan alias canonical di metadata workspace, contoh:

```yaml
topics:
  "00001": "1790078640411"
  "00002": "1790078640412"
```

Resolver harus:

- menerima ID modern secara langsung;
- resolve ID legacy melalui alias selama compatibility window;
- menampilkan warning bahwa ID sudah migrated;
- tidak membuat alias loop;
- menolak alias ke topic yang tidak ada;
- menjaga related/reference agar tetap dapat di-resolve.

Alias purge hanya boleh dilakukan melalui command eksplisit setelah backup.

## 8. Atomicity and idempotency

- Dry-run tidak mengubah byte apa pun.
- Upgrade sukses kedua menghasilkan `updated: 0` dan `migration_required: false`.
- Existing FileID/TopicID modern tidak digenerate ulang.
- Rename folder menggunakan staging directory dan atomic rename.
- `_meta.yaml` ditulis temporary + fsync + rename.
- SQLite dibangun di temporary path lalu divalidasi sebelum replace.
- Crash/retry tidak boleh menggandakan topic atau FileID.
- Lock stale/live process mengikuti policy allocator existing.

## 9. JSON output

```json
{
  "command": "upgrade",
  "ok": true,
  "data": {
    "dry_run": false,
    "migration_required": true,
    "topics_scanned": 11,
    "file_ids_migrated": 63,
    "topic_ids_migrated": 10,
    "index_rebuilt": true,
    "backup_path": ".historic-upgrades/20260922T153045Z",
    "aliases_written": 10,
    "warnings": []
  },
  "error": null
}
```

Error JSON harus menyertakan tahap gagal, path, backup path, dan remediation.

## 10. Compatibility policy

- Binary modern dapat membaca legacy dalam compatibility mode sebelum upgrade.
- Operasi non-destructive tetap boleh selama mode legacy.
- Operasi destructive atau rename identity memerlukan upgrade eksplisit.
- Binary lama menolak workspace format modern.
- Workspace lebih baru dari binary ditolak tanpa perubahan source.
- App version berbeda dengan schema compatible tidak otomatis memicu migration.
- Index mismatch memicu backup + rebuild sesuai policy version compatibility.

## 11. Test matrix

### Detection

- legacy topic only;
- legacy FileID only;
- mixed workspace;
- modern workspace;
- newer workspace rejection;
- missing/corrupt index.

### Pipeline

- dry-run byte-identical;
- FileID migration once;
- TopicID migration once;
- reference and alias update;
- open/closed pair preservation;
- SQLite/FTS rebuild;
- final doctor compatible.

### Failure recovery

- failure per stage;
- crash after staging rename;
- crash after metadata rename;
- crash before SQLite replace;
- backup restore;
- stale/live lock;
- concurrent upgrade rejection.

### Quality gate

```text
go test ./...
go vet ./...
go build .
git diff --check
```

## 12. Implementation order

1. Add workspace upgrade detection/report types.
2. Add backup manifest/checksum service.
3. Wrap existing FileID and TopicID migration services.
4. Implement alias/reference reconciliation.
5. Implement staged filesystem transaction and rollback.
6. Integrate temporary SQLite rebuild and validation.
7. Add CLI human/JSON output.
8. Integrate doctor/startup compatibility action.
9. Add failure-injection and end-to-end tests.
10. Update docs and Historic skill.

## 13. Acceptance criteria

- User legacy cukup menjalankan `historic upgrade`.
- `--dry-run` dapat direview sebelum perubahan.
- Semua Markdown dan metadata manual preserved.
- FileID dan TopicID migration idempotent.
- Alias legacy tetap resolve selama compatibility window.
- Backup dapat digunakan untuk rollback.
- SQLite rebuild atomic dan doctor compatible.
- Tidak ada silent filesystem rename saat startup biasa.
- Full quality gate lulus.

## Status

Complete. Implemented and validated in the current workspace. `historic upgrade` supports dry-run, JSON output, backup directory, FileID/TopicID migration controls, immutable backup, rollback-safe orchestration, idempotent rerun, SQLite rebuild, doctor compatibility, and upgrade tests. Commit reference pending Dev Agent commit.
