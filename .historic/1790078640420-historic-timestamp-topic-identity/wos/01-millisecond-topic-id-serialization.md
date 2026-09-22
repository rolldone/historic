---
id: "00010"
title: WO 01 Millisecond Topic ID Serialization and Migration
status: progress
created: "2026-09-22"
updated: "2026-09-22"
tags: [topic-id, timestamp, millisecond, lock, serialization, migration]
related: [../spec.md]
---
# WO 01 — Millisecond Topic ID Serialization and Migration

## 1. Scope

Implementasikan TopicID timestamp millisecond untuk topic baru tanpa mengubah domain `FileID` UUIDv7. Pembuatan topic harus serialized lintas process dan migration legacy harus aman.

## 2. Type contract

Tambahkan type terpisah:

```go
type TopicID string

func ParseTopicID(value string) (TopicID, error)
func (id TopicID) String() string
func (id TopicID) Valid() bool
```

Validator:

- hanya digit ASCII;
- integer positif;
- 13 digit untuk epoch millisecond modern;
- tidak menerima slug atau UUID;
- overflow ditolak;
- TopicID tidak boleh dipassing ke `ParseFileID`.

Pertahankan parser legacy lima digit selama migration compatibility:

```go
type LegacyTopicID string
```

## 3. Allocator API

Buat service repository/identifier yang injectable:

```go
type MillisecondClock interface {
    Now() time.Time
}

type TopicIDAllocator interface {
    Allocate(ctx context.Context) (TopicID, error)
}

func NewTopicIDAllocator(
    root string,
    clock MillisecondClock,
    lock Lock,
) TopicIDAllocator
```

`Allocate` wajib:

1. acquire inter-process lock;
2. scan open dan closed topic roots;
3. parse semua TopicID modern;
4. read `last_id` jika allocator state tersedia;
5. calculate `candidate = max(now_ms, max_existing + 1)`;
6. retry while candidate folder exists;
7. persist allocator state only after topic creation transaction succeeds;
8. release lock with defer.

Lock API:

```go
type Lock interface {
    Acquire(ctx context.Context) error
    Release() error
}
```

Gunakan lock file dengan exclusive create (`O_CREATE|O_EXCL`) atau mekanisme OS equivalent. Lock stale recovery harus memiliki PID, timestamp, dan policy yang tidak menghapus lock process hidup.

## 4. Create transaction

Ubah `repository.CreateTopic`:

1. Validate title/slug.
2. Allocate TopicID.
3. Build `<topic-id>-<slug>` path.
4. Create directory using exclusive operation.
5. Write `_meta.yaml` with TopicID atomically.
6. Commit result only after metadata fsync/rename succeeds.
7. On failure remove only the newly created directory.
8. Return TopicID in human and JSON output.

`--id` manual harus dihapus dari normal create atau dipindahkan ke explicit migration/admin command.

## 5. Parsing and storage changes

Update seluruh resolver agar mendukung TopicID modern:

- folder regex;
- `config.Workspace` discovery;
- `activeTopicID` selection;
- list/show/find filters;
- open/close/import/delete/purge/restore;
- related references;
- `_meta.yaml` validation;
- SQLite `topics.id`, foreign keys, and indexes;
- FTS topic identity output.

Do not reuse `domain.ID` five-digit parser for modern TopicID. Keep compatibility adapter for legacy folders during migration window.

## 6. Migration command

Tambahkan:

```text
historic migrate-topic-ids --dry-run
historic migrate-topic-ids --json
historic migrate-topic-ids
historic migrate-topic-ids --id 00001
```

Migration phases:

### Scan

- enumerate open/closed topics;
- identify legacy five-digit folders;
- validate metadata and duplicate topic identity;
- read existing alias/mapping state.

### Plan

- allocate timestamp IDs serialized under lock;
- deterministic order by legacy ID then path;
- report old ID, new TopicID, old path, new path, and action;
- dry-run writes nothing.

### Commit

- lock workspace;
- re-scan after lock;
- stage directory renames and metadata updates;
- write alias map atomically;
- update references;
- replace SQLite only after filesystem transaction succeeds;
- rollback all staged renames on failure.

Migration must not overwrite an existing topic folder or lose Markdown bytes.

## 7. Compatibility and rollback

- bump workspace format version;
- old binary rejects modern TopicID workspace;
- migration backup includes folder map, `_meta.yaml`, and alias map;
- interrupted migration can resume or rollback deterministically;
- rebuild after migration is idempotent;
- legacy aliases resolve until explicit alias purge.

## 8. Test matrix

- Parse valid/invalid/overflow TopicID.
- Same millisecond with 2, 100, and 1,000 concurrent creates.
- Multiple processes contending for lock.
- Clock rollback and future clock.
- Existing folder collision and gap allocation.
- Lock stale/live process behavior.
- Crash after directory creation and before metadata rename.
- Create rollback on metadata fsync failure.
- Legacy migration dry-run byte-preservation.
- Migration rerun idempotency.
- Open/closed duplicate storage mapping.
- Resolver compatibility for legacy and modern IDs.
- SQLite rebuild and FTS identity.

## 9. Acceptance criteria

- No duplicate TopicID under concurrent create.
- Allocated IDs are monotonic per workspace.
- No ID reuse after crash or deletion.
- Topic creation is atomic and rollback-safe.
- Legacy five-digit topics remain readable during migration.
- Migration preserves all Markdown and manual metadata.
- Rebuild does not regenerate TopicID or FileID.
- Version compatibility prevents old binary corruption.
- Full quality gate passes:

```text
go test ./...
go vet ./...
go build .
git diff --check
```

## 10. Non-goals

- Tidak mengubah FileID managed file UUIDv7.
- Tidak menghapus Markdown source.
- Tidak mengubah `PRD.md`.
- Tidak melakukan remote push.

## Status

Planned. Menunggu implementasi Dev Agent.
