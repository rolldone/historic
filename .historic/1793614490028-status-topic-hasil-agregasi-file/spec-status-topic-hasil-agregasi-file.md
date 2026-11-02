---
title: 'SPEC: Status Topic Hasil Agregasi File'
description: ""
status: complete
created: "2026-11-02"
updated: "2026-11-02"
---
# SPEC: Status Topic Hasil Agregasi File

## 1. Ringkasan

Historic sudah memiliki agregasi status per topic di SQLite (`topics.computed_status`),
tetapi agregasi tersebut tidak konsisten, tidak terlihat oleh pengguna, dan tidak menjadi
kontrak yang jelas. SPEC ini menetapkan **satu aturan agregasi status topic** yang
deterministik, derived-only, dan ditampilkan pada `historic list` serta `historic show`.

Status topic **bukan** operasi tulis pada Markdown. Tidak ada perubahan pada `_meta.yaml`
selain `files[]`/`assets[]` yang sudah ada. Markdown tetap source of truth; status topic
adalah nilai turunan (read model) yang selalu dapat dibangun ulang.

## 2. Latar belakang dan masalah

### 2.1 Dua implementasi agregasi yang berbeda

- `internal/indexer/indexer.go` (`aggregateStatus`) menulis `topics.computed_status`
  sebagai **status domain konkret** (mis. `progress`), diambil dari **status non-terminal
  pertama menurut urutan scan path** (praktis alfabetis). Semantiknya tidak deterministik:
  hasil bergantung pada urutan/nama file, bukan prioritas status.
- `internal/query/query.go` (`QueryTopicsPage`) **menghitung ulang sendiri di SQL** dan
  mengembalikan label sintetis `active` / `complete` / `cancelled`. `active` bukan status
  domain yang valid dan menghapus informasi (`blocked`, `progress`, `review` semuanya
  menjadi `active`).

Akibatnya `historic find` (jalur `search`, membaca `t.computed_status`) dan jalur query
agregat dapat melaporkan nilai berbeda untuk topic yang sama.

### 2.2 `forwarded` salah diklasifikasikan

- `aggregateStatus` memperlakukan `complete` dan `cancelled` sebagai terminal, tetapi
  **tidak** memperlakukan `forwarded` sebagai terminal, padahal `domain.StatusForwarded`
  terdaftar pada `closeStatuses` (`StatusForwarded.IsClose() == true`).
- `query.go` menghitung `active_files` dengan `status NOT IN ('complete', 'cancelled')`,
  sehingga `forwarded` selalu terhitung sebagai pekerjaan aktif.

Keduanya bertentangan dengan definisi domain sendiri, dan membuat topic yang seluruh
pekerjaannya sudah di-forward tampak belum selesai.

### 2.3 Status topic tidak terlihat

`historic list` dan `historic show` memakai `repository.TopicView` (walk filesystem) yang
tidak memiliki field status topic. Pengguna hanya dapat melihat status topic lewat
`historic find`, dan itu pun dengan label sintetis yang menyesatkan.

### 2.4 Preseden industri

Agregasi status anak → induk yang dipakai di sini mengikuti pola *state rollup* yang sudah
mapan: GitHub combined status (`failure` > `pending` > `success`), Jira Epic Status
(derived dari status anak), dan AWS CloudFormation `StackStatus` (status terburuk menang).
Pola ini **tidak** menggunakan aritmetika/skor; hanya prioritas.

## 3. Definisi dan istilah

| Istilah | Arti |
| --- | --- |
| **File terkelola** | Markdown dengan frontmatter Historic valid, `type = historic_file`. Asset (`type = asset`) **tidak** ikut agregasi. |
| **File terbuka** | File dengan `domain.Status.IsOpen()`: `create`, `draft`, `pending`, `planned`, `progress`, `review`, `blocked`. |
| **File tertutup** | File dengan `domain.Status.IsClose()`: `complete`, `failed`, `cancelled`, `archived`, `forwarded`. |
| **Status topic** | Nilai tunggal hasil agregasi seluruh file terkelola di satu topic. Derived, bukan authoritative. |
| **Topic tanpa file terkelola** | Topic yang `files[]` terkelolanya kosong (mis. baru `create`). |

## 4. Keputusan

1. **Status topic adalah nilai derived (read model).** Tidak ditulis ke `_meta.yaml` dan
   tidak menjadi status authoritative di Markdown. Tidak ada field status topic baru pada
   frontmatter atau manifest.
2. **Satu fungsi agregasi tunggal.** Aturan agregasi diimplementasikan sekali sebagai
   fungsi domain (mis. `domain.AggregateStatus`) dan dipakai oleh indexer, `list`/`show`,
   dan `search`. SQL **tidak** menghitung ulang status agregat sendiri.
3. **Label sintetis `active` dipisah dari status topic.** `active` tetap ada sebagai
   penanda/counter "masih ada file terbuka", bukan sebagai nilai `computed_status`.
   Vocabulary `computed_status` **hanya** memakai nilai yang ada di `domain` (plus kosong
   untuk topic tanpa file terkelola).
4. **`forwarded` setara `complete` untuk agregasi.** Pekerjaan yang dipindahkan ke topic
   lanjutan dianggap selesai di topic asal. Validitas target forward (`forwarded_to` ada,
   topic tujuan ada, `extends` cocok) adalah urusan integritas/`doctor`, bukan rollup.
5. **`cancelled` hanya menang bila seluruh topic cancelled.** `cancelled` bukan penutup
   "belum beres"; ia dilaporkan bila topic memang dibatalkan seluruhnya.
6. **`failed` adalah penutup negatif yang menang.** Bila terdapat file `failed`, status
   topic adalah `failed` (selama tidak ada file terbuka).
7. **`archived` setara `complete`** (netral/dipensiunkan, tidak mengubah kesimpulan).
8. **Agregasi dihitung ulang pada semua operasi tulis yang relevan**, bukan hanya
   `save`/`rebuild`.
9. **Status topic ditampilkan** pada `historic list` dan `historic show` (human dan
   `--json`), serta ikut pada filter `historic find --status` / `--status-not`.

## 5. Kontrak agregasi

### 5.1 Algoritma (deterministik, tanpa aritmetika)

Input: daftar status dari seluruh file terkelola di topic tersebut.

**Langkah 0 — topic tanpa file terkelola.**
Tidak ada file terkelola → status topic **kosong** (`null` pada JSON, `-` pada human).

**Langkah 1 — bila masih ada file terbuka, file terbuka menang.**
Diambil status terbuka dengan prioritas tertinggi:

```text
blocked > review > progress > planned > pending > draft > create
```

Status topic = status terbuka berprioritas tertinggi yang ada.
File tertutup tidak memengaruhi hasil pada langkah ini.

**Langkah 2 — semua file sudah tertutup.**

| Kondisi | Status topic |
| --- | --- |
| ada `failed` | `failed` |
| **semua** file `cancelled` | `cancelled` |
| selain itu (`complete` / `forwarded` / `archived`, atau campur) | `complete` |

Interpretasi "campur" yang berlaku: `complete` + `cancelled` → `complete`;
`complete` + `forwarded` → `complete`; `forwarded` + `archived` → `complete`;
`cancelled` + `archived` → `complete`; `cancelled` + `failed` → `failed`.

### 5.2 Tabel contoh (acuan uji)

| Isi topic | Status topic | Alasan |
| --- | --- | --- |
| 3 file `complete` | `complete` | semua tertutup, tidak ada `failed`, tidak semua cancelled |
| 1 `complete` + 1 `cancelled` | `complete` | tidak semua cancelled |
| 2 file `cancelled` | `cancelled` | seluruh topic dibatalkan |
| 1 `complete` + 1 `forwarded` | `complete` | `forwarded` setara `complete` |
| 2 file `forwarded` | `complete` | dipindah, bukan gagal |
| 1 `cancelled` + 1 `forwarded` | `complete` | tidak semua cancelled |
| 1 `archived` + 1 `cancelled` | `complete` | tidak semua cancelled |
| 1 `complete` + 1 `failed` | `failed` | ada penutup negatif |
| 1 `cancelled` + 1 `failed` | `failed` | `failed` menang |
| 1 `blocked` + 1 `complete` | `blocked` | masih ada file terbuka |
| 1 `progress` + 1 `forwarded` | `progress` | masih ada file terbuka |
| 1 `draft` + 1 `cancelled` | `draft` | masih ada file terbuka |
| 1 `review` + 1 `blocked` | `blocked` | prioritas terbuka |
| 1 `pending` + 1 `progress` | `progress` | prioritas terbuka |
| topic hanya berisi asset | kosong | tidak ada file terkelola |
| topic tanpa file | kosong | tidak ada file terkelola |

### 5.3 Sifat yang wajib dipenuhi

- **Deterministik**: hasil tidak bergantung pada urutan scan, nama file, atau urutan
  penyimpanan di SQLite.
- **Idempoten**: menghitung ulang pada input sama menghasilkan nilai sama.
- **Tidak ada aritmetika/skor** pada SPEC ini. Skor progres (`0–100`) **tidak** termasuk
  dan dicatat sebagai pertanyaan terbuka (Bagian 9).
- **Asset tidak berpengaruh**: file `type = asset` diabaikan sepenuhnya.

## 6. Perubahan perilaku

### 6.1 `computed_status` konsisten satu sumber

- `indexer` menulis `topics.computed_status` memakai fungsi agregasi yang sama.
- `query.QueryTopicsPage` **tidak** menghitung ulang; nilainya dibaca dari
  `topics.computed_status`.
- `search`/`find` tetap membaca `topics.computed_status`.

### 6.2 Bucket counter diperbaiki

`forwarded` (dan `archived`) tidak lagi dihitung sebagai pekerjaan aktif.

- `active_files` hanya menghitung file **terbuka** (`Status.IsOpen()`).
- `resolved_files` menghitung file tertutup (`Status.IsClose()`).
- Perlu keputusan field tambahan (Bagian 9) bila ingin counter per jenis penutupan.
- Bucket existing `complete_files` dan `cancelled_files` dipertahankan; `forwarded`
  tidak lagi masuk `active_files`.

### 6.3 Tampilan `list` dan `show`

- `historic list` menampilkan kolom status topic (human) dan field `computed_status`
  pada `--json`.
- `historic show` menampilkan status topic pada header/ringkasan topic, selain daftar
  file yang sudah ada.
- Sumber nilai: fungsi agregasi yang sama. Karena `list`/`show` membaca filesystem,
  perhitungan dapat dilakukan dari frontmatter file yang memang sudah dibaca.

### 6.4 Filter `find`

- `find --status <s>` mencocokkan:
  - file dengan `files.status = <s>`, **atau**
  - topic dengan `computed_status = <s>`.
- `find --status-not <s>` mengecualikan keduanya.
- Perilaku `--status` untuk kombinasi status tetap divalidasi seperti sekarang
  (status tidak dikenal / tumpang tindih ditolak).
- Karena `active` bukan lagi nilai `computed_status`, **filter `--status active`
  dihapus/ditolak** sebagai status tidak dikenal. Ini perubahan perilaku yang disengaja.

### 6.5 Recompute pada operasi tulis

Status topic dihitung ulang pada operasi tulis berikut:

| Operasi | Alasan |
| --- | --- |
| `historic status <path> <status>` | status file berubah |
| `historic add` | file terkelola baru |
| `historic forward` | status file menjadi `forwarded` |
| `historic delete` | file terkelola hilang |
| `historic save <id>` / `--all` | mirror open → closed |
| `historic rebuild` | pemulihan/penyelarasan global |

Implementasi boleh menyelaraskan lewat indexer yang memang sudah dipanggil oleh operasi
tersebut (`indexer.Rebuild`), selama nilai akhir konsisten dengan fungsi agregasi tunggal.

## 7. Non-tujuan

- Tidak menulis status topic ke Markdown (`_meta.yaml`, frontmatter).
- Tidak menambahkan command baru untuk mengubah status topic secara manual.
- Tidak menambahkan skor/progres `0–100`.
- Tidak mengubah arti `historic close`/`historic open` (storage only).
- Tidak mengubah `_meta.yaml.files[]`/`assets[]` selain sinkronisasi yang sudah ada.
- Tidak mengubah perilaku Git internal.

## 8. Dampak dan risiko

- **Perubahan perilaku filter `--status`**: topic yang sebelumnya muncul dengan
  `computed_status = active` tidak lagi muncul; `--status active` menjadi status tidak
  dikenal. Dokumentasi (`README.md`, `docs/commands.md`,
  `.agents/skills/historic/SKILL.md`) dan test yang meng-hardcode `"active"`
  (`internal/query/query_test.go`, `internal/indexer/indexer_test.go`,
  `internal/lifecycle/wo14_rebuild_test.go`) harus diperbarui.
- **Migrasi index**: nilai `computed_status` lama di SQLite harus dibangun ulang dengan
  aturan baru. Jalur aman: `historic rebuild`. Tidak perlu migrasi kolom baru; schema
  `computed_status TEXT` sudah ada.
- **Kompatibilitas binary**: `index_schema_version` tidak berubah oleh SPEC ini, tetapi
  nilai agregat berubah. Setelah perubahan, jalankan `./build-global.sh` sebelum validasi.

## 9. Pertanyaan terbuka

1. Apakah perlu counter eksplisit per jenis penutupan pada JSON
   (`failed_files`, `forwarded_files`, `archived_files`)?
2. Apakah skor progres berbobot `0–100` (`computed_progress`) diperlukan pada fase
   berikutnya sebagai topic lanjutan?
3. Apakah `doctor` perlu lint khusus untuk forward menggantung (`forwarded_to` ada tetapi
   topic tujuan tidak ada atau `extends` tidak cocok) pada SPEC terpisah?

## 10. Kriteria penerimaan

1. Terdapat satu fungsi agregasi domain yang dipakai oleh indexer, `list`/`show`, dan
   `search`. Tidak ada perhitungan ulang status agregat di SQL.
2. Tabel contoh pada Bagian 5.2 terpenuhi seluruhnya oleh unit test.
3. `historic list` dan `historic show` menampilkan status topic (human dan `--json`).
4. `find --status <s>` mencocokkan file dan topic sesuai Bagian 6.4; `--status active`
   ditolak sebagai status tidak dikenal.
5. `forwarded` tidak lagi dihitung sebagai file aktif pada bucket counter.
6. Menjalankan `historic rebuild --json` pada workspace contoh menghasilkan
   `computed_status` yang sesuai aturan Bagian 5 untuk setiap topic.
7. Sub-perintah lifecycle (`status`, `add`, `forward`, `delete`, `save`) menyelaraskan
   `computed_status` tanpa memerlukan `rebuild` manual.
8. `historic doctor --json` tidak melaporkan inkonsistensi baru pada workspace valid.
