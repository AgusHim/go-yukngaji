// Package shop mengelola marketplace merchandise penjual tunggal: katalog
// produk dan varian, pesanan, reservasi stok, dan buku besar stok.
//
// Paket ini terpisah dari internal/order dengan sengaja. internal/order adalah
// modul tiket event: barisnya terikat pada satu event dan pembuatannya
// menerbitkan user_tickets. Memakainya kembali untuk merchandise akan membuat
// pesanan produk tampak sebagai tiket, dan itu justru hal yang dilarang oleh
// kriteria selesai Fase 4.
//
// INVARIAN STOK yang dipegang seluruh paket ini:
//
//	product_variants.stock adalah jumlah fisik yang ada di tangan.
//	stock_reservations adalah tahanan SEMENTARA untuk pesanan yang BELUM
//	dibayar. Konfirmasi pembayaran menurunkan stock dan melepas reservasinya
//	dalam satu transaksi. Pembatalan/refund mengembalikan stock.
//
// Akibatnya ketersediaan = stock - SUM(reservasi aktif & belum kedaluwarsa)
// bergerak benar di setiap transisi tanpa pengecualian, dan reservasi yang
// lewat batas waktunya berhenti dihitung tanpa perlu ditulis apa pun.
package shop

import (
	"time"

	"mainyuk/internal/audit"
	"mainyuk/internal/payment"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Status produk.
const (
	ProductDraft     = "draft"
	ProductPublished = "published"
	ProductArchived  = "archived"
)

// Status varian.
const (
	VariantActive   = "active"
	VariantInactive = "inactive"
)

// Status pembayaran pesanan. Sumbu pertama dari dua sumbu status pesanan.
const (
	PaymentPending  = "pending"
	PaymentPaid     = "paid"
	PaymentRejected = "rejected"
	PaymentRefunded = "refunded"
)

// Status pemenuhan pesanan. Sumbu kedua.
const (
	FulfillmentUnfulfilled    = "unfulfilled"
	FulfillmentReadyForPickup = "ready_for_pickup"
	FulfillmentShipped        = "shipped"
	FulfillmentCompleted      = "completed"
	FulfillmentCancelled      = "cancelled"
)

// Cara pemenuhan.
const (
	MethodPickup   = "pickup"
	MethodShipping = "shipping"
)

// Alasan pergerakan stok.
const (
	ReasonRestock    = "restock"
	ReasonAdjustment = "adjustment"
	ReasonSale       = "sale"
	ReasonReturn     = "return"
)

// Urutan katalog.
const (
	SortTerbaru    = "terbaru"
	SortTermurah   = "termurah"
	SortTermahal   = "termahal"
	DefaultPerPage = 20
)

// MaxQtyPerLine adalah batas jumlah satu varian dalam satu baris keranjang.
// Angkanya sama dengan CHECK constraint pada shop_order_items.qty.
const MaxQtyPerLine = 10

// ReservationTTL adalah berapa lama stok ditahan untuk pesanan yang belum
// dibayar. Lewat batas ini tahanannya berhenti dihitung dan pesanannya
// ditandai batal saat diakses.
//
// Nilainya cukup panjang untuk transfer bank manual, tetapi tidak selama itu
// sampai satu pembeli dapat menahan persediaan orang lain berhari-hari.
const ReservationTTL = 24 * time.Hour

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Product adalah satu barang yang dijual. Harga tidak ada di sini: harga milik
// varian, dan harga katalog dihitung dari varian termurah yang aktif.
type Product struct {
	ID            string     `json:"id" gorm:"column:id;primaryKey"`
	PublicID      string     `json:"public_id" gorm:"column:public_id"`
	Slug          string     `json:"slug" gorm:"column:slug"`
	Name          string     `json:"name" gorm:"column:name"`
	Description   *string    `json:"description" gorm:"column:description"`
	CoverImageURL *string    `json:"cover_image_url" gorm:"column:cover_image_url"`
	Status        string     `json:"status" gorm:"column:status"`
	CreatedBy     *string    `json:"-" gorm:"column:created_by"`
	CreatedAt     time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt     *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (Product) TableName() string {
	return "products"
}

// ProductImage adalah satu foto pada galeri produk.
type ProductImage struct {
	ID        string    `json:"id" gorm:"column:id;primaryKey"`
	ProductID string    `json:"-" gorm:"column:product_id"`
	ImageURL  string    `json:"image_url" gorm:"column:image_url"`
	Position  int       `json:"position" gorm:"column:position"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

func (ProductImage) TableName() string {
	return "product_images"
}

// ProductVariant adalah satu kombinasi ukuran/warna yang bisa dibeli.
//
// Size dan Color memakai string kosong untuk "tidak berlaku", bukan NULL:
// itu yang membuat unique index (product_id, size, color) benar-benar menjaga
// keunikan, karena Postgres memperlakukan NULL sebagai berbeda satu sama lain.
type ProductVariant struct {
	ID        string     `json:"id" gorm:"column:id;primaryKey"`
	PublicID  string     `json:"public_id" gorm:"column:public_id"`
	ProductID string     `json:"-" gorm:"column:product_id"`
	SKU       string     `json:"sku" gorm:"column:sku"`
	Size      string     `json:"size" gorm:"column:size"`
	Color     string     `json:"color" gorm:"column:color"`
	Price     int64      `json:"price" gorm:"column:price"`
	Stock     int        `json:"stock" gorm:"column:stock"`
	Status    string     `json:"status" gorm:"column:status"`
	Position  int        `json:"position" gorm:"column:position"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (ProductVariant) TableName() string {
	return "product_variants"
}

// ShopOrder adalah satu pesanan merchandise.
//
// Dua sumbu status sengaja dipisah: uang dan barang bergerak dengan kecepatan
// yang berbeda, dan menggabungkannya membuat "sudah dibayar tetapi belum
// dikirim" tidak dapat dinyatakan sama sekali.
type ShopOrder struct {
	ID                string     `json:"id" gorm:"column:id;primaryKey"`
	PublicID          string     `json:"public_id" gorm:"column:public_id"`
	UserID            string     `json:"-" gorm:"column:user_id"`
	Subtotal          int64      `json:"subtotal" gorm:"column:subtotal"`
	ShippingCost      int64      `json:"shipping_cost" gorm:"column:shipping_cost"`
	PaymentStatus     string     `json:"payment_status" gorm:"column:payment_status"`
	FulfillmentStatus string     `json:"fulfillment_status" gorm:"column:fulfillment_status"`
	FulfillmentMethod string     `json:"fulfillment_method" gorm:"column:fulfillment_method"`
	RecipientName     string     `json:"recipient_name" gorm:"column:recipient_name"`
	RecipientPhone    string     `json:"recipient_phone" gorm:"column:recipient_phone"`
	RecipientAddress  *string    `json:"recipient_address" gorm:"column:recipient_address"`
	TrackingNumber    *string    `json:"tracking_number" gorm:"column:tracking_number"`
	PaymentMethodID   *string    `json:"-" gorm:"column:payment_method_id"`
	PaidAmount        *int64     `json:"paid_amount" gorm:"column:paid_amount"`
	PaymentReference  *string    `json:"payment_reference" gorm:"column:payment_reference"`
	ProofURL          *string    `json:"proof_url" gorm:"column:proof_url"`
	ConfirmedBy       *string    `json:"-" gorm:"column:confirmed_by"`
	ConfirmedAt       *time.Time `json:"confirmed_at" gorm:"column:confirmed_at"`
	DecisionReason    *string    `json:"decision_reason" gorm:"column:decision_reason"`
	// RewardedXP adalah snapshot XP yang benar-benar diberikan, dipakai untuk
	// membalik tepat sebesar yang pernah diberikan saat pesanan di-refund —
	// bukan sebesar aturan XP saat refund terjadi.
	RewardedXP int `json:"rewarded_xp" gorm:"column:rewarded_xp"`
	// ExpiresAt adalah batas tahanan stok. Kedaluwarsanya tidak dijalankan
	// pekerja latar: pesanan ditandai batal saat diakses.
	ExpiresAt time.Time  `json:"expires_at" gorm:"column:expires_at"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt *time.Time `json:"-" gorm:"column:deleted_at"`

	// Total dihitung, tidak disimpan — pola yang sama dengan order.Order.
	Total int64            `json:"total" gorm:"-"`
	Items []*ShopOrderItem `json:"items,omitempty" gorm:"-"`
}

func (ShopOrder) TableName() string {
	return "shop_orders"
}

// ComputeTotal mengisi Total dari subtotal + ongkir.
func (o *ShopOrder) ComputeTotal() {
	if o == nil {
		return
	}
	o.Total = o.Subtotal + o.ShippingCost
}

// AfterFind menjaga Total tetap benar di setiap jalur baca, supaya tidak ada
// respons yang menampilkan total yang berbeda dari komponennya.
func (o *ShopOrder) AfterFind(_ *gorm.DB) error {
	o.ComputeTotal()
	return nil
}

// ShopOrderItem adalah satu baris pesanan.
//
// ProductName, VariantLabel, dan UnitPrice adalah SNAPSHOT yang diambil server
// saat checkout. Nama produk boleh berubah dan harganya boleh naik sesudahnya;
// baris ini tetap menunjukkan apa yang benar-benar disepakati pembeli.
type ShopOrderItem struct {
	ID           string    `json:"id" gorm:"column:id;primaryKey"`
	OrderID      string    `json:"-" gorm:"column:order_id"`
	ProductID    *string   `json:"-" gorm:"column:product_id"`
	VariantID    *string   `json:"-" gorm:"column:variant_id"`
	ProductName  string    `json:"product_name" gorm:"column:product_name"`
	VariantLabel string    `json:"variant_label" gorm:"column:variant_label"`
	UnitPrice    int64     `json:"unit_price" gorm:"column:unit_price"`
	Qty          int       `json:"qty" gorm:"column:qty"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at"`
}

func (ShopOrderItem) TableName() string {
	return "shop_order_items"
}

// LineTotal adalah harga baris ini. Dihitung, tidak disimpan, supaya tidak ada
// angka yang bisa menyimpang dari unit_price x qty.
func (i *ShopOrderItem) LineTotal() int64 {
	if i == nil {
		return 0
	}
	return i.UnitPrice * int64(i.Qty)
}

// StockReservation adalah tahanan stok satu varian untuk satu pesanan.
type StockReservation struct {
	ID         string     `json:"id" gorm:"column:id;primaryKey"`
	VariantID  string     `json:"-" gorm:"column:variant_id"`
	OrderID    string     `json:"-" gorm:"column:order_id"`
	Qty        int        `json:"qty" gorm:"column:qty"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"column:expires_at"`
	ReleasedAt *time.Time `json:"released_at" gorm:"column:released_at"`
	CreatedAt  time.Time  `json:"created_at" gorm:"column:created_at"`
}

func (StockReservation) TableName() string {
	return "stock_reservations"
}

// StockMovement adalah satu baris buku besar stok. Append-only: koreksi berupa
// baris baru, bukan perubahan atau penghapusan.
type StockMovement struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey"`
	VariantID   string    `json:"-" gorm:"column:variant_id"`
	Delta       int       `json:"delta" gorm:"column:delta"`
	Reason      string    `json:"reason" gorm:"column:reason"`
	RefType     *string   `json:"ref_type" gorm:"column:ref_type"`
	RefID       *string   `json:"ref_id" gorm:"column:ref_id"`
	ActorUserID *string   `json:"-" gorm:"column:actor_user_id"`
	Note        *string   `json:"note" gorm:"column:note"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
}

func (StockMovement) TableName() string {
	return "stock_movements"
}

// ---------------------------------------------------------------------------
// DTO
// ---------------------------------------------------------------------------

// CartLine adalah satu baris keranjang yang dikirim klien.
//
// Tipe ini SENGAJA tidak punya field harga sama sekali. Itulah penegakan
// "validasi cart/harga server" pada tingkat tipe: tidak ada harga yang bisa
// dikirim klien, jadi tidak ada pemeriksaan yang bisa terlupa. Nama produk,
// label varian, dan harga selalu diambil server dari barisnya sendiri.
type CartLine struct {
	VariantID string `json:"variant_id"`
	Qty       int    `json:"qty"`
}

// CreateOrderRequest adalah badan checkout.
type CreateOrderRequest struct {
	Items             []CartLine `json:"items"`
	FulfillmentMethod string     `json:"fulfillment_method"`
	RecipientName     string     `json:"recipient_name"`
	RecipientPhone    string     `json:"recipient_phone"`
	RecipientAddress  string     `json:"recipient_address"`
	PaymentMethodID   *string    `json:"payment_method_id"`
}

// VariantView adalah varian yang tampil di katalog, sudah dilengkapi
// ketersediaan yang benar-benar bisa dijual.
type VariantView struct {
	ProductVariant
	Label     string `json:"label"`
	Available int    `json:"available"`
}

// ProductView adalah produk yang tampil di katalog publik.
type ProductView struct {
	Product
	Images    []*ProductImage `json:"images"`
	Variants  []*VariantView  `json:"variants"`
	MinPrice  int64           `json:"min_price"`
	MaxPrice  int64           `json:"max_price"`
	Available int             `json:"available"`
}

// OrderView adalah pesanan lengkap dengan baris dan instruksi pembayarannya.
type OrderView struct {
	ShopOrder
	Charge *payment.Charge `json:"charge,omitempty"`
}

// AdminOrderView adalah pesanan pada antrean pengurus. Sama dengan OrderView;
// dipisah sebagai tipe tersendiri supaya penambahan field khusus pengurus kelak
// tidak bocor ke respons anggota.
type AdminOrderView = OrderView

// CreateProductRequest adalah badan pembuatan produk.
type CreateProductRequest struct {
	Name          string   `json:"name"`
	Slug          string   `json:"slug"`
	Description   string   `json:"description"`
	CoverImageURL string   `json:"cover_image_url"`
	ImageURLs     []string `json:"image_urls"`
}

// UpdateProductRequest adalah badan pembaruan produk. Seluruh field opsional;
// yang nil berarti "jangan diubah".
type UpdateProductRequest struct {
	Name          *string   `json:"name"`
	Slug          *string   `json:"slug"`
	Description   *string   `json:"description"`
	CoverImageURL *string   `json:"cover_image_url"`
	ImageURLs     *[]string `json:"image_urls"`
}

// SetProductStatusRequest adalah badan perubahan status publikasi produk.
type SetProductStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// VariantRequest adalah badan pembuatan/pembaruan varian.
type VariantRequest struct {
	SKU      string `json:"sku"`
	Size     string `json:"size"`
	Color    string `json:"color"`
	Price    int64  `json:"price"`
	Stock    int    `json:"stock"`
	Status   string `json:"status"`
	Position int    `json:"position"`
}

// AdjustStockRequest adalah badan penyesuaian stok manual oleh pengurus.
//
// Delta ber-tanda: positif menambah, negatif mengurangi. Alasan wajib diisi
// karena inilah satu-satunya jalur yang boleh mengubah stok tanpa penjualan.
type AdjustStockRequest struct {
	Delta  int    `json:"delta"`
	Reason string `json:"reason"`
	Note   string `json:"note"`
}

// SubmitProofRequest adalah badan penyerahan bukti transfer oleh pembeli.
type SubmitProofRequest struct {
	ProofURL         string `json:"proof_url"`
	PaymentReference string `json:"payment_reference"`
	PaymentMethodID  string `json:"payment_method_id"`
}

// ConfirmOrderRequest adalah badan konfirmasi pembayaran oleh pengurus.
type ConfirmOrderRequest struct {
	PaidAmount       int64  `json:"paid_amount"`
	PaymentReference string `json:"payment_reference"`
	ProofURL         string `json:"proof_url"`
	Reason           string `json:"reason"`
}

// DecideOrderRequest adalah badan keputusan yang hanya butuh alasan.
type DecideOrderRequest struct {
	Reason string `json:"reason"`
}

// ShippingRequest adalah badan pengisian ongkir dan resi.
type ShippingRequest struct {
	ShippingCost   int64  `json:"shipping_cost"`
	TrackingNumber string `json:"tracking_number"`
}

// FulfillOrderRequest adalah badan perpindahan status pemenuhan.
type FulfillOrderRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// OrderFilter adalah penyaring antrean pesanan pengurus.
type OrderFilter struct {
	PaymentStatus     string
	FulfillmentStatus string
}

// ---------------------------------------------------------------------------
// Seam
// ---------------------------------------------------------------------------

// RewardGranter adalah seam sempit ke gamifikasi, dipenuhi oleh
// gamification.Service.
//
// Bentuknya sengaja cermin dari pasangan donasi: XP diberikan saat pembayaran
// terkonfirmasi dan dibalik tepat sebesar snapshot saat refund. Kegagalan
// memberi XP tidak boleh menggagalkan pesanan yang sudah sah.
type RewardGranter interface {
	GrantShopOrderReward(ctx *gin.Context, userID, orderID string) (int, error)
	ReverseShopOrderReward(ctx *gin.Context, userID, orderID string, xp int, reason string) error
}

// ---------------------------------------------------------------------------
// Kontrak lapisan
// ---------------------------------------------------------------------------

type Repository interface {
	// Katalog
	ListProducts(ctx *gin.Context, status, sort string, limit, offset int) ([]*Product, error)
	FindProductBySlug(ctx *gin.Context, slug string) (*Product, error)
	FindProductByID(ctx *gin.Context, id string) (*Product, error)
	ListProductsByIDs(ctx *gin.Context, ids []string) ([]*Product, error)
	CreateProduct(ctx *gin.Context, product *Product, images []*ProductImage) error
	UpdateProduct(ctx *gin.Context, id string, fields map[string]interface{}, images *[]*ProductImage) error
	SetProductStatus(ctx *gin.Context, id, from, to string) (bool, error)
	SoftDeleteProduct(ctx *gin.Context, id string) (bool, error)
	ListImagesByProducts(ctx *gin.Context, productIDs []string) (map[string][]*ProductImage, error)

	// Varian
	ListVariantsByProduct(ctx *gin.Context, productID string, activeOnly bool) ([]*ProductVariant, error)
	ListVariantsByProducts(ctx *gin.Context, productIDs []string) ([]*ProductVariant, error)
	ListVariantsByIDs(ctx *gin.Context, ids []string) ([]*ProductVariant, error)
	FindVariantByID(ctx *gin.Context, id string) (*ProductVariant, error)
	CreateVariant(ctx *gin.Context, variant *ProductVariant) error
	UpdateVariant(ctx *gin.Context, id string, fields map[string]interface{}) error
	SoftDeleteVariant(ctx *gin.Context, id string) (bool, error)

	// Ketersediaan
	ReservedQty(ctx *gin.Context, variantIDs []string, now time.Time) (map[string]int, error)

	// Pesanan
	ListOrdersByUser(ctx *gin.Context, userID string, limit, offset int) ([]*ShopOrder, error)
	ListOrders(ctx *gin.Context, filter OrderFilter, limit, offset int) ([]*ShopOrder, error)
	FindOrderByPublicID(ctx *gin.Context, publicID string) (*ShopOrder, error)
	FindOrderByID(ctx *gin.Context, id string) (*ShopOrder, error)
	ListOrderItems(ctx *gin.Context, orderIDs []string) (map[string][]*ShopOrderItem, error)
	CreateOrder(ctx *gin.Context, order *ShopOrder, items []*ShopOrderItem, lines []CartLine) error
	UpdateOrderFields(ctx *gin.Context, id, expectedPaymentStatus string, fields map[string]interface{}) (bool, error)
	TransitionFulfillment(ctx *gin.Context, id, from, to string, fields map[string]interface{}) (bool, error)
	SetOrderReward(ctx *gin.Context, id string, xp int) error
	OrderItemsWithVariant(ctx *gin.Context, orderID string) ([]*ShopOrderItem, error)

	// Perpindahan yang menyentuh stok. Ketiganya menggabungkan perubahan status
	// dan penyesuaian stok dalam SATU transaksi, karena status yang berpindah
	// tanpa stok yang ikut bergerak (atau sebaliknya) adalah keadaan yang tidak
	// boleh pernah terlihat oleh siapa pun.
	ConfirmPayment(ctx *gin.Context, id string, fields map[string]interface{}, actorID string) (bool, error)
	RejectPayment(ctx *gin.Context, id string, fields map[string]interface{}) (bool, error)
	RefundPayment(ctx *gin.Context, id string, fields map[string]interface{}, actorID string) (bool, error)
	CancelSettlement(ctx *gin.Context, id, fromFulfillment string, fields map[string]interface{}, actorID, reason string) (bool, error)

	// Stok
	AdjustStock(ctx *gin.Context, variantID string, delta int, reason, note, actorID string) error
	WriteAudit(ctx *gin.Context, entry *audit.Log) error
	ListAudit(ctx *gin.Context, entityTypes []string, limit, offset int) ([]*audit.Log, error)
}

type Service interface {
	// Katalog
	ListProducts(ctx *gin.Context, sort string, page, perPage int) ([]*ProductView, bool, error)
	ShowProduct(ctx *gin.Context, slug string) (*ProductView, error)

	// Pesanan anggota
	Checkout(ctx *gin.Context, req *CreateOrderRequest) (*OrderView, error)
	MyOrders(ctx *gin.Context, page, perPage int) ([]*OrderView, bool, error)
	MyOrder(ctx *gin.Context, publicID string) (*OrderView, error)
	SubmitProof(ctx *gin.Context, publicID string, req *SubmitProofRequest) (*OrderView, error)
	CancelOrder(ctx *gin.Context, publicID string) (*OrderView, error)

	// Katalog pengurus
	AdminProducts(ctx *gin.Context, status string, page, perPage int) ([]*ProductView, bool, error)
	CreateProduct(ctx *gin.Context, req *CreateProductRequest) (*ProductView, error)
	UpdateProduct(ctx *gin.Context, id string, req *UpdateProductRequest) (*ProductView, error)
	SetProductStatus(ctx *gin.Context, id string, req *SetProductStatusRequest) (*ProductView, error)
	DeleteProduct(ctx *gin.Context, id string) error
	CreateVariant(ctx *gin.Context, productID string, req *VariantRequest) (*ProductVariant, error)
	UpdateVariant(ctx *gin.Context, id string, req *VariantRequest) (*ProductVariant, error)
	DeleteVariant(ctx *gin.Context, id string) error
	AdjustStock(ctx *gin.Context, variantID string, req *AdjustStockRequest) (*ProductVariant, error)

	// Pesanan pengurus
	AdminOrders(ctx *gin.Context, filter OrderFilter, page, perPage int) ([]*OrderView, bool, error)
	ConfirmOrder(ctx *gin.Context, id string, req *ConfirmOrderRequest) (*OrderView, error)
	RejectOrder(ctx *gin.Context, id string, req *DecideOrderRequest) (*OrderView, error)
	RefundOrder(ctx *gin.Context, id string, req *DecideOrderRequest) (*OrderView, error)
	FulfillOrder(ctx *gin.Context, id string, req *FulfillOrderRequest) (*OrderView, error)
	SetShipping(ctx *gin.Context, id string, req *ShippingRequest) (*OrderView, error)
	AuditLogs(ctx *gin.Context, page, perPage int) ([]*audit.Log, bool, error)
}

type Handler interface {
	ListProducts(ctx *gin.Context)
	ShowProduct(ctx *gin.Context)

	Checkout(ctx *gin.Context)
	MyOrders(ctx *gin.Context)
	MyOrder(ctx *gin.Context)
	SubmitProof(ctx *gin.Context)
	CancelOrder(ctx *gin.Context)

	AdminProducts(ctx *gin.Context)
	CreateProduct(ctx *gin.Context)
	UpdateProduct(ctx *gin.Context)
	SetProductStatus(ctx *gin.Context)
	DeleteProduct(ctx *gin.Context)
	CreateVariant(ctx *gin.Context)
	UpdateVariant(ctx *gin.Context)
	DeleteVariant(ctx *gin.Context)
	AdjustStock(ctx *gin.Context)

	AdminOrders(ctx *gin.Context)
	ConfirmOrder(ctx *gin.Context)
	RejectOrder(ctx *gin.Context)
	RefundOrder(ctx *gin.Context)
	FulfillOrder(ctx *gin.Context)
	SetShipping(ctx *gin.Context)
	AuditLogs(ctx *gin.Context)
}
