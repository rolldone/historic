---
title: 01 Extract Mirror Operation
description: Pisahkan mekanisme penyalinan konten topic open ke .database dari langkah penutupan workdir open.
status: complete
created: "2026-10-13"
updated: "2026-10-13"
tags:
    - lifecycle
    - refactor
related:
    - ../spec-save-mirrors-open-workdir.md
---
# WO 01 — Extract Mirror Operation

## Tujuan

Menyediakan satu operasi internal `MirrorTopic(workspace, id)` pada package `lifecycle`
yang menyalin konten topic open ke `.database/<id>-<slug>/` **tanpa** menghapus workdir
open. Operasi ini menjadi fondasi bagi `save` (mirror saja) dan `close` (mirror + tutup).

## Scope

- Ekstrak tahap `stageDifferentialArchive` + `finalizeClosedTopic` di `internal/lifecycle/storage.go`
  menjadi komponen yang bisa dipanggil tanpa tahap pemindahan workdir open.
- Tambahkan opsi internal pada jalur finalisasi untuk melewatkan `os.Rename(source, sourceBackup)`
  dan `os.RemoveAll(sourceBackup)`.
- `close` memakai mirror lalu melanjutkan ke penutupan seperti sekarang.

## Aturan wajib

- Workdir open **tidak boleh** dihapus, dipindah, atau diubah oleh operasi mirror.
- Manifest/hash tetap dipakai untuk mendeteksi kasus snapshot identik (idempotent).
- Rebuild index tetap dijalankan setelah mirror agar read model konsisten.
- Jangan mengubah tanda tangan publik `Close`, `CloseAll`, atau `Open`.
- Operasi mirror tidak boleh menulis ke repo Git internal (tanpa commit).
- Resolver `TopicLocation` tetap memilih salinan open sebagai authoritative untuk operasi tulis.

## Acceptance criteria

- Operasi mirror dapat menghasilkan `.database/<id>-<slug>/` yang identik dengan workdir open.
- `MirrorTopic` idempotent: dipanggil dua kali tidak menghasilkan perubahan isi.
- `close` tetap lolos tes lifecycle yang ada setelah refactor.
