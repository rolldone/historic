---
id: 00001
title: Historic Technical SPEC Phase 1-4
status: progress
created: 2026-09-18
updated: 2026-09-19
tags: [spec, technical, cli, go, sqlite, git, fts5, phase-4]
---

# Technical SPEC — Historic Phase 1–4

## 1. Tujuan

Mendefinisikan kontrak teknis dan urutan pekerjaan untuk membangun CLI Historic sampai:

- Phase 1: Foundation/MVP
- Phase 2: Lifecycle
- Phase 3: Versioning
- Phase 4: Advanced Search

SPEC ini menjadi acuan Work Order. Implementasi source code belum termasuk dalam dokumen ini.

## 2. Baseline keputusan

Keputusan berikut dipakai sebagai baseline implementasi:

| Area | Keputusan |
|---|---|
| Bahasa | Go |
| CLI | Cobra |
| Markdown | Goldmark |
| Frontmatter | yaml.v3 |
| Index | SQLite sebagai cache/rebuildable index |
| Driver SQLite | `modernc.org/sqlite` untuk menghindari CGO pada single binary |
| Search MVP | Filesystem scan/grep; FTS5 disiapkan pada index phase |
| Search Phase 4 | SQLite FTS5 dengan fallback/error policy yang eksplisit |
| Git | Adapter Git yang dapat menggunakan `go-git`; tidak mengekspos detail Git ke user |
| ID topik | 5 digit, contoh `00014` |
| Relasi | Field `related` |
| `_meta.md` | Wajib pada setiap topik |
| Import | Copy sebagai default; move bukan bagian Phase 2 awal |
| Save | Manual melalui `historic save -m` |
| Working directory | Tidak wajib di-versioning internal pada Phase 1–3 |
| File watcher | Di luar Phase 1–3 |

## 3. Struktur data

```text
.historic/
├── .database/              # Arsip topik dan repository Git internal
├── 00014-admin-dashboard/  # Topik aktif
│   ├── _meta.md
│   ├── prd.md
│   └── wos/
│       └── 01-scaffold.md
└── .index.sqlite            # Nama/path final ditetapkan pada bootstrap
```

Aturan:

1. Folder topik berbentuk `<5-digit-id>-<slug>`.
2. `_meta.md` wajib ada dan menjadi sumber status topik.
3. File topik menggunakan frontmatter minimal: `id`, `title`, `status`, `created`.
4. `type` tidak disimpan di frontmatter; type diinferensikan dari path saat indexing.
5. Work Order disimpan pada `wos/` dengan nama sortable `NN-slug.md`.
6. Markdown adalah sumber kebenaran; index boleh dihapus dan dibangun ulang.

## 4. Status dan transisi

Status open: `create`, `pending`, `progress`, `review`, `blocked`.

Status close: `complete`, `failed`, `cancelled`, `archived`.

Transisi baseline:

- Topic baru: `create`.
- Pekerjaan aktif: `progress`.
- Menunggu input: `pending`.
- Menunggu verifikasi: `review`.
- Terhalang: `blocked`.
- Berhasil selesai: `complete`.
- Tidak berhasil: `failed`.
- Dihentikan: `cancelled`.
- Arsip final: `archived`.
- Status close memindahkan topik dari working directory ke `.database/`.

Validasi transition harus menolak ID tidak ditemukan, status tidak dikenal, dan operasi archive yang menghasilkan path konflik.

## 5. Kontrak command Phase 1

- `historic init`: membuat `.historic/`, `.database/`, dan index kosong.
- `historic create <title> [--id <id>]`: membuat topik dan `_meta.md`.
- `historic add <name>`: membuat file Markdown pada topik aktif; mendukung `wos/<name>`.
- `historic list [--json]`: menampilkan topik aktif/arsip sesuai scope yang disepakati.
- `historic show <id> [--json]`: menampilkan metadata dan daftar file topik.
- `historic find <keyword> [--json]`: mencari nama dan isi file.
- `historic rebuild`: scan Markdown dan membangun ulang index.
- `historic sync-meta <id|path> [--json]`: menyelaraskan section `Files` dan `Assets` pada `_meta.md`.

## 6. Kontrak command Phase 2

- `historic progress|pending|review|blocked <id>`: mengubah status open.
- `historic complete|failed|cancelled|archived <id>`: mengubah status close dan mengarsipkan topik.
- `historic import <id>`: menyalin topik dari `.database/` ke working directory.

Operasi perubahan file harus atomic sejauh platform memungkinkan. Jika langkah kedua gagal, sistem harus mengembalikan error jelas dan tidak menghapus sumber yang belum berhasil disalin.

## 7. Kontrak command Phase 3

- `historic save -m <message>`: commit perubahan pada repository internal.
- `historic log <id>`: menampilkan riwayat commit yang berkaitan dengan topik.
- `historic diff <id>`: menampilkan perubahan topik terhadap commit yang relevan.
- `historic restore <snapshot-id>`: memulihkan snapshot dengan validasi keamanan dan konfirmasi eksplisit bila berpotensi menimpa perubahan.

Tidak ada push ke remote pada Phase 3.

## 8. Kontrak command `sync-meta`

- `historic sync-meta <id>`: menyelaraskan section `Files` dan `Assets` pada `_meta.md` topic aktif.
- `historic sync-meta <path>`: menyelaraskan topic berdasarkan path topic atau `_meta.md`.
- `--json` menggunakan envelope standar `{ "command", "ok", "data", "error" }`.
- Markdown dengan frontmatter Historic valid diklasifikasikan sebagai managed file dan masuk `Files`.
- Markdown tanpa frontmatter valid, binary, gambar, PDF, Word, spreadsheet, archive, dan file lain diklasifikasikan sebagai asset dan masuk `Assets`.
- File asset bukan error dan tidak menghasilkan warning; file tetap tidak diubah.
- `_meta.md` dikecualikan dari `Files` dan `Assets`.
- Link existing dipertahankan; link stale tidak dihapus otomatis; link baru tidak digandakan.
- `_meta.md` ditulis secara atomic dan index direbuild setelah sukses.
- Path absolute/traversal, topic ambiguous, dan `_meta.md` invalid ditolak.
- Command tidak memindahkan atau mengarsipkan topic.

## 9. Model index

Index minimal menyimpan: `num`, `num_padded`, `type`, `title`, `status`, `tags`, `related`, `created_at`, `updated_at`, `path`, `folder_id`, `folder_slug`, `subfolder`, `filename`, `file_order`, `content`, `word_count`, `mtime`, dan `hash`.

Index harus memiliki index SQL pada nomor topik, status, type, dan folder. Rebuild dilakukan dalam transaction dan mengganti data lama hanya jika proses scan selesai valid.

## 9. Output dan error

1. Output manusia harus ringkas dan dapat dibaca di terminal.
2. Flag `--json` menghasilkan schema stabil dengan field `command`, `ok`, `data`, dan `error`.
3. Error JSON tidak boleh mencampur teks progress ke stdout.
4. Exit code non-zero digunakan untuk kegagalan validasi, file system, index, dan Git.
5. Path output harus relatif terhadap root Historic bila memungkinkan.
6. Query FTS invalid harus menghasilkan error actionable; tidak boleh jatuh diam-diam ke hasil yang salah.

## 10. Non-functional requirements

- Linux menjadi target validasi pertama; desain tidak boleh mengunci ke Linux.
- Search MVP untuk 1.000 file harus kurang dari 500 ms pada mesin pengembangan baseline.
- Tidak boleh ada kehilangan data ketika command gagal di tengah operasi.
- Semua parser menolak frontmatter wajib yang tidak valid dengan pesan lokasi file.
- Semua command utama memiliki unit test dan integration test CLI.
- Source of truth tetap file Markdown, bukan SQLite atau Git metadata.

## 11. Workspace Breaking Change

- Workspace canonical Phase 4+: `.historic/`.
- `.histories/` tidak didukung dan tidak memiliki compatibility layer.
- Migrasi workspace lama dilakukan manual dengan rename `.histories/` menjadi `.historic/`.
- Jika kedua folder ada, CLI menolak operasi dengan konflik; tidak ada auto-merge atau auto-rename.

## 12. Definition of Done Phase 1–3

- Semua command Phase 1–3 tersedia sesuai kontrak.
- Struktur folder dan frontmatter tervalidasi.
- Rebuild dapat memulihkan index dari Markdown setelah index dihapus.
- Lifecycle open/close dan archive/import lolos skenario normal, konflik, dan recovery.
- Git save/log/diff/restore lolos skenario perubahan dan snapshot.
- JSON output dan exit code terdokumentasi serta diuji.
- Benchmark search 1.000 file memenuhi target PRD.

## 13. Phase 4 Advanced Search

### Tujuan
Menyediakan pencarian berbasis SQLite FTS5 yang cepat, relevan, deterministik, dan tetap rebuildable dari Markdown.

### Scope
- Virtual table FTS5 untuk path, filename, title, dan content.
- Rebuild FTS atomik bersama index records.
- `historic find` memakai FTS5 untuk pencarian Phase 4.
- Filter `--status`, `--folder`, `--active`, `--archived`, `--type`, dan `--id`.
- Ranking relevansi dengan tie-breaker path.
- Snippet kontekstual dan highlight hanya pada output manusia.
- JSON envelope tetap `{ "command", "ok", "data", "error" }`.
- Query kosong, Unicode, special character, dan query invalid memiliki perilaku teruji.
- Benchmark 1.000, 10.000, dan 100.000 file.

### Keputusan Phase 4
- Markdown tetap source of truth.
- SQLite FTS5 adalah index/cache yang dapat dihapus dan dibangun ulang.
- Tidak menggunakan shell atau command grep eksternal.
- Jika index/FTS belum tersedia, command memberi error actionable yang menyarankan `historic rebuild`; fallback diam-diam tidak diperbolehkan.
- Hasil diurutkan berdasarkan relevansi, lalu path ascending sebagai tie-breaker.
- Tidak ada command baru; Phase 4 memperkuat `historic find` dan `historic rebuild`.

### Definition of Done Phase 4
- Semua WO 19–22 complete.
- FTS5 tersedia pada workspace `.historic/.index.sqlite`.
- Rebuild gagal secara aman jika Markdown invalid dan mempertahankan index lama.
- Filter dan ranking memiliki test unit/integration.
- Benchmark terdokumentasi dan memenuhi target yang disetujui.
- Dokumentasi `historic find` Phase 4 tersedia.

## 14. Out of scope

Webapp, GUI, multi-user, remote sync/push, AI chat, MCP server, file watcher, npm wrapper, dan monetisasi.

## 13. Open decisions untuk review berikutnya

1. Nama final file index SQLite.
2. Apakah `create` diperlakukan sebagai status persist atau hanya event awal.
3. Apakah `list` default menampilkan arsip selain topik aktif.
4. Detail schema JSON versi pertama.
5. Strategi restore: replace, copy snapshot, atau mode dry-run.
