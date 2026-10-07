# Migrasi Database

Sebelum Fase 0, skema dibuat dengan menjalankan `sql/migration.sql` secara
manual dan tidak ada catatan versi. Sekarang perubahan skema memakai berkas
berversi di `sql/migrations/` yang dijalankan runner Go.

## Struktur

```
sql/
  migration.sql          # baseline lama: membuat seluruh tabel dari nol
  poll_migration.sql     # baseline lama: tabel poll
  migrations/
    0001_users_source.up.sql
    0001_users_source.down.sql
    0002_otp_hardening.up.sql
    ...
```

Nama berkas: `<versi>_<nama>.<arah>.sql`. Versi diurutkan sebagai teks, jadi
pakai lebar tetap (`0001`, `0002`, …). Setiap migrasi **wajib** punya berkas
`up` dan `down`.

Setiap migrasi ditulis idempoten (`ADD COLUMN IF NOT EXISTS`,
`CREATE INDEX IF NOT EXISTS`, `DROP CONSTRAINT IF EXISTS`) supaya aman
dijalankan pada database yang sebagian perubahannya sudah ada.

## Perintah

```bash
go run ./cmd/migrate status      # daftar migrasi dan statusnya
go run ./cmd/migrate up          # terapkan semua migrasi tertunda
go run ./cmd/migrate up -n 1     # terapkan satu migrasi saja
go run ./cmd/migrate down -n 1   # batalkan satu migrasi terakhir (default 1)
go run ./cmd/migrate baseline    # catat migrasi sebagai sudah ada, tanpa menjalankan
```

Lewat Makefile:

```bash
make migrate-status
make migrate-up
make migrate-down
```

Runner membaca `.env` yang sama dengan server. Bila `.env` tidak ada,
variabel diambil dari environment.

## Cara kerja

- Riwayat disimpan di tabel `schema_migrations (version, name, applied_at)`.
- Setiap migrasi dijalankan dalam **satu transaksi** bersama pencatatannya.
  Postgres mendukung DDL transaksional, jadi migrasi yang gagal di tengah
  tidak meninggalkan skema setengah jadi.
- Berkas SQL dipecah menjadi pernyataan oleh `internal/migrate.SplitStatements`,
  yang sadar konteks: titik koma di dalam string, komentar, dan dollar-quote
  (`$$ … $$`) tidak dianggap pemisah.
- `down` selalu membalik dari migrasi terakhir.

## Database produksi yang sudah ada

Database lama sudah punya seluruh tabel dari `sql/migration.sql`, tetapi
belum punya baris di `schema_migrations`. Jalankan **satu kali**:

```bash
go run ./cmd/migrate baseline
```

Ini mencatat semua migrasi sebagai sudah diterapkan tanpa menjalankannya.
Setelah itu `up` hanya menjalankan migrasi baru.

Bila ingin benar-benar menerapkan migrasi lama pada database tersebut
(misalnya untuk menambahkan `participant_user_id` pada data yang sudah ada),
jalankan migrasi spesifik setelah baseline:

```bash
# contoh: hanya menjalankan satu migrasi yang belum tercatat
go run ./cmd/migrate down -n 1   # batalkan pencatatannya
go run ./cmd/migrate up -n 1     # lalu terapkan sungguhan
```

Migrasi ditulis idempoten, jadi menerapkannya pada skema yang sudah punya
kolom tersebut akan berakhir dengan `IF NOT EXISTS` dan tidak mengubah apa pun
selain backfill data.

## Backup dan rollback

**Selalu `pg_dump` sebelum `up` pada database berisi data produksi.**

```bash
pg_dump --format=custom --file=backup-$(date +%F-%H%M).dump "$DATABASE_URL"
```

Memulihkan:

```bash
pg_restore --clean --if-exists --dbname="$DATABASE_URL" backup-2026-10-02-0900.dump
```

Catatan:

- **Jangan** memakai `sql/drop_tables.sql` pada database aktif. Berkas itu
  hanya untuk membangun ulang lingkungan lokal dari nol.
- `down` membalik perubahan skema, **bukan** data. Migrasi `0004` melakukan
  soft delete pada baris `presence` duplikat; `down` tidak
  mengembalikannya. Pulihkan dari backup bila data itu dibutuhkan.
- Migrasi `0003` mengisi `user_tickets.participant_user_id` dari pencocokan
  email. `down` menghapus kolomnya beserta hasil backfill tersebut.
- Migrasi `0006` menambahkan `chk_orders_amounts` dengan `NOT VALID` supaya
  baris lama yang mungkin bernilai negatif tidak menggagalkan migrasi.
  Setelah data lama dibersihkan, validasi penuh bisa dijalankan manual:
  `ALTER TABLE orders VALIDATE CONSTRAINT chk_orders_amounts;`

## Isi migrasi

| Versi | Perubahan |
| --- | --- |
| `0001_users_source` | Kolom `users.source`. |
| `0002_otp_hardening` | `otp_tx.attempts`, `used_at`, `last_sent_at`; index `(email, created_at DESC)`. |
| `0003_user_tickets_participant` | Kolom `user_tickets.participant_user_id` + FK + backfill dari email. |
| `0004_presence_checkin_unique` | Kolom `presence.check_in_date` + backfill dari `created_at` (Asia/Jakarta) + unique index parsial `(user_ticket_id, check_in_date)`. |
| `0005_poll_responses_identity` | `poll_responses.user_id` jadi nullable; kolom `is_verified`. |
| `0006_constraints_indexes` | Unique `public_id`, check jumlah order, index untuk pencarian umum. |
| `0007_community_profiles` | Tabel `community_profiles` (alias, `public_id`, visibilitas, opt-out leaderboard, blokir). |
| `0008_gamification` | Tabel `xp_rules`, `xp_ledger` (append-only, `dedup_key` unik), `level_rules`. |
| `0009_missions` | Tabel `missions` dan `mission_claims` + unique parsial `uniq_mission_claims_aktif`. |
| `0010_fundraising` | Tabel `campaigns`, `donations`, `campaign_updates`, `audit_logs`; kolom `xp_rules.monthly_cap`; seed aturan XP donasi. |
| `0011_threads` | Tabel `threads`, `thread_comments`, `thread_reactions`, `thread_reports`, `thread_share_prefs` untuk feed komunitas anonim. |
| `0012_shop` | Tabel `products`, `product_images`, `product_variants`, `shop_orders`, `shop_order_items`, `stock_reservations`, `stock_movements` untuk marketplace merchandise; seed aturan XP pesanan. |

Catatan `0010`: `donations.amount`, `donations.paid_amount`, dan
`campaigns.target_amount` memakai `bigint`, bukan `integer` seperti `orders`.
Progres campaign adalah `SUM` atas banyak baris dan target kumulatif dapat
melampaui batas `integer`, sehingga penjumlahannya akan meluap di SQL sebelum
sempat dijumlahkan Go. Karena `SUM(bigint)` mengembalikan `numeric` di
Postgres, setiap pembacaan agregat di aplikasi harus di-cast `::bigint` agar
dapat dipindai ke `int` Go tanpa kehilangan presisi.

`xp_rules.monthly_cap` adalah batas **jumlah donasi ber-XP per bulan**, bukan
batas jumlah XP. Reward donasi besarnya tetap, sehingga batas dalam satuan XP
akan menghabiskan seluruh kuota pada donasi pertama.

Catatan `0011`: `uniq_threads_source (source_type, source_ref)` sengaja **tanpa**
predikat `deleted_at IS NULL`. Akibatnya pencabutan izin berbagi bersifat
permanen — mematikan lalu menyalakan kembali preferensi tidak menghidupkan
postingan lama — dan percobaan ulang penulisan sumber tidak pernah
menggandakan. `thread_reactions` juga sengaja tanpa `deleted_at`: batal reaksi
adalah hard delete, sehingga `uniq_thread_reactions_pengguna` cukup sebagai
penjaga keunikan. Tindakan moderator tidak memakai tabel catatan tersendiri;
jejaknya masuk ke `audit_logs` yang sudah ada sejak `0010`.

Prasyarat `0005`: tabel `poll_responses` harus sudah ada
(`sql/poll_migration.sql`).

Catatan `0012`: **invarian stok** yang menjadi dasar seluruh perhitungannya —
`product_variants.stock` adalah jumlah fisik di tangan, `stock_reservations`
adalah tahanan sementara untuk pesanan yang **belum dibayar**, konfirmasi
pembayaran menurunkan `stock` sekaligus melepas reservasinya, dan
pembatalan/refund mengembalikan `stock` lewat gerak `return`. Ketersediaan
adalah `stock - SUM(reservasi aktif & belum kedaluwarsa)`, sehingga reservasi
yang lewat `expires_at` berhenti dihitung tanpa perlu ditulis apa pun — itulah
sebabnya tidak ada scheduler. `uniq_stock_reservations_pesanan_varian` adalah
penjaga "konfirmasi berulang tidak memproses pesanan dua kali" sekaligus
"pembatalan mengembalikan stok tepat sekali". Pesanan merchandise sengaja
**tidak** memakai tabel `orders`: tabel itu terikat event dan menerbitkan
`user_tickets`. Batas XP `shop_order` terpisah dari batas `donation` — kuota
keduanya tidak saling mengurangi. Migrasi turun menghapus baris `xpr-shop` dari
`xp_rules` tetapi **tidak** menghapus `xp_ledger`, karena barisnya adalah
riwayat XP nyata milik anggota.

## Menambah migrasi baru

1. Buat `sql/migrations/0007_nama_perubahan.up.sql` dan `.down.sql`.
2. Tulis idempoten dan aman untuk data yang sudah ada (backfill dulu,
   batasan belakangan; pertimbangkan `NOT VALID` untuk check constraint).
3. `go run ./cmd/migrate status` untuk memastikan terbaca.
4. Uji pada database uji bila tersedia. **Jangan** menjalankan migrasi yang
   belum diuji langsung ke produksi.
5. Tambahkan barisnya ke tabel di atas.
