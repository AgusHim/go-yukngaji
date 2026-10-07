package shop

import (
	"errors"
	"time"

	"mainyuk/internal/audit"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// repository adalah satu-satunya tempat paket ini menyentuh database.
//
// Dua hal yang diulang di hampir setiap metode dan sengaja tidak disingkat:
// setiap query baca menambahkan `deleted_at IS NULL` (model memakai
// *time.Time, bukan gorm.DeletedAt, sehingga hapus di sini benar-benar hapus
// bila tidak dijaga), dan setiap perpindahan status memakai UPDATE bersyarat
// `status = from` yang mengembalikan RowsAffected.
type repository struct {
	db    *gorm.DB
	audit audit.Recorder
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db, audit: audit.NewRecorder(db)}
}

func (r *repository) WriteAudit(c *gin.Context, entry *audit.Log) error {
	return r.audit.Write(c, entry)
}

// ListAudit membaca riwayat toko dari tabel audit bersama.
//
// Penyaringnya adalah daftar entity_type, bukan kolom khusus: tabel audit
// dipakai bersama donasi, moderasi, dan toko, dan tiap modul memilih barisnya
// sendiri lewat jenis entitas yang memang hanya ia tulis.
func (r *repository) ListAudit(c *gin.Context, entityTypes []string, limit, offset int) ([]*audit.Log, error) {
	entries := []*audit.Log{}
	if len(entityTypes) == 0 {
		return entries, nil
	}
	err := r.db.Model(&audit.Log{}).
		Where("entity_type IN ?", entityTypes).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ---------------------------------------------------------------------------
// Katalog
// ---------------------------------------------------------------------------

// ListProducts membaca katalog dengan urutan yang dipilih pembeli.
//
// Harga terendah dihitung lewat subquery, bukan diurutkan di Go: mengurutkan
// setelah paginasi hanya akan mengurutkan satu halaman, sehingga "termurah"
// menampilkan barang yang bukan termurah. Subquery-nya hanya menghitung varian
// aktif — varian yang dinonaktifkan tidak boleh menentukan harga katalog.
func (r *repository) ListProducts(c *gin.Context, status, sort string, limit, offset int) ([]*Product, error) {
	products := []*Product{}

	query := r.db.
		Table("products AS p").
		Select("p.*").
		Joins(`LEFT JOIN (
			SELECT product_id, MIN(price) AS min_price
			  FROM product_variants
			 WHERE deleted_at IS NULL AND status = ?
			 GROUP BY product_id
		) AS v ON v.product_id = p.id`, VariantActive).
		Where("p.deleted_at IS NULL")

	if status != "" {
		query = query.Where("p.status = ?", status)
	}

	switch sort {
	case SortTermurah:
		query = query.Order("COALESCE(v.min_price, 0) ASC, p.created_at DESC, p.id DESC")
	case SortTermahal:
		query = query.Order("COALESCE(v.min_price, 0) DESC, p.created_at DESC, p.id DESC")
	default:
		query = query.Order("p.created_at DESC, p.id DESC")
	}

	if err := query.Limit(limit).Offset(offset).Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (r *repository) ListProductsByIDs(c *gin.Context, ids []string) ([]*Product, error) {
	products := []*Product{}
	if len(ids) == 0 {
		return products, nil
	}
	err := r.db.
		Where("id IN ?", ids).
		Where("deleted_at IS NULL").
		Find(&products).Error
	if err != nil {
		return nil, err
	}
	return products, nil
}

func (r *repository) FindProductBySlug(c *gin.Context, slug string) (*Product, error) {
	product := &Product{}
	err := r.db.
		Where("slug = ?", slug).
		Where("deleted_at IS NULL").
		First(product).Error
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (r *repository) FindProductByID(c *gin.Context, id string) (*Product, error) {
	product := &Product{}
	err := r.db.
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		First(product).Error
	if err != nil {
		return nil, err
	}
	return product, nil
}

// CreateProduct menyisipkan produk beserta galerinya dalam satu transaksi.
//
// Galeri diganti utuh, bukan ditambahi satu per satu, sehingga tidak pernah ada
// keadaan setengah jadi bila salah satu barisnya gagal.
func (r *repository) CreateProduct(c *gin.Context, product *Product, images []*ProductImage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(product).Error; err != nil {
			return err
		}
		for _, image := range images {
			image.ProductID = product.ID
			if err := tx.Create(image).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateProduct memperbarui kolom produk, dan mengganti galerinya bila images
// tidak nil. Nil berarti "galeri tidak disentuh".
func (r *repository) UpdateProduct(c *gin.Context, id string, fields map[string]interface{}, images *[]*ProductImage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(fields) > 0 {
			res := tx.Model(&Product{}).
				Where("id = ?", id).
				Where("deleted_at IS NULL").
				Updates(fields)
			if res.Error != nil {
				return res.Error
			}
		}
		if images == nil {
			return nil
		}
		if err := tx.Where("product_id = ?", id).Delete(&ProductImage{}).Error; err != nil {
			return err
		}
		for _, image := range *images {
			image.ProductID = id
			if err := tx.Create(image).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SetProductStatus memindahkan status produk secara atomik.
//
// RowsAffected = 0 berarti produknya sudah berpindah lebih dulu; pemanggil
// memperlakukannya sebagai sukses idempoten, bukan kegagalan.
func (r *repository) SetProductStatus(c *gin.Context, id, from, to string) (bool, error) {
	res := r.db.Model(&Product{}).
		Where("id = ?", id).
		Where("status = ?", from).
		Where("deleted_at IS NULL").
		Updates(map[string]interface{}{
			"status":     to,
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) SoftDeleteProduct(c *gin.Context, id string) (bool, error) {
	res := r.db.Model(&Product{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) ListImagesByProducts(c *gin.Context, productIDs []string) (map[string][]*ProductImage, error) {
	grouped := map[string][]*ProductImage{}
	if len(productIDs) == 0 {
		return grouped, nil
	}
	images := []*ProductImage{}
	if err := r.db.
		Where("product_id IN ?", productIDs).
		Order("product_id, position, id").
		Find(&images).Error; err != nil {
		return nil, err
	}
	for _, image := range images {
		grouped[image.ProductID] = append(grouped[image.ProductID], image)
	}
	return grouped, nil
}

// ---------------------------------------------------------------------------
// Varian
// ---------------------------------------------------------------------------

func (r *repository) ListVariantsByProduct(c *gin.Context, productID string, activeOnly bool) ([]*ProductVariant, error) {
	variants := []*ProductVariant{}
	query := r.db.Where("product_id = ?", productID).Where("deleted_at IS NULL")
	if activeOnly {
		query = query.Where("status = ?", VariantActive)
	}
	if err := query.Order("position, id").Find(&variants).Error; err != nil {
		return nil, err
	}
	return variants, nil
}

func (r *repository) ListVariantsByProducts(c *gin.Context, productIDs []string) ([]*ProductVariant, error) {
	variants := []*ProductVariant{}
	if len(productIDs) == 0 {
		return variants, nil
	}
	err := r.db.
		Where("product_id IN ?", productIDs).
		Where("deleted_at IS NULL").
		Order("product_id, position, id").
		Find(&variants).Error
	if err != nil {
		return nil, err
	}
	return variants, nil
}

func (r *repository) ListVariantsByIDs(c *gin.Context, ids []string) ([]*ProductVariant, error) {
	variants := []*ProductVariant{}
	if len(ids) == 0 {
		return variants, nil
	}
	err := r.db.
		Where("id IN ?", ids).
		Where("deleted_at IS NULL").
		Find(&variants).Error
	if err != nil {
		return nil, err
	}
	return variants, nil
}

func (r *repository) FindVariantByID(c *gin.Context, id string) (*ProductVariant, error) {
	variant := &ProductVariant{}
	err := r.db.
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		First(variant).Error
	if err != nil {
		return nil, err
	}
	return variant, nil
}

func (r *repository) CreateVariant(c *gin.Context, variant *ProductVariant) error {
	return r.db.Create(variant).Error
}

func (r *repository) UpdateVariant(c *gin.Context, id string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.Model(&ProductVariant{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(fields).Error
}

func (r *repository) SoftDeleteVariant(c *gin.Context, id string) (bool, error) {
	res := r.db.Model(&ProductVariant{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ---------------------------------------------------------------------------
// Ketersediaan
// ---------------------------------------------------------------------------

// ReservedQty menjumlahkan tahanan aktif per varian.
//
// Reservasi yang sudah lewat expires_at sengaja TIDAK dihitung: itulah
// "kedaluwarsa dicek saat diakses" pada tingkat query. Tidak ada pekerja latar
// yang perlu membebaskan tahanan, dan tahanan yang kedaluwarsa tidak pernah
// menahan penjualan lebih lama dari seharusnya.
func (r *repository) ReservedQty(c *gin.Context, variantIDs []string, now time.Time) (map[string]int, error) {
	reserved := map[string]int{}
	if len(variantIDs) == 0 {
		return reserved, nil
	}

	rows := []struct {
		VariantID string
		Total     int
	}{}
	err := r.db.Model(&StockReservation{}).
		Select("variant_id, COALESCE(SUM(qty), 0) AS total").
		Where("variant_id IN ?", variantIDs).
		Where("released_at IS NULL").
		Where("expires_at > ?", now).
		Group("variant_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		reserved[row.VariantID] = row.Total
	}
	return reserved, nil
}

// ---------------------------------------------------------------------------
// Pesanan
// ---------------------------------------------------------------------------

func (r *repository) ListOrdersByUser(c *gin.Context, userID string, limit, offset int) ([]*ShopOrder, error) {
	orders := []*ShopOrder{}
	err := r.db.
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *repository) ListOrders(c *gin.Context, filter OrderFilter, limit, offset int) ([]*ShopOrder, error) {
	orders := []*ShopOrder{}
	query := r.db.Where("deleted_at IS NULL")
	if filter.PaymentStatus != "" {
		query = query.Where("payment_status = ?", filter.PaymentStatus)
	}
	if filter.FulfillmentStatus != "" {
		query = query.Where("fulfillment_status = ?", filter.FulfillmentStatus)
	}
	err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *repository) FindOrderByPublicID(c *gin.Context, publicID string) (*ShopOrder, error) {
	order := &ShopOrder{}
	err := r.db.
		Where("public_id = ?", publicID).
		Where("deleted_at IS NULL").
		First(order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *repository) FindOrderByID(c *gin.Context, id string) (*ShopOrder, error) {
	order := &ShopOrder{}
	err := r.db.
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		First(order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *repository) ListOrderItems(c *gin.Context, orderIDs []string) (map[string][]*ShopOrderItem, error) {
	grouped := map[string][]*ShopOrderItem{}
	if len(orderIDs) == 0 {
		return grouped, nil
	}
	items := []*ShopOrderItem{}
	if err := r.db.
		Where("order_id IN ?", orderIDs).
		Order("order_id, created_at, id").
		Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		grouped[item.OrderID] = append(grouped[item.OrderID], item)
	}
	return grouped, nil
}

func (r *repository) OrderItemsWithVariant(c *gin.Context, orderID string) ([]*ShopOrderItem, error) {
	items := []*ShopOrderItem{}
	if err := r.db.
		Where("order_id = ?", orderID).
		Order("created_at, id").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// CreateOrder menyisipkan pesanan, barisnya, dan reservasi stoknya dalam satu
// transaksi — inilah penjaga overselling yang sesungguhnya.
//
// Urutannya mengikat dan tidak boleh diubah: kunci advisory diambil lebih dulu
// untuk SETIAP varian dalam urutan menaik, baru ketersediaannya dihitung ulang
// di dalam kunci. Mengambil kunci per varian sambil menghitung di antaranya
// akan membuat angka yang dipakai keputusan sudah basi sebelum dipakai.
//
// Urutan menaik itu bukan hiasan. Dua checkout bersamaan yang berbagi dua
// varian dengan urutan berbeda akan saling menunggu secara melingkar bila
// kuncinya diambil menurut urutan baris keranjang, dan deadlock di jalur
// checkout berarti pembeli melihat kegagalan yang tidak dapat mereka pahami.
//
// Seluruh pembacaan stok terjadi di dalam tx yang sama dengan pengambilan
// kunci. Membacanya lewat koneksi lain akan membuat kuncinya tidak menjaga apa
// pun, karena angka yang dipakai keputusan tidak lagi berasal dari transaksi
// yang memegang kuncinya.
func (r *repository) CreateOrder(c *gin.Context, order *ShopOrder, items []*ShopOrderItem, lines []CartLine) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, variantID := range ReservationOrdering(lines) {
			if err := tx.Exec(
				"SELECT pg_advisory_xact_lock(hashtext(?))",
				"shop_stock:"+variantID,
			).Error; err != nil {
				return err
			}
		}

		// Ketersediaan dihitung ulang setelah seluruh kunci dipegang. Satu
		// query agregat untuk semua varian sekaligus, bukan satu per varian:
		// jumlah varian dalam satu keranjang kecil, tetapi jumlah query-nya
		// tetap lebih baik satu.
		variantIDs := make([]string, 0, len(lines))
		for _, line := range lines {
			variantIDs = append(variantIDs, line.VariantID)
		}

		reserved := map[string]int{}
		rows := []struct {
			VariantID string
			Total     int
		}{}
		if err := tx.Model(&StockReservation{}).
			Select("variant_id, COALESCE(SUM(qty), 0) AS total").
			Where("variant_id IN ?", variantIDs).
			Where("released_at IS NULL").
			Where("expires_at > ?", time.Now()).
			Group("variant_id").
			Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			reserved[row.VariantID] = row.Total
		}

		available := map[string]int{}
		var subtotal int64
		for _, item := range items {
			if item.VariantID == nil {
				continue
			}
			// Stok fisik dan harga dibaca dari baris varian yang dipegang
			// kuncinya. Harga ikut dibaca di sini, bukan hanya di service,
			// supaya nominal yang tersimpan adalah harga pada saat kuncinya
			// dipegang — bukan harga yang sempat berubah di antaranya.
			variant := &ProductVariant{}
			if err := tx.
				Where("id = ?", *item.VariantID).
				Where("deleted_at IS NULL").
				First(variant).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errVariantGone
				}
				return err
			}
			item.UnitPrice = variant.Price
			subtotal += variant.Price * int64(item.Qty)
			available[*item.VariantID] = AvailableStock(variant.Stock, reserved[*item.VariantID])
		}

		if err := ValidateCartLines(lines, available); err != nil {
			// Sentinel, bukan galat validasi: transaksinya harus menggelinding
			// mundur, dan service yang menerjemahkannya menjadi pesan yang
			// menyebut variannya.
			return errStockShortage
		}
		order.Subtotal = subtotal

		if err := tx.Create(order).Error; err != nil {
			return err
		}

		for _, item := range items {
			item.OrderID = order.ID
			if err := tx.Create(item).Error; err != nil {
				return err
			}
		}

		for _, line := range lines {
			res := tx.Exec(
				`INSERT INTO stock_reservations
				   (id, variant_id, order_id, qty, expires_at, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)
				 ON CONFLICT (order_id, variant_id) DO NOTHING`,
				uuid.NewString(), line.VariantID, order.ID, line.Qty, order.ExpiresAt, time.Now(),
			)
			if res.Error != nil {
				return res.Error
			}
		}

		return nil
	})

	if err != nil {
		// Sentinel dikembalikan apa adanya: GORM meneruskan galat dari dalam
		// Transaction tanpa membungkusnya, sehingga service dapat mengenalinya
		// lewat errors.Is.
		return err
	}
	return nil
}

// UpdateOrderFields memperbarui kolom pesanan yang tidak mengubah statusnya —
// saat ini hanya penyerahan bukti transfer oleh pembeli.
//
// Syarat payment_status ikut dipasang supaya bukti tidak bisa menempel pada
// pesanan yang sudah diputuskan pengurus di sela antara pembacaan dan
// penulisan. RowsAffected = 0 berarti syaratnya tidak lagi terpenuhi, dan
// pemanggil memperlakukannya sebagai "sudah diputuskan", bukan kegagalan.
func (r *repository) UpdateOrderFields(c *gin.Context, id, expectedPaymentStatus string, fields map[string]interface{}) (bool, error) {
	if len(fields) == 0 {
		return false, nil
	}
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["updated_at"] = time.Now()

	query := r.db.Model(&ShopOrder{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL")
	if expectedPaymentStatus != "" {
		query = query.Where("payment_status = ?", expectedPaymentStatus)
	}

	res := query.Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) TransitionFulfillment(c *gin.Context, id, from, to string, fields map[string]interface{}) (bool, error) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["fulfillment_status"] = to
	fields["updated_at"] = time.Now()

	res := r.db.Model(&ShopOrder{}).
		Where("id = ?", id).
		Where("fulfillment_status = ?", from).
		Where("deleted_at IS NULL").
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// SetOrderReward menyimpan snapshot XP yang benar-benar diberikan.
//
// Nilai 0 tidak pernah menimpa snapshot yang sudah ada: bila pemberian reward
// gagal sebagian lalu diulang, snapshot lama tetap menjadi acuan pembalikan.
// Pola yang sama dengan donations.rewarded_xp.
func (r *repository) SetOrderReward(c *gin.Context, id string, xp int) error {
	if xp <= 0 {
		return nil
	}
	return r.db.Model(&ShopOrder{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(map[string]interface{}{
			"rewarded_xp": xp,
			"updated_at":  time.Now(),
		}).Error
}

// ---------------------------------------------------------------------------
// Stok
// ---------------------------------------------------------------------------

// ConfirmPayment memindahkan pesanan pending -> paid DAN mengubah tahanannya
// menjadi penjualan, keduanya dalam satu transaksi.
//
// Digabung bukan karena ringkas, melainkan karena urutan yang terpisah selalu
// punya celah: status yang sudah 'paid' sementara stoknya belum turun membuat
// pembeli lain melihat barang yang sebenarnya sudah terjual, dan stok yang
// sudah turun sementara statusnya belum berpindah membuat pesanan yang gagal
// mengunci persediaan tanpa alasan.
//
// RowsAffected = 0 pada langkah pertama berarti pesanan sudah diputuskan lebih
// dulu (oleh pengurus lain, atau oleh percobaan ulang yang sama). Itu SUKSES
// idempoten, bukan galat — dan justru itulah yang membuat konfirmasi berulang
// tidak memproses pesanan dua kali: seluruh isi transaksi ini dilewati.
func (r *repository) ConfirmPayment(c *gin.Context, id string, fields map[string]interface{}, actorID string) (bool, error) {
	changed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		ok, err := transitionPaymentTx(tx, id, PaymentPending, PaymentPaid, fields)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		changed = true
		return commitSaleTx(tx, id, actorID)
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// RejectPayment memindahkan pending -> rejected dan melepas tahanan stoknya.
//
// Stok fisik tidak pernah turun untuk pesanan yang belum dibayar, jadi yang
// perlu dilakukan hanyalah berhenti menahannya.
func (r *repository) RejectPayment(c *gin.Context, id string, fields map[string]interface{}) (bool, error) {
	changed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		ok, err := transitionPaymentTx(tx, id, PaymentPending, PaymentRejected, fields)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		changed = true
		if err := lockOrderTx(tx, id); err != nil {
			return err
		}
		_, err = releaseReservationsTx(tx, id)
		return err
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// RefundPayment memindahkan paid -> refunded dan mengembalikan stoknya.
//
// Pembalikan XP tidak ada di sini: ia milik gamification, seam yang terpisah
// dari database paket ini, dan dipanggil service setelah transaksi ini selesai.
func (r *repository) RefundPayment(c *gin.Context, id string, fields map[string]interface{}, actorID string) (bool, error) {
	changed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		ok, err := transitionPaymentTx(tx, id, PaymentPaid, PaymentRefunded, fields)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		changed = true
		if err := lockOrderTx(tx, id); err != nil {
			return err
		}
		return returnStockTx(tx, id, actorID, "")
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// CancelSettlement membatalkan pemenuhan pesanan dan menyesuaikan stoknya
// sesuai keadaan pembayarannya, dalam satu transaksi.
//
// Dua cabangnya sengaja diputuskan DI DALAM transaksi dengan membaca ulang
// barisnya, bukan dari nilai yang dipegang pemanggil: status pembayaran bisa
// saja berubah di sela antara pembacaan service dan penulisan ini, dan
// mengembalikan stok pesanan yang ternyata belum dibayar akan menggelembungkan
// persediaan.
func (r *repository) CancelSettlement(c *gin.Context, id, fromFulfillment string, fields map[string]interface{}, actorID, reason string) (bool, error) {
	changed := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if fields == nil {
			fields = map[string]interface{}{}
		}
		fields["fulfillment_status"] = FulfillmentCancelled
		fields["updated_at"] = time.Now()

		res := tx.Model(&ShopOrder{}).
			Where("id = ?", id).
			Where("fulfillment_status = ?", fromFulfillment).
			Where("deleted_at IS NULL").
			Updates(fields)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		changed = true

		if err := lockOrderTx(tx, id); err != nil {
			return err
		}

		order := &ShopOrder{}
		if err := tx.Where("id = ?", id).First(order).Error; err != nil {
			return err
		}

		if order.PaymentStatus == PaymentPaid {
			// Tahanannya sudah berubah menjadi penjualan saat pembayarannya
			// dikonfirmasi, jadi yang dikembalikan adalah stok fisiknya.
			return returnStockTx(tx, id, actorID, reason)
		}
		_, err := releaseReservationsTx(tx, id)
		return err
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// ---------------------------------------------------------------------------
// Pembantu transaksi stok
// ---------------------------------------------------------------------------

// transitionPaymentTx adalah satu-satunya tempat status pembayaran berpindah.
// Guard-nya adalah pasangan (id, status lama), sehingga dua pemanggil yang
// berlomba tidak mungkin dua-duanya berhasil.
func transitionPaymentTx(tx *gorm.DB, id, from, to string, fields map[string]interface{}) (bool, error) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["payment_status"] = to
	fields["updated_at"] = time.Now()

	res := tx.Model(&ShopOrder{}).
		Where("id = ?", id).
		Where("payment_status = ?", from).
		Where("deleted_at IS NULL").
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// lockOrderTx menyerialkan seluruh perpindahan stok satu pesanan, sehingga
// pelepasan dan pengembalian tidak pernah berjalan bersamaan.
func lockOrderTx(tx *gorm.DB, orderID string) error {
	return tx.Exec(
		"SELECT pg_advisory_xact_lock(hashtext(?))",
		"shop_order:"+orderID,
	).Error
}

// pendingReservationsTx membaca tahanan yang belum dilepas milik satu pesanan.
func pendingReservationsTx(tx *gorm.DB, orderID string) ([]*StockReservation, error) {
	pending := []*StockReservation{}
	err := tx.
		Where("order_id = ?", orderID).
		Where("released_at IS NULL").
		Order("variant_id").
		Find(&pending).Error
	if err != nil {
		return nil, err
	}
	return pending, nil
}

// releaseReservationsTx menandai tahanan sebagai dilepas dan mengembalikan yang
// BARU dilepas.
//
// Inilah penjaga "tepat sekali": conditional UPDATE `released_at IS NULL`
// membuat pelepasan kedua menghasilkan RowsAffected = 0, sehingga tidak ada
// baris yang ikut dilepas dua kali.
func releaseReservationsTx(tx *gorm.DB, orderID string) ([]*StockReservation, error) {
	pending, err := pendingReservationsTx(tx, orderID)
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(pending))
	for _, reservation := range pending {
		ids = append(ids, reservation.ID)
	}
	now := time.Now()
	if err := tx.Model(&StockReservation{}).
		Where("id IN ?", ids).
		Where("released_at IS NULL").
		Updates(map[string]interface{}{"released_at": now}).Error; err != nil {
		return nil, err
	}
	return pending, nil
}

// commitSaleTx menurunkan stok fisik, mencatat pergerakan 'sale', dan melepas
// tahanannya.
//
// Reservasi yang sudah dilepas tidak ikut dihitung, sehingga memanggilnya dua
// kali untuk pesanan yang sama tidak menurunkan stok dua kali — penjaga lapis
// kedua setelah conditional UPDATE pada status pembayaran.
//
// Penurunan stok memakai `stock >= ?` sebagai syarat. Kondisi itu seharusnya
// selalu terpenuhi karena tahanannya memegang barangnya; bila tidak, berarti
// ada ketidakcocokan nyata (tahanannya sudah kedaluwarsa dan stoknya diambil
// pembeli lain), dan pesanannya harus gagal dengan jelas alih-alih menjual
// barang yang tidak ada.
func commitSaleTx(tx *gorm.DB, orderID, actorID string) error {
	if err := lockOrderTx(tx, orderID); err != nil {
		return err
	}

	pending, err := pendingReservationsTx(tx, orderID)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}

	now := time.Now()
	for _, reservation := range pending {
		res := tx.Exec(
			`UPDATE product_variants
			    SET stock = stock - ?, updated_at = ?
			  WHERE id = ? AND stock >= ? AND deleted_at IS NULL`,
			reservation.Qty, now, reservation.VariantID, reservation.Qty,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errStockShortage
		}
		if err := tx.Create(&StockMovement{
			ID:          uuid.NewString(),
			VariantID:   reservation.VariantID,
			Delta:       -reservation.Qty,
			Reason:      ReasonSale,
			RefType:     strPtr("shop_order"),
			RefID:       &orderID,
			ActorUserID: optionalID(actorID),
			CreatedAt:   now,
		}).Error; err != nil {
			return err
		}
	}

	_, err = releaseReservationsTx(tx, orderID)
	return err
}

// returnStockTx mengembalikan stok pesanan yang dibatalkan setelah dibayar.
//
// Acuannya adalah pergerakan 'sale' pesanan ini, bukan qty yang tertulis di
// baris pesanan: yang dikembalikan adalah apa yang benar-benar pernah keluar.
// Bila tidak ada pergerakan 'sale', tidak ada yang dikembalikan — itulah yang
// membuat pembatalan pesanan yang belum dibayar tidak menambah stok.
//
// Pergerakan 'return' dengan ref yang sama menjadi penanda "sudah pernah
// dikembalikan", sehingga pembatalan kedua tidak menambah stok lagi. Penanda
// ini dipakai alih-alih menghitung selisih, karena buku besar stok bersifat
// append-only dan tidak menyimpan status pesanan.
func returnStockTx(tx *gorm.DB, orderID, actorID, reason string) error {
	sales := []*StockMovement{}
	if err := tx.
		Where("ref_type = ?", "shop_order").
		Where("ref_id = ?", orderID).
		Where("reason = ?", ReasonSale).
		Order("variant_id").
		Find(&sales).Error; err != nil {
		return err
	}
	if len(sales) == 0 {
		return nil
	}

	returned := []*StockMovement{}
	if err := tx.
		Where("ref_type = ?", "shop_order").
		Where("ref_id = ?", orderID).
		Where("reason = ?", ReasonReturn).
		Find(&returned).Error; err != nil {
		return err
	}
	if len(returned) > 0 {
		return nil
	}

	now := time.Now()
	for _, sale := range sales {
		qty := -sale.Delta
		if qty <= 0 {
			continue
		}
		if err := tx.Exec(
			`UPDATE product_variants
			    SET stock = stock + ?, updated_at = ?
			  WHERE id = ? AND deleted_at IS NULL`,
			qty, now, sale.VariantID,
		).Error; err != nil {
			return err
		}
		if err := tx.Create(&StockMovement{
			ID:          uuid.NewString(),
			VariantID:   sale.VariantID,
			Delta:       qty,
			Reason:      ReasonReturn,
			RefType:     strPtr("shop_order"),
			RefID:       &orderID,
			ActorUserID: optionalID(actorID),
			Note:        TrimOrNil(reason),
			CreatedAt:   now,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// AdjustStock mengubah stok secara manual dan mencatatnya di buku besar.
//
// Alasan diwajibkan pemanggil; ia ikut tersimpan di note supaya buku besar
// dapat menjelaskan dirinya sendiri tanpa harus membuka jejak audit.
func (r *repository) AdjustStock(c *gin.Context, variantID string, delta int, reason, note, actorID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		res := tx.Exec(
			`UPDATE product_variants
			    SET stock = stock + ?, updated_at = ?
			  WHERE id = ? AND stock + ? >= 0 AND deleted_at IS NULL`,
			delta, now, variantID, delta,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return invalid("penyesuaian membuat stok negatif, atau variannya tidak ada")
		}

		moveReason := ReasonAdjustment
		if delta > 0 {
			// Penambahan stok dari pengurus hampir selalu berarti barang
			// datang, bukan koreksi. Membedakannya membuat buku besar lebih
			// berguna tanpa menambah kolom apa pun.
			moveReason = ReasonRestock
		}

		return tx.Create(&StockMovement{
			ID:          uuid.NewString(),
			VariantID:   variantID,
			Delta:       delta,
			Reason:      moveReason,
			RefType:     strPtr("manual"),
			ActorUserID: optionalID(actorID),
			Note:        TrimOrNil(note),
			CreatedAt:   now,
		}).Error
	})
}

// ---------------------------------------------------------------------------
// Bantuan
// ---------------------------------------------------------------------------

func strPtr(value string) *string {
	return &value
}

func optionalID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}
