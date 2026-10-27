---
title: 'WO 05 — Tampilkan Relasi extends pada historic show'
description: "Mengekspos parent topic pada historic show human dan JSON melalui read model, tanpa mengubah output induk dengan backlink atau output list/find di MVP."
status: create
created: "2026-10-25"
updated: "2026-10-28"
tags:
    - show
    - cli
    - extension
related:
    - "1792889892100"
---
# WO 05 — Tampilkan Relasi `extends` pada `historic show`

## Konteks

Referensi SPEC: `../spec-topic-berlalu-dan-extension.md` §5.3, §6, §9, AC-5.
`historic show <child-id>` harus menampilkan parent extension bila tersedia. Induk tidak
berubah dan tidak ada backlink/children di MVP.

## Ruang lingkup

- Ekspos `extends` dari metadata/indexer ke hasil `show`.
- JSON `data[]` mendapat field `extends` (kosong/null jika tidak ada; pilih dan dokumentasikan
  satu representasi konsisten dengan envelope existing).
- Human output menampilkan parent ID hanya jika relasi ada.
- Tidak menambah extends ke `list` atau `find`; tidak menambah backlink `extends_children`.
- Topic legacy/no-extends mempertahankan perilaku/output kompatibel.

## Langkah implementasi

1. Periksa model/DTO `show` serta deserialisasi metadata dan index.
2. Tambahkan field ke response data tanpa mengubah field lama.
3. Render human output dengan format ringkas, misalnya `extends: <topic-id>`.
4. Tambahkan command tests untuk child, topic biasa, dual-copy, dan parent.

## Kriteria penerimaan

- **AC-1** `show <child-id> --json` memuat `data[].extends` tepat sama dengan parent ID.
- **AC-2** Human show menampilkan parent untuk child.
- **AC-3** Show topic tanpa parent tetap valid dan tidak menampilkan relasi palsu.
- **AC-4** Show parent tidak memuat/menulis backlink.
- **AC-5** Bentuk data existing tetap kompatibel.

## Test plan

| Kasus | Ekspektasi |
| --- | --- |
| Child dengan extends | ID parent muncul pada JSON dan human output |
| Topic tanpa extends | Null/kosong sesuai kontrak, tanpa error |
| Parent show | Tidak ada daftar children/backlink |
| Dual copy | Kedua storage menampilkan relasi metadata yang sama |

## Risiko dan dependensi

Risiko sedang pada kompatibilitas JSON/human output. **WO-03** dan **WO-04** diperlukan
agar field tersedia dari metadata/read model.
