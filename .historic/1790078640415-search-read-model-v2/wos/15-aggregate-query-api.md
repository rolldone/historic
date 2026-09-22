---
id: 00015
title: WO 15 Aggregate Topic File Query API
status: complete
created: 2026-09-21
updated: 2026-09-21
tags: [query, aggregate, sqlite, ai]
related: [../spec.md]
---
# WO 15 — Aggregate Topic/File Query API

## Tujuan
Menyediakan query SQL lintas `topics` dan `files` untuk context dan summary AI.

## Scope
- JOIN topics/files.
- COUNT/SUM/MAX/GROUP BY/HAVING.
- total/active/resolved/complete/cancelled.
- computed status.
- Combined filters status/tags/storage/type/topic/folder/dates.
- Topic context pada file result.

## Acceptance Criteria
- Complete/cancelled resolved.
- Asset excluded from work aggregate.
- Empty topic safe.
- Combined filters deterministic.
- JSON aggregate stable.

## Status
Complete. Implemented and validated in commit `26a1d81`.
