---
id: 00005
title: Historic Search Read Model and Workspace Lifecycle SPEC
status: draft
created: 2026-09-21
updated: 2026-09-21
tags: [spec, search, sqlite, fts5, aggregate, ai, lifecycle, open, close]
related: [./wos/01-two-table-search-read-model.md]
---
# Historic Search Read Model and Workspace Lifecycle SPEC

## 1. Tujuan

Mendefinisikan model data dan perilaku workspace Historic untuk kebutuhan AI memory:

- Markdown tetap source of truth;
- SQLite menjadi read model/cache yang dapat dibangun ulang;
- status pekerjaan berasal dari file managed Markdown;
- topic dan file dapat dicari dengan query terstruktur, FTS5, dan aggregate;
- storage open/closed dikelola terpisah dari work status;
- open/close mempertahankan snapshot `.database` melalui operasi copy.

## 2. Model domain

### 2.1 Topic

Topic adalah wrapper identitas dan container filesystem. Topic tidak memiliki work status authoritative.

Storage state ditentukan oleh lokasi:

```text
open   = .historic/<id>-<slug>/
closed = .historic/.database/<id>-<slug>/
```

Topic dapat memiliki computed summary dari child files, tetapi summary itu bukan source of truth dan tidak ditulis kembali sebagai status topic pada `_meta.yaml`.

### 2.2 Managed historic file

Managed historic file adalah file Markdown dengan frontmatter Historic yang valid. Rebuild memindai seluruh file di dalam folder topic secara rekursif; setiap file Markdown yang mengikuti format Historic valid adalah member `files` topic tersebut, apa pun nama file, subfolder, atau ID frontmatter-nya. Folder topic menentukan parent/storage file; `frontmatter.id` tetap menjadi identitas dokumen atau Work Order dan tidak harus sama dengan ID topic.

File Markdown valid di folder `wos/` maupun subfolder lain wajib diklasifikasikan sebagai managed file (`type=historic_file`). File di bawah `wos/` menggunakan `type=task` pada manifest dan read model turunan. Status authoritative selalu dibaca dari `frontmatter.status` dan ditulis pada `files[].status` di manifest serta kolom `files.status` di SQLite.

Managed historic file memiliki work status authoritative:

```text
create
draft
pending
progress
review
blocked
complete
failed
cancelled
```

`complete` dan `cancelled` adalah resolved untuk aggregate, tetapi nilai aslinya tetap disimpan.

### 2.3 Asset

Asset adalah file selain managed historic Markdown, termasuk plain Markdown tanpa frontmatter Historic valid, image, PDF, office file, archive, binary, dan file lain.

Asset memiliki metadata file tetapi tidak memiliki work status dan tidak memengaruhi aggregate status.

`_meta.yaml` adalah metadata canonical topic dan dikecualikan dari `files` maupun `assets`. `_meta.md` bukan metadata canonical; file tersebut diperlakukan sebagai asset Markdown biasa.

Klasifikasi wajib konsisten pada seluruh read model:

| Kondisi filesystem | Manifest | SQLite | Status |
|---|---|---|---|
| Markdown + frontmatter Historic valid di mana pun dalam folder topic | `files` | `type=historic_file` | wajib dari `frontmatter.status` |
| Markdown di `wos/` + frontmatter Historic valid | `files`, `type: task` | `type=historic_file`, inferred task | wajib dari `frontmatter.status` |
| Markdown tanpa frontmatter valid | `assets`, `type: markdown` | `type=asset` | tidak ada |
| `_meta.md` | `assets`, `type: markdown` | `type=asset` | tidak ada |
| file non-Markdown | `assets` | `type=asset` | tidak ada |

## 3. SQLite read model

Read model menggunakan dua tabel utama.

### `topics`

```text
id
num_padded
title
description
slug
storage                    open | closed
created_at
updated_at
tags
related
computed_status            nullable/cache only
```

### `files`

```text
id
topic_id
type                      historic_file | asset
path                      relative POSIX path within topic
title
description
status                    nullable; only historic_file
tags
related
content
asset_kind
created_at
updated_at
mtime
hash
size
word_count
```

`files.path` tidak menyimpan `.historic/`, `.database/`, topic folder, atau `archive_path`. Physical path diturunkan dari topic storage state:

```text
topic_base(storage) + files.path
```

Satu logical file tidak boleh menjadi duplicate hanya karena memiliki current copy dan `.database` snapshot.

## 4. Source of truth dan description

- `_meta.yaml` adalah metadata topic canonical dan satu-satunya metadata topic yang dibaca/ditulis oleh workspace baru.
- `_meta.md` bukan metadata canonical dan tidak dimigrasikan; ia diperlakukan sebagai asset Markdown.
- File dan frontmatter Markdown managed adalah source of truth untuk keberadaan, isi, identitas dokumen, dan status pekerjaan.
- `status: draft` adalah status Historic yang valid untuk topic maupun managed file dan termasuk status active/unresolved.
- `related` pada frontmatter menerima ID Historic lima digit atau logical relative path Markdown seperti `../spec.md` dan `./wos/01-task.md`; path absolut dan traversal unsafe ditolak.
- Metadata manifest `files` dan `assets` di `_meta.yaml` adalah hasil generate filesystem/rebuild yang terlihat oleh user, bukan field manual.
- SQLite menyalin manifest dan metadata file sebagai read model/cache; SQLite dan FTS tidak boleh dianggap sebagai sumber status atau isi utama.
- Status managed file berasal dari `frontmatter.status` dan harus tercermin identik pada `files[].status` serta `files.status` SQLite.
- Validitas managed file tidak mensyaratkan `frontmatter.id` sama dengan ID topic parent. ID topic berasal dari folder topic, sedangkan ID file/WO berasal dari frontmatter file.
- File Markdown valid di subfolder `wos/` tetap managed Historic file dan diklasifikasikan sebagai `type: task` pada manifest.

## 4.1 Format `_meta.yaml`

Contoh:

```yaml
id: "00005"
title: Search Read Model v2
description: Read model SQLite untuk pencarian dan memory AI.
created: "2026-09-21"
updated: "2026-09-21"
tags:
  - search
  - sqlite
  - ai
related: []
```

Field manifest di bawah ini adalah field generated yang ditulis ulang penuh berdasarkan filesystem saat `sync-meta` atau `rebuild`:

```yaml
files:
  - path: wos/01-two-table-search-read-model.md
    type: task
    status: complete
assets:
  - path: diagram.pdf
    type: pdf
```

Aturan manifest:

- `files` berisi seluruh Markdown dengan frontmatter Historic valid di dalam folder topic secara rekursif, termasuk WO, SPEC, dan Markdown valid yang tidak berada di `wos/`.
- `files[].status` wajib dan nilainya harus sama dengan `frontmatter.status`.
- `files[].path` adalah logical POSIX relative path terhadap folder topic.
- `files[].type` menggunakan klasifikasi Historic (`prd`, `spec`, `issue`, `note`, `decision`, `task`, atau `file`); file di `wos/` default ke `task`.
- `assets` berisi seluruh file non-managed, termasuk `_meta.md`; asset tidak memiliki `status`.
- `_meta.yaml` tidak muncul pada `files` atau `assets`.
- `files` dan `assets` diurutkan deterministik dan stale entries dihapus pada sync berikutnya.
- Field manual topic tetap dipertahankan saat manifest di-generate.
- File existence dan classification diturunkan dari filesystem/rebuild.

## 4.2 Canonical metadata policy

- Workspace baru langsung menggunakan `_meta.yaml`.
- `_meta.yaml` adalah satu-satunya metadata canonical dan write target.
- `_meta.md` tidak dimigrasikan, tidak dibaca sebagai metadata, dan selalu diperlakukan sebagai asset Markdown.
- Workspace lama yang masih memiliki `_meta.md` harus dikonversi secara eksplisit oleh user atau command migrasi khusus di masa depan; `rebuild` dan `sync-meta` tidak boleh menganggapnya sebagai metadata.
- `sync-meta` dan unified `rebuild` menulis ulang `files` dan `assets` sebagai manifest generated yang terlihat user, sambil mempertahankan field manual topic.
- Manifest generated harus sama dengan hasil scan filesystem dan SQLite read model.
- Manifest write dilakukan atomic dan idempotent jika filesystem tidak berubah.
- Jika `_meta.yaml` tidak ada atau invalid, rebuild/sync menolak topic tersebut dan tidak membuat metadata pengganti secara diam-diam.

Jika `_meta.yaml` hilang pada folder topic yang valid, `rebuild` melakukan recovery metadata minimal sebelum membangun read model:

- `id` diambil dari nama folder topic;
- `title` diambil dari slug folder dengan fallback aman, bukan wajib dari child WO;
- `description` kosong;
- `created` memakai tanggal managed file paling awal jika tersedia;
- `updated` memakai tanggal managed file terbaru jika tersedia;
- `tags` dan `related` kosong;
- `files` dan `assets` direconcile dari filesystem;
- metadata hasil recovery ditulis atomic dan dihapus kembali jika scan/index gagal;
- index lama dipertahankan jika rebuild gagal;
- recovery topic open dan closed harus didukung dan rebuild kedua idempotent.

Jika `_meta.yaml` invalid, rebuild tidak menimpanya secara diam-diam; topic gagal dilaporkan dan metadata/index lama dilindungi.

## 5. Open dan close berbasis copy

### `open`

```text
.database topic → copy ke .historic/ workdir
```

- Snapshot `.database` tetap ada.
- Workdir dibuat melalui staging dan validasi.
- Topic storage state menjadi `open` pada read model.
- Search menggunakan current workdir copy.
- Snapshot `.database` tidak menjadi logical duplicate result.
- Jika snapshot `.database` dan folder open memiliki slug berbeda untuk ID yang sama, keduanya tetap satu logical topic; open menjadi current dan nama folder terbaru menjadi canonical saat close berikutnya.

### `close`

```text
workdir topic → manifest diff → staging → overwrite/rename .database topic
```

- Isi workdir menjadi snapshot terbaru.
- Archive dicari berdasarkan topic ID, bukan nama folder.
- Nama folder `.database` dinormalisasi mengikuti slug folder open terbaru.
- Target `.database` ditimpa setelah staging dan validasi berhasil.
- Setelah replacement berhasil, workdir dihapus.
- Storage state menjadi `closed`.
- Jika proses gagal, snapshot `.database` lama tetap utuh dan workdir tetap tersedia.
- Close tidak mengubah work status file.
- Sebelum copy, hitung manifest logical path dan SHA-256 content.
- File dengan logical path dan hash sama tidak disalin ulang.
- File baru disalin, file berubah diganti, file hilang dihapus dari staged snapshot.
- Jika seluruh manifest dan slug sama, close menjadi no-op untuk content dan hanya menyelesaikan state storage.
- Rename folder archive karena perubahan slug tidak dianggap perubahan content.

### Archive normalization and differential close

Identitas topic adalah ID. Nama folder adalah slug yang dapat berubah. Rebuild boleh memperbarui logical slug/path di read model, tetapi tidak melakukan rename archive destruktif secara otomatis.

Saat close berhasil, hanya boleh ada satu snapshot `.database` untuk satu topic ID. Snapshot dengan slug lama harus dinormalisasi atau diganti secara atomic setelah target baru tervalidasi.

Perbandingan file menggunakan:

```text
topic ID + relative POSIX path + SHA-256(content)
```

`mtime` dan ukuran file bukan bukti utama perubahan. Internal Git tetap dapat mencatat rename dan diff content secara efisien.

### Safety

- Copy, overwrite, replacement, dan deletion harus atomic sejauh platform memungkinkan.
- Gunakan staging dan conflict detection.
- Symlink topic/file ditolak.
- Rollback wajib melindungi snapshot lama dan workdir.
- Internal Git `.historic/.database/.git` tetap lokal dan tidak memiliki remote.

## 6. Delete and purge

Delete adalah operasi berbeda dari close:

```text
close  = menyimpan topic secara reversible
delete = menghapus file atau topic dari current state
purge  = penghapusan permanen dengan konfirmasi ekstra
```

### File delete

Command user-facing yang direncanakan:

```text
historic delete <path> [--json]
```

- Menghapus satu managed historic file atau asset dari current filesystem setelah konfirmasi.
- Tidak mengubah status file lain atau status topic computed secara langsung.
- `historic rebuild` setelah delete mereconcile `Files`/`Assets`, menghapus record file dari SQLite/FTS, dan menghapus stale link.
- Asset dapat dihapus sebagai file biasa dan tidak memengaruhi aggregate work status.
- Delete pada topic closed tidak boleh memodifikasi snapshot `.database` secara diam-diam; user harus membuka/copy topic terlebih dahulu atau memakai alur command yang eksplisit.

### Topic delete

Command topic delete/purge harus dibedakan dari `close`:

```text
historic delete-topic <id> [--open|--closed] [--json]
historic purge <id> [--open|--closed] [--json]
```

- `delete-topic` menghapus current topic setelah konfirmasi dan harus memiliki recovery/trash policy yang jelas.
- `purge` adalah operasi permanen dan membutuhkan konfirmasi eksplisit ekstra.
- Untuk topic open, operasi menargetkan workdir.
- Untuk topic closed, operasi menargetkan snapshot `.database`.
- Jika open dan closed copy sama-sama ada, scope wajib eksplisit.
- Tidak boleh menghapus internal `.historic/.database/.git` sebagai efek samping.
- SQLite/FTS dibersihkan melalui `rebuild`, bukan manipulasi index manual.

### Safety

- Default delete tidak boleh menghapus snapshot `.database` ketika user hanya menghapus current open copy.
- Operasi destructive menggunakan staging/backup/trash dan rollback bila gagal.
- Symlink, traversal, absolute path, missing target, dan conflict ditolak.
- Human output menjelaskan target, storage state, dan konsekuensi operasi.
- JSON output menggunakan envelope standar.

## 7. Unified rebuild

```text
historic rebuild [--json]
```

Rebuild adalah workflow utama:

1. scan seluruh topic open dan closed;
2. reconcile penuh `files` dan `assets` pada `_meta.yaml` untuk current physical copy;
3. scan seluruh file topic secara rekursif dan klasifikasikan setiap Markdown dengan frontmatter Historic valid sebagai member `files`, tanpa syarat nama file, subfolder, atau kesamaan ID file dengan ID topic;
4. baca status managed file dari `frontmatter.status` dan salin ke manifest serta read model;
5. build aggregate topic secara computed;
6. build tabel `topics` dan `files`;
7. build FTS5;
8. validate temporary index;
9. atomic replace index.

Rebuild tidak mengubah work status file dan tidak menulis computed topic status sebagai authoritative field ke `_meta.yaml`.

`historic sync-meta` targeted tetap tersedia sebagai operasi metadata khusus, tetapi rebuild menjadi workflow normal untuk kalibrasi penuh workspace.

Rebuild tidak melakukan rename fisik folder archive secara destruktif; normalisasi nama archive dilakukan pada `close` setelah differential staging tervalidasi.
## 8. Search and aggregate

`historic find` default mencari logical records open dan closed. Filter storage menggunakan:

```text
--open
--closed
```

Flag lama `--active` dan `--archived` tidak menjadi kontrak baru.

Structured filters minimal:

```text
--status
--tag
--title
--type
--topic
--folder
--open
--closed
--created-after
--created-before
--updated-after
--updated-before
```

Query dapat menggabungkan `topics` dan `files` menggunakan join, `GROUP BY`, `COUNT`, `SUM`, `MAX`, dan `HAVING`.

Aggregate minimal:

```text
total_files
active_files
resolved_files
complete_files
cancelled_files
computed_status
```

Aturan resolved:

```text
complete + cancelled = resolved
create + draft + pending + progress + review + blocked + failed = active/unresolved
```

- Jika tidak ada managed files, summary nullable atau dipertahankan sebagai unknown; tidak boleh mengubah `_meta.yaml` status secara otomatis.

## 9. FTS5

FTS5 mencakup:

```text
topic title
topic description
file filename
file path
file title
file description
file content
tags
```

Bobot ranking:

```text
title/description/tags > filename/path > content
```

Vector DB dan embedding tidak termasuk scope awal. FTS5 dan description manual menjadi fondasi pencarian AI yang ringan, lokal, rebuildable, dan deterministik.

## 10. AI output

JSON search result minimal:

```json
{
  "type": "historic_file",
  "topic_id": "00005",
  "title": "Schema Calibration",
  "description": "Upgrade aman untuk schema index lama.",
  "status": "progress",
  "storage": "open",
  "tags": ["schema", "upgrade"],
  "path": "wos/task.md",
  "score": 3.82,
  "matched_in": ["title", "description"],
  "snippet": "Upgrade aman untuk schema index lama..."
}
```

Path logical tetap relatif terhadap topic. Physical resolved path dapat ditambahkan sebagai output turunan jika diperlukan, tetapi bukan field utama read model.

## 11. Schema compatibility

- Binary version, workspace format version, dan index schema version dipisahkan.
- `doctor`, `upgrade`, schema-aware rebuild, backup, lock, rollback, dan atomic replacement mengikuti schema calibration SPEC.
- Index lama tidak boleh menghasilkan raw `no such column` tanpa diagnosis actionable.
- Markdown checksum harus tetap sama setelah upgrade/rebuild.

## 12. Non-functional requirements

- Semua read model dapat dibangun ulang dari filesystem.
- Query dua tabel dan aggregate harus deterministic.
- Rebuild kedua tanpa perubahan menjadi no-op/equivalent result.
- Open/closed topic tidak boleh menjadi duplicate logical result.
- Asset tidak memengaruhi status pekerjaan.
- Rename slug harus memperbarui canonical archive saat close berikutnya.
- Close harus menghindari copy/overwrite file yang hash-nya identik.
- Tidak ada remote sync, Vector DB, atau perubahan `PRD.md` pada scope ini.

## 14. Open decisions

1. `computed_status` disimpan sebagai cache pada `topics` atau selalu dihitung saat query.
2. Schema FTS5 final dan weighting detail.
3. Bentuk final filter tanggal dan tag pada CLI.
4. Apakah `sync-meta` dipertahankan jangka panjang atau menjadi internal service.
5. Format backup `.database` dan retention policy.
6. Manifest format, hash algorithm, dan apakah differential close wajib mempertahankan file mode/permission.
7. Apakah migrasi metadata non-canonical diperlukan untuk workspace lama; `_meta.md` tetap asset pada policy canonical saat ini.
8. Apakah close/open wajib menjalankan sync manifest sebelum index rebuild.

## 15. Status

Draft. SPEC ini menjadi dasar penyelarasan lanjutan terhadap WO read model, lifecycle, rebuild, dan schema compatibility.
