-- Dinding komunitas Instagram di landing page (section "Kegiatan Bersama YN Solo").
--
-- Isinya dua macam baris yang hidup di tabel yang sama:
--   * kurasi admin  -> source = 'manual', ig_media_id NULL
--   * hasil sinkron -> source = 'sync',   ig_media_id diisi dari Graph API
--
-- like_count/comments_count sengaja boleh NULL. NULL berarti "angkanya tidak
-- diketahui", bukan nol — kartu di landing page menyembunyikan baris
-- like/komentar sepenuhnya selama kedua angkanya NULL, jadi membedakan
-- "tidak tahu" dari "nol" itu penting.

CREATE TABLE IF NOT EXISTS instagram_posts (
    id             varchar(36) PRIMARY KEY,
    -- ID media Instagram. NULL untuk kurasi admin. Dipakai sebagai kunci dedup
    -- sinkronisasi supaya mengulang sync tidak menggandakan baris.
    ig_media_id    varchar(64),
    source         varchar(16) NOT NULL DEFAULT 'manual'
                   CHECK (source IN ('manual', 'sync')),
    -- Tanpa tanda '@'.
    username       varchar(64) NOT NULL,
    caption        text,
    media_type     varchar(16)
                   CHECK (media_type IS NULL OR media_type IN ('IMAGE', 'VIDEO', 'CAROUSEL_ALBUM')),
    -- Untuk VIDEO, media_url menunjuk berkas videonya; yang dipakai kartu
    -- adalah thumbnail_url. Untuk IMAGE keduanya boleh sama.
    media_url      text,
    thumbnail_url  text,
    permalink      text,
    like_count     integer CHECK (like_count IS NULL OR like_count >= 0),
    comments_count integer CHECK (comments_count IS NULL OR comments_count >= 0),
    -- Diambil dari field `timestamp` Graph API.
    posted_at      timestamptz,
    is_published   boolean NOT NULL DEFAULT false,
    sort_order     integer NOT NULL DEFAULT 0,
    -- Sinkronisasi menulis created_by NULL; hanya kurasi admin yang mengisinya.
    created_by     varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    deleted_at     timestamptz
);

-- Dedup sinkronisasi. PostgreSQL memperlakukan NULL sebagai nilai yang berbeda,
-- jadi baris kurasi admin (ig_media_id NULL) tidak pernah saling menabrak.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_instagram_posts_media_aktif
    ON instagram_posts (ig_media_id) WHERE deleted_at IS NULL;

-- Urutan dinding publik: sort_order dulu (kurasi admin menang), lalu yang
-- paling baru.
CREATE INDEX IF NOT EXISTS idx_instagram_posts_publik
    ON instagram_posts (is_published, sort_order, posted_at DESC)
    WHERE deleted_at IS NULL;
