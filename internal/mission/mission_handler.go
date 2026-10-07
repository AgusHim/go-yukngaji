package mission

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
	return &handler{
		s,
	}
}

func (h *handler) ListPublished(c *gin.Context) {
	res, err := h.Service.ListPublished(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"missions": res})
}

func (h *handler) ShowPublished(c *gin.Context) {
	res, err := h.Service.ShowPublished(c, c.Param("id"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"mission": res})
}

func (h *handler) Claim(c *gin.Context) {
	var req ClaimMission
	// Body opsional: misi tanpa kewajiban bukti boleh diklaim tanpa payload.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
	}

	res, err := h.Service.Claim(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"claim": res})
}

func (h *handler) MyClaims(c *gin.Context) {
	page, perPage := paginationParams(c)

	res, hasMore, err := h.Service.MyClaims(c, page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"claims":   res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) ListAll(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}
	res, err := h.Service.ListAll(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"missions": res})
}

func (h *handler) Create(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}

	var req CreateMission
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.Create(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"mission": res})
}

func (h *handler) Update(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}

	var req UpdateMission
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.Update(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"mission": res})
}

func (h *handler) Delete(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}
	if err := h.Service.Delete(c, c.Param("id")); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Misi dihapus"})
}

func (h *handler) ClaimsForReview(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}
	page, perPage := paginationParams(c)

	res, hasMore, err := h.Service.ClaimsForReview(c, c.Query("status"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"claims":   res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) Approve(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}
	h.decide(c, h.Service.Approve)
}

func (h *handler) Reject(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}
	h.decide(c, h.Service.Reject)
}

// decide menghindari duplikasi penanganan body opsional antara approve dan
// reject. Body boleh kosong pada approve; reject menolak alasan kosong di
// lapisan service.
func (h *handler) decide(c *gin.Context, fn func(*gin.Context, string, *DecideClaim) (*MissionClaim, error)) {
	var req DecideClaim
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
	c.JSON(http.StatusOK, gin.H{"claim": res})
}

func paginationParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	return gamification.NormalizePagination(page, perPage)
}

// requireMissionManage memastikan pemanggil punya izin mengelola misi.
func requireMissionManage(c *gin.Context) bool {
	return auth.RequirePermission(c, authz.PermissionMissionManage)
}
