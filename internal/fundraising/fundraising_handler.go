package fundraising

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
// Permukaan publik
// ---------------------------------------------------------------------------

func (h *handler) ListCampaigns(c *gin.Context) {
	res, err := h.Service.ListCampaigns(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaigns": res})
}

func (h *handler) ShowCampaign(c *gin.Context) {
	res, err := h.Service.ShowCampaign(c, c.Param("slug"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": res})
}

func (h *handler) ListDonors(c *gin.Context) {
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.ListDonors(c, c.Param("slug"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"donations": res,
		"page":      page,
		"per_page":  perPage,
		"has_more":  hasMore,
	})
}

func (h *handler) ListMessages(c *gin.Context) {
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.ListMessages(c, c.Param("slug"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"messages": res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) ListUpdates(c *gin.Context) {
	res, err := h.Service.ListUpdates(c, c.Param("slug"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"updates": res})
}

func (h *handler) PublicReport(c *gin.Context) {
	res, err := h.Service.PublicReport(c, c.Param("slug"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": res})
}

// ---------------------------------------------------------------------------
// Anggota
// ---------------------------------------------------------------------------

func (h *handler) CreateDonation(c *gin.Context) {
	var req CreateDonation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.CreateDonation(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *handler) MyDonations(c *gin.Context) {
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.MyDonations(c, page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"donations": res,
		"page":      page,
		"per_page":  perPage,
		"has_more":  hasMore,
	})
}

func (h *handler) MyDonation(c *gin.Context) {
	res, err := h.Service.MyDonation(c, c.Param("public_id"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ---------------------------------------------------------------------------
// Pengurus
// ---------------------------------------------------------------------------

func (h *handler) ListAllCampaigns(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}
	res, err := h.Service.ListAllCampaigns(c, c.Query("status"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaigns": res})
}

func (h *handler) CreateCampaign(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}

	var req CreateCampaign
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.CreateCampaign(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"campaign": res})
}

func (h *handler) UpdateCampaign(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}

	var req UpdateCampaign
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.UpdateCampaign(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": res})
}

func (h *handler) SetCampaignStatus(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}

	var req SetCampaignStatus
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.SetCampaignStatus(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": res})
}

func (h *handler) DeleteCampaign(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}
	if err := h.Service.DeleteCampaign(c, c.Param("id")); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Campaign dihapus"})
}

func (h *handler) CampaignReport(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}
	res, err := h.Service.CampaignReport(c, c.Param("id"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": res})
}

func (h *handler) DonationsForReview(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}
	page, perPage := paginationParams(c)

	filter := DonationFilter{
		Status:        c.Query("status"),
		MessageStatus: c.Query("message_status"),
		CampaignID:    c.Query("campaign_id"),
	}
	res, hasMore, err := h.Service.DonationsForReview(c, filter, page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"donations": res,
		"page":      page,
		"per_page":  perPage,
		"has_more":  hasMore,
	})
}

// ConfirmDonation menerima body opsional: nominal yang diterima dan referensi
// transfer boleh dikosongkan, dan service akan memakai nominal donasi apa
// adanya. Karena itu body kosong tidak dianggap galat.
func (h *handler) ConfirmDonation(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}

	var req ConfirmDonation
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
	}

	res, err := h.Service.ConfirmDonation(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"donation": res})
}

func (h *handler) RejectDonation(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}
	h.decide(c, h.Service.RejectDonation)
}

func (h *handler) RefundDonation(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}
	h.decide(c, h.Service.RefundDonation)
}

// decide menghindari duplikasi penanganan body antara tolak dan refund.
// Alasan wajib ditolak di lapisan service, bukan di sini.
func (h *handler) decide(c *gin.Context, fn func(*gin.Context, string, *DecideDonation) (*Donation, error)) {
	var req DecideDonation
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
	}

	res, err := fn(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"donation": res})
}

func (h *handler) ModerateMessage(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}

	var req ModerateMessage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.ModerateMessage(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"donation": res})
}

func (h *handler) CreateUpdate(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}

	var req CampaignUpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.CreateUpdate(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"update": res})
}

func (h *handler) UpdateUpdate(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}

	var req CampaignUpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.UpdateUpdate(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"update": res})
}

func (h *handler) DeleteUpdate(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}
	if err := h.Service.DeleteUpdate(c, c.Param("id")); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Update dihapus"})
}

func (h *handler) ListAuditLogs(c *gin.Context) {
	if !requireFundsManage(c) {
		return
	}
	page, perPage := paginationParams(c)

	res, hasMore, err := h.Service.ListAuditLogs(c, c.Query("entity_type"), c.Query("entity_id"), page, perPage)
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

// requireFundsManage memastikan pemanggil punya izin mengelola dana.
func requireFundsManage(c *gin.Context) bool {
	return auth.RequirePermission(c, authz.PermissionFundsManage)
}
