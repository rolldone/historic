---
id: 00001
title: WO 03 Markdown Frontmatter
status: complete
created: 2026-09-18
updated: 2026-09-18
---

# WO 03 — Markdown and Frontmatter

## Tujuan
Menyediakan parser dan writer Markdown dengan frontmatter yang sesuai PRD.

## Tasks
- Parse YAML frontmatter dan body Markdown.
- Validasi field wajib `id`, `title`, `status`, `created`.
- Parse field opsional `updated`, `tags`, dan `related`.
- Generate `_meta.md` dengan section deskripsi, files, dan progress.
- Generate file Markdown baru dengan template frontmatter.
- Tulis file secara atomic dan menjaga newline/encoding UTF-8.
- Kembalikan error yang menyebut path dan field bermasalah.

## Dependensi
WO 01, WO 02.

## Acceptance Criteria
- Round-trip parse/write tidak menghilangkan metadata atau body.
- Frontmatter invalid ditolak tanpa overwrite file sumber.
- File tanpa frontmatter wajib menghasilkan error terstruktur.
- Test mencakup tags/related kosong dan format tanggal PRD.

## Output
Parser, validator, template writer, dan test fixtures.
