---
id: 00001
title: WO 24 Interactive Search TUI
status: progress
created: 2026-09-19
updated: 2026-09-19
tags: [phase-5, tui, search, interactive]
related: [../spec-interactive-search-tui.md, ../spec.md]
---

# WO 24 — Interactive Search TUI MVP

## Tujuan

Mengimplementasikan `historic search` sebagai antarmuka pencarian interaktif berbasis terminal untuk pengguna manusia, dengan tetap mempertahankan `historic find` sebagai command one-shot untuk AI, script, dan automation.

## Referensi

- SPEC utama: `../spec-interactive-search-tui.md`
- Baseline search Phase 4: `../spec.md`
- Command existing: `historic find`

## Scope

### Termasuk

- Menambahkan command `historic search`.
- Membuka sesi TUI hanya pada terminal interaktif.
- TUI read-only dengan panel daftar hasil dan preview.
- Input query interaktif.
- Limit default `20`.
- Pagination untuk hasil yang melebihi limit.
- Filter status: `all`, `create`, `pending`, `progress`, `review`, `blocked`, `complete`, `failed`, `cancelled`, `archived`.
- Filter scope: `active`, `archived`, `all`.
- Filter type minimal `all`, `topic`, dan `work-order`.
- Navigasi keyboard untuk query, hasil, filter, pagination, refresh, dan exit.
- Preview judul, status, path, dan snippet/isi yang relevan.
- Preview asset atau binary sebagai metadata tanpa merender isi binary.
- Reuse search service, ranking, snippet, filter, dan index FTS5 yang digunakan `historic find`.
- Unit test state query/filter/pagination.
- Integration test command dan validasi terminal non-interaktif.
- Smoke test terisolasi untuk memastikan TUI tidak mengubah workspace.
- Dokumentasi command dan shortcut dasar.

### Tidak termasuk

- Aksi mutasi dari TUI seperti `complete`, `archive`, `status`, atau `sync-meta`.
- Editor Markdown penuh.
- Dukungan mouse sebagai requirement MVP.
- Web UI, remote sync, multi-user, AI chat, atau filesystem watcher.
- Search index baru khusus TUI.
- Perubahan terhadap kontrak JSON `historic find`.

## Kontrak command

```text
historic search
```

Perilaku minimum:

1. Membuka alternate screen bila didukung library terminal.
2. Menampilkan query, filter aktif, limit, daftar hasil, preview, dan shortcut.
3. Query kosong menampilkan topic aktif terbaru dengan limit default.
4. Perubahan query/filter memperbarui hasil tanpa keluar dari sesi.
5. Hasil kosong ditampilkan sebagai keadaan normal.
6. Index yang tidak tersedia memberi pesan actionable yang menyarankan `historic rebuild`.
7. Query invalid tidak menyebabkan TUI crash.
8. `q`, `Esc`, atau `Ctrl+C` mengakhiri sesi dan mengembalikan terminal ke keadaan semula.
9. Jika dijalankan tanpa terminal interaktif, command gagal dengan error jelas dan tidak mengubah workspace.
10. Command tidak mengubah Markdown, status, archive, asset, atau metadata topic.

Perilaku `historic search --json` harus mengikuti keputusan final pada SPEC sebelum implementasi. Jika belum dikunci, command harus menolak kombinasi tersebut dengan error yang jelas, bukan mencampur output JSON dan TUI.

## Interaksi keyboard MVP

- `↑` / `↓`: memilih hasil.
- `Enter`: memilih hasil dan memperbarui preview.
- `/`: fokus input query.
- `f`: fokus filter.
- `n`: halaman berikutnya.
- `p`: halaman sebelumnya.
- `r`: menjalankan ulang query.
- `Esc`: menutup input/filter aktif atau kembali.
- `q`: keluar.
- `Ctrl+C`: keluar dengan aman.

## Acceptance Criteria

- `historic search` dapat dibuka dari workspace Historic yang valid.
- TUI menolak lingkungan non-interaktif tanpa memodifikasi workspace.
- Query dapat diketik, dihapus, dan dijalankan ulang.
- Query kosong menampilkan topic aktif terbaru.
- Limit default adalah 20 dan pagination bekerja deterministik.
- Filter status, scope, dan type konsisten dengan `historic find`.
- Daftar hasil dapat dinavigasi tanpa mouse.
- Preview berubah mengikuti hasil yang dipilih.
- Result list memiliki viewport yang dibatasi oleh tinggi terminal yang tersedia setelah header dan footer.
- Item aktif selalu tetap terlihat saat navigasi keyboard.
- Result list tidak menggambar atau tenggelam di bawah footer.
- Judul/path panjang dipotong atau di-wrap secara aman tanpa mendorong footer keluar layar.
- Terminal kecil menampilkan layout ringkas yang tetap dapat dinavigasi.
- Pagination atau scrolling tetap memungkinkan semua hasil dalam halaman diakses.
- Preview tidak mendorong footer keluar viewport.
- Query menggunakan search service dan FTS5 yang sama dengan `historic find`.
- Hasil kosong, index hilang, dan query invalid ditangani tanpa crash.
- Preview read-only dan tidak mengubah Markdown atau status.
- `historic find` dan `historic find --json` tetap berfungsi dengan kontrak yang ada.
- Tidak ada file temporary tertinggal setelah exit atau error.
- Unit, integration, dan smoke test tersedia serta lulus.
- Dokumentasi shortcut dan penggunaan tersedia.

## Dogfooding Notes

### Blocker — Result list tenggelam pada viewport terminal

Ditemukan saat dogfooding: result list dapat melebihi tinggi area yang tersedia sehingga sebagian hasil tampak tenggelam/terpotong dan berpotensi menggambar di bawah footer. Perilaku ini analog dengan masalah `overflow: hidden` pada layout web.

Perbaikan harus menghitung tinggi viewport berdasarkan ukuran terminal aktual, header, footer, dan preview; menjaga item aktif tetap terlihat; serta menangani judul/path panjang dan terminal kecil secara aman. Blocker ini harus diselesaikan sebelum WO 24 dapat ditandai `complete`.

## Validasi wajib

```text
gofmt -d <changed-go-files>
go test ./...
go vet ./...
go build -o /tmp/historic .
git diff --check
```

Smoke test harus menggunakan direktori temporary terpisah dan memverifikasi checksum atau diff workspace sebelum dan sesudah sesi TUI. Jangan menggunakan workspace utama untuk simulasi interaktif kecuali diminta secara eksplisit.

## Catatan implementasi

- Pilih library TUI yang kompatibel dengan single binary Go dan platform target yang didukung.
- Pisahkan state/input/rendering dari search service.
- Jangan membaca SQLite langsung dari layer tampilan.
- Pertahankan output manusia `historic find` dan JSON envelope `{ "command", "ok", "data", "error" }`.
- Jangan mengubah `PRD.md`.
- Jangan melakukan push ke remote repository.

## Status

Progress. Implementasi dapat dimulai setelah keputusan terbuka pada SPEC Phase 5 dikunci oleh tim.
