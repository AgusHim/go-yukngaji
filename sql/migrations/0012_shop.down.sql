-- Membalik 0012_shop.
--
-- Hanya menjatuhkan tabel yang dibuat migrasi ini, plus baris xp_rules yang
-- disisipkannya. Tidak ada kolom yang ditambahkan ke tabel lama, jadi tidak ada
-- yang perlu dipulihkan di users, payment_methods, maupun xp_rules selain baris
-- seed itu. Urutannya mengikuti arah FK: anak dulu, induk belakangan.

DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS stock_reservations;
DROP TABLE IF EXISTS shop_order_items;
DROP TABLE IF EXISTS shop_orders;
DROP TABLE IF EXISTS product_variants;
DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS products;

-- Ledger XP tidak ikut dihapus: baris xp_ledger yang sudah terlanjur tertulis
-- adalah riwayat nyata milik anggota, dan menghapusnya akan mengubah total XP
-- mereka. Yang dibatalkan hanyalah aturannya, supaya tidak ada aturan yang
-- menggantung menunjuk sumber yang tabelnya sudah tiada.
DELETE FROM xp_rules WHERE id = 'xpr-shop';
