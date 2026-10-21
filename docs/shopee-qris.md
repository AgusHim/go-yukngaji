# Route ShopeePay / QRIS

Route yang sebelumnya dijalankan oleh `server.js` sekarang menjadi paket Go
`internal/shopee` dan ikut berjalan bersama server Gin utama. Tidak ada port,
process manager, atau server Node terpisah.

Seluruh route operasional selain `GET /api/health` dan `GET /` mewajibkan
header `X-API-Key` (atau query `api_key`) yang nilainya sama dengan `API_KEY`.
Health menampilkan `token_configured`, `qris_configured`, dan
`auto_validation`; ketiganya dapat diperiksa tanpa membocorkan nilai rahasia.

| Method | Route | Fungsi |
| --- | --- | --- |
| POST | `/update-token` | Memperbarui token Shopee di memori. Body: `token`. |
| GET | `/token-status` | Status konfigurasi token. |
| POST | `/create-qris` | Membuat QRIS nominal dinamis dari `QRIS_STATIC`. Body: `amount`. |
| GET | `/qr/:id` | Membaca QRIS yang belum kedaluwarsa (30 menit). |
| GET | `/transactions` | Membaca mutasi ShopeePay tiga hari terakhir. |
| GET | `/transactions/all` | Membaca mutasi ShopeePay bulan berjalan dengan pagination. |
| POST | `/check-payment` | Membaca pembayaran yang sudah diklaim worker berdasarkan `amount` dan `startTime` opsional. |
| GET | `/api/logs` | Maksimum 100 log operasional terakhir. |

Mutasi diambil memakai token sesi `SHOPEE_TOKEN`; token per request dapat
dikirim lewat `X-Shopee-Token` untuk route inspeksi. Jalur transaksi aplikasi
tidak memakai penanda in-memory: charge, nominal unik, ID mutasi, lease worker,
dan status settlement disimpan pada `shopee_qris_charges`.

## Aktivasi transaksi aplikasi

1. Isi `SHOPEE_TOKEN`, `QRIS_STATIC`, dan `API_KEY` pada environment server.
2. Terapkan migrasi: `go run ./cmd/migrate up`.
3. Jalankan server Go. Log `auto-validasi aktif setiap 15s` menandakan worker
   sudah berjalan.
4. Pilih payment method `QRIS ShopeePay` (`id=shopee-qris`) pada checkout.

Worker mengambil mutasi sukses, mencocokkan `payable_amount`, lalu mengklaim
`transactionId` dengan conditional update dan unique index. Setelah itu domain
event, donasi, atau shop diselesaikan secara idempoten. Settlement yang gagal
akan disimpan pesannya dan dicoba ulang; mutasi tidak pernah dilepas untuk
transaksi lain. Endpoint legacy `/check-payment` hanya membaca klaim persisten
tersebut sehingga tidak dapat berbenturan dengan worker.

Integrasi merchant Shopee ini bukan kontrak publik; pantau respons upstream dan
siapkan provider resmi bila Shopee mengubah endpoint atau skema responsnya.
