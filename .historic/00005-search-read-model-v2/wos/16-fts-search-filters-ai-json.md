---
id: 00016
title: WO 16 FTS Search Filters and AI JSON
status: planned
created: 2026-09-21
updated: 2026-09-21
tags: [fts5, search, filters, ai, json]
related: [../spec.md]
---
# WO 16 — FTS Search, Filters, and AI JSON

## Tujuan
Menyediakan pencarian weighted FTS5 dan structured filters dengan output yang kaya untuk AI.

## Scope
- FTS title/description/tags/filename/path/content.
- Weighted ranking.
- status/tag/type/topic/folder/date/open/closed filters.
- score, matched_in, snippet, topic context, storage.
- Human open/closed labels.
- Remove old active/archived flags.
- No Vector DB.

## Acceptance Criteria
- Topic/file combined search.
- Structured filters tidak menjadi FTS syntax.
- Title/description/tags lebih berbobot.
- JSON AI-friendly stable.
- Open/closed tidak duplicate.
- Query invalid actionable.

## Status
Complete. Implemented and validated in commit `c6ef492`.
