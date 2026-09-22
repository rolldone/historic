---
id: "00004"
title: WO 03 Automatic Slugging for Add Filenames
status: complete
created: "2026-09-21"
updated: "2026-09-21"
---
# WO 03 — Automatic Slugging for `historic add`

## 1. Ringkasan Teknis

Tambahkan normalisasi slug pada pipeline `historic add` agar argumen `<name>` tidak langsung digunakan sebagai basename filesystem. Normalisasi harus dilakukan di layer repository yang membentuk relative entry path, bukan hanya pada output CLI.

Target perilaku:

`historic add "Remove stale Cost Dashboard breakdowns" --id 00032 --json`

menghasilkan:

`.historic/00032-<topic-slug>/remove-stale-cost-dashboard-breakdowns.md`

Untuk input Work Order:

`historic add "wos/Remove stale Cost Dashboard breakdowns" --id 00032 --json`

menghasilkan:

`.historic/00032-<topic-slug>/wos/NN-remove-stale-cost-dashboard-breakdowns.md`

`NN` harus tetap mengikuti allocator Work Order yang sudah ada.

## 2. Lokasi Perubahan yang Disarankan

- `internal/repository/topics.go`
  - Pertahankan `TopicStore.AddEntry` sebagai entry point.
  - Perketat/ubah `normalizeEntryName` agar basename dinormalisasi menjadi slug.
  - Pastikan prefix directory seperti `wos/` dipisahkan dari basename sebelum slugification.
  - Jalankan slugification sebelum `nextWorkOrderPath`.
- `internal/repository/add_test.go`
  - Tambahkan table-driven unit test untuk `TopicStore.AddEntry`.
- `internal/cmd/add_test.go`
  - Tambahkan integration/command test untuk output JSON dan actual filesystem path.
- Jangan mengubah `internal/domain.SlugTitle` kecuali ada alasan kuat; fungsi tersebut saat ini berkontrak untuk topic folder dan perubahan dapat berdampak pada storage topic.

## 3. Pipeline yang Wajib Dipertahankan

Urutan pemrosesan `AddEntry` harus secara logis menjadi:

1. Validasi `topicID`.
2. Resolve active topic path.
3. Validasi raw relative path terhadap absolute path, traversal, dan hidden path sesuai kontrak existing.
4. Pisahkan directory prefix dari basename.
5. Slugify basename.
6. Pastikan ekstensi `.md` ditambahkan/ditangani sekali saja.
7. Jika directory adalah `wos/` dan basename belum memiliki prefix nomor `NN-`, alokasikan nomor melalui `nextWorkOrderPath`.
8. Bentuk `entryPath` dengan `filepath.Join` dari komponen yang telah tervalidasi.
9. Cek collision dan hormati `force`.
10. Tulis Markdown dan update `_meta.md` seperti perilaku existing.
11. Return `domain.Entry.Filename` dengan path relatif POSIX-compatible untuk output dan metadata.

Catatan penting: slugification tidak boleh dilakukan sebelum validasi traversal. Input seperti `../Remove stale` atau `/tmp/Remove stale` harus tetap ditolak, bukan diubah menjadi nama yang tampak aman.

## 4. Kontrak Fungsi Normalisasi

Disarankan memisahkan helper pure, misalnya:

`normalizeEntryName(name string) (string, error)`

atau helper internal tambahan untuk basename, dengan kontrak:

- Input non-empty setelah trim.
- Output selalu relative path.
- Separator directory menggunakan `/` secara logis, lalu dikonversi memakai `filepath.FromSlash` saat akses filesystem.
- Basename tanpa extension menjadi slug lowercase dengan separator `-`.
- `.md` case-insensitive diterima dan tidak diduplikasi.
- Input `foo.md` menghasilkan `foo.md`.
- Input `Foo Bar.MD` menghasilkan `foo-bar.md` atau, bila case extension harus dipertahankan sebagai kontrak existing, dokumentasikan keputusan tersebut dan test secara eksplisit.
- Tidak menghasilkan `--`, leading `-`, trailing `-`, atau basename kosong.
- Karakter non-ASCII/non-alfanumerik harus memiliki perilaku deterministik: dipertahankan bila aman, atau diubah menjadi separator/dihapus sesuai keputusan implementasi. Pilih satu kebijakan dan cover dengan test.
- Nama yang hanya berisi separator/karakter tidak valid menghasilkan `domain.ErrConflict` atau error domain yang konsisten dengan validasi existing.

## 5. Aturan Slugification yang Definitif

Implementasi harus menetapkan aturan berikut:

| Input | Output |
|---|---|
| `Remove stale Cost Dashboard breakdowns` | `remove-stale-cost-dashboard-breakdowns.md` |
| `  Multiple   spaces  ` | `multiple-spaces.md` |
| `Already-slugged-name` | `already-slugged-name.md` |
| `name_with_separator` | `name-with-separator.md` |
| `file.md` | `file.md` |
| `FILE.MD` | `file.md` |
| `wos/Review Login Flow` | `wos/NN-review-login-flow.md` sebelum numbering final |

Prefix directory selain `wos/` harus tetap mengikuti validasi path existing. Jangan slugify atau merename directory prefix secara implisit.

## 6. Work Order Numbering

- Slugification harus terjadi sebelum `nextWorkOrderPath` dipanggil agar allocator memeriksa basename yang sudah canonical.
- Existing file dengan format `NN-<slug>.md` tidak boleh diberi nomor kedua.
- Untuk input `wos/01-Review Login Flow`, implementasi harus menetapkan kontrak eksplisit: apakah `01-` dipertahankan sebagai explicit numbered filename atau dianggap basename dan dinormalisasi. Rekomendasi: pertahankan nomor valid, slugify hanya bagian setelah `NN-`, lalu jangan alokasi nomor baru.
- Collision tetap ditolak tanpa `--force`.
- `--force` tetap hanya mengizinkan overwrite pada path final yang sudah dinormalisasi.

## 7. Metadata dan JSON Contract

Tidak mengubah schema output JSON. Field berikut harus tetap konsisten:

- `data.id`: ID topic lima digit.
- `data.file`: path aktual relatif dari project root, menggunakan separator `/`.
- `data.title`: judul entry yang ditetapkan oleh kontrak existing; implementer harus memastikan apakah title berasal dari slug/path atau raw input, lalu menambahkan test agar tidak terjadi perubahan tidak disengaja.
- `error`: tetap `null` pada operasi sukses.

`_meta.md` harus menerima link ke path final yang sama dengan `data.file`, tanpa spasi mentah dan tanpa duplicate extension.

## 8. Error Handling dan Security

- Pertahankan penolakan absolute path.
- Pertahankan penolakan traversal (`..`) setelah normalisasi path separator.
- Jangan gunakan shell command untuk membentuk nama file.
- Jangan mengikuti symlink sebagai topic root/file target bila aturan repository existing melarangnya.
- Jangan menghapus atau menimpa file lain akibat hasil slug collision.
- Jika dua input berbeda menghasilkan slug sama, perlakukan sebagai collision normal dan ikuti aturan `--force`.

## 9. Test Matrix Minimum

Tambahkan test untuk:

1. Nama biasa tanpa spasi.
2. Kalimat dengan spasi tunggal.
3. Leading/trailing whitespace.
4. Multiple whitespace.
5. Uppercase/lowercase.
6. Separator `_` dan `-` berulang.
7. Extension `.md`, `.MD`, dan extension ganda.
8. Prefix `wos/` dengan auto-numbering.
9. Prefix `wos/NN-` bila dipertahankan sebagai explicit number.
10. Collision setelah slugification.
11. `--force` terhadap path final.
12. Absolute path.
13. `../` dan variasi separator traversal.
14. Input kosong atau hanya separator.
15. Verifikasi file aktual, `data.file`, link `_meta.md`, frontmatter, dan isi heading.

## 10. Definition of Done

- Semua test existing tetap lulus.
- Test baru untuk repository dan command lulus.
- `go vet ./...` lulus.
- `git diff --check` lulus.
- Tidak ada perubahan pada topic folder slugification atau lifecycle command.
- Dokumentasi command/reference diperbarui bila kontrak user-facing berubah.
- Perubahan source code dilakukan oleh coding agent, bukan Manager Agent.
- Tidak ada push ke remote repository.

## 11. Status

- SPEC/WO: siap diimplementasikan.
- Source code: belum diubah oleh Manager Agent.
- Push remote: tidak dilakukan oleh Manager Agent.
