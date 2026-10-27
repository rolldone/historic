---
title: 'SPEC: Update Status Topic Berlalu, Topic Extension, Status review dan forwarded'
description: "Topic berlalu boleh diubah status filenya; pekerjaan lanjutan dibuat sebagai topic baru yang meng-extend topic lama; status file review ditegaskan; status terminal forwarded menandai task yang dipindahkan ke topic lanjutan beserta tujuan forwarded_to."
status: draft
created: "2026-10-25"
updated: "2026-10-28"
tags:
    - lifecycle
    - topic
    - status
    - extension
    - closed
    - forwarded
related:
    - "1790078640413"
    - "1791851149892"
---
# SPEC: Update Status Topic Berlalu, Topic Extension, Status `review` dan `forwarded`

## 1. Ringkasan

Empat hal:

1. **Status file pada topic berlalu boleh diubah.** Tidak perlu `historic open` lebih
   dahulu. Perilaku `historic add` pada topic berlalu **tidak diubah** (tetap seperti
   sekarang: `add` hanya bekerja pada topic open).
2. **Pekerjaan lanjutan dapat dibuat sebagai topic baru yang meng-extend** topic yang
   sudah berlalu, tanpa memodifikasi topic lama.
3. **`review` ditegaskan sebagai status file yang sah** dan konsisten di seluruh
   permukaan (domain, CLI, pencarian, TUI, manifest, dokumentasi, test).
4. **`forwarded` adalah status file terminal baru** untuk task yang tidak dapat
   dikerjakan di topic tersebut dan dipindahkan ke topic lanjutan. File asal tetap ada,
   statusnya menjadi `forwarded`, dan mencatat tujuan pada `forwarded_to`.

> **Revisi 2026-10-28:** aturan "topic berlalu tidak boleh membuat file baru"
> (`domain.ErrTopicReadOnly`) **dicabut**. `add` pada topic berlalu kembali ke perilaku
> lama, yaitu gagal karena tidak ada workdir open. Pengguna yang ingin menambah file
> dapat menjalankan `historic open <id>` terlebih dahulu.

> **Tambahan 2026-10-28:** status `forwarded` beserta command `historic forward`
> ditambahkan. Keputusan desain: `forwarded` bersifat **terminal (close-like)**,
> memakai command khusus, dan topic tujuan **wajib** meng-extend topic asal.

## 2. Latar belakang

Saat ini `historic add` hanya bekerja pada topic open (mencari workdir open melalui
`activeTopicPath`), sedangkan `historic status <path> <status>` sudah dapat menjangkau
`.historic/.database/`. Namun:

- Belum ada mekanisme resmi untuk menyatakan bahwa topic baru merupakan **lanjutan** dari
  topic yang sudah berlalu, sehingga rantai pekerjaan antar-topic hilang.
- `review` sudah ada di `internal/domain/model.go` (`validStatuses` dan `openStatuses`),
  tetapi belum diverifikasi konsisten di semua permukaan dan belum ada kontrak eksplisit
  bahwa status ini boleh dipakai pada file di topic closed.

## 3. Definisi

| Istilah | Definisi |
| --- | --- |
| **Topic terbuka (open)** | Topic dengan workdir `.historic/<topic-id>-<slug>/`. |
| **Topic berlalu (closed)** | Topic yang hanya ada di `.historic/.database/<topic-id>-<slug>/`. |
| **File terkelola** | Markdown dengan frontmatter Historic valid yang terdaftar di `_meta.yaml.files`. |
| **Topic extension** | Topic baru yang menyatakan `extends: <topic-id>` pada `_meta.yaml`-nya dan menunjuk topic berlalu sebagai induk. |
| **Forwarded** | Status file terminal yang menandai task dipindahkan ke topic lanjutan; file asal tetap ada dan mencatat `forwarded_to: <topic-id>` tujuan. |

Catatan: topic yang punya salinan open **dan** closed (hasil `historic save`) dianggap
**topic terbuka** selama workdir open masih ada; salinan open tetap otoritatif.

## 4. Keputusan

1. **Status file pada topic berlalu adalah operasi yang didukung.** Tidak perlu
   `historic open` terlebih dahulu.
2. **Tidak ada aturan khusus untuk `add` pada topic berlalu.** Perilaku lama
   dipertahankan: `add` gagal karena tidak menemukan workdir open (`ErrTopicMissing`).
   Pengguna dapat menjalankan `historic open <id>` bila memang ingin menambah file.
3. **Topik extension tidak memodifikasi topic induk**: fitur `--extends` tidak menulis
   file, `_meta.yaml`, atau backlink apa pun ke topic lama. Relasi disimpan satu arah
   pada topic baru (`extends`).
4. **`review` adalah status file yang sah**, termasuk untuk file di topic berlalu.
   `review` tetap tergolong status *open/aktif/unresolved* (bukan terminal), sehingga
   ikut terhitung pada `computed_status` agregat sebagai pekerjaan berjalan.
5. **Repo Git internal tidak berubah perilakunya**: tidak ada commit/push otomatis.
6. **`forwarded` adalah status file terminal (close-like).** Task yang tidak dapat
   dikerjakan di topic asal dipindahkan ke topic lanjutan; file asal tidak dihapus dan
   tidak disalin, hanya berubah status. `forwarded` tergolong status terminal sehingga
   berkontribusi pada agregasi `computed_status` sebagai pekerjaan yang sudah
   diselesaikan di topic tersebut.
7. **Forward hanya sah ke topic yang meng-extend topic asal.** Validasi ini wajib
   (strict) agar rantai extension tetap konsisten; tidak ada override pada SPEC ini.

## 5. Perubahan perilaku

### 5.1 Update status pada topic berlalu

```sh
historic status .historic/.database/1791851149892-save-mirror-open-workdir/spec-save-mirrors-open-workdir.md review
historic status wos/01-extract-mirror-operation.md progress
```

- Resolusi path tetap memindai `workspace.Histories` dan `workspace.Database`.
- Jika path yang sama ada di salinan open dan closed, **wajib path eksplisit**; error
  `ErrConflict` "matches multiple Markdown files" dipertahankan. Tidak boleh memilih
  diam-diam.
- Perubahan hanya menyentuh frontmatter file target: `status` dan `updated` (format
  `YYYY-MM-DD`, UTC), lalu `indexer.Rebuild`.
- Operasi ini **tidak** membuat/menghapus file dan tidak mengubah storage.

### 5.2 Perilaku `add` pada topic berlalu (tidak berubah)

```sh
historic add "Catatan tambahan" --id 1791851149892
```

- Perilaku lama dipertahankan: `add` gagal dengan `ErrTopicMissing` karena topic tidak
  memiliki workdir open.
- **Tidak ada sentinel atau pesan khusus baru** untuk kasus ini (aturan
  `ErrTopicReadOnly` dicabut).
- Bila pengguna memang ingin menambah file, jalankan `historic open <id>` lalu `add`.
- Berlaku sama untuk `--force`: `--force` hanya relevan pada topic open.

### 5.3 Topic extension

```sh
historic create "Lanjutan Save Mirror" --extends 1791851149892
historic create "Lanjutan Save Mirror" --extends 1791851149892 --json
```

Kontrak:

- Flag baru `--extends <topic-id>` pada `historic create`.
- Target harus topic yang **sudah ada** (open atau closed) dan dapat di-resolve oleh
  `TopicLocation`. Target tidak ditemukan → `domain.ErrTopicMissing`.
- Hanya satu induk; pemberian `--extends` lebih dari sekali atau kosong → error validasi.
- ID topic baru tetap dialokasikan normal (monotonic epoch-ms) dan **tidak** diturunkan
  dari ID induk.
- Relasi ditulis di `_meta.yaml` topic baru:

```yaml
id: "1792889892200"
title: Lanjutan Save Mirror
created: "2026-10-25"
extends: "1791851149892"
files: []
assets: []
```

- `extends` **tidak** otomatis menyalin `related`, tag, atau file apa pun.
- `extends` bersifat satu arah dan **immutable setelah dibuat** (tidak ada
  `historic stage`/`update --extends` pada SPEC ini).
- Rantai diperbolehkan: A (closed) ← B (closed) ← C, dengan `extends` menunjuk induk
  langsung.
- Fitur extension **tidak memodifikasi topic induk**: tidak menambah file, tidak menulis
  `_meta.yaml`, tidak membuat backlink. Backlink, jika dibutuhkan, hanya ditampilkan
  sebagai hasil turunan (derived) dari index, bukan ditulis ke Markdown induk.
- `historic show <id>` (human dan `--json`) menampilkan field `extends` bila ada,
  termasuk pada data topic anak; `historic show` milik induk **tidak** berubah.
- `--extends` boleh menunjuk topic open, walau intent utamanya adalah topic berlalu.
  Pembatasan "hanya closed" dicatat pada Bagian 12 sebagai pertanyaan terbuka.

### 5.4 Status file `review`

`review` sudah ada di `validStatuses` dan `openStatuses`. SPEC ini menetapkan:

- `historic status <path> review` valid untuk file di topic open maupun closed.
- Status tersimpan pada frontmatter file dan `_meta.yaml.files[].status` setelah rebuild.
- `historic find --status review` dan `--status-not review` berfungsi.
- TUI (`f` filter status) memuat `review` dalam siklus status.
- Pesan validasi status pada `internal/search/search.go` dan dokumentasi
  (`README.md`, `docs/commands.md`, `.agents/skills/historic/SKILL.md`) menyebut
  `review`.
- `review` dianggap aktif/unresolved pada agregasi `computed_status`.
- Tidak ada command topic-level baru untuk `review`; status tetap milik file
  (lihat `ChangeStatus` yang sudah menolak status topic).

### 5.5 Sinkronisasi manifest setelah perubahan status

Temuan saat penyusunan SPEC: `historic status <path> <status>` memperbarui frontmatter
file dan index, tetapi `_meta.yaml.files[].status` serta `files[].type` baru ikut
terbarui setelah `historic rebuild` dijalankan. Contoh: file SPEC menjadi `status: draft`
di frontmatter, sementara `_meta.yaml` masih mencatat `status: create` dan
`type: file` sampai rebuild dijalankan.

Kontrak yang ditetapkan:

- `historic status` harus menyinkronkan `_meta.yaml.files` (status dan type) dalam
  operasi yang sama, tanpa mewajibkan rebuild manual.
- Sinkronisasi harus idempoten dan tidak mengubah `assets` atau field lain.
- Bila sinkronisasi manifest gagal, frontmatter yang sudah ditulis harus dilaporkan
  sebagai kegagalan parsial yang jelas (bukan sukses senyap), dan index tetap konsisten
  dengan Markdown.
- `historic rebuild` tetap menjadi jalur pemulihan utama bila manifest diubah manual.

### 5.6 Status `forwarded` dan command `historic forward`

Kasus: sebuah task/file tidak dapat dikerjakan di topic tersebut sehingga harus
dipindahkan ke topic baru yang merupakan kelanjutan topic tersebut.

```sh
# Opsi 1: topic tujuan sudah dibuat lebih dulu dengan --extends
historic create "Lanjutan Proyek X" --extends 1791851149892
historic forward wos/03-analisis-kuota.md --to 1792889892200

# Opsi 2: buat topic tujuan sekaligus
historic forward wos/03-analisis-kuota.md --create "Lanjutan Proyek X"
```

Kontrak:

- File asal **tidak dihapus dan tidak disalin**. Isi Markdown di topic asal tetap utuh.
- Frontmatter file asal berubah: `status: forwarded` dan `forwarded_to: "<topic-id>"`.
  Field `updated` ikut diperbarui (format `YYYY-MM-DD`, UTC).
- `_meta.yaml.files[]` entri file tersebut memperoleh `status: forwarded` dan
  `forwarded_to: "<topic-id>"`, disinkronkan tanpa menunggu `rebuild`.
- `forwarded` masuk `validStatuses` dan `closeStatuses` (terminal), **tidak** masuk
  `openStatuses`.
- `--to <topic-id>` wajib menunjuk topic yang **ada** dan `extends`-nya sama dengan
  topic asal file tersebut. Pelanggaran → error validasi, tidak ada perubahan filesystem.
- `--create "<title>"` membuat topic baru dengan `extends` = topic asal, lalu langsung
  menandai file sebagai `forwarded`. Jika pembuatan topic gagal, status file tidak
  berubah (atomik).
- `--to` dan `--create` tidak boleh dipakai bersamaan.
- File harus file terkelola (Markdown dengan frontmatter Historic valid) di **topic
  open**. Forward dari salinan closed ditolak; jalankan `historic open` lebih dulu.
- Forward ulang diperbolehkan dan bersifat idempoten: `forwarded_to` diganti ke tujuan
  baru yang valid.
- Mengubah status file kembali ke status non-`forwarded` (mis. `progress`) **menghapus**
  `forwarded_to` dari frontmatter dan manifest.
- Konten task **tidak** otomatis dipindahkan ke topic tujuan pada SPEC ini; membuat file
  padanan di topic tujuan tetap tanggung jawab pengguna.

## 6. Kontrak CLI dan JSON

| Command | Perubahan |
| --- | --- |
| `historic create <title> [--extends <id>] [--json]` | Flag baru; `data.extends` pada JSON (string, kosong bila tidak ada). |
| `historic add <name> [--id <id>] [--force] [--json]` | Tidak berubah; tetap hanya untuk topic open. |
| `historic status <path> <status> [--json]` | Tidak berubah; `review` diterima. Status non-`forwarded` menghapus `forwarded_to`. |
| `historic forward <path> --to <id>` atau `--create "<title>"` `[--json]` | Command baru; `data.forwarded_to` pada JSON. |
| `historic show <id> [--json]` | `data[].extends` dan `forwarded_to` per file ditambahkan. |
| `historic list [--json]` | Tidak berubah (tanpa perlu menampilkan `extends`). |
| `historic find/search` | Daftar status memuat `review` dan `forwarded`. |

Semua output JSON tetap memakai envelope `{ "command", "ok", "data", "error" }` dan
error JSON tidak diduplikasi ke stderr.

## 7. Dampak teknis

- `internal/domain/model.go`: tambah `StatusForwarded Status = "forwarded"` ke
  `validStatuses` dan `closeStatuses` (bukan `openStatuses`).
- `internal/domain/model.go`: tambah `ForwardedTo TopicID` pada `Frontmatter`
  (`yaml:"forwarded_to,omitempty"`).
- `internal/markdown/metadata.go`: tambah field `Extends domain.TopicID \`yaml:"extends,omitempty"\``
  pada `TopicMetadata`, termasuk validasi (`Valid()` dan keberadaan referensi dilakukan
  di lapisan repository, bukan saat parse agar rebuild tetap toleran).
- `internal/markdown/metadata.go`: tambah `ForwardedTo domain.TopicID`
  (`yaml:"forwarded_to,omitempty"`) pada `ManifestFile`.
- `internal/repository/topics.go`: `CreateTopic` menerima parameter `extends`. `AddEntry`
  tidak diubah.
- `internal/repository/topics.go` (baca): expose `Extends` pada `domain.Topic`.
- `internal/lifecycle/file_status.go`: bersihkan `ForwardedTo` saat status berubah ke
  non-`forwarded`; tambahkan fungsi forward atomik atau helper bersama.
- `internal/cmd/create.go`: flag `--extends`. `internal/cmd/add.go` tidak diubah.
- `internal/cmd/forward.go`: command baru `historic forward`.
- `internal/indexer/schema.go` + `index_migration`: kolom `topics.extends_id` dan
  `files.forwarded_to`; bump versi index; data lama diisi `NULL`.
- `internal/cmd/list_show.go`: tampilkan `extends` dan `forwarded_to`.
- `internal/search/search.go` + `internal/tui/search.go`: status `forwarded` masuk filter
  dan daftar validasi status.
- Dokumentasi: `README.md`, `docs/commands.md`, `.agents/skills/historic/SKILL.md`.

## 7.1 Tabel status file

| Status | Klasifikasi | Arti |
| --- | --- | --- |
| `create`, `draft`, `pending`, `planned`, `progress`, `review`, `blocked` | open | pekerjaan berjalan |
| `complete`, `failed`, `cancelled`, `archived` | close | selesai tanpa pindah topic |
| `forwarded` | close | dipindahkan ke topic lanjutan (`forwarded_to`) |

## 8. Edge cases

1. `--extends` menunjuk ID yang tidak ada → `ErrTopicMissing`, tidak ada topic dibuat.
2. `--extends` menunjuk topic dengan dua salinan (open + closed) → diterima, memakai ID
   sebagai identitas; salinan tidak memengaruhi relasi.
3. `add --id <closed-id>` → gagal `ErrTopicMissing` seperti perilaku lama; setelah
   `historic open`, `add` kembali normal.
4. `status` pada file di salinan closed saat salinan open ada, dengan path ambigu →
   `ErrConflict`, tidak memilih otomatis.
5. `_meta.yaml` legacy tanpa `extends` → tetap valid, `extends` kosong.
6. `extends` menunjuk dirinya sendiri tidak mungkin (ID baru selalu baru); rantai siklik
   tidak dapat terjadi karena `extends` immutable dan menunjuk topic yang sudah ada.
7. `rebuild` pada topic berlalu yang memiliki anak extension → tidak menulis apa pun ke
   induk; hanya index yang membaca `extends_id`.
8. `close`/`save` pada topic anak → `extends` dipertahankan byte-for-byte.
9. `forward --to` menunjuk topic yang tidak meng-extend topic asal → error validasi,
   tidak ada perubahan pada frontmatter maupun manifest.
10. `forward --create` gagal membuat topic → status file tetap seperti semula (atomik).
11. `forward` pada file di salinan closed → ditolak; pengguna harus `historic open`
    terlebih dahulu.
12. File yang sudah `forwarded` lalu diubah ke `progress` → `forwarded_to` hilang dari
    frontmatter dan manifest.
13. `forward --to` dan `--create` bersamaan → error validasi.
14. `rebuild` pada file `forwarded` dengan `forwarded_to` yang topicnya sudah dihapus →
    index tetap dibangun; nilai `forwarded_to` dipertahankan apa adanya (tidak ada
    kegagalan rebuild), dan `doctor` dapat melaporkannya sebagai referensi tak
    ter-resolve.

## 9. Non-goals

- Tidak ada command untuk mengubah/menghapus `extends`.
- Tidak ada larangan pembuatan file pada topic berlalu (aturan ini dicabut).
- Tidak ada penulisan backlink ke topic induk oleh fitur extension.
- Tidak ada penyalinan otomatis file dari topic induk ke topic anak.
- Tidak ada pemindahan otomatis isi task ke topic tujuan saat `forward`.
- Tidak ada penghapusan file asal saat `forward`.
- Tidak ada forward massal (bulk) pada SPEC ini.
- Tidak ada status topic-level, termasuk `review` dan `forwarded` di level topic.
- Tidak ada perubahan pada mekanisme commit/push Git internal.
- Tidak ada UI pohon (tree) antar-extension pada SPEC ini.

## 10. Acceptance criteria

- **AC-1** `historic status` berhasil mengubah status file di `.historic/.database/`
  tanpa `historic open`, termasuk ke `review`.
- **AC-2** `historic create "<title>" --extends <closed-id>` membuat topic baru dengan
  `extends: "<closed-id>"` pada `_meta.yaml`, dan ID induk tidak berubah.
- **AC-3** `historic show <child-id> --json` memuat `data[].extends`; `show` induk
  identik dengan sebelum fitur.
- **AC-4** `--extends` dengan ID tidak valid/tidak ada gagal tanpa membuat direktori topic.
- **AC-5** `review` diterima oleh `historic status` pada file open dan closed, tersimpan
  di frontmatter dan manifest, serta dapat difilter lewat `historic find --status review`.
- **AC-6** `historic rebuild --json` dan `--dry-run` tetap idempotent pada workspace yang
  memiliki topic extension; index lama tanpa kolom `extends_id` termigrasi/rebuild tanpa
  error.
- **AC-7** Dokumentasi dan pesan validasi status menyebut `review` secara konsisten.
- **AC-8** `go test ./...` lulus, termasuk test regresi baru.
- **AC-9** Setelah `historic status <path> draft`, `_meta.yaml.files[].status` sudah
  `draft` tanpa perlu menjalankan `historic rebuild` manual.
- **AC-10** `historic add --id <closed-id>` tetap berperilaku seperti sebelumnya
  (`ErrTopicMissing`) dan tidak ada sentinel baru; setelah `historic open`, `add` normal.
- **AC-11** `historic forward <path> --to <child-id>` mengubah status file asal menjadi
  `forwarded`, mengisi `forwarded_to: "<child-id>"` di frontmatter dan
  `_meta.yaml.files[]`, tanpa menghapus atau menyalin isi file.
- **AC-12** `forward --to` ke topic yang tidak meng-extend topic asal ditolak dan tidak
  mengubah filesystem apa pun.
- **AC-13** `historic forward <path> --create "<title>"` membuat topic baru dengan
  `extends` = topic asal lalu menandai file `forwarded`; jika pembuatan gagal, status file
  tidak berubah.
- **AC-14** `forwarded` tersedia sebagai filter `historic find --status forwarded` dan
  masuk siklus filter status TUI.
- **AC-15** `historic status <path> progress` pada file `forwarded` menghapus
  `forwarded_to` dari frontmatter dan manifest.
- **AC-16** `historic show <topic-id> --json` menampilkan `forwarded_to` per file bila
  ada.

## 11. Test matrix

| Kasus | Input | Ekspektasi |
| --- | --- | --- |
| Status pada topic closed | `status .historic/.database/<id>/wos/01-x.md review` | Sukses, frontmatter `review`, manifest sinkron |
| Status pada topic open | `status wos/01-x.md progress` | Sukses seperti sebelumnya |
| Path ambigu (open + closed) | `status wos/01-x.md complete` | `ErrConflict`, tidak memilih otomatis |
| Add pada topic closed | `add "Catatan" --id <closed-id>` | `ErrTopicMissing` (perilaku lama, tanpa sentinel baru) |
| Add setelah `open` | `open <id>` lalu `add "Catatan" --id <id>` | Sukses |
| Create extension | `create "Lanjutan" --extends <closed-id>` | `_meta.yaml` memuat `extends: "<closed-id>"` |
| Create extension ke ID tidak ada | `create "Lanjutan" --extends 1799999999999` | `ErrTopicMissing`, tidak ada direktori dibuat |
| Rantai extension | A closed ← B closed ← C | `extends` masing-masing menunjuk induk langsung |
| Show anak | `show <child-id> --json` | `data[].extends` terisi |
| Show induk | `show <parent-id> --json` | Identik dengan sebelum fitur |
| Filter status review | `find "" --status review` | Hanya file berstatus `review` |
| Forward ke child valid | `forward wos/01-x.md --to <child-id>` | Status `forwarded`, `forwarded_to` terisi |
| Forward ke topic tak terkait | `forward wos/01-x.md --to <other-id>` | Error validasi, tanpa perubahan |
| Forward dengan --create | `forward wos/01-x.md --create "Lanjutan"` | Topic baru `extends` asal, file `forwarded` |
| Forward dua opsi sekaligus | `--to` + `--create` | Error validasi |
| Forward pada salinan closed | `forward .historic/.database/<id>/wos/01-x.md --to <child>` | Ditolak, arahkan ke `open` |
| Forward ulang | `forward` dua kali ke child berbeda | `forwarded_to` diperbarui, idempoten |
| Batalkan forward | `status wos/01-x.md progress` | `forwarded_to` hilang |
| Filter forwarded | `find "" --status forwarded` | Hanya file `forwarded` |
| Show forwarded_to | `show <topic-id> --json` | Field `forwarded_to` per file |
| Rebuild workspace ber-extension | `rebuild --json` | Idempotent, `errors: 0` |
| Legacy `_meta.yaml` tanpa `extends` | `rebuild` | Valid, `extends` kosong |

## 12. Pertanyaan terbuka

> **Keputusan 2026-10-28 (ditutup/tetap):**
> 1. `--extends` **menerima topic open maupun closed** (menegaskan §5.3). Tidak ada
>    pembatasan closed-only.
> 2. Nomor WO dirapikan menjadi 01–10 (WO-03 lama dibatalkan dan tidak lagi menyisakan
>    nomor bolong). Judul topic juga disesuaikan (nama folder tetap).
> 3. Pertanyaan lain (§12 no.2–6) **ditunda** sesuai rekomendasi.
>
> Sisa daftar di bawah ini dipertahankan sebagai catatan rekomendasi.

1. Apakah `--extends` harus **dibatasi hanya untuk topic closed** (sesuai niat awal),
   atau tetap mengizinkan topic open? ~~Rekomendasi: batasi hanya closed~~ **Diputuskan
   2026-10-28: menerima open maupun closed.**
2. Apakah `extends` perlu ditampilkan juga pada `historic list` dan `historic find`
   (mis. kolom/field `extends_id`)? Rekomendasi: tidak untuk MVP; cukup `show`.
3. Apakah backlink turunan (derived, dari index) perlu diekspos sebagai
   `extends_children` pada `show`? Rekomendasi: ditunda.
4. Apakah konten task perlu ikut dipindahkan otomatis ke topic tujuan saat `forward`?
   Rekomendasi SPEC ini: tidak; pengguna membuat file padanan sendiri. Bila dibutuhkan,
   dapat menjadi `--move` pada iterasi berikutnya.
5. Apakah `forward` perlu mendukung banyak path sekaligus (bulk)? Rekomendasi: ditunda.
6. Apakah `forwarded` perlu tampil sebagai kolom khusus pada `historic list`?
   Rekomendasi: tidak untuk MVP; cukup `find --status` dan `show`.
