---
id: 00001
title: Historic Interactive Search TUI SPEC
status: draft
created: 2026-09-19
updated: 2026-09-19
tags: [spec, tui, search, interactive, phase-5]
related: [spec.md]
---

# Interactive Search TUI SPEC — Historic Phase 5

## 1. Tujuan

Menyediakan antarmuka pencarian interaktif berbasis terminal untuk pengguna manusia, tanpa mengubah CLI one-shot yang sudah digunakan oleh AI, script, dan automation.

Fitur ini menjadi lapisan presentasi di atas search engine Historic yang sudah ada. Markdown tetap menjadi source of truth, sedangkan SQLite/FTS5 tetap menjadi index yang dapat dibangun ulang.

## 2. Prinsip desain

1. CLI tetap menjadi interface utama untuk AI dan automation.
2. TUI menjadi interface eksplorasi untuk user yang ingin browsing dan memfilter hasil.
3. TUI tidak memiliki search engine atau database terpisah.
4. Query dan filter TUI menggunakan service search yang sama dengan `historic find`.
5. TUI Phase 5 bersifat read-only.
6. Tidak ada perubahan terhadap file Markdown hanya karena user membuka atau mencari data.
7. Tidak ada push, remote sync, atau ketergantungan webapp.

## 3. Kontrak command

```text
historic find "<keyword>" [options] [--json]
historic search
```

- `historic find` tetap menjadi command one-shot dan kontrak untuk AI, script, serta automation.
- `historic search` membuka sesi TUI interaktif.
- `historic search` tidak memerlukan query awal; user dapat mengetik query setelah TUI terbuka.
- `--json` tidak berlaku untuk sesi TUI interaktif. Jika diberikan, command menolak dengan error yang jelas atau menggunakan command `find` sebagai gantinya; perilaku final harus dikunci sebelum implementasi.
- Jika stdout/stderr bukan terminal interaktif, command menolak dengan error actionable dan tidak mengubah workspace.

## 4. Layout dan pengalaman pengguna

Layout minimum yang ditargetkan:

```text
┌─ Historic Search ─────────────────────────────────────────────┐
│ Query: sync                                                   │
│ Filter: status=all  scope=active  type=all  limit=20          │
├──────────────────────────┬────────────────────────────────────┤
│ Results                  │ Preview                            │
│                          │                                    │
│ > 00001 Sync Meta       │ # Sync Meta                         │
│   00007 Search Design   │ status: progress                    │
│   00012 Remote Sync     │                                    │
│                          │ ## Progress                         │
│                          │ ...                                │
├──────────────────────────┴────────────────────────────────────┤
│ ↑↓ Navigate  Enter Preview  / Query  f Filter  n/p Page  q Quit│
└────────────────────────────────────────────────────────────────┘
```

Tampilan boleh menyesuaikan ukuran terminal, tetapi harus mempertahankan tiga informasi utama:

- query dan filter aktif;
- daftar hasil yang sedang ditampilkan;
- preview hasil yang sedang dipilih.

## 5. Search dan filter MVP

TUI harus mendukung:

- query teks yang dapat diubah secara interaktif;
- hasil yang diperbarui setelah query berubah dengan debounce atau mekanisme input yang setara;
- limit default `20`;
- pagination ketika hasil melebihi limit;
- filter status: `all`, `create`, `pending`, `progress`, `review`, `blocked`, `complete`, `failed`, `cancelled`, `archived`;
- filter scope: `active`, `archived`, `all`;
- filter type minimal `all`, `topic`, dan `work-order`;
- reset query dan filter tanpa keluar dari sesi.

Search menggunakan field dan ranking yang tersedia pada `historic find` Phase 4, termasuk path, filename, title, content, BM25, snippet, dan tie-breaker path.

Query kosong pada MVP menampilkan topic aktif terbaru, dengan limit default, agar TUI berguna sebagai topic browser sekaligus search interface.

## 6. Navigasi keyboard MVP

- `↑` / `↓`: memilih hasil.
- `Enter`: memilih hasil dan memperbarui preview.
- `/`: memfokuskan input query.
- `f`: membuka atau memfokuskan filter.
- `n`: halaman berikutnya.
- `p`: halaman sebelumnya.
- `r`: menjalankan ulang query saat ini.
- `Esc`: menutup input/filter aktif atau kembali ke tampilan sebelumnya.
- `q`: keluar dari TUI.
- `Ctrl+C`: keluar dengan aman.

MVP tidak mewajibkan dukungan mouse. Dukungan mouse dapat ditambahkan kemudian tanpa mengubah kontrak search.

## 7. Preview dan pembukaan file

- Preview hanya-baca dan tidak mengubah file.
- Preview menampilkan judul, status, path, dan potongan isi yang relevan bila tersedia.
- Hasil yang dipilih boleh berupa topic, Work Order, atau file terindeks sesuai filter.
- `Enter` pada MVP hanya memilih hasil dan menampilkan preview.
- Membuka editor eksternal atau menjalankan aksi lifecycle bukan bagian dari MVP.
- Preview untuk binary/asset menampilkan metadata path dan tipe file tanpa mencoba merender isi biner sebagai Markdown.

## 8. Integrasi dengan search engine

Alur data yang diharapkan:

```text
historic search
      │
      ▼
interactive input + filter state
      │
      ▼
shared search service
      │
      ▼
SQLite FTS5 index
      │
      ▼
results + snippets + metadata
      │
      ▼
TUI result list and preview
```

TUI tidak boleh membaca SQLite secara langsung dari layer tampilan. Query, filter, ranking, pagination, dan error handling harus berada pada service yang dapat digunakan bersama oleh `find` dan TUI.

## 9. Error dan recovery

- Index yang tidak tersedia menghasilkan pesan yang menyarankan `historic rebuild`.
- Query invalid menghasilkan error yang dapat dipahami dan tidak membuat sesi crash.
- Hasil kosong ditampilkan sebagai keadaan normal, bukan error fatal.
- Ukuran terminal yang terlalu kecil menghasilkan tampilan ringkas atau pesan actionable.
- Keluar dari TUI tidak meninggalkan file temporary, perubahan Markdown, atau perubahan status.
- Error tidak boleh mencetak output JSON ke dalam sesi TUI.
- Kegagalan query tidak boleh merusak atau mengganti index yang ada.

## 10. Non-functional requirements

- Target validasi awal: Linux, dengan desain yang tidak mengunci platform lain.
- TUI harus tetap responsif untuk query pada index yang memenuhi benchmark Phase 4.
- Pagination harus membatasi jumlah record yang dimuat dan dirender per halaman.
- Keyboard navigation harus dapat digunakan tanpa mouse.
- TUI harus menggunakan alternate screen bila library terminal yang dipilih mendukungnya.
- Terminal dikembalikan ke keadaan semula setelah exit normal maupun interrupt.
- Library TUI harus kompatibel dengan single binary Historic dan tidak memerlukan runtime web.

## 11. Definition of Done Phase 5 MVP

- `historic search` membuka TUI pada terminal interaktif.
- Query dapat diketik, dihapus, dan dijalankan ulang tanpa keluar dari sesi.
- Query kosong menampilkan topic aktif terbaru dengan limit default.
- Hasil memiliki limit dan pagination.
- Filter status, scope, dan type bekerja konsisten dengan `historic find`.
- Daftar hasil dapat dinavigasi dengan keyboard.
- Preview berubah mengikuti hasil yang dipilih.
- Search menggunakan service/index yang sama dengan `historic find`.
- Index hilang atau query invalid menghasilkan error actionable tanpa crash.
- TUI tidak mengubah Markdown, status, archive, atau asset.
- `historic find --json` tetap bekerja dengan envelope JSON yang sama.
- Unit test untuk state query/filter/pagination dan integration test CLI tersedia.
- Smoke test terisolasi membuktikan sesi search tidak mengubah workspace.

## 12. Out of scope

- Aksi mutasi seperti `complete`, `archive`, `status`, atau `sync-meta` dari shortcut TUI.
- Editor Markdown penuh.
- Dukungan mouse sebagai requirement MVP.
- Web UI, remote sync, multi-user, dan AI chat.
- Search index baru khusus TUI.
- Sinkronisasi real-time filesystem watcher.
- Konfigurasi layout yang kompleks.

## 13. Keputusan yang masih terbuka

1. Library TUI terminal yang digunakan untuk implementasi.
2. Perilaku final jika `--json` diberikan pada `historic search`.
3. Apakah filter type pada MVP cukup `topic` dan `work-order`, atau langsung mencakup `file` dan `asset`.
4. Apakah preview menampilkan Markdown terformat atau plain text terlebih dahulu.
5. Apakah `Enter` pada fase setelah MVP membuka editor eksternal.
6. Apakah query diperbarui setiap keystroke atau setelah user menekan Enter.

Keputusan terbuka ini tidak mengubah prinsip bahwa `historic find` tetap menjadi interface one-shot dan TUI menggunakan search service yang sama.
