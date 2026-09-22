---
id: 00001
title: WO 02 Domain Model
status: complete
created: 2026-09-18
updated: 2026-09-18
---

# WO 02 — Domain Model

## Tujuan
Mendefinisikan model domain dan aturan validasi yang dipakai seluruh command.

## Tasks
- Definisikan Topic, Entry, WorkOrder, Frontmatter, dan IndexRecord.
- Definisikan status open/close serta parser status case-sensitive yang konsisten.
- Implementasikan format ID lima digit dan validasi ID manual.
- Implementasikan aturan slug title ke nama folder.
- Definisikan error domain untuk ID duplikat, status invalid, topic missing, dan conflict.

## Dependensi
WO 01.

## Output
Package domain tervalidasi dan transition matrix terdokumentasi.

## Review Validation
- `go test ./...` — berhasil
- `go vet ./...` — berhasil
- `go build -o /tmp/historic .` — berhasil
- Transition matrix saat ini dipertahankan untuk WO 02.
- Matrix lifecycle yang lebih ketat wajib dikunci sebelum WO 11.
- Commit lokal: `67889a9 feat: add historic domain model`
