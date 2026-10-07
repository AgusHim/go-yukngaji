-- Feed komunitas anonim: thread, komentar, reaksi, laporan, dan preferensi
-- berbagi aktivitas.
--
-- Tidak ada perubahan pada tabel yang sudah ada. Blokir akun memakai kolom
-- community_profiles.is_blocked yang sudah ada sejak 0007, dan tindakan
-- moderator berjejak di audit_logs yang sudah ada sejak 0010 — karena itu
-- "catatan moderasi" tidak menjadi tabel tersendiri: satu tabel jejak lebih
-- baik daripada dua sumber kebenaran untuk riwayat keputusan yang sama.

-- ---------------------------------------------------------------------------
-- threads
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS threads (
    id              varchar(36) PRIMARY KEY,
    -- public_id adalah satu-satunya rujukan yang boleh tampil di URL dan
    -- response; id akun penulis tidak pernah dikirim ke luar.
    public_id       varchar(32) NOT NULL,
    user_id         varchar(36) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title           varchar(140) NOT NULL,
    body            text NOT NULL,
    status          varchar(16) NOT NULL DEFAULT 'published',
    -- Auto-post: jejak ke aktivitas sumber. Thread manual selalu NULL pada
    -- kedua kolom. Keduanya NULL atau keduanya terisi.
    source_type     varchar(24),
    source_ref      varchar(36),
    comment_count   integer NOT NULL DEFAULT 0,
    reaction_count  integer NOT NULL DEFAULT 0,
    hidden_by       varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    hidden_at       timestamptz,
    decision_reason text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);

-- Batas panjang ditegakkan di sini, bukan hanya di Go: fungsi murni
-- CheckThreadContent memberi pesan galat yang ramah, constraint ini yang
-- menjaminnya tetap benar walau ada jalur tulis baru di kemudian hari.
DO $$
BEGIN
  ALTER TABLE threads ADD CONSTRAINT chk_threads_status
    CHECK (status IN ('published', 'hidden', 'deleted'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
  ALTER TABLE threads ADD CONSTRAINT chk_threads_source_type
    CHECK (source_type IS NULL OR source_type IN ('event_registration', 'donation', 'mission'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
  ALTER TABLE threads ADD CONSTRAINT chk_threads_source_lengkap
    CHECK ((source_type IS NULL) = (source_ref IS NULL));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
  ALTER TABLE threads ADD CONSTRAINT chk_threads_panjang
    CHECK (char_length(title) BETWEEN 3 AND 140 AND char_length(body) BETWEEN 1 AND 5000);
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
  ALTER TABLE threads ADD CONSTRAINT chk_threads_counter
    CHECK (comment_count >= 0 AND reaction_count >= 0);
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_threads_public_id
    ON threads (public_id);

-- Dedup auto-post. Postgres menganggap NULL berbeda satu sama lain, jadi
-- thread manual (source_type NULL) tidak pernah bertabrakan.
--
-- Sengaja TANPA predikat deleted_at IS NULL: pencabutan izin bersifat
-- permanen. Mematikan lalu menyalakan kembali preferensi tidak menghidupkan
-- postingan lama, dan percobaan ulang penulisan sumber tidak pernah
-- menggandakan.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_threads_source
    ON threads (source_type, source_ref);

CREATE INDEX IF NOT EXISTS idx_threads_feed
    ON threads (status, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_threads_penulis
    ON threads (user_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- Dipakai saat mencabut izin per aktivitas: hapus semua auto-post sejenis
-- milik satu akun.
CREATE INDEX IF NOT EXISTS idx_threads_sumber_penulis
    ON threads (source_type, user_id)
    WHERE source_type IS NOT NULL;

-- ---------------------------------------------------------------------------
-- thread_comments
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS thread_comments (
    id              varchar(36) PRIMARY KEY,
    public_id       varchar(32) NOT NULL,
    thread_id       varchar(36) NOT NULL REFERENCES threads (id) ON DELETE CASCADE,
    user_id         varchar(36) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    body            text NOT NULL,
    status          varchar(16) NOT NULL DEFAULT 'published',
    hidden_by       varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    hidden_at       timestamptz,
    decision_reason text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);

DO $$
BEGIN
  ALTER TABLE thread_comments ADD CONSTRAINT chk_thread_comments_status
    CHECK (status IN ('published', 'hidden', 'deleted'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
  ALTER TABLE thread_comments ADD CONSTRAINT chk_thread_comments_panjang
    CHECK (char_length(body) BETWEEN 1 AND 1000);
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_thread_comments_public_id
    ON thread_comments (public_id);

CREATE INDEX IF NOT EXISTS idx_thread_comments_urut
    ON thread_comments (thread_id, status, created_at, id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_thread_comments_penulis
    ON thread_comments (user_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- thread_reactions
-- ---------------------------------------------------------------------------

-- Tanpa deleted_at: batal reaksi adalah hard delete, sehingga index unik biasa
-- sudah cukup dan reaksi ulang berfungsi bersih. Inilah penjaga keunikan yang
-- tidak dimiliki tabel likes (satu akun bisa menyukai komentar yang sama
-- berkali-kali di sana). Tidak ada riwayat reaksi — audit hanya untuk tindakan
-- moderator, bukan reaksi anggota.
CREATE TABLE IF NOT EXISTS thread_reactions (
    id         varchar(36) PRIMARY KEY,
    thread_id  varchar(36) NOT NULL REFERENCES threads (id) ON DELETE CASCADE,
    user_id    varchar(36) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       varchar(16) NOT NULL DEFAULT 'like',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_thread_reactions_pengguna
    ON thread_reactions (thread_id, user_id);

CREATE INDEX IF NOT EXISTS idx_thread_reactions_thread
    ON thread_reactions (thread_id);

-- ---------------------------------------------------------------------------
-- thread_reports
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS thread_reports (
    id               varchar(36) PRIMARY KEY,
    public_id        varchar(32) NOT NULL,
    reporter_user_id varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    -- target_id sengaja tanpa foreign key: targetnya polimorfik (thread atau
    -- komentar). Integritasnya ditegakkan service, bukan constraint.
    target_type      varchar(16) NOT NULL,
    target_id        varchar(36) NOT NULL,
    reason           varchar(32) NOT NULL,
    note             text,
    status           varchar(16) NOT NULL DEFAULT 'open',
    handled_by       varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    handled_at       timestamptz,
    decision_reason  text,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
  ALTER TABLE thread_reports ADD CONSTRAINT chk_thread_reports_target
    CHECK (target_type IN ('thread', 'thread_comment'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
  ALTER TABLE thread_reports ADD CONSTRAINT chk_thread_reports_alasan
    CHECK (reason IN ('spam', 'sara', 'pornografi', 'penipuan', 'perundungan', 'lainnya'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
  ALTER TABLE thread_reports ADD CONSTRAINT chk_thread_reports_status
    CHECK (status IN ('open', 'actioned', 'dismissed'));
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_thread_reports_public_id
    ON thread_reports (public_id);

-- Satu laporan per akun per target. Ini yang menahan spam report; pelanggaran
-- constraint-nya diperlakukan sebagai "sudah pernah dilaporkan", bukan galat.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_thread_reports_sekali
    ON thread_reports (reporter_user_id, target_type, target_id);

CREATE INDEX IF NOT EXISTS idx_thread_reports_antrean
    ON thread_reports (status, created_at DESC, id DESC);

-- ---------------------------------------------------------------------------
-- thread_share_prefs
-- ---------------------------------------------------------------------------

-- Preferensi berbagi aktivitas. Dipisah dari community_profiles karena
-- community_profiles dibaca di jalur panas (leaderboard, daftar donatur, klaim
-- misi) dan tidak perlu ikut membesar; dengan begitu seluruh urusan auto-post
-- tetap di dalam modul thread.
--
-- Default true (kebijakan opt-out). Ketiadaan baris berarti semua true, jadi
-- akun lama tidak perlu di-backfill.
CREATE TABLE IF NOT EXISTS thread_share_prefs (
    user_id                 varchar(36) PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    share_event_registration boolean NOT NULL DEFAULT true,
    share_donation           boolean NOT NULL DEFAULT true,
    share_mission            boolean NOT NULL DEFAULT true,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);
