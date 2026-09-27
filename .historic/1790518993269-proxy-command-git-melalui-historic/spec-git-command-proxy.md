---
title: "SPEC: `historic git` command proxy"
description: Proxy Git CLI commands with the Historic database directory as the working directory.
status: complete
created: "2026-09-27"
updated: "2026-09-27"
---
# SPEC: Proxy command `historic git <args...>`

## Tujuan
Menyediakan entry point Historic yang menjalankan Git pada repository internal di `.historic/.database`, tanpa pengguna harus berpindah direktori atau memanggil Git secara langsung.

## Sintaks
```sh
historic git <git-args...>
```
Contoh: `historic git status`, `historic git log -1 --oneline`, `historic git diff --stat`.

## Perilaku
- Cari executable `git` melalui `PATH`; jangan mengunduh/memasang Git otomatis.
- Set working directory proses Git ke `.historic/.database` (root kerja repository internal), bukan project root, `.historic` root, atau folder topic.
- Teruskan argumen sebagai argumen proses tanpa shell interpretation; jangan menyusun shell command string.
- Inherit/forward stdin, stdout, dan stderr secara langsung agar interaktif, pager, dan output Git bekerja sewajarnya.
- Kembalikan exit status Git kepada pemanggil secara akurat. Jika executable tidak tersedia, error actionable dan exit non-zero.
- Tanpa argumen: tampilkan usage `historic git <git-args...>` dan contoh command, non-zero.
- Tidak menambahkan `--no-pager`, tidak menonaktifkan terminal/prompt, tidak mengubah konfigurasi Git.
- Semua subcommand dan flags Git diteruskan apa adanya, termasuk `push`, `pull`, `commit`, `fetch`, `merge`, dan `init`; proxy tidak menjalankan tindakan tersebut secara otomatis.
- Jangan mengubah command Historic internal yang sudah ada (`save`, `log`, `diff`, `restore`); command proxy ini hanya meneruskan subcommand setelah `git`.
- Workspace `.database` yang tidak ada atau belum memiliki repository Git ditangani dengan pesan error jelas; proxy tidak menjalankan `git init` otomatis. Jika pengguna secara eksplisit meminta `historic git init`, perintah itu diteruskan sebagai subcommand Git biasa.

## Keamanan dan batas
- Argumen Git dianggap instruksi eksplisit pengguna; command tidak melewati shell.
- Proxy tidak membatasi subcommand atau flags Git. Perintah destruktif hanya terjadi bila secara eksplisit diminta pengguna.
- Remote/konfigurasi repository internal mengikuti repository lokal yang ada. Tidak melakukan push otomatis atau menambah remote.

## Kriteria penerimaan
- `historic git status` menjalankan Git dengan cwd `.historic/.database` dan hasil menunjukkan repository internal.
- Argumen/options diteruskan utuh, termasuk beberapa argumen dan pathspec.
- stdin/stdout/stderr tersedia untuk proses child; exit status sukses/gagal dipertahankan.
- Tidak ada shell interpolation untuk karakter khusus dalam argumen.
- Error mencakup kondisi Git tidak terpasang, database directory hilang, dan bukan repository.
- Argumen kosong menampilkan usage proxy.
- Command terlihat di help/completion dan terdokumentasi.
