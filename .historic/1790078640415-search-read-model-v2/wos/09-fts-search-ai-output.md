---
id: 00009
title: WO 09 FTS Search and AI Output
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [search, fts5, filters, ai, json]
related: [../spec.md, ./07-unified-rebuild-read-model.md, ./08-aggregate-topic-file-queries.md]
---
# WO 09 — FTS Search and AI Output

## Tujuan
Meningkatkan `historic find` agar memahami title, description, tags, file content, storage, dan aggregate context.

## Scope
- FTS fields topic/file title, description, tags, path, filename, content.
- Weighted ranking.
- `--status`, `--tag`, `--type`, `--topic`, date filters, `--open`, `--closed`.
- JSON fields type, topic_id, status, storage, path, score, matched_in, snippet.
- Human output storage label.
- Reject old `--active`/`--archived` flags.

## Acceptance Criteria
- Query topic dan file dapat digabung.
- Structured filter tidak dipaksakan menjadi FTS syntax.
- Ranking title/description/tags lebih tinggi dari content.
- JSON membantu AI memahami alasan relevansi.
- Open/closed search tidak duplicate.
- Vector DB tidak diperlukan.

## Status
Complete. Implemented and validated in commit `957daae`.
