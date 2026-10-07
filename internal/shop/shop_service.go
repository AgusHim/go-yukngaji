package shop

import (
	"errors"
	"log"
	"strings"
	"time"

	"mainyuk/internal/apperr"
	"mainyuk/internal/audit"
	"mainyuk/internal/gamification"
	"mainyuk/internal/payment"
	"mainyuk/internal/ratelimit"
	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Jenis entitas pada jejak audit toko.
//
// Ketiganya dipakai sebagai penyaring saat membaca riwayat toko: tabel audit
// dipakai bersama donasi dan moderasi, dan setiap modul memilih barisnya lewat
// jenis entitas yang memang hanya ia tulis.
const (
	entityProduct = "product"
	entityVariant = "product_variant"
	entityOrder   = "shop_order"
)

// service menyatukan aturan toko. Semua jalur tulis lewat sini supaya validasi,
// transisi status, stok, XP, dan audit tidak bisa dilewati satu pun.
type service struct {
	Repository
	// Rewards boleh nil: toko tetap berjalan tanpa pemberian XP.
	Rewards RewardGranter
	// Provider boleh nil: pesanan tetap dibuat, hanya instruksi transfernya
	// yang tidak tampil.
	Provider payment.Provider
}

func NewService(repository Repository, rewards RewardGranter, provider payment.Provider) Service {
	return &service{Repository: repository, Rewards: rewards, Provider: provider}
}

// ---------------------------------------------------------------------------
// Katalog publik
// ---------------------------------------------------------------------------

func (s *service) ListProducts(c *gin.Context, sort string, page, perPage int) ([]*ProductView, bool, error) {
	page, perPage = gamification.NormalizePagination(page, perPage)

	products, err := s.Repository.ListProducts(c, ProductPublished, normalizeSort(sort), perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(products) > perPage
	if hasMore {
		products = products[:perPage]
	}

	views, err := s.buildViews(c, products, true)
	if err != nil {
		return nil, false, err
	}
	return views, hasMore, nil
}

func (s *service) ShowProduct(c *gin.Context, slug string) (*ProductView, error) {
	product, err := s.findProductBySlug(c, slug)
	if err != nil {
		return nil, err
	}
	// Produk yang belum terbit tidak boleh bocor ke publik hanya karena
	// tautannya ditebak. Bagi publik ia tidak ada, bukan "ada tapi terlarang".
	if product.Status != ProductPublished {
		return nil, ErrProductNotFound
	}

	views, err := s.buildViews(c, []*Product{product}, true)
	if err != nil {
		return nil, err
	}
	return views[0], nil
}

// buildViews melengkapi produk dengan galeri, varian, dan ketersediaannya.
//
// Pembacaannya dikumpulkan per kumpulan produk, bukan per produk: satu halaman
// katalog berisi dua puluh produk, dan mengerjakannya satu per satu berarti
// tiga kali dua puluh query untuk halaman yang sama.
//
// activeOnly memisahkan dua pemakaian yang memang berbeda: pembeli hanya boleh
// melihat varian yang dijual, sedangkan pengurus harus melihat semuanya —
// termasuk yang sedang dinonaktifkan — supaya dapat mengaktifkannya kembali.
func (s *service) buildViews(c *gin.Context, products []*Product, activeOnly bool) ([]*ProductView, error) {
	views := make([]*ProductView, 0, len(products))
	if len(products) == 0 {
		return views, nil
	}

	productIDs := make([]string, 0, len(products))
	for _, product := range products {
		productIDs = append(productIDs, product.ID)
	}

	variants, err := s.Repository.ListVariantsByProducts(c, productIDs)
	if err != nil {
		return nil, err
	}
	images, err := s.Repository.ListImagesByProducts(c, productIDs)
	if err != nil {
		return nil, err
	}

	byProduct := map[string][]*ProductVariant{}
	variantIDs := make([]string, 0, len(variants))
	for _, variant := range variants {
		if activeOnly && variant.Status != VariantActive {
			continue
		}
		byProduct[variant.ProductID] = append(byProduct[variant.ProductID], variant)
		variantIDs = append(variantIDs, variant.ID)
	}

	reserved, err := s.Repository.ReservedQty(c, variantIDs, time.Now())
	if err != nil {
		return nil, err
	}

	for _, product := range products {
		views = append(views, buildProductView(product, byProduct[product.ID], images[product.ID], reserved))
	}
	return views, nil
}

// buildProductView adalah perakitan murni: seluruh bahannya sudah dibaca.
func buildProductView(product *Product, variants []*ProductVariant, images []*ProductImage, reserved map[string]int) *ProductView {
	view := &ProductView{
		Product:  *product,
		Images:   images,
		Variants: make([]*VariantView, 0, len(variants)),
	}
	if view.Images == nil {
		view.Images = []*ProductImage{}
	}

	available := 0
	first := true
	for _, variant := range variants {
		sisa := AvailableStock(variant.Stock, reserved[variant.ID])
		view.Variants = append(view.Variants, &VariantView{
			ProductVariant: *variant,
			Label:          VariantLabel(variant.Size, variant.Color),
			Available:      sisa,
		})
		if first || variant.Price < view.MinPrice {
			view.MinPrice = variant.Price
		}
		if first || variant.Price > view.MaxPrice {
			view.MaxPrice = variant.Price
		}
		first = false
		available += sisa
	}
	view.Available = available
	return view
}

// ---------------------------------------------------------------------------
// Pesanan anggota
// ---------------------------------------------------------------------------

// Checkout membuat pesanan beserta tahanan stoknya.
//
// Seluruh keterangan baris diambil server dari varian yang dipilih: nama
// produk, label varian, dan harganya. Klien hanya mengirim id varian dan
// jumlahnya — bentuk CartLine memang tidak punya tempat untuk harga.
//
// Ketersediaan tidak diperiksa di sini. Pemeriksaan yang sebenarnya terjadi di
// dalam transaksi repository, di bawah kunci per varian; memeriksanya lebih
// dulu di sini hanya akan menghasilkan jawaban yang sudah basi saat dipakai.
func (s *service) Checkout(c *gin.Context, req *CreateOrderRequest) (*OrderView, error) {
	currentUser, ok := user.FromContext(c)
	if !ok || currentUser == nil {
		return nil, apperr.ErrUnauthorized
	}
	// Checkout menahan stok fisik. Tanpa batas, satu akun dapat mengunci
	// seluruh persediaan hanya dengan membuat pesanan yang tidak pernah
	// dibayar.
	if !ratelimit.ShopOrder.Allow(ratelimit.Key(c, currentUser.ID)) {
		return nil, ratelimit.ErrTooManyRequests
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	lines := NormalizeCart(req.Items)
	if len(lines) == 0 {
		return nil, invalid("keranjang kosong")
	}

	method := strings.TrimSpace(req.FulfillmentMethod)
	if err := ValidateFulfillmentInput(method, req.RecipientName, req.RecipientPhone, req.RecipientAddress); err != nil {
		return nil, err
	}

	variantIDs := make([]string, 0, len(lines))
	for _, line := range lines {
		variantIDs = append(variantIDs, line.VariantID)
	}
	variants, err := s.Repository.ListVariantsByIDs(c, variantIDs)
	if err != nil {
		return nil, err
	}

	byID := map[string]*ProductVariant{}
	productIDs := make([]string, 0, len(variants))
	for _, variant := range variants {
		if variant.Status != VariantActive {
			continue
		}
		byID[variant.ID] = variant
		productIDs = append(productIDs, variant.ProductID)
	}
	for _, line := range lines {
		if _, found := byID[line.VariantID]; !found {
			return nil, invalid("varian yang dipilih tidak lagi dijual")
		}
	}

	products, err := s.Repository.ListProductsByIDs(c, productIDs)
	if err != nil {
		return nil, err
	}
	productName := map[string]string{}
	published := map[string]bool{}
	for _, product := range products {
		productName[product.ID] = product.Name
		published[product.ID] = product.Status == ProductPublished
	}
	// Varian yang masih aktif tetapi produknya belum terbit atau sudah
	// dihapus tidak boleh dibeli. Tanpa pemeriksaan ini, pembeli yang menyimpan
	// id varian lama dapat menembus penarikan produk dari peredaran.
	for _, variant := range byID {
		if !published[variant.ProductID] {
			return nil, invalid("produk untuk varian yang dipilih tidak lagi dijual")
		}
	}

	now := time.Now()
	order := &ShopOrder{
		ID:                uuid.NewString(),
		PublicID:          newPublicID(),
		UserID:            currentUser.ID,
		PaymentStatus:     PaymentPending,
		FulfillmentStatus: FulfillmentUnfulfilled,
		FulfillmentMethod: method,
		RecipientName:     strings.TrimSpace(req.RecipientName),
		RecipientPhone:    strings.TrimSpace(req.RecipientPhone),
		RecipientAddress:  TrimOrNil(req.RecipientAddress),
		PaymentMethodID:   trimPtr(req.PaymentMethodID),
		ExpiresAt:         OrderExpiry(now),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	items := make([]*ShopOrderItem, 0, len(lines))
	for _, line := range lines {
		variant := byID[line.VariantID]
		items = append(items, &ShopOrderItem{
			ID:           uuid.NewString(),
			ProductID:    strPtr(variant.ProductID),
			VariantID:    strPtr(variant.ID),
			ProductName:  productName[variant.ProductID],
			VariantLabel: VariantLabel(variant.Size, variant.Color),
			// UnitPrice dan Subtotal diisi repository di dalam kunci, dari
			// harga varian pada saat itu — bukan dari harga yang dibaca di sini.
			Qty:       line.Qty,
			CreatedAt: now,
		})
	}

	if err := s.Repository.CreateOrder(c, order, items, lines); err != nil {
		switch {
		case errors.Is(err, errStockShortage):
			return nil, ErrStockShortage
		case errors.Is(err, errVariantGone):
			return nil, ErrVariantNotFound
		}
		return nil, err
	}

	return s.orderViewByID(c, order.ID, true)
}

func (s *service) MyOrders(c *gin.Context, page, perPage int) ([]*OrderView, bool, error) {
	currentUser, ok := user.FromContext(c)
	if !ok || currentUser == nil {
		return nil, false, apperr.ErrUnauthorized
	}
	page, perPage = gamification.NormalizePagination(page, perPage)

	orders, err := s.Repository.ListOrdersByUser(c, currentUser.ID, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(orders) > perPage
	if hasMore {
		orders = orders[:perPage]
	}

	// Kedaluwarsa diselesaikan lebih dulu, baru barisnya dibaca ulang, supaya
	// daftar yang tampil tidak memuat pesanan yang sebenarnya sudah batal.
	settled := make([]*ShopOrder, 0, len(orders))
	for _, order := range orders {
		updated, err := s.settleExpiry(c, order)
		if err != nil {
			return nil, false, err
		}
		settled = append(settled, updated)
	}

	orderIDs := make([]string, 0, len(settled))
	for _, order := range settled {
		orderIDs = append(orderIDs, order.ID)
	}
	items, err := s.Repository.ListOrderItems(c, orderIDs)
	if err != nil {
		return nil, false, err
	}

	// Daftar tidak menyertakan instruksi transfer: satu halaman berisi dua
	// puluh pesanan, dan setiap instruksi menuntut satu pembacaan metode
	// pembayaran. Instruksinya ada di halaman rincian pesanannya.
	views := make([]*OrderView, 0, len(settled))
	for _, order := range settled {
		order.Items = items[order.ID]
		views = append(views, &OrderView{ShopOrder: *order})
	}
	return views, hasMore, nil
}

func (s *service) MyOrder(c *gin.Context, publicID string) (*OrderView, error) {
	order, err := s.ownedOrder(c, publicID)
	if err != nil {
		return nil, err
	}
	return s.orderView(c, order, true)
}

// SubmitProof mencatat bukti transfer yang diserahkan pembeli.
//
// Status pembayaran tidak berpindah di sini: yang menilai buktinya adalah
// pengurus. Karena itu penulisannya dijaga oleh status 'pending' — bukti tidak
// boleh menempel pada pesanan yang sudah diputuskan di sela antara pembacaan
// dan penulisan.
func (s *service) SubmitProof(c *gin.Context, publicID string, req *SubmitProofRequest) (*OrderView, error) {
	currentUser, ok := user.FromContext(c)
	if !ok || currentUser == nil {
		return nil, apperr.ErrUnauthorized
	}
	if !ratelimit.ShopProof.Allow(ratelimit.Key(c, currentUser.ID)) {
		return nil, ratelimit.ErrTooManyRequests
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	order, err := s.ownedOrder(c, publicID)
	if err != nil {
		return nil, err
	}
	if order.PaymentStatus != PaymentPending {
		return nil, ErrOrderAlreadyDecided
	}

	proofURL := strings.TrimSpace(req.ProofURL)
	if proofURL == "" {
		return nil, invalid("tautan bukti transfer wajib diisi")
	}

	fields := map[string]interface{}{
		"proof_url":         proofURL,
		"payment_reference": TrimOrNil(req.PaymentReference),
	}
	if methodID := strings.TrimSpace(req.PaymentMethodID); methodID != "" {
		fields["payment_method_id"] = methodID
	}

	changed, err := s.Repository.UpdateOrderFields(c, order.ID, PaymentPending, fields)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, ErrOrderAlreadyDecided
	}
	return s.orderViewByID(c, order.ID, true)
}

// CancelOrder membatalkan pesanan pembeli selama uangnya belum dikonfirmasi.
//
// Pesanan yang sudah dibayar tidak dapat dibatalkan sendiri: uang yang sudah
// masuk harus diputuskan pengurus lewat refund, bukan dihilangkan oleh pembeli.
func (s *service) CancelOrder(c *gin.Context, publicID string) (*OrderView, error) {
	currentUser, ok := user.FromContext(c)
	if !ok || currentUser == nil {
		return nil, apperr.ErrUnauthorized
	}

	order, err := s.ownedOrder(c, publicID)
	if err != nil {
		return nil, err
	}
	if order.FulfillmentStatus == FulfillmentCancelled {
		// Sudah batal, mungkin oleh kedaluwarsa. Membatalkan lagi bukan
		// kegagalan.
		return s.orderView(c, order, true)
	}
	if order.PaymentStatus != PaymentPending {
		return nil, invalid("pesanan yang sudah dibayar tidak dapat dibatalkan sendiri")
	}
	if !IsCancellableFulfillment(order.FulfillmentStatus) {
		return nil, ErrInvalidTransition
	}

	reason := "dibatalkan pembeli"
	changed, err := s.Repository.CancelSettlement(c, order.ID, order.FulfillmentStatus,
		map[string]interface{}{"decision_reason": reason}, currentUser.ID, reason)
	if err != nil {
		return nil, err
	}
	if changed {
		s.writeAudit(c, "shop_order.cancel", entityOrder, order.ID, &reason, nil)
	}
	return s.orderViewByID(c, order.ID, true)
}

// ---------------------------------------------------------------------------
// Katalog pengurus
// ---------------------------------------------------------------------------

func (s *service) AdminProducts(c *gin.Context, status string, page, perPage int) ([]*ProductView, bool, error) {
	status = strings.TrimSpace(status)
	if status != "" && !isProductStatus(status) {
		return nil, false, invalid("status produk tidak dikenal: %s", status)
	}

	page, perPage = gamification.NormalizePagination(page, perPage)
	products, err := s.Repository.ListProducts(c, status, SortTerbaru, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(products) > perPage
	if hasMore {
		products = products[:perPage]
	}

	// Pengurus melihat seluruh varian, termasuk yang dinonaktifkan.
	views, err := s.buildViews(c, products, false)
	if err != nil {
		return nil, false, err
	}
	return views, hasMore, nil
}

func (s *service) CreateProduct(c *gin.Context, req *CreateProductRequest) (*ProductView, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	name := strings.TrimSpace(req.Name)
	slug := NormalizeSlug(req.Slug)
	if slug == "" {
		// Slug yang tidak diisi diturunkan dari nama. Produk tanpa slug tidak
		// dapat ditautkan, dan memaksanya diisi hanya menambah satu langkah
		// yang dapat dilakukan server dengan benar.
		slug = NormalizeSlug(name)
	}
	imageURLs := NormalizeImageURLs(req.ImageURLs)
	if err := ValidateProduct(name, slug, imageURLs); err != nil {
		return nil, err
	}

	now := time.Now()
	product := &Product{
		ID:            uuid.NewString(),
		PublicID:      newPublicID(),
		Slug:          slug,
		Name:          name,
		Description:   TrimOrNil(req.Description),
		CoverImageURL: TrimOrNil(req.CoverImageURL),
		Status:        ProductDraft,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if actor := actorID(c); actor != "" {
		product.CreatedBy = &actor
	}

	images := make([]*ProductImage, 0, len(imageURLs))
	for position, url := range imageURLs {
		images = append(images, &ProductImage{
			ID:        uuid.NewString(),
			ImageURL:  url,
			Position:  position,
			CreatedAt: now,
		})
	}

	if err := s.Repository.CreateProduct(c, product, images); err != nil {
		if isUniqueViolation(err) {
			return nil, invalid("slug %q sudah dipakai produk lain", slug)
		}
		return nil, err
	}

	s.writeAudit(c, "product.create", entityProduct, product.ID, nil, map[string]any{
		"slug": slug,
		"name": name,
	})
	return s.showAdminProduct(c, product.ID)
}

func (s *service) UpdateProduct(c *gin.Context, id string, req *UpdateProductRequest) (*ProductView, error) {
	product, err := s.findProductByID(c, id)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	// Yang divalidasi adalah hasil akhirnya, bukan tiap field terpisah: nama
	// dan slug hanya bermakna bersama-sama, dan mengubah salah satunya tetap
	// harus menghasilkan produk yang sah.
	name := product.Name
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	slug := product.Slug
	if req.Slug != nil {
		slug = NormalizeSlug(*req.Slug)
	}
	var imageURLs []string
	if req.ImageURLs != nil {
		imageURLs = NormalizeImageURLs(*req.ImageURLs)
	}
	if err := ValidateProduct(name, slug, imageURLs); err != nil {
		return nil, err
	}

	now := time.Now()
	fields := map[string]interface{}{"updated_at": now}
	if req.Name != nil {
		fields["name"] = name
	}
	if req.Slug != nil {
		fields["slug"] = slug
	}
	if req.Description != nil {
		fields["description"] = TrimOrNil(*req.Description)
	}
	if req.CoverImageURL != nil {
		fields["cover_image_url"] = TrimOrNil(*req.CoverImageURL)
	}

	var images *[]*ProductImage
	if req.ImageURLs != nil {
		built := make([]*ProductImage, 0, len(imageURLs))
		for position, url := range imageURLs {
			built = append(built, &ProductImage{
				ID:        uuid.NewString(),
				ProductID: product.ID,
				ImageURL:  url,
				Position:  position,
				CreatedAt: now,
			})
		}
		images = &built
	}

	if err := s.Repository.UpdateProduct(c, product.ID, fields, images); err != nil {
		if isUniqueViolation(err) {
			return nil, invalid("slug %q sudah dipakai produk lain", slug)
		}
		return nil, err
	}

	s.writeAudit(c, "product.update", entityProduct, product.ID, nil, map[string]any{"slug": slug})
	return s.showAdminProduct(c, product.ID)
}

func (s *service) SetProductStatus(c *gin.Context, id string, req *SetProductStatusRequest) (*ProductView, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	product, err := s.findProductByID(c, id)
	if err != nil {
		return nil, err
	}

	target := strings.TrimSpace(req.Status)
	if !CanTransitionProduct(product.Status, target) {
		return nil, ErrInvalidTransition
	}

	changed, err := s.Repository.SetProductStatus(c, product.ID, product.Status, target)
	if err != nil {
		return nil, err
	}
	if changed {
		s.writeAudit(c, "product.status", entityProduct, product.ID, TrimOrNil(req.Reason), map[string]any{
			"from": product.Status,
			"to":   target,
		})
	}
	return s.showAdminProduct(c, product.ID)
}

func (s *service) DeleteProduct(c *gin.Context, id string) error {
	product, err := s.findProductByID(c, id)
	if err != nil {
		return err
	}

	changed, err := s.Repository.SoftDeleteProduct(c, product.ID)
	if err != nil {
		return err
	}
	if changed {
		s.writeAudit(c, "product.delete", entityProduct, product.ID, nil, map[string]any{
			"slug": product.Slug,
			"name": product.Name,
		})
	}
	return nil
}

// ---------------------------------------------------------------------------
// Varian
// ---------------------------------------------------------------------------

func (s *service) CreateVariant(c *gin.Context, productID string, req *VariantRequest) (*ProductVariant, error) {
	product, err := s.findProductByID(c, productID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	size := strings.TrimSpace(req.Size)
	color := strings.TrimSpace(req.Color)
	if err := ValidateVariant(size, color, req.Price, req.Stock); err != nil {
		return nil, err
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = VariantActive
	}
	if err := ValidateVariantStatus(status); err != nil {
		return nil, err
	}

	now := time.Now()
	variant := &ProductVariant{
		ID:        uuid.NewString(),
		PublicID:  newPublicID(),
		ProductID: product.ID,
		SKU:       strings.TrimSpace(req.SKU),
		Size:      size,
		Color:     color,
		Price:     req.Price,
		Stock:     req.Stock,
		Status:    status,
		Position:  req.Position,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.Repository.CreateVariant(c, variant); err != nil {
		if isUniqueViolation(err) {
			return nil, invalid("varian dengan ukuran/warna itu sudah ada")
		}
		return nil, err
	}

	s.writeAudit(c, "variant.create", entityVariant, variant.ID, nil, map[string]any{
		"product_id": product.ID,
		"label":      VariantLabel(size, color),
		"price":      req.Price,
		"stock":      req.Stock,
	})
	return variant, nil
}

// UpdateVariant memperbarui keterangan varian.
//
// Stok sengaja TIDAK ikut diperbarui di sini, walaupun field-nya ada di badan
// permintaan: satu-satunya jalur yang boleh mengubah stok adalah AdjustStock,
// yang mencatat alasannya di buku besar. Mengubahnya diam-diam lewat formulir
// varian akan membuat buku besar stok berhenti dapat dipercaya.
func (s *service) UpdateVariant(c *gin.Context, id string, req *VariantRequest) (*ProductVariant, error) {
	variant, err := s.findVariantByID(c, id)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	size := strings.TrimSpace(req.Size)
	color := strings.TrimSpace(req.Color)
	if err := ValidateVariant(size, color, req.Price, variant.Stock); err != nil {
		return nil, err
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = variant.Status
	}
	if err := ValidateVariantStatus(status); err != nil {
		return nil, err
	}

	fields := map[string]interface{}{
		"sku":        strings.TrimSpace(req.SKU),
		"size":       size,
		"color":      color,
		"price":      req.Price,
		"status":     status,
		"position":   req.Position,
		"updated_at": time.Now(),
	}
	if err := s.Repository.UpdateVariant(c, variant.ID, fields); err != nil {
		if isUniqueViolation(err) {
			return nil, invalid("varian dengan ukuran/warna itu sudah ada")
		}
		return nil, err
	}

	s.writeAudit(c, "variant.update", entityVariant, variant.ID, nil, map[string]any{
		"product_id": variant.ProductID,
		"label":      VariantLabel(size, color),
	})
	return s.Repository.FindVariantByID(c, variant.ID)
}

func (s *service) DeleteVariant(c *gin.Context, id string) error {
	variant, err := s.findVariantByID(c, id)
	if err != nil {
		return err
	}

	changed, err := s.Repository.SoftDeleteVariant(c, variant.ID)
	if err != nil {
		return err
	}
	if changed {
		s.writeAudit(c, "variant.delete", entityVariant, variant.ID, nil, map[string]any{
			"product_id": variant.ProductID,
			"label":      VariantLabel(variant.Size, variant.Color),
		})
	}
	return nil
}

// AdjustStock mengubah stok secara manual dan mencatatnya di buku besar.
//
// Inilah satu-satunya jalur yang boleh mengubah stok tanpa penjualan, dan
// karena itu alasannya wajib: baris buku besar tanpa alasan tidak dapat
// menjelaskan selisih apa pun kepada siapa pun yang memeriksanya kelak.
func (s *service) AdjustStock(c *gin.Context, variantID string, req *AdjustStockRequest) (*ProductVariant, error) {
	variant, err := s.findVariantByID(c, variantID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if req.Delta == 0 {
		return nil, invalid("penyesuaian stok tidak boleh nol")
	}
	if err := RequireReason(req.Reason); err != nil {
		return nil, err
	}

	if err := s.Repository.AdjustStock(c, variant.ID, req.Delta,
		strings.TrimSpace(req.Reason), strings.TrimSpace(req.Note), actorID(c)); err != nil {
		return nil, err
	}

	s.writeAudit(c, "stock.adjust", entityVariant, variant.ID, TrimOrNil(req.Reason), map[string]any{
		"product_id": variant.ProductID,
		"delta":      req.Delta,
	})
	return s.Repository.FindVariantByID(c, variant.ID)
}

// ---------------------------------------------------------------------------
// Pesanan pengurus
// ---------------------------------------------------------------------------

func (s *service) AdminOrders(c *gin.Context, filter OrderFilter, page, perPage int) ([]*OrderView, bool, error) {
	filter.PaymentStatus = strings.TrimSpace(filter.PaymentStatus)
	filter.FulfillmentStatus = strings.TrimSpace(filter.FulfillmentStatus)
	if filter.PaymentStatus != "" && !isPaymentStatus(filter.PaymentStatus) {
		return nil, false, invalid("status pembayaran tidak dikenal: %s", filter.PaymentStatus)
	}
	if filter.FulfillmentStatus != "" && !isFulfillmentStatus(filter.FulfillmentStatus) {
		return nil, false, invalid("status pemenuhan tidak dikenal: %s", filter.FulfillmentStatus)
	}

	page, perPage = gamification.NormalizePagination(page, perPage)
	orders, err := s.Repository.ListOrders(c, filter, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(orders) > perPage
	if hasMore {
		orders = orders[:perPage]
	}

	// Antrean pengurus juga menyelesaikan kedaluwarsa: pengurus yang membuka
	// antreannya adalah orang yang paling mungkin melihat pesanan yang
	// tahanannya sudah lepas.
	settled := make([]*ShopOrder, 0, len(orders))
	for _, order := range orders {
		updated, err := s.settleExpiry(c, order)
		if err != nil {
			return nil, false, err
		}
		settled = append(settled, updated)
	}

	orderIDs := make([]string, 0, len(settled))
	for _, order := range settled {
		orderIDs = append(orderIDs, order.ID)
	}
	items, err := s.Repository.ListOrderItems(c, orderIDs)
	if err != nil {
		return nil, false, err
	}

	views := make([]*OrderView, 0, len(settled))
	for _, order := range settled {
		order.Items = items[order.ID]
		views = append(views, &OrderView{ShopOrder: *order})
	}
	return views, hasMore, nil
}

// ConfirmOrder memverifikasi pembayaran dan mengubah tahanan menjadi penjualan.
//
// Nominal diperiksa terhadap total pesanan, bukan terhadap subtotal: ongkir
// yang diisi pengurus adalah bagian dari tagihan, dan transfer yang kurang
// serupiah pun tidak boleh diam-diam dianggap lunas.
func (s *service) ConfirmOrder(c *gin.Context, id string, req *ConfirmOrderRequest) (*OrderView, error) {
	if !ratelimit.ShopReview.Allow(ratelimit.Key(c, actorID(c))) {
		return nil, ratelimit.ErrTooManyRequests
	}

	order, err := s.findOrderByID(c, id)
	if err != nil {
		return nil, err
	}
	order, err = s.settleExpiry(c, order)
	if err != nil {
		return nil, err
	}

	if order.PaymentStatus == PaymentPaid {
		// Sudah dikonfirmasi sebelumnya. Pemberian XP diulang supaya percobaan
		// yang gagal di tengah tetap pulih — kunci dedup gamifikasi membuatnya
		// tidak pernah memberi dua kali.
		if err := s.grantReward(c, order); err != nil {
			return nil, err
		}
		return s.orderViewByID(c, order.ID, true)
	}
	if order.PaymentStatus != PaymentPending {
		return nil, ErrOrderAlreadyDecided
	}
	// Pesanan yang pemenuhannya sudah batal — biasanya karena tahanannya
	// kedaluwarsa — tidak boleh dikonfirmasi: barangnya sudah dilepas, dan
	// menandainya lunas hanya akan mencatat uang untuk barang yang tidak
	// pernah diserahkan. Pengurus harus menolaknya.
	if order.FulfillmentStatus == FulfillmentCancelled {
		return nil, ErrOrderAlreadyDecided
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	paidAmount := req.PaidAmount
	if paidAmount <= 0 {
		paidAmount = order.Total
	}
	if err := payment.VerifyManual(int(order.Total), int(paidAmount), req.PaymentReference); err != nil {
		return nil, err
	}

	fields := map[string]interface{}{
		"paid_amount":       paidAmount,
		"payment_reference": strings.TrimSpace(req.PaymentReference),
		"confirmed_at":      time.Now(),
		"decision_reason":   TrimOrNil(req.Reason),
	}
	if proofURL := strings.TrimSpace(req.ProofURL); proofURL != "" {
		fields["proof_url"] = proofURL
	}
	if actor := actorID(c); actor != "" {
		fields["confirmed_by"] = actor
	}

	changed, err := s.Repository.ConfirmPayment(c, order.ID, fields, actorID(c))
	if err != nil {
		// Stok tidak lagi mencukupi berarti tahanannya sudah kedaluwarsa dan
		// barangnya diambil pembeli lain. Pesanannya harus ditolak pengurus,
		// bukan dipaksa jalan.
		if errors.Is(err, errStockShortage) {
			return nil, ErrStockShortage
		}
		return nil, err
	}

	if !changed {
		refreshed, err := s.findOrderByID(c, order.ID)
		if err != nil {
			return nil, err
		}
		if refreshed.PaymentStatus != PaymentPaid {
			return nil, ErrOrderAlreadyDecided
		}
		order = refreshed
	} else {
		s.writeAudit(c, "shop_order.confirm", entityOrder, order.ID, TrimOrNil(req.Reason), map[string]any{
			"subtotal":      order.Subtotal,
			"shipping_cost": order.ShippingCost,
			"paid_amount":   paidAmount,
		})
		if order, err = s.findOrderByID(c, order.ID); err != nil {
			return nil, err
		}
	}

	if err := s.grantReward(c, order); err != nil {
		return nil, err
	}
	return s.orderViewByID(c, order.ID, true)
}

func (s *service) RejectOrder(c *gin.Context, id string, req *DecideOrderRequest) (*OrderView, error) {
	if !ratelimit.ShopReview.Allow(ratelimit.Key(c, actorID(c))) {
		return nil, ratelimit.ErrTooManyRequests
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if err := RequireReason(req.Reason); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Reason)

	order, err := s.findOrderByID(c, id)
	if err != nil {
		return nil, err
	}
	order, err = s.settleExpiry(c, order)
	if err != nil {
		return nil, err
	}
	if order.PaymentStatus == PaymentRejected {
		return s.orderViewByID(c, order.ID, true)
	}
	if order.PaymentStatus != PaymentPending {
		return nil, ErrOrderAlreadyDecided
	}

	changed, err := s.Repository.RejectPayment(c, order.ID, map[string]interface{}{"decision_reason": reason})
	if err != nil {
		return nil, err
	}
	if !changed {
		refreshed, err := s.findOrderByID(c, order.ID)
		if err != nil {
			return nil, err
		}
		if refreshed.PaymentStatus != PaymentRejected {
			return nil, ErrOrderAlreadyDecided
		}
	} else {
		s.writeAudit(c, "shop_order.reject", entityOrder, order.ID, &reason, nil)
	}
	return s.orderViewByID(c, order.ID, true)
}

// RefundOrder mengembalikan dana, mengembalikan stok, dan membalik XP.
//
// Pembalikan XP memakai snapshot rewarded_xp, bukan nilai aturan saat ini,
// supaya yang dikembalikan persis sebesar yang pernah diberikan walaupun
// xp_rules berubah setelahnya. Kunci dedup deterministik membuat pengulangan
// refund tidak pernah membalik dua kali.
func (s *service) RefundOrder(c *gin.Context, id string, req *DecideOrderRequest) (*OrderView, error) {
	if !ratelimit.ShopReview.Allow(ratelimit.Key(c, actorID(c))) {
		return nil, ratelimit.ErrTooManyRequests
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if err := RequireReason(req.Reason); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(req.Reason)

	order, err := s.findOrderByID(c, id)
	if err != nil {
		return nil, err
	}
	if order.PaymentStatus == PaymentRefunded {
		return s.orderViewByID(c, order.ID, true)
	}
	if order.PaymentStatus != PaymentPaid {
		return nil, ErrOrderAlreadyDecided
	}

	changed, err := s.Repository.RefundPayment(c, order.ID,
		map[string]interface{}{"decision_reason": reason}, actorID(c))
	if err != nil {
		return nil, err
	}

	updated, err := s.findOrderByID(c, order.ID)
	if err != nil {
		return nil, err
	}
	if !changed && updated.PaymentStatus != PaymentRefunded {
		return nil, ErrOrderAlreadyDecided
	}
	if changed {
		s.writeAudit(c, "shop_order.refund", entityOrder, order.ID, &reason, map[string]any{
			"rewarded_xp": updated.RewardedXP,
		})
	}

	if updated.RewardedXP > 0 && s.Rewards != nil {
		if err := s.Rewards.ReverseShopOrderReward(c, updated.UserID, updated.ID, updated.RewardedXP, reason); err != nil {
			log.Printf("[xp] gagal membalik reward pesanan %s: %v", updated.ID, err)
			return nil, err
		}
	}
	return s.orderViewByID(c, order.ID, true)
}

// FulfillOrder memindahkan status pemenuhan.
//
// Barang tidak boleh bergerak sebelum uangnya jelas: pesanan yang belum
// dikonfirmasi pembayarannya selalu ditolak di sini, apa pun tujuan
// perpindahannya. Pembatalan ditangani jalur tersendiri karena ia menyentuh
// stok, dan karena itu menuntut alasan.
func (s *service) FulfillOrder(c *gin.Context, id string, req *FulfillOrderRequest) (*OrderView, error) {
	if !ratelimit.ShopReview.Allow(ratelimit.Key(c, actorID(c))) {
		return nil, ratelimit.ErrTooManyRequests
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	order, err := s.findOrderByID(c, id)
	if err != nil {
		return nil, err
	}
	order, err = s.settleExpiry(c, order)
	if err != nil {
		return nil, err
	}

	target := strings.TrimSpace(req.Status)
	if !CanTransitionFulfillment(order.FulfillmentMethod, order.FulfillmentStatus, target) {
		return nil, ErrInvalidTransition
	}
	if order.PaymentStatus != PaymentPaid {
		return nil, invalid("pesanan belum dibayar")
	}

	if target == FulfillmentCancelled {
		if err := RequireReason(req.Reason); err != nil {
			return nil, err
		}
		reason := strings.TrimSpace(req.Reason)
		changed, err := s.Repository.CancelSettlement(c, order.ID, order.FulfillmentStatus,
			map[string]interface{}{"decision_reason": reason}, actorID(c), reason)
		if err != nil {
			return nil, err
		}
		if changed {
			s.writeAudit(c, "shop_order.cancel", entityOrder, order.ID, &reason, nil)
		}
		return s.orderViewByID(c, order.ID, true)
	}

	fields := map[string]interface{}{}
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		fields["decision_reason"] = reason
	}

	changed, err := s.Repository.TransitionFulfillment(c, order.ID, order.FulfillmentStatus, target, fields)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, ErrInvalidTransition
	}

	s.writeAudit(c, "shop_order.fulfill", entityOrder, order.ID, TrimOrNil(req.Reason), map[string]any{
		"from": order.FulfillmentStatus,
		"to":   target,
	})
	return s.orderViewByID(c, order.ID, true)
}

// SetShipping mengisi ongkir dan nomor resi.
//
// Ongkir hanya boleh berubah selama pembayaran belum dikonfirmasi: setelah itu
// nominal tagihannya sudah disepakati pembeli, dan mengubahnya akan membuat
// transfer yang sudah masuk tiba-tiba tidak cocok dengan tagihannya.
func (s *service) SetShipping(c *gin.Context, id string, req *ShippingRequest) (*OrderView, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	order, err := s.findOrderByID(c, id)
	if err != nil {
		return nil, err
	}
	if order.FulfillmentMethod != MethodShipping {
		return nil, invalid("pesanan ini diambil sendiri, bukan dikirim")
	}
	if order.FulfillmentStatus == FulfillmentCancelled || order.FulfillmentStatus == FulfillmentCompleted {
		return nil, ErrInvalidTransition
	}

	expectedStatus := ""
	fields := map[string]interface{}{}

	if req.ShippingCost != order.ShippingCost {
		if order.PaymentStatus != PaymentPending {
			return nil, invalid("ongkos kirim tidak dapat diubah setelah pembayaran dikonfirmasi")
		}
		if req.ShippingCost < 0 {
			return nil, invalid("ongkos kirim tidak boleh negatif")
		}
		fields["shipping_cost"] = req.ShippingCost
		expectedStatus = PaymentPending
	}

	if tracking := strings.TrimSpace(req.TrackingNumber); tracking != "" {
		if current := derefString(order.TrackingNumber); current != tracking {
			fields["tracking_number"] = tracking
		}
	}

	if len(fields) == 0 {
		return s.orderViewByID(c, order.ID, true)
	}

	changed, err := s.Repository.UpdateOrderFields(c, order.ID, expectedStatus, fields)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, ErrOrderAlreadyDecided
	}

	s.writeAudit(c, "shop_order.shipping", entityOrder, order.ID, nil, map[string]any{
		"shipping_cost": req.ShippingCost,
		"tracking":      strings.TrimSpace(req.TrackingNumber),
	})
	return s.orderViewByID(c, order.ID, true)
}

func (s *service) AuditLogs(c *gin.Context, page, perPage int) ([]*audit.Log, bool, error) {
	page, perPage = gamification.NormalizePagination(page, perPage)

	entries, err := s.Repository.ListAudit(c, []string{entityProduct, entityVariant, entityOrder},
		perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(entries) > perPage
	if hasMore {
		entries = entries[:perPage]
	}
	return entries, hasMore, nil
}

// ---------------------------------------------------------------------------
// Pembantu
// ---------------------------------------------------------------------------

// settleExpiry menandai pesanan yang tahanannya sudah lewat batas sebagai
// batal, lalu mengembalikan baris terbarunya.
//
// Ini pengganti pekerja latar: aplikasi ini tidak punya scheduler, jadi
// kedaluwarsa dijalankan saat pesanannya diakses. Pesanan yang tidak pernah
// dibuka pun tidak merugikan siapa pun, karena tahanan yang lewat batas sudah
// berhenti dihitung oleh query ketersediaan — yang tertinggal hanyalah status
// yang belum rapi.
func (s *service) settleExpiry(c *gin.Context, order *ShopOrder) (*ShopOrder, error) {
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if !ShouldExpireOrder(order.PaymentStatus, order.ExpiresAt, time.Now()) {
		return order, nil
	}
	// Pesanan yang sudah dikirim atau selesai tentu tidak lagi menahan stok,
	// dan membatalkannya sekarang berarti menarik kembali barang yang sudah di
	// tangan pembeli.
	if !IsCancellableFulfillment(order.FulfillmentStatus) {
		return order, nil
	}

	reason := "kedaluwarsa: batas waktu pembayaran terlewat"
	changed, err := s.Repository.CancelSettlement(c, order.ID, order.FulfillmentStatus,
		map[string]interface{}{"decision_reason": reason}, "", reason)
	if err != nil {
		return nil, err
	}
	if changed {
		s.writeAudit(c, "shop_order.cancel", entityOrder, order.ID, &reason, map[string]any{"expired": true})
	}
	return s.findOrderByID(c, order.ID)
}

// ownedOrder membaca pesanan milik pemanggil.
//
// Pesanan milik orang lain dijawab "tidak ditemukan", bukan "terlarang":
// keberadaan pesanan orang lain bukan sesuatu yang perlu diketahui pemanggil
// yang tidak berhak.
func (s *service) ownedOrder(c *gin.Context, publicID string) (*ShopOrder, error) {
	currentUser, ok := user.FromContext(c)
	if !ok || currentUser == nil {
		return nil, apperr.ErrUnauthorized
	}

	order, err := s.findOrderByPublicID(c, publicID)
	if err != nil {
		return nil, err
	}
	if order.UserID != currentUser.ID {
		return nil, ErrOrderNotFound
	}
	return s.settleExpiry(c, order)
}

// orderViewByID membaca ulang pesanan beserta barisnya, lalu merakit tampilannya.
func (s *service) orderViewByID(c *gin.Context, orderID string, withCharge bool) (*OrderView, error) {
	order, err := s.findOrderByID(c, orderID)
	if err != nil {
		return nil, err
	}
	return s.orderView(c, order, withCharge)
}

// orderView melengkapi pesanan dengan barisnya dan, bila diminta, instruksi
// pembayarannya.
//
// Instruksi hanya disusun untuk pesanan yang masih menunggu pembayaran: pesanan
// yang sudah diputuskan tidak lagi punya tagihan untuk dibayar, dan menampilkan
// tujuannya justru mengundang transfer kedua.
func (s *service) orderView(c *gin.Context, order *ShopOrder, withCharge bool) (*OrderView, error) {
	if order == nil {
		return nil, ErrOrderNotFound
	}

	items, err := s.Repository.OrderItemsWithVariant(c, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items

	view := &OrderView{ShopOrder: *order}
	if !withCharge || s.Provider == nil || order.PaymentStatus != PaymentPending {
		return view, nil
	}

	charge, err := s.Provider.CreateCharge(c, payment.ChargeRequest{
		Amount:          int(order.Total),
		PaymentMethodID: order.PaymentMethodID,
	})
	if err != nil {
		return nil, err
	}
	view.Charge = charge
	return view, nil
}

func (s *service) showAdminProduct(c *gin.Context, id string) (*ProductView, error) {
	product, err := s.findProductByID(c, id)
	if err != nil {
		return nil, err
	}
	views, err := s.buildViews(c, []*Product{product}, false)
	if err != nil {
		return nil, err
	}
	return views[0], nil
}

// grantReward memberi XP untuk pesanan yang pembayarannya sudah terkonfirmasi
// dan menyimpan snapshot-nya.
//
// Nilai yang dikembalikan gamification adalah yang benar-benar masuk ledger —
// termasuk bila batas bulanan membuatnya nol — dan itulah yang disimpan supaya
// pembalikan saat refund tepat. Pesanan yang kena batas tetap sah; hanya XP-nya
// nol.
func (s *service) grantReward(c *gin.Context, order *ShopOrder) error {
	if s.Rewards == nil || order == nil || order.PaymentStatus != PaymentPaid {
		return nil
	}
	xp, err := s.Rewards.GrantShopOrderReward(c, order.UserID, order.ID)
	if err != nil {
		return err
	}
	if xp <= 0 {
		return nil
	}
	return s.Repository.SetOrderReward(c, order.ID, xp)
}

// writeAudit mencatat satu keputusan pengurus.
//
// Hanya dipanggil setelah penulisan terjaga yang mendahuluinya benar-benar
// mengubah baris, sehingga percobaan ulang tidak menghasilkan baris audit
// ganda. Kegagalan menulis audit tidak membatalkan aksi yang sudah terjadi —
// ia dilaporkan ke log, bukan dikembalikan sebagai galat.
func (s *service) writeAudit(c *gin.Context, action, entityType, entityID string, reason *string, detail map[string]any) {
	entry := audit.NewEntry(actorID(c), action, entityType, entityID, reason, detail)

	if err := s.Repository.WriteAudit(c, entry); err != nil {
		log.Printf("[audit] gagal menulis audit %s %s/%s: %v", action, entityType, entityID, err)
	}
}

func (s *service) findProductBySlug(c *gin.Context, slug string) (*Product, error) {
	product, err := s.Repository.FindProductBySlug(c, strings.TrimSpace(slug))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return product, nil
}

func (s *service) findProductByID(c *gin.Context, id string) (*Product, error) {
	product, err := s.Repository.FindProductByID(c, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return product, nil
}

func (s *service) findVariantByID(c *gin.Context, id string) (*ProductVariant, error) {
	variant, err := s.Repository.FindVariantByID(c, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVariantNotFound
		}
		return nil, err
	}
	return variant, nil
}

func (s *service) findOrderByID(c *gin.Context, id string) (*ShopOrder, error) {
	order, err := s.Repository.FindOrderByID(c, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

func (s *service) findOrderByPublicID(c *gin.Context, publicID string) (*ShopOrder, error) {
	order, err := s.Repository.FindOrderByPublicID(c, strings.TrimSpace(publicID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return order, nil
}

// normalizeSort memetakan urutan yang tidak dikenal ke urutan bawaan, bukan
// menolaknya: urutan katalog adalah preferensi tampilan, dan menolak seluruh
// permintaan karena satu nilai yang tidak dikenal akan membuat katalognya
// hilang.
func normalizeSort(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case SortTermurah:
		return SortTermurah
	case SortTermahal:
		return SortTermahal
	default:
		return SortTerbaru
	}
}

func isProductStatus(status string) bool {
	switch status {
	case ProductDraft, ProductPublished, ProductArchived:
		return true
	}
	return false
}

func isPaymentStatus(status string) bool {
	switch status {
	case PaymentPending, PaymentPaid, PaymentRejected, PaymentRefunded:
		return true
	}
	return false
}

func isFulfillmentStatus(status string) bool {
	switch status {
	case FulfillmentUnfulfilled, FulfillmentReadyForPickup, FulfillmentShipped,
		FulfillmentCompleted, FulfillmentCancelled:
		return true
	}
	return false
}

// isUniqueViolation melaporkan apakah err berasal dari pelanggaran constraint
// unik PostgreSQL (SQLSTATE 23505). Dipakai pada jalur yang memang sebaiknya
// memberi pesan ramah — slug dan pasangan ukuran/warna — bukan pada jalur
// idempotensi, yang memakai ON CONFLICT DO NOTHING.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// actorID mengambil identitas pemanggil dari konteks autentikasi.
func actorID(c *gin.Context) string {
	if currentUser, ok := user.FromContext(c); ok {
		return currentUser.ID
	}
	return ""
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// newPublicID membuat rujukan publik pesanan: 20 karakter heksadesimal dari
// UUID tanpa tanda hubung. Cukup panjang untuk tidak mudah ditebak, cukup
// pendek untuk ditempel di tautan.
func newPublicID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
}
