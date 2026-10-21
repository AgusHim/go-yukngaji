# Route ShopeePay / QRIS

Route yang sebelumnya dijalankan oleh `server.js` sekarang menjadi paket Go
`internal/shopee` dan ikut berjalan bersama server Gin utama. Tidak ada port,
process manager, atau server Node terpisah.

Seluruh route operasional selain `GET /api/health` dan `GET /` mewajibkan
header `X-API-Key` (atau query `api_key`) yang nilainya sama dengan `API_KEY`.

| Method | Route | Fungsi |
| --- | --- | --- |
| POST | `/update-token` | Memperbarui token Shopee di memori. Body: `token`. |
| GET | `/token-status` | Status konfigurasi token. |
| POST | `/create-qris` | Membuat QRIS nominal dinamis dari `QRIS_STATIC`. Body: `amount`. |
| GET | `/qr/:id` | Membaca QRIS yang belum kedaluwarsa (30 menit). |
| GET | `/transactions` | Membaca mutasi ShopeePay tiga hari terakhir. |
| GET | `/transactions/all` | Membaca mutasi ShopeePay bulan berjalan dengan pagination. |
| POST | `/check-payment` | Mengecek mutasi sukses berdasarkan `amount` dan `startTime` opsional. |
| GET | `/api/logs` | Maksimum 100 log operasional terakhir. |

Mutasi diambil memakai token sesi `SHOPEE_TOKEN`; token per request dapat
dikirim lewat `X-Shopee-Token`. `POST /check-payment` menandai satu ID transaksi
sebagai terpakai selama 24 jam agar satu mutasi tidak bisa diklaim dua kali.
Integrasi merchant Shopee ini bukan kontrak publik; pantau respons upstream dan
siapkan provider resmi bila Shopee mengubah endpoint atau skema responsnya.
