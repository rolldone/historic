---
id: 00001
title: WO 01 Project Bootstrap
status: complete
created: 2026-09-18
updated: 2026-09-18
---

# WO 01 — Project Bootstrap

## Tujuan
Menyiapkan project Go dan binary `historic` yang dapat dibuild dan dites.

## Tasks
- Buat module Go dan entrypoint CLI.
- Tambahkan Cobra, Goldmark, yaml.v3, SQLite driver, dan adapter Git sesuai keputusan SPEC.
- Siapkan struktur package: `cmd`, `internal/domain`, `internal/markdown`, `internal/repository`, `internal/indexer`, `internal/lifecycle`, `internal/search`, `internal/gitproxy`, dan `internal/config`.
- Tambahkan command root, version, help, dan error handling dasar.
- Siapkan format build untuk Linux; jangan mengunci desain ke OS tertentu.

## Dependensi
Tidak ada.

## Output
Project skeleton, dependency manifest, build/test baseline.

## Review Validation
- Standard test command: `go test ./...`
- Result: berhasil
- Build validation: `go build -o /tmp/historic .` berhasil
- CLI validation: `/tmp/historic --help` dan `/tmp/historic version` berhasil
- Commit: `4ed7be8 chore: bootstrap historic cli`
- Scope check: `PRD.md` dan file di luar scope tidak diubah
