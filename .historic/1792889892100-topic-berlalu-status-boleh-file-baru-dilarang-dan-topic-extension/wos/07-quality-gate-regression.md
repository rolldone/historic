---
title: WO 07 — Quality Gate dan Regression Suite Topic Extension
description: Mengintegrasikan test lintas paket untuk status/manifest, create --extends, forward, index migration, show JSON, serta menjalankan quality gate final.
status: complete
created: "2026-10-25"
updated: "2026-10-27"
tags:
    - testing
    - quality-gate
    - regression
related:
    - "1792889892100"
---
# WO 07 — Quality Gate dan Regression Suite Topic Extension

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §8, §10, §11.
WO ini menyatukan coverage lintas komponen dan memvalidasi acceptance criteria setelah
WO implementasi selesai. Tidak mengganti test unit yang menjadi tanggung jawab WO lain.

## Ruang lingkup

- Tambahkan/rapikan regression/integration tests lintas CLI/repository/lifecycle/indexer.
- Pastikan semua skenario tabel Test Matrix SPEC tercakup, khususnya:
  - status file open/closed dan path ambigu;
  - perilaku `add` pada topic closed tetap seperti sebelumnya;
  - create extension parent open/closed/absent/invalid;
  - parent immutability, chain extension, show child/parent;
  - legacy metadata, migration lama, rebuild idempotence;
  - `review` filter dan sinkronisasi manifest;
  - `forwarded`: forward ke child valid, target tidak terkait, `--create`, forward ulang,
    pembatalan forward, dan filter `--status forwarded`.
- Validasi error JSON envelope dan exit status.
- Jalankan `go test ./...`; jangan push/commit otomatis.

## Langkah implementasi

1. Audit test yang sudah ditambahkan di WO-01–06; hindari duplikasi tanpa alasan.
2. Buat integration fixture workspace dengan parent closed, parent open, child, dan
   dual-copy topic.
3. Assertions mencakup filesystem bytes sebelum/sesudah operasi yang ditolak.
4. Jalankan targeted tests lalu seluruh `go test ./...`.
5. Laporkan test yang gagal sebagai blocker; jangan mengubah kontrak SPEC diam-diam.

## Kriteria penerimaan

- **AC-1** Setiap acceptance criteria SPEC mempunyai setidaknya satu assertion test.
- **AC-2** Tidak ada mutasi parent pada create extension atau add yang ditolak.
- **AC-3** Migrasi/rebuild lama dan baru idempoten serta tanpa errors.
- **AC-4** Error CLI memiliki JSON envelope dan exit code yang diharapkan.
- **AC-5** `go test ./...` lulus.
- **AC-6** Tidak ada commit atau push otomatis.

## Test plan

| Grup | Skenario utama |
| --- | --- |
| Lifecycle | Status review pada open/closed, ambiguous paths, manifest sync |
| Repository | Add pada closed (perilaku lama) dan add setelah open |
| Create | Parent validation, extension metadata, ID allocator, immutability |
| Forward | Validasi strict, `--create` atomik, forward ulang, pembatalan |
| Index | Schema migration, rebuild, dry-run, idempotence, rollback |
| CLI/show | JSON envelope, human output, parent vs child, `forwarded_to` |
| Search | Filter `review` dan `forwarded`, status list |
| Full suite | `go test ./...` |

## Risiko dan dependensi

Risiko sedang karena coverage lintas paket dan fixture dual-copy. WO-01, WO-02, dan WO-03–06
serta WO-08–10 adalah input; selesaikan regression gaps tanpa memperluas scope yang belum
disetujui.

## Keputusan implementasi (final)

Keputusan 2026-10-28: `--extends` menerima parent **open maupun closed** (sesuai SPEC §5.3).
Pertanyaan terbuka SPEC §12 no.1 ditutup; tidak ada pembatasan closed-only.

## Hasil pelaksanaan (2026-10-28)

Regression suite lintas-paket ditambahkan tanpa menduplikasi unit test yang sudah ada:

| Grup | Test baru |
| --- | --- |
| Lifecycle (CLI end-to-end) | `TestRegressionStatusOnClosedTopicSyncsManifestAndReviewFilter` |
| Lifecycle/ambigu | `TestRegressionAmbiguousPathAcrossOpenAndClosedCopies` |
| Repository/perilaku lama | `TestRegressionAddOnClosedTopicMatchesLegacyBehavior` |
| Create/extension | `TestRegressionExtensionChainAndParentImmutability` |
| Forward | `TestRegressionForwardLifecycleAndFilters` |
| Index/rebuild | `TestRegressionRebuildIsIdempotentWithExtensions` |

Gap lain yang sudah tertutup oleh WO sebelumnya: migrasi schema lama
(`TestInitializeMigratesLegacyTopicsReadModelWithExtendsColumn`), rollback index
(`TestRebuildRecoveryRollbackPreservesMissingMetadataAndIndex`), dry-run non-destruktif
(`TestUpgradeWorkspaceDryRunIsNonDestructive`, `TestMigrateFileIDsDryRunPreservesBytes`),
dan immutability parent (`TestCreateTopicExtendsClosedParent`).

Dokumentasi "binary stale": ditambahkan ke `README.md`, `docs/commands.md`, dan
`.agents/skills/historic/SKILL.md` (schema index dimiliki binary terpasang; reinstall
dulu sebelum validasi schema).

Temuan di luar scope → dipindahkan ke **WO 11** (standarisasi tanggal `created`/`updated`
ke UTC).

Verifikasi: `gofmt` bersih, `go vet ./...` bersih, `go test ./...` lulus,
`historic rebuild --json` → `errors: 0`.
