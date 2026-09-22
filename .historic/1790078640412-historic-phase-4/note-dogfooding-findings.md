---
id: "00002"
title: note dogfooding findings
status: progress
created: "2026-09-19"
---
# Dogfooding Findings

## Observations

- `historic add <name>` nyaman ketika hanya satu topic aktif karena `--id` tidak diperlukan.
- Ketika dua topic aktif, command tanpa `--id` ditolak dengan error yang jelas: `specify --id when active topic count is 2`.
- Output JSON konsisten memakai envelope `command`, `ok`, `data`, dan `error`.
- `historic complete` memindahkan topic ke `.historic/.database/`; `historic import` mengembalikan salinan aktif dan mempertahankan archive.
- `historic save` pada topic yang sudah di-import tidak mendeteksi perubahan di working directory aktif karena internal Git berakar di `.historic/.database/`; perubahan aktif baru masuk snapshot setelah topic diarsipkan.
- `updated` pada lifecycle terisi, tetapi tanggal yang terlihat satu hari lebih awal dari tanggal environment saat pengujian.
- Tidak ditemukan blocker kritis atau kebutuhan command baru untuk workflow ini.

## Follow-up

- Jadikan ketepatan timezone/tanggal `updated` sebagai input review Phase 4.
- Dokumentasikan atau tinjau scope `save`, `log`, dan `diff` terhadap topic aktif hasil import.
- Pertahankan rejection untuk add tanpa `--id` saat ada beberapa topic aktif.
