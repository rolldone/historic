---
title: Historic Workspace Validation Middleware
description: Memastikan command yang membutuhkan workspace hanya berjalan pada workspace Historic yang valid dan memiliki read model siap pakai, tanpa rebuild otomatis.
status: complete
created: "2026-09-23"
updated: "2026-09-23"
tags:
    - workspace
    - validation
    - middleware
    - sqlite
    - rebuild
    - errors
    - cli
---
# SPEC — Historic Workspace Validation Middleware

## 1. Latar belakang

Command seperti `historic find` dapat dijalankan pada folder yang belum memiliki workspace Historic lengkap atau sebelum index SQLite dibuat. Kondisi ini menghasilkan kegagalan yang kurang jelas atau mendorong command melakukan asumsi tersembunyi.

Historic menggunakan `.historic/` sebagai workspace canonical. Markdown adalah source of truth, sedangkan `.historic/.index.sqlite` adalah read model/cache yang harus dibuat eksplisit melalui `historic rebuild`.

## 2. Tujuan

Menyediakan validasi workspace terpusat, konsisten, dan **reject-only** sebelum command yang membutuhkan workspace berjalan.

Middleware harus:

- menolak folder yang belum memiliki `.historic/`;
- menolak konflik ketika `.historic/` dan `.histories/` sama-sama ada;
- menolak workspace ketika index SQLite belum tersedia, rusak, atau tidak kompatibel;
- memberikan pesan perbaikan yang actionable;
- tidak pernah menjalankan `rebuild` secara otomatis;
- tidak mengubah source Markdown, metadata, index, atau repository internal saat validasi.

## 3. Non-goals

- Tidak mengimplementasikan rebuild baru.
- Tidak mengubah format Markdown atau schema index.
- Tidak melakukan migrasi `.histories/` otomatis.
- Tidak menjalankan command lain secara rekursif.
- Tidak melakukan perubahan source code sebagai bagian dari operasi runtime middleware.

## 4. Klasifikasi command

### 4.1 Bootstrap/inspeksi

Command yang boleh berjalan tanpa workspace siap:

- `historic init`;
- `historic doctor` dalam mode read-only;
- `historic version`;
- `historic help`.

### 4.2 Workspace-dependent

Command berikut wajib melewati validasi readiness sebelum mengakses read model atau lifecycle data:

- `find` dan `search`;
- `list` dan `show`;
- `create` dan `add`;
- `status`;
- `close`, `open`, `import`, dan `restore`;
- command lifecycle lain yang membaca atau mengubah topic.

Command `rebuild` memiliki preflight tersendiri dan tidak boleh memanggil middleware readiness yang sama.

## 5. Urutan validasi

```text
parse dan validasi argumen command
→ cek root workspace
→ cek .historic/
→ cek konflik .historic/ dan .histories/
→ cek struktur minimum .historic/
→ cek .historic/.index.sqlite dan kompatibilitas schema
→ lanjutkan handler command
```

Kegagalan pada tahap mana pun menghentikan command sebelum query atau perubahan data dilakukan.

## 6. Kontrak hasil dan error

Helper validasi mengembalikan hasil terstruktur atau typed error, bukan pesan yang dirakit oleh setiap command.

State minimum:

- `not_historic_workspace` — `.historic/` tidak ditemukan;
- `legacy_conflict` — `.historic/` dan `.histories/` ditemukan bersamaan;
- `missing_index` — `.historic/.index.sqlite` tidak ditemukan;
- `invalid_index` — index tidak dapat dibuka, rusak, atau schema tidak kompatibel;
- `invalid_structure` — struktur minimum workspace tidak valid;
- `ready` — workspace siap digunakan.

Pesan perbaikan minimum:

- workspace belum ada: `Jalankan historic init terlebih dahulu.`
- index belum ada atau tidak valid: `Jalankan historic rebuild terlebih dahulu.`
- konflik direktori: `Pisahkan atau migrasikan .histories secara manual; Historic tidak menggabungkan direktori otomatis.`

Mode JSON wajib mempertahankan envelope `{command, ok, data, error}` dan tidak menggandakan error JSON ke stderr.

## 7. Aturan keamanan dan konsistensi

- Validasi tidak boleh melakukan rebuild otomatis.
- Validasi tidak boleh menulis file atau mengubah database.
- Pemeriksaan schema harus memakai mekanisme kompatibilitas existing.
- Rebuild tetap eksplisit dan dapat diaudit oleh user.
- Lock yang sudah digunakan oleh operasi index tetap dihormati; middleware tidak boleh mengambil alih atau mengganti index.

## 8. Acceptance criteria

1. Command workspace-dependent pada folder tanpa `.historic/` ditolak dengan instruksi `historic init`.
2. Command pada folder dengan `.historic/` dan `.histories/` ditolak tanpa rename, merge, atau migrasi otomatis.
3. Command pada `.historic/` tanpa `.index.sqlite` ditolak dengan instruksi `historic rebuild`.
4. Command pada index rusak atau schema tidak kompatibel ditolak dengan instruksi `historic rebuild`.
5. Workspace valid dengan index valid dapat menjalankan command seperti `historic find`.
6. Middleware tidak memanggil command `rebuild` dan tidak menyebabkan rekursi.
7. Perilaku human-readable dan JSON konsisten.
8. Test membuktikan validasi terjadi sebelum database query atau mutasi lifecycle.

## 9. Batasan implementasi

Perubahan implementasi dilakukan oleh Work Order terkait. SPEC ini tidak mengizinkan perubahan source code oleh Manager Agent.
