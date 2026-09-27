---
title: 'SPEC: close all open topics'
description: Add a bulk command to close every currently open topic with per-topic results.
status: complete
created: "2026-09-27"
updated: "2026-09-27"
---
# SPEC: `historic close all`

## Tujuan
Menyediakan satu perintah untuk memindahkan semua topic yang sedang open ke closed storage, dengan perilaku aman dan hasil tiap topic dapat diketahui.

## Perintah
```sh
historic close all [--json]
```

`all` adalah selector khusus untuk bulk operation. Perintah tetap menerima argumen ID tunggal seperti perilaku `historic close <id>` yang sudah ada.

## Perilaku
1. Ambil daftar topic yang open pada awal operasi dari work dir `.historic/<id>-<slug>/`; abaikan `.database`, staging folder, dan topic yang sudah closed.
2. Urutkan target secara deterministik berdasarkan TopicID lalu path.
3. Proses tiap target menggunakan lifecycle close individual yang sudah ada; jangan mengimplementasikan ulang mekanisme archive/snapshot.
4. Per topic, pertahankan perilaku close eksisting: snapshot di `.historic/.database/`, hapus work dir hanya jika close dan index update sukses, serta rollback bila operasi gagal.
5. Kegagalan satu topic tidak menghentikan pemrosesan target lain.
6. Hasil akhir merangkum topic berhasil dan gagal beserta ID/path dan alasan gagal. Bila ada kegagalan, command mengembalikan exit code non-zero; bila semua berhasil atau tidak ada target, exit code zero.
7. Topic yang dibuat/dibuka setelah daftar awal diambil tidak wajib ikut diproses pada pemanggilan tersebut.
8. Command tidak mengubah status member/frontmatter dan tidak menghapus topic secara permanen.

## Output
Human output menampilkan hasil per topic dan total berhasil/gagal. `--json` memakai envelope standar Historic `{command,ok,data,error}`; `ok` true jika seluruh target berhasil (termasuk daftar kosong), false jika satu atau lebih gagal. `data` memuat daftar hasil per topic dan jumlah sukses/gagal; error merangkum kegagalan tanpa menduplikasi error ke stderr.

Contoh struktur data JSON:
```json
{
  "command": "close",
  "ok": false,
  "data": {
    "succeeded": [{"id": "...", "path": ".historic/.database/..."}],
    "failed": [{"id": "...", "path": ".historic/...", "error": "..."}],
    "total": 2,
    "closed": 1,
    "failed_count": 1
  },
  "error": "1 topic gagal ditutup"
}
```

## Kriteria penerimaan
- `historic close all` menutup semua topic yang open saat daftar target diambil.
- Hanya topic valid tingkat pertama yang diproses; closed/staging/database folder tidak ikut.
- Tiap close memakai lifecycle existing dan tidak merusak snapshot/work dir jika gagal.
- Satu kegagalan tidak mencegah topic lain diproses.
- Ringkasan human dan JSON menunjukkan hasil individual serta total akurat.
- Jika ada kegagalan, process exit non-zero; jika kosong/semua sukses, exit zero.
- `historic close <id>` tetap kompatibel dan tidak berubah perilaku.
- Tes mencakup nol target, satu/beberapa target, sebagian gagal, output JSON, determinisme, dan regresi perintah tunggal.

## Di luar ruang lingkup
- Menutup topic tertentu melalui selector/filter selain `all`.
- Penghapusan permanen, commit/push remote, atau perubahan status work member.
- Paralelisasi operasi close.
