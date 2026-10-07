-- Fase 2 (F2-01..F2-05): penggalangan dana — campaign, donasi, verifikasi
-- pembayaran manual, laporan penggunaan dana, dan jejak audit.
--
-- Donasi adalah transaksi yang TERPISAH dari order tiket. Kolom orders.donation
-- tidak dipakai ulang: ia komponen checkout tiket, bukan penggalangan dana.
--
-- Tipe uang: campaigns.target_amount, donations.amount, donations.paid_amount,
-- dan campaign_updates.amount memakai bigint, bukan integer seperti orders.
-- Alasannya progres campaign adalah SUM atas banyak baris dan target kumulatif
-- dapat melampaui batas integer (~2,1 miliar); penjumlahannya akan meluap di
-- SQL sebelum sempat dijumlahkan Go. Kolom orders tidak diubah.
--
-- Tidak ada tabel payments terpisah. Dengan verifikasi manual satu donasi
-- tepat punya satu pembayaran, sehingga tabel 1:1 hanya menambah join dan
-- menciptakan sumber kebenaran status kedua. Bila gateway ditambahkan kelak,
-- tabel percobaan pembayaran dapat diperkenalkan di belakang seam
-- PaymentProvider tanpa mengubah kontrak donasi maupun laporan.

CREATE TABLE IF NOT EXISTS campaigns (
    id              varchar(36) PRIMARY KEY,
    slug            varchar(120) NOT NULL,
    title           varchar(200) NOT NULL,
    summary         varchar(300),
    story           text,
    cover_image_url text,
    -- Jenis dana. Dipakai untuk penyaringan dan pelaporan.
    fund_type       varchar(32) NOT NULL
                    CHECK (fund_type IN ('operasional', 'dakwah', 'sosial', 'pendidikan', 'lainnya')),
    -- Penerima dana, dikelola pengurus. Bukan referensi akun: penerima bisa
    -- pihak luar yang tidak punya akun di sistem.
    recipient       varchar(200) NOT NULL,
    -- 0 berarti tanpa target nominal.
    target_amount   bigint NOT NULL DEFAULT 0 CHECK (target_amount >= 0),
    status          varchar(16) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'published', 'closed')),
    starts_at       timestamptz,
    ends_at         timestamptz,
    published_at    timestamptz,
    closed_at       timestamptz,
    created_by      varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz,
    CONSTRAINT campaigns_rentang_waktu
        CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at)
);

-- Slug unik hanya di antara campaign yang belum dihapus, supaya slug lama
-- dapat dipakai ulang tanpa menabrak riwayat.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_campaigns_slug_aktif
    ON campaigns (slug) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_campaigns_publikasi
    ON campaigns (status, starts_at, ends_at) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_campaigns_jenis
    ON campaigns (fund_type, created_at DESC) WHERE deleted_at IS NULL;

-- Satu baris donasi = satu donasi = satu pembayaran.
--
-- Status di sini adalah status UANG, bukan status pengiriman. Alurnya
-- pending -> confirmed | rejected | cancelled, dan confirmed -> refunded.
-- Tidak ada 'expired': tidak ada scheduler pada Fase 2, dan donasi pending
-- dibiarkan menunggu verifikasi pengurus.
CREATE TABLE IF NOT EXISTS donations (
    id                varchar(36) PRIMARY KEY,
    -- Rujukan publik, seperti orders.public_id. Satu-satunya identifier yang
    -- boleh dikirim ke donatur lewat tautan.
    public_id         varchar(32) NOT NULL,
    campaign_id       varchar(36) NOT NULL REFERENCES campaigns (id) ON DELETE RESTRICT,
    -- Identitas donatur selalu diambil server dari konteks auth, tidak pernah
    -- dari body. ON DELETE CASCADE: donasi ikut terhapus bila akunnya dihapus,
    -- supaya tidak ada baris tanpa pemilik.
    user_id           varchar(36) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    amount            bigint NOT NULL CHECK (amount > 0),
    status            varchar(16) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'confirmed', 'rejected', 'cancelled', 'refunded')),
    is_anonymous      boolean NOT NULL DEFAULT false,
    -- show_amount=false menyembunyikan nominal di daftar donatur publik,
    -- terpisah dari is_anonymous yang menyembunyikan identitas.
    show_amount       boolean NOT NULL DEFAULT true,
    message           text,
    -- 'none' berarti donatur tidak menulis pesan, sehingga tidak ada yang
    -- perlu dimoderasi dan donasi itu tidak masuk antrean moderasi.
    message_status    varchar(16) NOT NULL DEFAULT 'none'
                      CHECK (message_status IN ('none', 'pending', 'approved', 'hidden')),
    payment_method_id varchar(36) REFERENCES payment_methods (id) ON DELETE SET NULL,
    -- Token idempotensi dari klien. Menekan tombol "Donasi" dua kali tidak
    -- menghasilkan dua donasi.
    client_token      varchar(64),
    -- Nominal yang benar-benar diterima, diisi saat verifikasi.
    paid_amount       bigint CHECK (paid_amount IS NULL OR paid_amount > 0),
    payment_reference varchar(120),
    proof_url         text,
    confirmed_by      varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    confirmed_at      timestamptz,
    decision_reason   text,
    -- Snapshot XP yang benar-benar diberikan. Dipakai untuk membalik XP saat
    -- refund, sehingga perubahan aturan XP setelahnya tidak mengubah besarnya.
    rewarded_xp       integer NOT NULL DEFAULT 0 CHECK (rewarded_xp >= 0),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz,
    -- Pesan dan status pesannya harus konsisten: 'none' berarti tidak ada
    -- pesan, selain itu pesannya wajib ada dan tidak kosong. Ditulis sebagai
    -- dua cabang yang keduanya total, bukan ekuivalensi, supaya tidak ada
    -- cabang yang bernilai NULL (yang akan dilewatkan CHECK).
    CONSTRAINT donations_pesan_ada CHECK (
        (message_status = 'none' AND (message IS NULL OR btrim(message) = ''))
        OR (message_status <> 'none' AND message IS NOT NULL AND btrim(message) <> '')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_donations_public_id
    ON donations (public_id);

-- Idempotensi pembuatan donasi. Parsial pada deleted_at supaya donasi yang
-- sudah dihapus tidak memblokir pemakaian ulang token yang sama.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_donations_client_token
    ON donations (user_id, client_token)
    WHERE client_token IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_donations_campaign
    ON donations (campaign_id, status) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_donations_anggota
    ON donations (user_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_donations_antrean
    ON donations (status, created_at) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_donations_pesan
    ON donations (message_status, created_at)
    WHERE message_status = 'pending' AND deleted_at IS NULL;

-- Update penggunaan dana dan catatan biaya.
--
-- kind='update' adalah narasi biasa; 'usage' dan 'fee' wajib membawa amount
-- supaya dapat muncul sebagai baris di laporan. Biaya TIDAK mengurangi progres
-- — progres selalu bruto dari donasi terkonfirmasi; baris biaya hanya dicatat
-- terpisah agar tidak ada uang yang hilang dari laporan.
CREATE TABLE IF NOT EXISTS campaign_updates (
    id           varchar(36) PRIMARY KEY,
    campaign_id  varchar(36) NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE,
    title        varchar(200) NOT NULL,
    body         text NOT NULL,
    kind         varchar(16) NOT NULL DEFAULT 'update'
                 CHECK (kind IN ('update', 'usage', 'fee')),
    amount       bigint CHECK (amount IS NULL OR amount > 0),
    proof_url    text,
    is_published boolean NOT NULL DEFAULT false,
    created_by   varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz,
    CONSTRAINT campaign_updates_jumlah CHECK (kind = 'update' OR amount IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_campaign_updates_campaign
    ON campaign_updates (campaign_id, is_published, created_at DESC) WHERE deleted_at IS NULL;

-- Jejak audit append-only.
--
-- Tidak ada deleted_at dan tidak ada jalur update: koreksi berupa baris baru,
-- sama seperti xp_ledger. dedup_key unik membuat percobaan ulang yang sama
-- tidak menggandakan baris audit.
CREATE TABLE IF NOT EXISTS audit_logs (
    id            varchar(36) PRIMARY KEY,
    actor_user_id varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    action        varchar(64) NOT NULL,
    entity_type   varchar(32) NOT NULL,
    entity_id     varchar(36) NOT NULL,
    reason        text,
    detail        jsonb,
    dedup_key     varchar(200) NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_audit_logs_dedup_key
    ON audit_logs (dedup_key);

CREATE INDEX IF NOT EXISTS idx_audit_logs_entitas
    ON audit_logs (entity_type, entity_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_aktor
    ON audit_logs (actor_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_aksi
    ON audit_logs (action, created_at DESC);

-- Batas bulanan XP sebagai data, bukan konstanta kode. NULL berarti tanpa
-- batas. Hanya sumber donasi yang memakainya pada Fase 2.
ALTER TABLE IF EXISTS xp_rules ADD COLUMN IF NOT EXISTS monthly_cap integer
    CHECK (monthly_cap IS NULL OR monthly_cap > 0);

-- Satuan monthly_cap adalah JUMLAH DONASI ber-XP per bulan, bukan jumlah XP.
-- Reward donasi besarnya tetap, sehingga batas dalam satuan XP akan
-- menghabiskan seluruh kuota pada donasi pertama.
INSERT INTO xp_rules (id, source_type, xp, description, monthly_cap) VALUES
    ('xpr-donation', 'donation', 50, 'Donasi terkonfirmasi (maksimum 3 per bulan)', 3)
ON CONFLICT DO NOTHING;
