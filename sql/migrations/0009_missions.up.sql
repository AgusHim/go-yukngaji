-- Fase 1 (F1-04, F1-05, F1-06): definisi misi dan klaim anggota.
--
-- Rentang waktu disimpan sebagai timestamptz (instant UTC). Perhitungan
-- periode (harian/mingguan/bulanan) dilakukan di aplikasi pada zona
-- Asia/Jakarta, lalu hasilnya disimpan sebagai period_key pada klaim — jadi
-- tidak ada perhitungan zona waktu di dalam query.

CREATE TABLE IF NOT EXISTS missions (
    id                varchar(36) PRIMARY KEY,
    code              varchar(64) NOT NULL,
    title             varchar(200) NOT NULL,
    description       text,
    type              varchar(16) NOT NULL
                      CHECK (type IN ('daily', 'weekly', 'special')),
    verification_mode varchar(24) NOT NULL
                      CHECK (verification_mode IN ('auto', 'self_claim', 'proof_approval')),
    reward_xp         integer NOT NULL DEFAULT 0 CHECK (reward_xp >= 0),
    claim_limit       integer NOT NULL DEFAULT 1 CHECK (claim_limit >= 1),
    version           integer NOT NULL DEFAULT 1 CHECK (version >= 1),
    starts_at         timestamptz NOT NULL,
    ends_at           timestamptz NOT NULL,
    is_published      boolean NOT NULL DEFAULT false,
    sort_order        integer NOT NULL DEFAULT 0,
    created_by        varchar(36),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz,
    CONSTRAINT missions_rentang_waktu CHECK (ends_at > starts_at)
);

-- Kode misi unik hanya di antara misi yang belum dihapus, supaya kode misi
-- lama dapat dipakai ulang tanpa menabrak riwayat.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_missions_code_aktif
    ON missions (code) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_missions_publikasi
    ON missions (is_published, starts_at, ends_at) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_missions_tipe
    ON missions (type, starts_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS mission_claims (
    id              varchar(36) PRIMARY KEY,
    mission_id      varchar(36) NOT NULL REFERENCES missions (id) ON DELETE RESTRICT,
    -- Versi definisi misi saat klaim dibuat, supaya perubahan definisi tidak
    -- mengubah arti klaim lama.
    mission_version integer NOT NULL,
    user_id         varchar(36) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Periode yang dihitung aplikasi di zona Asia/Jakarta, mis. "weekly:2026-W40".
    period_key      varchar(64) NOT NULL,
    status          varchar(16) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    -- Snapshot batas klaim saat klaim dibuat. Dipakai index unik parsial di
    -- bawah: penegakan "satu klaim aktif" hanya berlaku untuk misi berbatas 1.
    claim_limit     integer NOT NULL DEFAULT 1 CHECK (claim_limit >= 1),
    -- Reward di-snapshot saat disetujui; NULL selama belum diputuskan.
    reward_xp       integer CHECK (reward_xp IS NULL OR reward_xp >= 0),
    proof_url       text,
    proof_note      text,
    decision_reason text,
    reviewed_by     varchar(36),
    reviewed_at     timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);

-- Penegakan batas klaim tanpa read-modify-write. Predikat index boleh mengacu
-- kolom tabelnya sendiri, jadi batas "satu klaim aktif" hanya berlaku untuk
-- misi dengan claim_limit = 1; misi berbatas lebih besar diserialisasi
-- aplikasi memakai pg_advisory_xact_lock.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_mission_claims_aktif
    ON mission_claims (mission_id, user_id, period_key)
    WHERE status IN ('pending', 'approved') AND deleted_at IS NULL AND claim_limit = 1;

CREATE INDEX IF NOT EXISTS idx_mission_claims_antrean
    ON mission_claims (status, created_at) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mission_claims_anggota
    ON mission_claims (user_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_mission_claims_misi
    ON mission_claims (mission_id, status) WHERE deleted_at IS NULL;
