---
id: "00009"
title: Strict Historic Markdown Classification
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [historic, markdown, validation, classification, assets, managed-files]
related: [./wos/01-strict-historic-markdown-classification.md]
---
# Strict Historic Markdown Classification SPEC

## Tujuan

Memastikan file Markdown yang dimaksudkan sebagai file Historic managed tidak diam-diam masuk ke `assets[]` ketika frontmatter-nya salah. File Historic yang valid harus masuk ke `files[]`; file non-Historic tetap boleh menjadi asset.

## Aturan klasifikasi

1. Markdown dengan frontmatter valid (`title`, `status`, `created`, dan metadata opsional valid) diklasifikasikan sebagai managed Historic file.
2. Markdown tanpa frontmatter dan Markdown biasa tanpa metadata Historic diklasifikasikan sebagai asset.
3. Markdown yang memiliki bentuk frontmatter Historic tetapi salah nilai, field wajib hilang, atau status tidak dikenal harus ditolak sebagai error actionable.
4. File `wos/` dengan frontmatter valid tetap managed file walaupun tidak memiliki `id` child.
5. ID managed file hanya berasal dari `_meta.yaml.files[].id`; Markdown child tidak menulis ID.

## Metadata dan identity

- Topic ID tetap numerik lima digit.
- FileID managed disimpan di `_meta.yaml.files[].id`.
- Identity file adalah `topic_id + logical_path`.
- Rebuild mempertahankan FileID yang sudah ada dan hanya menghasilkan FileID untuk file managed baru.
- `path`, `type`, dan `status` manifest dipertahankan selama file masih ada.
- Asset tidak menerima FileID.

## Error dan atomicity

- Rebuild/sync harus menyertakan path file dan alasan validasi pada error.
- Metadata invalid tidak boleh ditimpa otomatis.
- SQLite lama harus tetap aman jika validasi atau rebuild gagal.
- Metadata yang berubah ditulis atomic.

## Acceptance criteria

- WO01 pada Topic 00006 masuk `files[]`, bukan `assets[]`.
- Status `planned` diterima sebagai status managed yang valid.
- File Historic dengan status invalid menghasilkan error, bukan asset.
- File Markdown biasa tanpa frontmatter tetap asset.
- Rebuild kedua idempotent dan mempertahankan FileID.
- `go test ./...`, `go vet ./...`, build, dan `git diff --check` lulus.

## Status

Complete. Implemented in commit `5e5c71e`.
