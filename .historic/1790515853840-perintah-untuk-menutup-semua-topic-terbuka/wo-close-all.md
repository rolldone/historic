---
title: 'WO: implement `historic close all`'
description: Add bulk-close selector while reusing existing topic lifecycle close behavior.
status: complete
created: "2026-09-27"
updated: "2026-09-27"
tags:
    - lifecycle
    - close
    - bulk-operation
    - cli
related:
    - ./spec-close-all.md
---
# WO — Implement `historic close all`

## Tujuan
Implement command `historic close all [--json]` yang menutup semua topic open dengan memanggil ulang lifecycle close individual, melanjutkan proses setelah kegagalan per-topic, dan melaporkan hasil agregat.

## Konteks arsitektur yang harus dipertahankan
- Registrasi command ada di `internal/cmd/root.go`; command close saat ini dibangun melalui `newStorageCommand` di `internal/cmd/storage.go`.
- Lifecycle operasi single close ada di `internal/lifecycle/storage.go` (`Service.Close`); gunakan implementasi itu, jangan menduplikasi staging, snapshot, rollback, atau index rebuild.
- Resolusi storage/topic location serta validasi folder tersedia di package `internal/lifecycle`.
- Index SQLite direbuild oleh lifecycle close. Jangan mengakses SQLite langsung dari command/presentation layer.
- Envelope JSON CLI mengikuti `{command,ok,data,error}` dan output/error ditangani utilitas di `internal/cmd`.
- Tes close/lifecycle berada di `internal/lifecycle/archive_test.go`; tes command terkait ada di `internal/cmd/lifecycle_test.go` dan tes root/storage lain. Sesuaikan lokasi setelah memeriksa pola aktual.

## Desain implementasi

### 1. Sintaks dan dispatch
- Pertahankan `historic close <id>` tanpa perubahan kompatibilitas.
- Terima selector literal `all` sebagai bentuk bulk: `historic close all [--json]`.
- Dispatch bulk di handler command close sebelum parsing argumen sebagai TopicID; jangan mengubah command `open` atau `import`.
- Pastikan `all` tidak diterima oleh operasi single-topic lain.

### 2. Penemuan target
- Tambahkan API lifecycle khusus, misalnya `Service.CloseAll()`, agar enumerasi filesystem dan proses operasi berada di package lifecycle.
- Enumerasi hanya direktori topic langsung pada work root `.historic/`; exclude `.database`, `.index.sqlite`, `.git`, `.staging-*`, file non-direktori, symlink, dan entri invalid.
- Validasi nama folder/TopicID memakai helper konvensi yang sudah ada. Jangan ikut memproses folder nested.
- Ambil snapshot daftar target satu kali di awal, lalu urutkan deterministik berdasarkan ID dan path. Topic yang muncul setelah enumerasi tidak termasuk run saat itu.
- Duplikat ID atau entri symlink/invalid: tentukan sebagai hasil gagal yang actionable, tanpa menghapus atau memindahkan target yang tak tervalidasi.

### 3. Orkestrasi per-target
- Untuk setiap target snapshot, panggil `Service.Close(id)` yang sama dengan close tunggal.
- Tangkap error target secara independen lalu lanjutkan ke berikutnya.
- Hindari menahan lock global di sepanjang batch jika mekanisme lock existing tidak reentrant; lock lifecycle per operasi dan penanganan duplicate/race harus mencegah close tidak aman.
- Hasil domain memuat daftar success (`ID`, title/path/storage bila tersedia) dan failure (`ID`, path, error string), beserta total/closed/failed counts.
- Kasus daftar kosong sukses dengan counts nol.

### 4. Command/output/exit status
- Ubah konstruksi close secukupnya agar mendukung dispatch `all`; jangan mengubah output close tunggal.
- Human output: satu baris hasil per topic dan ringkasan akhir. Error parsial harus terlihat.
- JSON: satu JSON envelope valid saja; `data` selalu memuat `succeeded`, `failed`, `total`, `closed`, `failed_count`; `ok=false` bila ada failure.
- Bila ada kegagalan, emit hasil JSON/human terlebih dahulu lalu kembalikan error bertipe/terkendali agar exit status non-zero. Pastikan jalur error tidak menulis envelope kedua atau mencemari stdout JSON.
- Semua sukses/kosong menghasilkan exit status zero.

### 5. Konsistensi lifecycle
- Per topic pertahankan jaminan existing: closed snapshot tervalidasi, workdir baru dihapus setelah operasi/index rebuild sukses, rollback dilakukan jika close gagal.
- Jangan mengubah status frontmatter atau menghapus topic permanen.
- Rebuild index per topic sesuai close existing pada tahap awal; optimasi batch satu kali rebuild hanya boleh dilakukan jika dirancang tanpa melemahkan rollback/atomicity dan diuji terpisah.

## Tes wajib
- Lifecycle: enumerasi nol/satu/beberapa topic; hanya direktori tingkat pertama; closed, staging, metadata files dan symlink tidak ikut; urutan deterministik.
- Lifecycle: semua close sukses menghasilkan closed snapshots dan menghapus work dirs.
- Partial failure: satu topic close gagal tetapi topic berikutnya tetap dicoba; hasil mencatat sukses/gagal benar.
- Failure safety: snapshot/workdir tetap aman sesuai kontrak single close; tidak ada staging tertinggal.
- Command: `close all` dispatch tepat; `close <id>` regression mempertahankan output/semantik.
- Output: human detail+summary; JSON valid satu envelope dan counts/records akurat; error parsial exit non-zero; kosong dan semua sukses exit zero.
- Concurrency/race: topic yang berubah/ditutup oleh proses lain antara enumerasi dan pemrosesan dilaporkan gagal per topic, tidak menyebabkan batch abort atau korupsi.
- Jalankan `go test ./internal/lifecycle ./internal/cmd`, lalu `go test ./...`.

## File yang diperkirakan terdampak
- `internal/cmd/root.go`, `internal/cmd/storage.go`, tipe/serializer output terkait.
- `internal/lifecycle/storage.go` dan API/tipe hasil lifecycle terkait.
- Tes di `internal/cmd` dan `internal/lifecycle`.
- Dokumentasi command (`docs/commands.md`, README/help yang relevan).

Daftar indikatif; dev perlu mengikuti struktur aktual dan menjaga diff minimal. Tidak ada perubahan PRD, operasi push, atau penghapusan permanent.

## Acceptance Criteria
- Semua acceptance criteria di `spec-close-all.md` terpenuhi.
- Close tunggal tetap backward-compatible.
- Batch memanggil lifecycle close yang ada dan deterministik.
- Kegagalan per topic terisolasi, dicatat, dan menghasilkan exit non-zero setelah seluruh target diproses.
- JSON/human output menunjukkan hasil tiap target dan ringkasan akurat tanpa output ganda.
- Tes paket terkait dan `go test ./...` lulus.

## Status approval
Draft untuk direview. Belum disetujui untuk implementasi; tunggu approval pengguna.
