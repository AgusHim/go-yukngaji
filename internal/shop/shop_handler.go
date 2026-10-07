package shop

import (
	"net/http"
	"strconv"

	"mainyuk/internal/auth"
	"mainyuk/internal/authz"
	"mainyuk/internal/gamification"
	"mainyuk/internal/httperr"

	"github.com/gin-gonic/gin"
)

type handler struct {
	Service
}

func NewHandler(s Service) Handler {
	return &handler{s}
}

// ---------------------------------------------------------------------------
// Katalog publik
// ---------------------------------------------------------------------------

func (h *handler) ListProducts(c *gin.Context) {
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.ListProducts(c, c.Query("sort"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"products": res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) ShowProduct(c *gin.Context) {
	res, err := h.Service.ShowProduct(c, c.Param("slug"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": res})
}

// ---------------------------------------------------------------------------
// Pesanan anggota
// ---------------------------------------------------------------------------

func (h *handler) Checkout(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.Checkout(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"order": res})
}

func (h *handler) MyOrders(c *gin.Context) {
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.MyOrders(c, page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"orders":   res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) MyOrder(c *gin.Context) {
	res, err := h.Service.MyOrder(c, c.Param("public_id"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": res})
}

func (h *handler) SubmitProof(c *gin.Context) {
	var req SubmitProofRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.SubmitProof(c, c.Param("public_id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": res})
}

func (h *handler) CancelOrder(c *gin.Context) {
	res, err := h.Service.CancelOrder(c, c.Param("public_id"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": res})
}

// ---------------------------------------------------------------------------
// Katalog pengurus
// ---------------------------------------------------------------------------

func (h *handler) AdminProducts(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.AdminProducts(c, c.Query("status"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"products": res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) CreateProduct(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.CreateProduct(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"product": res})
}

func (h *handler) UpdateProduct(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.UpdateProduct(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": res})
}

func (h *handler) SetProductStatus(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req SetProductStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.SetProductStatus(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"product": res})
}

func (h *handler) DeleteProduct(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	if err := h.Service.DeleteProduct(c, c.Param("id")); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "produk dihapus"})
}

func (h *handler) CreateVariant(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req VariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.CreateVariant(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"variant": res})
}

func (h *handler) UpdateVariant(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req VariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.UpdateVariant(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"variant": res})
}

func (h *handler) DeleteVariant(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	if err := h.Service.DeleteVariant(c, c.Param("id")); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "varian dihapus"})
}

func (h *handler) AdjustStock(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.AdjustStock(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"variant": res})
}

// ---------------------------------------------------------------------------
// Pesanan pengurus
// ---------------------------------------------------------------------------

func (h *handler) AdminOrders(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	page, perPage := paginationParams(c)
	filter := OrderFilter{
		PaymentStatus:     c.Query("payment_status"),
		FulfillmentStatus: c.Query("fulfillment_status"),
	}
	res, hasMore, err := h.Service.AdminOrders(c, filter, page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"orders":   res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) ConfirmOrder(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req ConfirmOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.ConfirmOrder(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": res})
}

func (h *handler) RejectOrder(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req DecideOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.RejectOrder(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": res})
}

func (h *handler) RefundOrder(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req DecideOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.RefundOrder(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": res})
}

func (h *handler) FulfillOrder(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req FulfillOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.FulfillOrder(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": res})
}

func (h *handler) SetShipping(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	var req ShippingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.SetShipping(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": res})
}

func (h *handler) AuditLogs(c *gin.Context) {
	if !requireProductManage(c) {
		return
	}

	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.AuditLogs(c, page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"audit_logs": res,
		"page":       page,
		"per_page":   perPage,
		"has_more":   hasMore,
	})
}

// ---------------------------------------------------------------------------
// Pembantu
// ---------------------------------------------------------------------------

func paginationParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	return gamification.NormalizePagination(page, perPage)
}

// requireProductManage memastikan pemanggil berhak mengelola katalog.
//
// Rutenya sudah dilindungi AuthPJ, yang meloloskan himpunan yang sama persis
// dengan pemegang product:manage. Pemeriksaan ini tetap ada supaya izinnya
// benar-benar dinyatakan, bukan hanya kebetulan cocok dengan peran yang
// diloloskan middleware.
func requireProductManage(c *gin.Context) bool {
	return auth.RequirePermission(c, authz.PermissionProductManage)
}
