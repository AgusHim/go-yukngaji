package gamification

import (
	"net/http"
	"strconv"

	"mainyuk/internal/auth"
	"mainyuk/internal/authz"
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

// AuditLogs menampilkan jejak koreksi XP dan perubahan aturan XP/level.
//
// Rutenya dijaga AuthPJ di tingkat route; pemeriksaan izin di sini menyatakan
// haknya secara eksplisit, sama seperti pasangan tulisnya.
func (h *handler) AuditLogs(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}

	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.AuditLogs(c, c.Query("entity_type"), c.Query("entity_id"), page, perPage)
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

func (h *handler) Summary(c *gin.Context) {
	res, err := h.Service.Summary(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": res})
}

func (h *handler) History(c *gin.Context) {
	page, perPage := paginationParams(c)

	res, hasMore, err := h.Service.History(c, page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"entries":  res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

// Adjust adalah koreksi XP oleh admin. Alasan wajib diisi agar jejak ledger
// dapat diaudit; ledgernya sendiri tidak pernah diubah atau dihapus.
func (h *handler) Adjust(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}

	var req AdjustXP
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.Adjust(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"entry": res})
}

func (h *handler) LevelRules(c *gin.Context) {
	res, err := h.Service.LevelRules(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": res})
}

func (h *handler) UpdateLevelRules(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}

	var req UpdateLevelRules
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.UpdateLevelRules(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": res})
}

func (h *handler) XPRules(c *gin.Context) {
	res, err := h.Service.XPRules(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": res})
}

func (h *handler) UpdateXPRule(c *gin.Context) {
	if !requireMissionManage(c) {
		return
	}

	var req UpdateXPRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.UpdateXPRule(c, c.Param("source_type"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": res})
}

// Leaderboard bersifat publik: hanya identitas publik yang dikirim.
func (h *handler) Leaderboard(c *gin.Context) {
	page, perPage := paginationParams(c)

	res, err := h.Service.Leaderboard(c, c.Query("period"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// paginationParams membaca ?page=&per_page= dengan pengaman rentang.
func paginationParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	return NormalizePagination(page, perPage)
}

// requireMissionManage memastikan pemanggil punya izin mengelola misi dan XP.
func requireMissionManage(c *gin.Context) bool {
	return auth.RequirePermission(c, authz.PermissionMissionManage)
}
