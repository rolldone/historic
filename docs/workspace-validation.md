# Perilaku Validasi Workspace

Command yang membaca atau mengubah workspace menjalankan preflight readiness secara read-only sebelum membuka read model atau menjalankan lifecycle.

## State dan recovery

- `not_historic_workspace`: `.historic/` tidak ditemukan. Jalankan `historic init`.
- `legacy_conflict`: `.historic/` dan `.histories/` ditemukan bersama. Pisahkan atau migrasikan `.histories` secara manual; Historic tidak menggabungkan direktori otomatis.
- `invalid_structure`: struktur root, `.historic`, `.database`, atau repository internal tidak valid. Perbaiki struktur workspace secara manual.
- `missing_index`: `.historic/.index.sqlite` tidak ditemukan. Jalankan `historic rebuild`.
- `invalid_index`: index SQLite rusak atau metadata/schema tidak kompatibel. Jalankan `historic rebuild`.
- `ready`: workspace lolos seluruh pemeriksaan dan command dilanjutkan.

Validasi tidak membuat direktori/file, tidak memperbarui metadata, tidak memindahkan `.histories/`, dan tidak menjalankan rebuild otomatis. `doctor` menggunakan jalur inspeksi recovery, sedangkan `rebuild` memiliki preflight sendiri sehingga tetap bisa memulihkan index yang hilang atau rusak.

## Output JSON

Error JSON menggunakan envelope standar:

```json
{"command":"find","ok":false,"data":null,"error":{"code":"missing_index","message":"Jalankan historic rebuild terlebih dahulu."}}
```

Error ditulis sekali ke output command dan tidak diduplikasi ke stderr.
