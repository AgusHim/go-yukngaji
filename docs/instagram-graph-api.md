# Menyalakan Sinkronisasi Instagram

Dinding komunitas di landing page (section "Kegiatan Bersama YN Solo") bisa
diisi dua cara:

1. **Kurasi manual** lewat `/dashboard/instagram`. Ini selalu bisa dipakai dan
   tidak butuh apa pun dari Meta.
2. **Sinkronisasi Graph API** lewat tombol "Sync sekarang". Ini yang butuh
   token, dan dokumen ini menjelaskan cara mendapatkannya.

Tanpa token, sinkronisasi dijawab dengan pesan "belum dikonfigurasi" dan tidak
menulis apa pun. Dinding tetap tampil dengan postingan kurasi.

> Instagram **tidak bisa** dibaca tanpa autentikasi. Membuka
> `instagram.com/yukngajisolo` tanpa login hanya menghasilkan halaman login,
> bukan data postingan. Graph API adalah satu-satunya jalur yang sah.

## Apa saja yang bisa ditarik

| Endpoint | Isinya |
| --- | --- |
| `GET /<IG_ID>/media` | Postingan milik akun `@yukngajisolo` sendiri. |
| `GET /<IG_ID>/tags` | Postingan orang lain yang **menandai** akun ini. |

Keduanya membawa angka like dan komentar yang asli. Yang perlu diketahui:
postingan anggota yang **tidak** menandai akun tidak bisa diambil dengan cara
apa pun, jadi postingan seperti itu harus ditambahkan manual lewat dashboard.

Angka like/komentar tidak diminta pada `/tags`. Graph API sering menolaknya
untuk media milik orang lain, dan karena kartu menyembunyikan baris like saat
angkanya kosong, hasilnya tetap benar.

## Prasyarat

- Akun `@yukngajisolo` harus berjenis **Bisnis** atau **Creator**, bukan akun
  pribadi.
- Akun itu harus tertaut ke sebuah **Halaman Facebook**. Graph API menolak
  akun Bisnis yang belum punya halaman tertaut.

Keduanya diatur dari aplikasi Instagram: **Pengaturan → Akun → Beralih ke akun
profesional**, lalu tautkan halaman Facebook-nya.

## Langkah mendapatkan token

1. Buka [developers.facebook.com](https://developers.facebook.com) dan buat
   satu **app** bertipe Business.
2. Di dasbor app, tambahkan produk **Instagram**, lalu tautkan akun
   `@yukngajisolo` sebagai Instagram Business Account.
3. Tambahkan izin berikut:
   - `instagram_business_basic` — membaca profil dan media.
   - `instagram_business_manage_comments` — dibutuhkan `/tags`.

   `instagram_business_manage_comments` tidak dipakai untuk membalas komentar;
   ia yang membuka akses ke daftar media bertanda. Tanpa izin ini, `/tags`
   mengembalikan daftar kosong tanpa pesan kesalahan.
4. Buka **Graph API Explorer**, pilih app di atas, lalu pilih akun
   `@yukngajisolo` pada pemilih akun. Klik **Generate Access Token**.
5. Tukar token itu dengan **long-lived token** (masa berlaku ~60 hari):
   ```
   GET https://graph.instagram.com/access_token
       ?grant_type=ig_exchange_token
       &client_secret=<APP_SECRET>
       &access_token=<SHORT_LIVED_TOKEN>
   ```
6. Catat **IG user ID** numeriknya — ambil dari:
   ```
   GET https://graph.instagram.com/v21.0/me?fields=id,username&access_token=<TOKEN>
   ```
   Yang dipakai adalah `id` (deretan angka), bukan `username`.
7. Isi `.env` di server:
   ```
   INSTAGRAM_ACCESS_TOKEN=<long-lived token>
   INSTAGRAM_IG_USER_ID=<id numerik>
   INSTAGRAM_GRAPH_VERSION=v21.0
   ```
8. Restart API, lalu tekan **Sync sekarang** di `/dashboard/instagram`.
   Postingan yang masuk selalu tersimpan **belum terbit** — pengurus memilih
   mana yang tampil di landing page.

## Memperpanjang token

Token long-lived habis dalam ~60 hari. Perpanjang sebelum lewat:

```
GET https://graph.instagram.com/refresh_access_token
    ?grant_type=ig_refresh_token
    &access_token=<TOKEN>
```

Token hasil perpanjangan **harus disalin kembali ke `.env`** — tidak ada
penyimpanan token di aplikasi ini, jadi token lama tidak diperbarui otomatis.
Pasang pengingat kalender di sekitar hari ke-50.

> Token ini adalah rahasia. Jangan pernah commit, jangan tempel di issue atau
> chat, dan jangan kirim ke frontend: ia hanya dibaca server lewat
> `os.Getenv("INSTAGRAM_ACCESS_TOKEN")`. `.env` sudah masuk `.gitignore` dan
> tidak ikut ke dalam image Docker.

## Kalau gagal

| Gejala | Penyebab biasanya |
| --- | --- |
| `Invalid OAuth access token` (kode 190) | Token salah ketik atau sudah kedaluwarsa. Perpanjang, lalu salin ulang ke `.env`. |
| `(#10) Application does not have permission` | Izin belum ditambahkan ke app, atau app-nya belum di-review untuk akun ini. |
| `/tags` kosong tanpa error | Akun bukan Bisnis/Creator, atau `instagram_business_manage_comments` belum diberikan. |
| Sync jalan tetapi `skipped` tinggi | Item dari Graph API tidak punya `id`, sehingga tidak bisa dideduplikasi dan dilewati. |
| Sync jalan, `created` 0 dan `updated` banyak | Normal: postingan itu sudah pernah masuk dan hanya disegarkan angkanya. |

## Yang tidak dilakukan sinkronisasi

- **Tidak menerbitkan apa pun.** Baris baru selalu `is_published = false`.
- **Tidak menimpa keputusan pengurus.** `is_published` dan `sort_order`
  tidak pernah ditulis ulang oleh sync, jadi postingan yang disembunyikan
  atau digeser tetap begitu setelah sync berikutnya.
- **Tidak menghapus.** Postingan yang hilang dari Instagram tetap ada di
  database sampai pengurus menghapusnya sendiri.

## Penjadwalan

Backend ini tidak punya penjadwal internal — hanya `go hub.Run()` untuk
websocket. Sinkronisasi dijalankan manual dari dashboard, atau otomatis oleh
cron di host yang memanggil:

```
POST https://<host>/admin_api/instagram/sync
Authorization: Bearer <JWT admin>
```

Satu panggilan menarik maksimal 5 halaman (50 postingan per halaman) per
endpoint, jadi ia tidak akan berjalan tanpa ujung.
