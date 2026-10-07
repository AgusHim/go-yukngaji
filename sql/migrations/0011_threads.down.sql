-- Membalik 0011_threads.
--
-- Hanya menjatuhkan tabel yang dibuat migrasi ini. Tidak ada kolom yang
-- ditambahkan ke tabel lama, jadi tidak ada yang perlu dipulihkan di
-- community_profiles maupun audit_logs. Urutannya mengikuti arah FK: anak
-- dulu, induk belakangan.

DROP TABLE IF EXISTS thread_reports;
DROP TABLE IF EXISTS thread_reactions;
DROP TABLE IF EXISTS thread_comments;
DROP TABLE IF EXISTS threads;
DROP TABLE IF EXISTS thread_share_prefs;
