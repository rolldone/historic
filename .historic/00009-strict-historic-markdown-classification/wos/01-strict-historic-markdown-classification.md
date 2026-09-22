---
title: Strict Historic Markdown Classification and StatusPlanned
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [classification, validation, status, planned, rebuild, sync-meta]
related: [../spec.md]
---
# WO 01 — Strict Historic Markdown Classification and StatusPlanned

## Tujuan

Mencegah file Markdown berbentuk Historic yang invalid diam-diam masuk ke `assets[]`, dan menambahkan `planned` sebagai status managed yang valid.

## Perubahan yang diterapkan

### 1. `internal/domain/model.go`

Tambahkan `StatusPlanned` ke daftar `validStatuses` dan `openStatuses`:

```go
StatusPlanned Status = "planned"
```

### 2. `internal/markdown/markdown.go`

Tambahkan deteksi frontmatter bergaya Historic:

```go
func LooksLikeHistoricFile(input []byte) bool
```

Fungsi ini memeriksa apakah frontmatter mengandung setidaknya satu dari field `title:`, `status:`, atau `created:`. Tidak memvalidasi nilai; hanya mendeteksi bentuk.

### 3. `internal/lifecycle/sync_meta.go`

Ubah `scanTopicFiles` agar:
- Membaca file secara manual untuk deteksi frontmatter.
- Jika `markdown.Parse` gagal dan `LooksLikeHistoricFile` true → return error dengan path dan alasan.
- Jika file bukan bentuk Historic → tetap asset.

### 4. `internal/indexer/indexer.go`

Ubah `scanTopic` agar:
- Membaca file secara manual untuk deteksi frontmatter.
- Jika `markdown.Parse` gagal dan `LooksLikeHistoricFile` true → return error.
- Jika file bukan bentuk Historic → tetap asset.

### 5. `internal/tui/search.go`

Tambahkan `StatusPlanned` ke cycle filter status di TUI.

### 6. `internal/domain/model_test.go`

Perbarui test validasi status dan transition matrix untuk menyertakan `StatusPlanned`.

## Acceptance criteria

- WO01 pada Topic 00006 masuk `files[]`, bukan `assets[]`.
- Status `planned` diterima sebagai status managed yang valid.
- File dengan frontmatter invalid ditolak, bukan diam-diam menjadi asset.
- File Markdown biasa tanpa frontmatter tetap asset.
- `go test ./...`, `go vet ./...`, build, dan `git diff --check` lulus.

## Status

Complete. Implemented in commit `5e5c71e`.
