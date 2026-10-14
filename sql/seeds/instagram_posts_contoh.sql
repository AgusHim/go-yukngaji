-- Data contoh dinding komunitas Instagram.
--
-- Ini BUKAN data asli. Tanpa INSTAGRAM_ACCESS_TOKEN, sinkronisasi Graph API
-- belum bisa jalan, jadi enam baris ini hanya untuk mengisi tampilan section
-- "Kegiatan Bersama YN Solo" supaya bisa dibandingkan dengan rancangannya.
--
-- Semuanya source = 'manual' dan is_published = true, jadi bisa disunting,
-- disembunyikan, atau dihapus lewat /dashboard/instagram seperti postingan
-- kurasi biasa. Menghapus semuanya sekaligus:
--
--   DELETE FROM instagram_posts WHERE source = 'manual';
--
-- Gambarnya gambar placeholder lokal di web/public/images/landing/kegiatan/,
-- bukan foto Instagram.
--
-- Idempoten: id-nya tetap, jadi menjalankan ulang seed ini tidak menggandakan
-- apa pun.

INSERT INTO instagram_posts
    (id, source, username, caption, media_type, media_url, permalink,
     like_count, comments_count, posted_at, is_published, sort_order)
VALUES
    ('c0e1a000-0000-4000-8000-000000000001', 'manual', 'masudizzulmujahid',
     'Ngobrol santai soal rencana program semester depan. Terima kasih sudah mampir ke markas.',
     'IMAGE', '/images/landing/kegiatan/contoh-1.jpg',
     'https://www.instagram.com/yukngajisolo/', 120, 16,
     '2026-09-28T10:15:00+07:00', true, 1),

    ('c0e1a000-0000-4000-8000-000000000002', 'manual', 'yukngajisolo',
     'BUKAN QURBAN BIASA',
     'IMAGE', '/images/landing/kegiatan/contoh-2.jpg',
     'https://www.instagram.com/yukngajisolo/', 120, 16,
     '2026-09-21T09:00:00+07:00', true, 2),

    ('c0e1a000-0000-4000-8000-000000000003', 'manual', 'mni__srj',
     '~merakit puzzle kehidupan~ Carilah tempat di mana dirimu diterima apa adanya, dengan lingkungan yang sesuai tujuan hidupmu. Nyatanya jalan sendiri itu berat, butuh teman untuk jalan bersama.',
     'IMAGE', '/images/landing/kegiatan/contoh-3.jpg',
     'https://www.instagram.com/yukngajisolo/', 120, 16,
     '2026-09-15T19:30:00+07:00', true, 3),

    ('c0e1a000-0000-4000-8000-000000000004', 'manual', 'muthiah_jasmine',
     'Jangan lelah belajar soal Baitul Maqdis. Setiap usaha dan tiap amal, insya Allah tidak akan sia-sia walaupun kita belum melihat hasilnya sekarang.',
     'IMAGE', '/images/landing/kegiatan/contoh-4.jpg',
     'https://www.instagram.com/yukngajisolo/', 120, 16,
     '2026-09-09T13:45:00+07:00', true, 4),

    -- Baris ini sengaja tanpa angka like/komentar, supaya aturan
    -- "sembunyikan barisnya" ikut terlihat di halaman.
    ('c0e1a000-0000-4000-8000-000000000005', 'manual', 'nisaagma',
     'Alhamdulillah, pertengahan bulan lalu berkesempatan mengikuti kegiatan "Sisters Day Out" Yuk Ngaji Solo dengan tema belajar mengelola pangan mandiri di rumah. Banyak ilmu baru soal pengelolaan limbah rumah tangga dan memberdayakan apa yang ada di sekitar.',
     'IMAGE', '/images/landing/kegiatan/contoh-5.jpg',
     'https://www.instagram.com/yukngajisolo/', NULL, NULL,
     '2026-09-02T08:20:00+07:00', true, 5),

    ('c0e1a000-0000-4000-8000-000000000006', 'manual', 'yukngajisolo',
     'Satu hari penuh kebersamaan di kebun. Terima kasih untuk semua yang datang dan ikut merapikan.',
     'IMAGE', '/images/landing/kegiatan/contoh-6.jpg',
     'https://www.instagram.com/yukngajisolo/', 96, 8,
     '2026-08-25T16:10:00+07:00', true, 6)
ON CONFLICT (id) DO NOTHING;
