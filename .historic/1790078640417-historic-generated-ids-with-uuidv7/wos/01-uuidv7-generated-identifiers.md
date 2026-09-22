---
id: "00007"
title: WO 01 UUIDv7 Generated Identifiers and Canonical Metadata
status: complete
created: "2026-09-22"
updated: "2026-09-22"
tags: [uuidv7, identifiers, metadata, migration, rebuild, implementation]
related: [../spec.md]
---
# WO 01 — UUIDv7 Generated Identifiers for Historic Files

## 1. Tujuan dan keputusan final

Implementasikan UUIDv7 **hanya untuk managed Historic files**. ID topic tidak berubah dan tetap memakai `domain.ID` numerik lima digit.

| Entitas | ID canonical | Generator | Disimpan di |
|---|---|---|---|
| Topic | `00001` | generator numerik existing | folder name + `_meta.yaml.id` |
| Managed file/WO | UUIDv7 lowercase | generator baru | `_meta.yaml.files[].id` |
| Asset | tidak wajib punya ID | tidak ada | `_meta.yaml.assets[]` |

User/AI hanya mengirimkan title/path. ID file tidak boleh ditulis pada template Markdown dan tidak boleh diterima sebagai input normal CLI.

## 2. Kontrak data wajib

### 2.1 Domain ID

Pertahankan `domain.ID` untuk topic. Tambahkan tipe terpisah agar ID file tidak tercampur dengan ID topic:

```go
type FileID string

func ParseFileID(value string) (FileID, error)
func (id FileID) String() string
func (id FileID) Valid() bool
```

`ParseFileID` hanya menerima UUID canonical lowercase dengan format 8-4-4-4-12, UUID version `7`, dan variant RFC 9562. Jangan memperluas `domain.ParseID`; parser topic harus tetap menerima lima digit saja.

### 2.2 Manifest

Ubah struktur manifest menjadi:

```go
type ManifestFile struct {
    ID     domain.FileID `yaml:"id"`
    Path   string        `yaml:"path"`
    Type   string        `yaml:"type"`
    Status domain.Status  `yaml:"status"`
}
```

`ValidateTopicMetadata` wajib memeriksa:

- setiap `files[].id` valid;
- ID file unik dalam satu topic;
- path `files` dan `assets` tidak overlap;
- path tetap logical POSIX relative;
- `assets` tidak diberi ID file;
- metadata topic tetap divalidasi dengan `domain.ID` numerik.

### 2.3 YAML canonical

Output canonical wajib mempertahankan ID file:

```yaml
id: "00007"
title: Historic Generated IDs with UUIDv7
created: "2026-09-22"
files:
    - id: 0192f3b5-1e20-7abc-8def-0123456789ab
      path: wos/01-uuidv7-generated-identifiers.md
      type: task
      status: pending
assets: []
```

`rebuild` tidak boleh menghasilkan ID baru jika pasangan logical identity `(topic_id, path)` sudah memiliki ID di metadata canonical.

## 3. Generator UUIDv7

Buat package/service kecil, misalnya `internal/identifier`, dengan API minimal:

```go
type Clock interface { Now() time.Time }
type Generator interface { New() (domain.FileID, error) }

func NewUUIDv7Generator(clock Clock, entropy io.Reader) Generator
```

Implementasi:

1. Ambil Unix timestamp millisecond dari `clock.Now().UTC()`.
2. Isi 48 bit pertama UUID dengan timestamp millisecond.
3. Set version nibble `0111` pada byte 6.
4. Set RFC variant `10` pada byte 8.
5. Isi bit acak dengan `crypto/rand.Reader` atau `entropy` injectable.
6. Serialize lowercase canonical dengan hyphen.
7. Validasi hasil sebelum return.

Persyaratan concurrency:

- generator aman dipanggil concurrent;
- gunakan mutex atau atomic state untuk monotonic millisecond/random state;
- jika clock mundur, jangan menghasilkan ID duplikat;
- collision harus retry maksimal 3 kali, lalu return error;
- jangan pernah overwrite file karena collision.

Dependency yang boleh digunakan: library UUID yang mendukung UUIDv7 dan RFC 9562. Jangan mengimplementasikan algoritma random sendiri bila library standard yang tervalidasi tersedia.

## 4. Perubahan repository dan command

### 4.1 `create`

Tidak mengubah perilaku ID topic:

```text
historic create "Topic Title"
```

Tetap membuat folder `<five-digit>-<slug>` dan `_meta.yaml.id` lima digit.

### 4.2 `add`

Alur normal:

```text
historic add "wos/work-order-title" --id 00007 --json
```

Langkah atomic:

1. Resolve topic dari ID numerik.
2. Validasi dan normalisasi logical path.
3. Pastikan destination belum ada, kecuali `--force` untuk isi file tanpa mengganti ID.
4. Generate `FileID` UUIDv7.
5. Buat Markdown tanpa field `id` pada frontmatter.
6. Tulis file ke temporary path lalu rename.
7. Baca `_meta.yaml` terbaru di bawah lock.
8. Append manifest entry `{id, path, type, status}`.
9. Tulis `_meta.yaml` temporary, fsync, lalu rename atomic.
10. Jika langkah 6–9 gagal, rollback file baru dan jangan mengubah metadata lama.

Output JSON:

```json
{
  "command": "add",
  "ok": true,
  "data": {
    "id": "0192f3b5-1e20-7abc-8def-0123456789ab",
    "topic_id": "00007",
    "file": ".historic/00007-topic/wos/work-order-title.md"
  },
  "error": null
}
```

Hapus atau nonaktifkan input ID member manual. Flag `--id` yang sudah ada hanya berarti target topic dan tetap numerik.

## 5. Rebuild dan klasifikasi

`rebuild` harus memakai manifest sebagai sumber identity managed file:

1. Parse `_meta.yaml` topic.
2. Index `files[].id` dan `files[].path`.
3. Walk filesystem.
4. Untuk path yang ada di `files`, parse Markdown dan update status/title/content tanpa mengganti ID.
5. Untuk Markdown valid yang belum ada di manifest, generate FileID baru, update manifest secara atomic, lalu index sebagai `historic_file`.
6. Untuk path `wos/` dengan Markdown valid, klasifikasikan sebagai managed file; tidak perlu `child.frontmatter.ID == topic.ID`.
7. Markdown tanpa frontmatter atau invalid tetap asset sesuai policy dan tidak mendapat FileID otomatis kecuali migration mode eksplisit.
8. File hilang dihapus dari manifest/index.
9. Manifest hasil rebuild diurutkan berdasarkan `path` dan tidak menduplikasi path/ID.

Jika rebuild membutuhkan perubahan `_meta.yaml`, gunakan temporary metadata plus rollback. Index SQLite lama tetap dipertahankan bila scan atau validasi gagal.

## 6. Migration legacy

Migration tidak mengubah ID topic numerik.

Untuk member lama:

- jika manifest sudah memiliki UUIDv7, preserve;
- jika manifest memiliki ID numerik/format lama, generate UUIDv7 sekali dan tulis mapping;
- mapping harus berdasarkan `(topic_id, logical_path)`;
- rerun migration harus menghasilkan output identik;
- tampilkan old ID, new UUIDv7, topic ID, dan path pada report;
- collision, duplicate path, atau missing source harus menghentikan commit atomic;
- sediakan dry-run sebelum menulis.

Jangan mengubah referensi topic `00001` menjadi UUID. Referensi file lama dipetakan melalui migration map.

## 7. Perubahan komponen

- `internal/domain`: tambahkan `FileID` dan validator UUIDv7.
- `internal/identifier` atau `internal/id`: generator, clock, entropy, mutex/monotonic state.
- `internal/markdown`: tambahkan `ManifestFile.ID`, YAML parse/write/validation, backward-compatible decode.
- `internal/repository`: generate ID saat `AddEntry`, update return type/output tanpa mengubah topic ID.
- `internal/lifecycle`: preserve file IDs pada open/close/delete; sync/rebuild metadata tanpa ID regeneration.
- `internal/indexer`: gunakan manifest path/ID sebagai identity dan hilangkan syarat child ID sama dengan topic ID.
- `internal/search`/`internal/query`: expose file ID dan topic ID terpisah jika result contract membutuhkannya.
- `internal/cmd`: bedakan `--id` target topic dari ID managed file; update JSON schema dan help.
- `docs`/skill: dokumentasikan format ID yang baru.

## 8. Acceptance criteria terukur

- `go test ./...` lulus.
- `go vet ./...` lulus.
- Build binary lulus.
- Test generator menghasilkan UUID version 7 dan variant RFC yang benar minimal 1.000 kali.
- Test concurrent minimal 100 goroutine × 100 ID tanpa duplicate.
- Test injected clock yang mundur tidak menghasilkan duplicate.
- Test add memastikan Markdown tidak memiliki `id` child manual dan `_meta.yaml.files[].id` terisi.
- Test rebuild kedua byte-equivalent untuk ID manifest dan tidak menambah ID.
- Test Work Order dengan path `wos/` dan ID berbeda dari topic menjadi `historic_file`.
- Test invalid/duplicate metadata menjaga index lama dan tidak meninggalkan partial metadata.
- Test open/close/delete mempertahankan `(topic_id, file_id, path)`, kecuali path memang diubah eksplisit.
- Test legacy migration idempotent dan tidak mengubah ID topic.
- `git diff --check` lulus.

## 9. Non-goals

- Tidak mengubah format ID topic pada WO ini.
- Tidak mengganti UUIDv7 dengan UUIDv4, slug, timestamp string, atau nomor urut file.
- Tidak mengubah `PRD.md`.
- Tidak melakukan remote push.

## 10. Urutan implementasi

1. Tambahkan `FileID` dan test validator.
2. Tambahkan generator UUIDv7 dan test concurrency/clock.
3. Tambahkan field ID pada manifest dan validator YAML.
4. Ubah repository `add` dan output JSON.
5. Ubah rebuild/indexer dan klasifikasi managed file.
6. Ubah lifecycle open/close/delete.
7. Tambahkan migration dry-run/idempotency.
8. Update search/query/docs.
9. Jalankan full test, vet, build, dan diff check.

## Status

Complete. Implemented and committed in `8f59db2` (`feat: add UUIDv7 managed file identifiers`). Legacy migration was completed by WO 02 in `1e89ed2`; command documentation, Historic skill updates, rebuild, doctor, and quality validation are complete.
