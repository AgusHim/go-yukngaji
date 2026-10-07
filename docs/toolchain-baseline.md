# Baseline Toolchain dan Regresi

Dicatat pada 2 Oktober 2026 sebagai titik awal Fase 0.

## Versi

| Komponen | Versi |
| --- | --- |
| Go (mesin pengembang) | 1.25.0 darwin/arm64 |
| Go (Dockerfile) | `golang:1.24-alpine` |
| `go.mod` | `go 1.22`, `toolchain go1.22.8` |
| Node.js | 24.6.0 |
| npm | 11.5.1 |
| Next.js | 15.5.7 |
| React | 19.2.3 |
| Tailwind CSS | 4.3.3 |
| TypeScript | 5.x |
| PostgreSQL | dijalankan lewat `postgres:alpine` (lihat Makefile) |

`go.mod` menyebut `go 1.22` sementara mesin pengembang memakai 1.25 dan
Docker memakai 1.24. Selama ini tidak menimbulkan masalah karena kode tidak
memakai fitur bahasa di atas 1.22. Bila nanti diseragamkan, ubah `go.mod` dan
`Dockerfile` bersamaan.

## Perintah

Backend (dari `server/`):

```bash
go build ./...
go vet ./...
go test ./...
```

Frontend (dari `web/`):

```bash
npx tsc --noEmit
npm run build
```

Migrasi:

```bash
go run ./cmd/migrate status
```

## Status suite test

Sebelum Fase 0 **tidak ada satu pun berkas `*_test.go`** di repositori ini,
dan `go.mod` tidak memuat `testify` — jadi test ditulis dengan `testing`
standar saja.

Fase 0 menambahkan test untuk logika murni yang bisa diuji tanpa Postgres:

| Paket | Yang diuji |
| --- | --- |
| `internal/authz` | `IsMember` (satu tier `user`/`jamaah`), `Can`, `IsRanger`, `IsStaff`, `HasRole`. |
| `internal/order` | `ComputeOrderAmounts`, `DeriveStatus`, `IsValidStatus`, `CanTransition`, `ValidateTicketEvents`, `ValidateTicketWindow`, `NormalizeParticipantEmails`, `MatchParticipants`. |
| `internal/otp` | `NormalizeEmail`, `ValidateOTP` (sekali pakai, batas percobaan, kedaluwarsa), `CanResend`. |
| `internal/presence` | `CheckInDate` (batas hari Asia/Jakarta, kestabilan tanggal untuk idempotensi). |
| `internal/ratelimit` | Jendela bergulir, isolasi antar kunci, `Reset`. |
| `internal/migrate` | `SplitStatements` (string, komentar, dollar-quote). |
| `internal/origins` | Parsing `CORS_ALLOWED_ORIGINS`, penolakan origin asing. |

Yang **belum** tercakup dan menjadi utang teknis: repository, service, dan
handler yang menyentuh database. Menutupnya memerlukan database uji
(lihat "Database uji" di bawah).

## Database uji

Saat Fase 0 dikerjakan, Docker daemon tidak berjalan dan `psql`/`pg_dump`
tidak terpasang di mesin pengembang. Akibatnya:

- migrasi **tidak** pernah dijalankan terhadap database sungguhan;
- verifikasi migrasi hanya statis (pembacaan berkas + uji pemecah pernyataan).

Untuk menutup celah ini, siapkan database uji:

```bash
make postgresinit          # menjalankan postgres:alpine di port 5433
docker exec -it mainyuk-db createdb --username=root --owner=root mainyuk_test
```

lalu arahkan `DB_*` ke database tersebut dan jalankan
`go run ./cmd/migrate up`, disusul `down -n 1` untuk memastikan keduanya
sama-sama jalan.

## Regresi manual yang harus dijalankan sebelum rilis

Tidak ada yang bisa dijalankan otomatis saat Fase 0 karena tidak ada
lingkungan. Jalankan daftar ini pada lingkungan uji:

1. Masuk dengan Google, OTP, dan kata sandi.
2. Buka profil, ubah data, lalu keluar — pastikan token ikut terhapus.
3. Checkout tiket gratis, berbayar, dan rombongan (beberapa peserta).
4. Pindai tiket pada event multi-hari, dua kali di hari yang sama.
5. Tulis komentar, sukai komentar, kirim poll, kirim masukan.
6. Buka agenda dan dashboard ranger.
7. Buka WebSocket komentar pada satu event.

### Fase 1–2 — profil, XP, misi

8. Lengkapi profil sampai penuh; pastikan XP penyelesaian profil hanya
   diberikan sekali, bukan setiap kali disimpan.
9. Selesaikan satu misi, lalu setujui klaimnya dari dashboard. Periksa XP
   bertambah tepat sebesar reward, dan batas bulanan (`xp_rules`) dipatuhi.
10. Ajukan penyesuaian XP dari `/admin_api/xp/adjustments`, lalu pastikan
    barisnya muncul di `GET /admin_api/xp/audit_logs` (baru) — inilah satu-
    satunya jejak koreksi manual selain `xp_ledger`.
11. Buka leaderboard untuk periode mingguan, bulanan, dan sepanjang waktu.

### Fase 3 — donasi dan komunitas

12. Buat donasi, konfirmasi dari dashboard, dan pastikan XP donasi masuk
    sekali saja. Ulangi permintaan konfirmasi yang sama — harus idempoten,
    bukan menggandakan reward.
13. Tolak dan refund donasi; pastikan reward XP ikut dibalik tepat sekali.
14. Tulis thread dan komentar, laporkan salah satunya, lalu selesaikan
    laporan dari dashboard moderasi. Periksa jejaknya di
    `GET /admin_api/moderation/audit_logs`.
15. Batasi (`restrict`) satu akun, lalu buka kembali.

### Fase 4 — toko merchandise

Empat kriteria selesai Fase 4 diuji langsung di sini:

16. **Tidak ada overselling.** Siapkan satu varian berstok 1, lalu lakukan
    dua checkout bersamaan dari dua akun. Satu berhasil, satu ditolak —
    stok tidak pernah menjadi negatif.
17. **Konfirmasi berulang tidak memproses dua kali.** Kirim permintaan
    konfirmasi pembayaran yang sama dua kali; XP reward hanya bertambah
    sekali dan `shop_orders.payment_status` berpindah sekali.
18. **Pembatalan mengembalikan stok tepat sekali.** Batalkan pesanan yang
    sudah dikonfirmasi, lalu ulangi permintaannya. Stok kembali tepat satu
    kali, dan `stock_movements` hanya memuat satu baris `return`.
19. **Pesanan toko tidak menyentuh tiket event.** Setelah checkout
    merchandise, pastikan tidak ada baris baru di `orders`/`user_tickets`
    dan tidak ada tiket yang terbit.
20. Uji kedaluwarsa: buat pesanan, lewati `expires_at`, lalu buka halaman
    pesanannya. Reservasi dilepas saat diakses (tidak ada pekerja latar),
    dan stok kembali.
21. Uji alur kirim: isi ongkir dan nomor resi, lalu tandai selesai.
22. Pastikan setiap perubahan stok manual dan setiap keputusan pesanan
    muncul di `GET /admin_api/shop/audit_logs`.

### Lintas modul

23. Buka `/dashboard/metrics` dan `/dashboard` — angkanya harus berasal
    dari data nyata, bukan angka tetap.
24. Buka `/dashboard/users` sebagai admin; daftarnya berhalaman dan berisi
    pengguna sebenarnya.
25. Buka rute dashboard yang bukan hak peran tersebut (mis. masuk sebagai
    anggota lalu buka `/dashboard/orders`); harus muncul panel "Tidak
    berizin", bukan isi halaman.
