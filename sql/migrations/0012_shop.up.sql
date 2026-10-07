-- Marketplace merchandise penjual tunggal: katalog, varian, pesanan,
-- reservasi stok, dan buku besar stok.
--
-- Tidak ada perubahan pada tabel yang sudah ada selain satu baris seed di
-- xp_rules. Pesanan merchandise sengaja TIDAK memakai tabel orders: tabel itu
-- terikat event dan menerbitkan user_tickets, sehingga memakainya kembali akan
-- membuat pesanan produk tampak sebagai tiket.
--
-- INVARIAN STOK — satu-satunya aturan yang membuat seluruh perhitungan di
-- bawah ini benar, dan karenanya wajib dipegang setiap jalur tulis:
--
--   product_variants.stock adalah jumlah fisik yang ada di tangan.
--   stock_reservations adalah tahanan SEMENTARA untuk pesanan yang BELUM
--   dibayar. Konfirmasi pembayaran mengubah tahanan menjadi penjualan: stock
--   turun (gerak 'sale') dan reservasinya dilepas, dalam satu transaksi.
--   Pembatalan/refund pesanan yang sudah dibayar mengembalikan stock (gerak
--   'return'). Reservasi yang lewat expires_at berhenti dihitung tanpa perlu
--   ditulis apa pun.
--
-- Akibatnya ketersediaan = stock - SUM(reservasi aktif & belum kedaluwarsa)
-- bergerak benar di setiap transisi: turun saat checkout, TETAP saat dibayar
-- (kedua sisi bergerak sebesar qty), naik saat batal/refund/kedaluwarsa.

-- ---------------------------------------------------------------------------
-- products
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS products (
    id              varchar(36) PRIMARY KEY,
    public_id       varchar(32) NOT NULL,
    slug            varchar(160) NOT NULL,
    name            varchar(140) NOT NULL,
    description     text,
    cover_image_url text,
    status          varchar(16) NOT NULL DEFAULT 'draft',
    created_by      varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);

-- Harga tidak ada di sini: harga milik varian. Harga yang tampil di katalog
-- dihitung dari MIN(price) varian aktif, sehingga tidak ada angka yang bisa
-- menyimpang dari harga yang benar-benar ditagih.

DO $$
BEGIN
    ALTER TABLE products ADD CONSTRAINT chk_products_status
        CHECK (status IN ('draft', 'published', 'archived'));
    ALTER TABLE products ADD CONSTRAINT chk_products_nama
        CHECK (char_length(name) BETWEEN 3 AND 140);
    -- Slug dipakai di URL publik, jadi bentuknya dibatasi di sini juga. Fungsi
    -- NormalizeSlug di Go yang merapikannya lebih dulu; constraint ini yang
    -- menjaminnya tetap rapi walau ada jalur tulis baru kelak.
    ALTER TABLE products ADD CONSTRAINT chk_products_slug
        CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_products_public_id ON products (public_id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_products_slug ON products (slug);

CREATE INDEX IF NOT EXISTS idx_products_katalog
    ON products (status, created_at DESC, id DESC) WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- product_images
-- ---------------------------------------------------------------------------

-- Galeri foto sebagai tabel anak, bukan kolom array: tidak ada satu pun kolom
-- array di skema ini, dan tabel anak membuat urutan foto tersimpan eksplisit.
-- Seluruh isinya diganti utuh saat produk diperbarui, sehingga tidak perlu ada
-- endpoint pengelolaan galeri tersendiri.
CREATE TABLE IF NOT EXISTS product_images (
    id         varchar(36) PRIMARY KEY,
    product_id varchar(36) NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    image_url  text NOT NULL,
    position   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_product_images_produk
    ON product_images (product_id, position, id);

-- ---------------------------------------------------------------------------
-- product_variants
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS product_variants (
    id         varchar(36) PRIMARY KEY,
    public_id  varchar(32) NOT NULL,
    product_id varchar(36) NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    sku        varchar(64) NOT NULL DEFAULT '',
    -- size dan color sengaja NOT NULL DEFAULT '' alih-alih nullable: uniq di
    -- bawah hanya berfungsi sebagai penjaga keunikan bila Postgres tidak
    -- memperlakukan NULL sebagai berbeda. String kosong berarti "tidak
    -- berlaku", dan itu juga membuat label varian deterministik.
    size       varchar(32) NOT NULL DEFAULT '',
    color      varchar(32) NOT NULL DEFAULT '',
    price      bigint NOT NULL,
    stock      integer NOT NULL DEFAULT 0,
    status     varchar(16) NOT NULL DEFAULT 'active',
    position   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

DO $$
BEGIN
    ALTER TABLE product_variants ADD CONSTRAINT chk_product_variants_status
        CHECK (status IN ('active', 'inactive'));
    ALTER TABLE product_variants ADD CONSTRAINT chk_product_variants_harga
        CHECK (price >= 0);
    -- Stok negatif berarti buku besar dan kolom ini sudah menyimpang; lebih
    -- baik transaksinya gagal daripada angka mustahil tersimpan.
    ALTER TABLE product_variants ADD CONSTRAINT chk_product_variants_stok
        CHECK (stock >= 0);
    -- Varian tanpa pembeda apa pun tidak bermakna: satu produk hanya boleh
    -- punya satu varian seperti itu, dan pembeli tidak bisa membedakannya.
    ALTER TABLE product_variants ADD CONSTRAINT chk_product_variants_pembeda
        CHECK (char_length(size) > 0 OR char_length(color) > 0);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_product_variants_public_id
    ON product_variants (public_id);

-- Penjaga keunikan varian. Sengaja tanpa predikat deleted_at: varian yang
-- dihapus tidak boleh dihidupkan kembali lewat jalur ini, dan membiarkannya
-- tetap terhitung membuat percobaan ulang penambahan varian selalu ditolak
-- dengan jelas alih-alih diam-diam membuat baris kembar.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_product_variants_pembeda
    ON product_variants (product_id, size, color);

CREATE INDEX IF NOT EXISTS idx_product_variants_produk
    ON product_variants (product_id, position, id) WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- shop_orders
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS shop_orders (
    id                 varchar(36) PRIMARY KEY,
    public_id          varchar(32) NOT NULL,
    user_id            varchar(36) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- Snapshot jumlah yang disepakati saat checkout. Total TIDAK disimpan: ia
    -- subtotal + shipping_cost, dihitung di Go seperti orders.total.
    subtotal           bigint NOT NULL,
    -- Ongkir diisi pengurus, bukan dihitung otomatis. Karena itu total baru
    -- final setelah pengurus mengisinya, dan pembeli melihatnya di halaman
    -- pesanannya sebelum transfer.
    shipping_cost      bigint NOT NULL DEFAULT 0,
    -- Dua sumbu status yang sengaja dipisah: uang dan barang bergerak dengan
    -- kecepatan yang berbeda, dan menggabungkannya membuat "sudah dibayar tapi
    -- belum dikirim" tidak dapat dinyatakan.
    payment_status     varchar(16) NOT NULL DEFAULT 'pending',
    fulfillment_status varchar(16) NOT NULL DEFAULT 'unfulfilled',
    fulfillment_method varchar(16) NOT NULL,
    recipient_name     varchar(140) NOT NULL DEFAULT '',
    recipient_phone    varchar(32) NOT NULL DEFAULT '',
    recipient_address  text,
    tracking_number    text,
    payment_method_id  varchar(36) REFERENCES payment_methods (id) ON DELETE SET NULL,
    paid_amount        bigint,
    payment_reference  varchar(140),
    proof_url          text,
    confirmed_by       varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    confirmed_at       timestamptz,
    decision_reason    text,
    -- Snapshot XP yang benar-benar diberikan, dipakai untuk membalik tepat
    -- sebesar yang pernah diberikan saat pesanan di-refund. Pola yang sama
    -- dengan donations.rewarded_xp.
    rewarded_xp        integer NOT NULL DEFAULT 0,
    -- Batas waktu tahanan stok. Kedaluwarsanya tidak dijalankan pekerja latar
    -- apa pun: reservasi yang lewat batas berhenti dihitung pada query
    -- ketersediaan, dan pesanannya ditandai batal saat diakses.
    expires_at         timestamptz NOT NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    deleted_at         timestamptz
);

DO $$
BEGIN
    ALTER TABLE shop_orders ADD CONSTRAINT chk_shop_orders_pembayaran
        CHECK (payment_status IN ('pending', 'paid', 'rejected', 'refunded'));
    ALTER TABLE shop_orders ADD CONSTRAINT chk_shop_orders_pemenuhan
        CHECK (fulfillment_status IN ('unfulfilled', 'ready_for_pickup', 'shipped', 'completed', 'cancelled'));
    ALTER TABLE shop_orders ADD CONSTRAINT chk_shop_orders_cara
        CHECK (fulfillment_method IN ('pickup', 'shipping'));
    ALTER TABLE shop_orders ADD CONSTRAINT chk_shop_orders_angka
        CHECK (subtotal >= 0 AND shipping_cost >= 0 AND rewarded_xp >= 0);
    -- Pesanan kirim tanpa alamat tidak boleh ada di DB, bukan hanya ditolak di
    -- Go: alamat adalah satu-satunya cara barang sampai ke pembeli.
    ALTER TABLE shop_orders ADD CONSTRAINT chk_shop_orders_penerima
        CHECK (
            fulfillment_method <> 'shipping'
            OR (
                char_length(recipient_name) > 0
                AND char_length(recipient_phone) > 0
                AND char_length(coalesce(recipient_address, '')) > 0
            )
        );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_shop_orders_public_id
    ON shop_orders (public_id);

CREATE INDEX IF NOT EXISTS idx_shop_orders_pemilik
    ON shop_orders (user_id, created_at DESC, id DESC) WHERE deleted_at IS NULL;

-- Antrean pengurus: yang menunggu pembayaran lebih dulu.
CREATE INDEX IF NOT EXISTS idx_shop_orders_antrean
    ON shop_orders (payment_status, created_at DESC, id DESC) WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- shop_order_items
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS shop_order_items (
    id            varchar(36) PRIMARY KEY,
    order_id      varchar(36) NOT NULL REFERENCES shop_orders (id) ON DELETE CASCADE,
    -- ON DELETE SET NULL, bukan CASCADE: menghapus produk tidak boleh
    -- menghapus riwayat pembelian orang. Snapshot di bawah ini yang membuat
    -- barisnya tetap bermakna walau produknya sudah tiada.
    product_id    varchar(36) REFERENCES products (id) ON DELETE SET NULL,
    variant_id    varchar(36) REFERENCES product_variants (id) ON DELETE SET NULL,
    product_name  varchar(140) NOT NULL,
    variant_label varchar(80) NOT NULL DEFAULT '',
    unit_price    bigint NOT NULL,
    qty           integer NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
    -- Batas atas qty ditegakkan di sini juga supaya tidak ada jalur tulis yang
    -- bisa melewatinya. Angkanya sama dengan MaxQtyPerLine di Go.
    ALTER TABLE shop_order_items ADD CONSTRAINT chk_shop_order_items_qty
        CHECK (qty > 0 AND qty <= 10);
    ALTER TABLE shop_order_items ADD CONSTRAINT chk_shop_order_items_harga
        CHECK (unit_price >= 0);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Satu baris per varian per pesanan. Ini yang membuat perhitungan stok tidak
-- pernah ambigu, dan variant_id yang NULL (variannya sudah terhapus) tetap
-- boleh berulang karena Postgres memperlakukan NULL sebagai berbeda.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_shop_order_items_varian
    ON shop_order_items (order_id, variant_id);

CREATE INDEX IF NOT EXISTS idx_shop_order_items_pesanan
    ON shop_order_items (order_id);

-- ---------------------------------------------------------------------------
-- stock_reservations
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS stock_reservations (
    id          varchar(36) PRIMARY KEY,
    variant_id  varchar(36) NOT NULL REFERENCES product_variants (id) ON DELETE CASCADE,
    order_id    varchar(36) NOT NULL REFERENCES shop_orders (id) ON DELETE CASCADE,
    qty         integer NOT NULL,
    expires_at  timestamptz NOT NULL,
    released_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
    ALTER TABLE stock_reservations ADD CONSTRAINT chk_stock_reservations_qty
        CHECK (qty > 0);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Penjaga "konfirmasi berulang tidak memproses pesanan dua kali" sekaligus
-- "pembatalan mengembalikan stok tepat sekali": barisnya dibuat sekali per
-- (pesanan, varian), dan pelepasan memakai conditional UPDATE
-- WHERE released_at IS NULL yang RowsAffected-nya menjadi buktinya.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_stock_reservations_pesanan_varian
    ON stock_reservations (order_id, variant_id);

-- Dipakai perhitungan ketersediaan; hanya baris yang belum dilepas yang
-- relevan, sehingga index parsial jauh lebih kecil dari tabelnya.
CREATE INDEX IF NOT EXISTS idx_stock_reservations_aktif
    ON stock_reservations (variant_id, expires_at) WHERE released_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_stock_reservations_pesanan
    ON stock_reservations (order_id);

-- ---------------------------------------------------------------------------
-- stock_movements
-- ---------------------------------------------------------------------------

-- Buku besar stok yang diminta F4-03. Append-only seperti xp_ledger dan
-- audit_logs: tidak ada deleted_at dan tidak ada jalur update — koreksi berupa
-- baris baru. Inilah yang membuat kolom product_variants.stock dapat
-- direkonstruksi dan diperiksa, bukan sekadar dipercaya.
CREATE TABLE IF NOT EXISTS stock_movements (
    id            varchar(36) PRIMARY KEY,
    variant_id    varchar(36) NOT NULL REFERENCES product_variants (id) ON DELETE CASCADE,
    delta         integer NOT NULL,
    reason        varchar(24) NOT NULL,
    ref_type      varchar(16),
    ref_id        varchar(36),
    actor_user_id varchar(36) REFERENCES users (id) ON DELETE SET NULL,
    note          text,
    created_at    timestamptz NOT NULL DEFAULT now()
);

DO $$
BEGIN
    ALTER TABLE stock_movements ADD CONSTRAINT chk_stock_movements_delta
        CHECK (delta <> 0);
    ALTER TABLE stock_movements ADD CONSTRAINT chk_stock_movements_alasan
        CHECK (reason IN ('restock', 'adjustment', 'sale', 'return'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_stock_movements_varian
    ON stock_movements (variant_id, created_at DESC, id DESC);

-- Menelusuri seluruh pergerakan stok satu pesanan.
CREATE INDEX IF NOT EXISTS idx_stock_movements_rujukan
    ON stock_movements (ref_type, ref_id);

-- ---------------------------------------------------------------------------
-- Aturan XP
-- ---------------------------------------------------------------------------

-- Batas bulanan pesanan merchandise, sejajar dengan batas donasi: maksimum 3
-- pesanan ber-XP per bulan kalender Asia/Jakarta. Kuotanya TERPISAH dari kuota
-- donasi — membeli merchandise tidak menghabiskan kuota donasi, dan sebaliknya.
INSERT INTO xp_rules (id, source_type, xp, description, monthly_cap) VALUES
    ('xpr-shop', 'shop_order', 50, 'Pesanan merchandise yang pembayarannya terkonfirmasi (maksimum 3 per bulan)', 3)
ON CONFLICT DO NOTHING;
