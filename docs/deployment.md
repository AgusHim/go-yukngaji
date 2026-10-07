# Deployment

Catatan lingkungan produksi dan hal yang harus diperhatikan saat merilis.

Dokumen ini mulai sebagai catatan **temuan** pada Fase 0, ketika konfigurasi
server belum disentuh. Temuan itu kini sudah ditutup pada pekerjaan lintas
modul: `.env` tidak lagi dibakar ke image, kredensial database tidak lagi
tertulis di MakeFile, nama variabel lingkungan sudah diselaraskan dengan
kode, dan panduan rollback sudah ada. Bagian yang masih berupa temuan
ditandai apa adanya.

## Topologi saat ini

| Domain | Diteruskan ke | Isi |
| --- | --- | --- |
| `be.ynsolo.id` | `localhost:8000` | API Go (termasuk `/ws/`) |
| `go.ynsolo.id` | `localhost:3030` | Aplikasi web Next.js |
| `ppob.bantukamujadiawardee.com` | `localhost:8080` | Layanan lain, tidak terkait |
| `jualquota.com` | — | Dialihkan ke HTTPS |

Backend dijalankan sebagai satu container (`gushim/yukngaji`), satu proses.
Frontend dijalankan terpisah.

## Variabel lingkungan

Lihat `.env.example`. Yang wajib — daftar ini sama persis dengan yang dibaca
`os.Getenv` di kode, tidak lebih dan tidak kurang:

`HOST`, `GIN_MODE`, `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`,
`DB_DATABASE`, `JWT_SECRET_KEY`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`,
`GOOGLE_REDIRECT_URL`, `SMTP_LOGIN`, `SMTP_PASSWORD`.

Dua nama pernah tertulis salah di dokumen ini dan diwarisi oleh konfigurasi
lama; keduanya **tidak** dibaca kode sama sekali:

| Tertulis | Yang benar | Akibat bila salah |
| --- | --- | --- |
| `JWT_SECRET` | `JWT_SECRET_KEY` | Tidak ada kunci penanda tangan; token gagal dibuat/diverifikasi. |
| `SMTP_*` (mis. `SMTP_USER`, `SMTP_HOST`) | `SMTP_LOGIN` dan `SMTP_PASSWORD` | OTP tidak terkirim. Hanya dua variabel ini yang dibaca. |

Yang baru dan perlu diset saat rilis Fase 0:

| Variabel | Fungsi | Bila tidak diset |
| --- | --- | --- |
| `CORS_ALLOWED_ORIGINS` | Daftar origin yang boleh mengakses API, dipisah koma. | Jatuh ke `http://localhost:3000`, `https://ynsolo.id`, `https://www.ynsolo.id`. |
| `COOKIE_SECURE` | `true` untuk mengirim cookie `oauthstate` hanya lewat HTTPS. | Cookie tidak memakai flag `Secure`. Set `true` di produksi. |

`CORS_ALLOWED_ORIGINS` **harus** memuat origin aplikasi web yang sebenarnya
(mis. `https://go.ynsolo.id`) dan origin WebSocket-nya, kalau berbeda.

## WebSocket

- **Satu proses, satu hub.** `ws.NewHub()` menyimpan koneksi di memori
  proses. Menjalankan lebih dari satu instance backend akan memecah siaran:
  komentar yang masuk ke instance A tidak sampai ke klien yang terhubung ke
  instance B. Selama hanya ada satu container, ini tidak masalah. Untuk
  menambah instance, hub harus diganti dengan pub/sub (Redis) lebih dulu.
- **Origin diperiksa.** Sebelum Fase 0, `CheckOrigin` selalu mengembalikan
  `true` sehingga situs mana pun bisa membuka koneksi atas nama pengunjung.
  Sekarang origin dicek terhadap `CORS_ALLOWED_ORIGINS`; permintaan tanpa
  header `Origin` (klien non-browser) tetap diterima.
- **nginx mengirim `Origin` kosong** pada blok `go.ynsolo.id`
  (`proxy_set_header Origin ""`). Dengan aturan baru, koneksi itu tetap
  diterima karena origin kosong diizinkan. Blok `be.ynsolo.id` meneruskan
  `Origin` asli, jadi origin web harus terdaftar di `CORS_ALLOWED_ORIGINS`.
- **Timeout.** Blok `be.ynsolo.id` tidak menyetel `proxy_read_timeout`,
  sedangkan `go.ynsolo.id` menyetel 86400 detik. Bila koneksi WebSocket
  terputus tiap menit lewat `be.ynsolo.id`, tambahkan
  `proxy_read_timeout 86400;` pada blok `location /ws/`.
- **Token.** Klien mengirim `?token=<JWT>` saat handshake. Token juga masih
  diterima lewat header `Authorization` untuk klien non-browser.

## nginx

`nginx.conf` di root repositori mencampur beberapa domain milik layanan lain
(`jualquota.com`, `ppob.bantukamujadiawardee.com`). **Fase 0 tidak
mengubahnya.** Yang perlu diperhatikan bila nanti dirapikan:

- Blok `ppob...` menambahkan `Access-Control-Allow-Origin *` pada `/api/v1/`.
  Header itu tidak ada hubungannya dengan API ynsolo.id, tetapi jangan
  ditiru: wildcard + credentials tidak sah menurut spesifikasi CORS.
- Sertifikat `ynsolo.pem`/`ynsolo.key` dipakai bersama oleh `be.` dan `go.`.

## Docker

`Dockerfile` membangun `cmd/main.go` dan `cmd/migrate`, lalu menyalin biner,
`template/`, dan `sql/migrations/`. **Rahasia tidak dibakar ke dalam image.**

Catatan untuk tiga hal yang sebelumnya menjadi temuan:

1. **`.env` tidak lagi disalin ke image.** `godotenv.Load()` ternyata sudah
   bersifat opsional sejak Fase 0 — kegagalannya hanya dicatat sebagai info
   (`cmd/main.go:44`), tidak menghentikan proses. Karena itu `COPY .env .env`
   dihapus dan `.env` didaftarkan di `.dockerignore` supaya `COPY . .` pada
   stage build juga tidak menyertakannya. Variabel datang dari
   `--env-file .env` saat `docker run`; berkasnya tetap tinggal di host.
2. **Runner migrasi ikut dibangun.** `migrate` membaca `sql/migrations`
   relatif terhadap direktori kerja, jadi kedua berkas itu ikut disalin.
   Migrasi dijalankan lewat `docker exec` — lihat target
   `migrate-container-*` di MakeFile.
3. **Kredensial database tidak lagi ada di MakeFile.** Target `runimage`
   memakai `--env-file .env`, bukan `-e DB_PASSWORD=…`.

## Urutan rilis

Backend dan frontend adalah **dua repositori terpisah** (`server/` dan
`web/`). Urutannya selalu backend lebih dulu: API baru harus sudah ada
sebelum UI yang memanggilnya dirilis. Rilis frontend lebih dulu akan
menghasilkan layar kosong pada halaman yang endpoint-nya belum ada.

1. `pg_dump` database produksi (lihat `docs/migrations.md`).
2. Bangun image backend: `make buildimage`.
3. Jalankan `make runimage`. Container baru sudah memuat biner `migrate`.
4. `make migrate-container-status` — pastikan migrasi yang tertunda sesuai
   harapan sebelum dijalankan.
5. `make migrate-container-baseline` **sekali saja**, hanya bila database
   itu sudah punya skema sebelum runner ini dipakai. Melewati langkah ini
   pada database lama membuat semua migrasi dianggap tertunda dan dijalankan
   ulang.
6. `make migrate-container-up`.
7. Pastikan `CORS_ALLOWED_ORIGINS` dan `COOKIE_SECURE=true` sudah ada di
   `.env` host.
8. Rilis frontend (repositori `web/`).
9. Jalankan daftar regresi manual di `docs/toolchain-baseline.md`.

## Rollback

Rollback backend dan frontend berdiri sendiri; keduanya tidak memerlukan
langkah DB kecuali migrasi terbaru memang harus dibatalkan.

| Bagian | Cara | Catatan |
| --- | --- | --- |
| Backend | Hentikan container, jalankan image bertag sebelumnya dengan `--env-file .env` yang sama. | `make runimage` selalu memakai `gushim/yukngaji:latest`, jadi tag sebelumnya harus disimpan saat `buildimage`. Sebelum menimpa `latest`, tandai dulu: `docker tag gushim/yukngaji:latest gushim/yukngaji:sebelum-<tanggal>`. |
| Migrasi DB | `make migrate-container-down` membatalkan **satu** migrasi terakhir. | Setiap migrasi punya berkas `.down.sql`, tetapi pembatalan tidak mengembalikan data yang sudah dihapus. Bila ada keraguan, pulihkan dari `pg_dump` langkah 1 — bukan dari `down`. |
| Frontend | Deploy ulang build sebelumnya. | Simpan direktori build rilis terakhir sebelum menimpanya. Tidak ada rollback otomatis. |

Urutan rollback: hentikan frontend lebih dulu bila ia memanggil endpoint yang
baru dibatalkan, lalu backend, lalu DB.

**Yang tidak ada:** tidak ada CI, tidak ada `docker-compose`, dan tidak ada
skrip rilis otomatis. Seluruh langkah di atas dijalankan manual.

## Di luar cakupan Fase 0

- Unggah bukti misi dan media: belum ada fitur unggah sama sekali. Perlu
  rencana penyimpanan (objek storage, batas ukuran, validasi tipe berkas)
  sebelum Fase 1 dimulai.
- Hub WebSocket multi-instance.
- Integrasi gateway pembayaran.
