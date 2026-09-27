---
title: 'WO: implement `historic git` command proxy'
description: Add a safe Git process passthrough rooted at `.historic/.database`.
status: complete
created: "2026-09-27"
updated: "2026-09-27"
tags:
    - git
    - cli
    - process
    - proxy
related:
    - ./spec-git-command-proxy.md
---
# WO: Implement `historic git <args...>`

## Tujuan
Implementasikan command proxy sesuai SPEC tanpa mengubah semantics command Git internal Historic yang sudah ada.

## Inventaris kode yang perlu ditinjau
- Cobra root/command registration dan error/status handling di `internal/cmd/root.go` serta command files.
- Workspace resolution/config untuk memperoleh path `.historic/.database` (jangan menebak cwd relatif).
- Pola exec/process dan dukungan stdin/stdout/stderr; cari utilities yang ada sebelum menambah abstraksi.
- Existing internal `save`, `log`, `diff`, `restore` commands: pastikan `historic git log` tidak ambigu dengan `historic log` karena dispatch hanya pada child command `git`.
- Build/test setup untuk PATH/executable stubbing yang deterministik.

## Rencana implementasi
1. Tambahkan Cobra command `git <git-args...>` dengan variadic args dan registrasikan pada root/help/completion.
2. Resolve workspace secara konsisten dengan command Historic lain; set `cmd.Dir` ke absolute `.historic/.database`.
3. Gunakan `exec.Command("git", args...)` tanpa shell. Hubungkan `Stdin`, `Stdout`, `Stderr` ke stream command parent.
4. Propagasikan child exit code melalui konvensi error yang dikenali root `Execute`; jangan meratakan semua Git failures menjadi status 1 bila kode tersedia.
5. Tangani error `exec.LookPath`/`Start` (Git tidak tersedia), direktori database tidak ada, dan fatal not-a-repository dengan pesan tanpa mengubah atau menginisialisasi repository.
6. Buat usage untuk argumen kosong. Pastikan empty args tidak diam-diam berubah menjadi `git` tanpa subcommand.
7. Teruskan semua subcommand Git secara transparan, termasuk `push`, `pull`, `commit`, `fetch`, `merge`, dan `init`, hanya ketika diminta eksplisit oleh pengguna; jangan menambahkan operasi otomatis.
8. Jangan mengubah konfigurasi Git, menambahkan `--no-pager`, atau menonaktifkan interaktivitas.
9. Tambahkan docs di `docs/commands.md` dan lokasi help/README relevan. Jelaskan cwd `.historic/.database`, passthrough penuh, dan bahwa tindakan seperti push hanya terjadi atas argumen eksplisit pengguna.

## Tes
- Command menggunakan temp workspace/repository lokal dan assert bahwa cwd Git adalah `.historic/.database`.
- Stub executable/script atau fixture terisolasi untuk verifikasi urutan argumen, environment/cwd, stream forwarding, dan exit code.
- Tes missing Git executable, missing `.database`, no args, serta child exit non-zero.
- Tes memastikan karakter khusus pada args tidak di-shell-evaluate.
- Tes root/help memastikan command terdaftar tanpa mengubah command `historic log`, `historic diff`, `historic save`, atau `historic restore`.
- Jalankan `go test ./internal/cmd`, kemudian `go test ./...`.

## Area file kandidat
- `internal/cmd/git.go` (baru), `internal/cmd/root.go` dan tes command terkait.
- Workspace/config helper hanya jika perlu accessor path yang didukung API existing.
- `docs/commands.md` dan help docs.

Jaga diff minimal; jangan ubah source lain sebelum kebutuhan terbukti. Tidak ada push.

## Acceptance Criteria
- Seluruh acceptance criteria di SPEC terpenuhi.
- Tes memverifikasi cwd `.historic/.database`, passthrough stream/args/status, serta error paths.
- Tidak ada `git init`, push, atau shell execution otomatis.
- Semua tes paket terkait dan `go test ./...` lulus.

## Status approval
Approved for implementation; implementation and validation completed.

## Hasil implementasi
- Added `historic git <git-args...>` as a direct process passthrough rooted at `.historic/.database`; no shell or implicit repository initialization is used.
- Forwarded stdin/stdout/stderr and preserved Git child exit status; covered argument safety, missing executable/database, not-a-repository, no arguments, help registration, and internal command coexistence with isolated tests.
- Documented the proxy in `README.md` and `docs/commands.md`.
Tidak ada push.
