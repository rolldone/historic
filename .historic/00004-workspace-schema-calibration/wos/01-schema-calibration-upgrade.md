---
id: 00001
title: WO 01 Schema Calibration and Upgrade
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [compatibility, schema, doctor, upgrade, rebuild, recovery]
related: []
---
# WO 01 — Schema Calibration and Upgrade

## Tujuan

Menyediakan mekanisme resmi untuk mendeteksi dan memulihkan ketidakcocokan antara versi binary Historic, format workspace, dan schema index SQLite. User tidak perlu menambahkan kolom atau menghapus `.index.sqlite` secara manual.

## Latar Belakang

Perubahan model storage menambahkan kolom `storage` pada index. Workspace yang masih memiliki schema lama dapat menyebabkan command gagal dengan error SQL seperti `no such column: storage`, meskipun Markdown sebagai source of truth tetap aman.

## Kontrak command

```text
historic doctor [--json]
historic upgrade [--json]
historic rebuild [--json]
```

### `historic doctor`

Diagnosis read-only yang menampilkan minimal:

- binary version dan executable path;
- workspace format version;
- current index schema version;
- required index schema version;
- status: compatible, upgrade required, rebuild required, binary too old, atau workspace invalid;
- keberadaan dan validitas source Markdown;
- rekomendasi recovery yang actionable.

`doctor` tidak mengubah workspace, index, atau Markdown.

### `historic upgrade`

Upgrade eksplisit yang:

1. mendeteksi versi binary, workspace, dan index;
2. memvalidasi workspace;
3. membuat backup index lama;
4. membangun schema baru pada SQLite temporary;
5. memindai Markdown sebagai source of truth;
6. memvalidasi index baru;
7. melakukan atomic replace ke `.historic/.index.sqlite`;
8. membersihkan temporary file setelah sukses;
9. mengembalikan index lama jika tahap upgrade gagal.

Markdown tidak boleh diubah oleh upgrade.

### `historic rebuild`

Rebuild harus schema-aware. Jika index hilang, schema lama, atau schema rusak, command tidak boleh gagal dengan error SQL mentah. Command harus memberi pesan actionable atau menjalankan rebuild aman ke schema terbaru dengan atomic replacement.

## Versioning

Pisahkan tiga versi:

- binary version, misalnya `0.2.0`;
- index schema version, misalnya `2`;
- workspace format version, misalnya `1`.

Index baru wajib menyimpan schema version yang dapat dibaca sebelum query terhadap kolom yang mungkin berubah.

Kebijakan kompatibilitas:

| Kondisi | Perilaku |
|---|---|
| Schema sama dengan requirement binary | gunakan index |
| Schema lebih lama | `doctor` menyarankan `upgrade`; command mutasi/query memberi error actionable |
| Schema lebih baru dari binary | tolak dan minta binary yang lebih baru |
| Index hilang | buat index baru dari Markdown |
| Index rusak | rebuild aman dari Markdown |
| Markdown invalid | pertahankan index lama dan laporkan file invalid |
| Workspace invalid | hentikan tanpa perubahan |

## Backup dan recovery

- Backup dibuat sebelum replacement, dengan nama yang dapat diidentifikasi waktu dan versi.
- Source index lama dipertahankan sampai index baru tervalidasi.
- Temporary database tidak boleh tertinggal setelah sukses atau failure.
- Jika replacement atau rebuild gagal, index lama tetap dapat digunakan atau dipulihkan.
- Upgrade tidak boleh melakukan perubahan parsial pada Markdown.
- Gunakan lock untuk mencegah dua proses upgrade bersamaan.
- Internal Git `.historic/.database/.git` tetap lokal dan tidak memiliki remote.

## Scope

### Termasuk

- Schema version metadata.
- Workspace compatibility check.
- `historic doctor` human dan JSON output.
- `historic upgrade` human dan JSON output.
- Schema-aware `historic rebuild`.
- Atomic index replacement.
- Backup, rollback, temporary-file cleanup, dan upgrade lock.
- Error actionable untuk binary lama, schema lama/baru, index rusak, dan Markdown invalid.
- Test untuk workspace lama yang membutuhkan kolom `storage`.
- Isolated smoke test upgrade dari index schema lama ke schema terbaru.

### Tidak termasuk

- Migrasi atau perubahan isi Markdown.
- SQL manual sebagai langkah user.
- Remote sync atau push.
- Perubahan source-of-truth topic.
- Perubahan `PRD.md`.
- Backward compatibility tanpa batas untuk semua schema lama.

## Acceptance Criteria

- `historic doctor --json` dapat mendiagnosis schema lama tanpa mengubah workspace.
- Error `no such column` tidak muncul sebagai satu-satunya petunjuk kepada user.
- `historic upgrade` mengubah index lama ke schema yang dibutuhkan binary.
- Upgrade tidak mengubah checksum Markdown.
- Upgrade gagal aman jika Markdown invalid dan index lama tetap terlindungi.
- Index baru memiliki schema version yang dapat dideteksi sebelum query.
- Binary yang terlalu lama menolak index schema yang lebih baru dengan pesan actionable.
- Index hilang dapat dibangun ulang dari Markdown.
- Rebuild menggunakan temporary database dan atomic replacement.
- Backup dan rollback tervalidasi.
- Concurrent upgrade ditolak atau diserialisasi dengan aman.
- Setelah upgrade berhasil, `historic list`, `add`, `find`, `close`, dan `open` berjalan tanpa schema error.
- JSON tetap menggunakan envelope `{ "command", "ok", "data", "error" }`.
- Unit, integration, dan isolated CLI smoke test lulus.
- Tidak ada SQL manual yang diperlukan user.

## Status

Planned. Implementasi belum dimulai.
