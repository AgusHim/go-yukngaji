package thread

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

func (h *handler) ListThreads(c *gin.Context) {
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.ListThreads(c, c.Query("sort"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"threads":  res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) ShowThread(c *gin.Context) {
	res, err := h.Service.ShowThread(c, c.Param("public_id"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"thread": res})
}

func (h *handler) ListComments(c *gin.Context) {
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.ListComments(c, c.Param("public_id"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"comments": res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

// ---------------------------------------------------------------------------
// Aksi anggota
// ---------------------------------------------------------------------------

func (h *handler) CreateThread(c *gin.Context) {
	var req CreateThread
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.CreateThread(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"thread": res})
}

func (h *handler) DeleteThread(c *gin.Context) {
	if err := h.Service.DeleteThread(c, c.Param("public_id")); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Thread dihapus"})
}

func (h *handler) CreateComment(c *gin.Context) {
	var req CreateThreadComment
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.CreateComment(c, c.Param("public_id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"comment": res})
}

func (h *handler) DeleteComment(c *gin.Context) {
	err := h.Service.DeleteComment(c, c.Param("public_id"), c.Param("comment_public_id"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Komentar dihapus"})
}

func (h *handler) React(c *gin.Context) {
	if err := h.Service.React(c, c.Param("public_id")); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Reaksi tersimpan"})
}

func (h *handler) Unreact(c *gin.Context) {
	if err := h.Service.Unreact(c, c.Param("public_id")); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Reaksi dibatalkan"})
}

// Report selalu menjawab 200, termasuk ketika laporannya sudah pernah dikirim.
// Laporan kedua atas target yang sama tidak menambah baris apa pun, dan
// memberi tahu pelapornya bahwa laporannya gagal hanya akan mendorongnya
// mencoba lagi.
func (h *handler) Report(c *gin.Context) {
	var req CreateReport
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if err := h.Service.Report(c, &req); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Laporan diterima"})
}

// ---------------------------------------------------------------------------
// Preferensi berbagi aktivitas
// ---------------------------------------------------------------------------

func (h *handler) MySharePrefs(c *gin.Context) {
	res, err := h.Service.MySharePrefs(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"share_prefs": res})
}

func (h *handler) UpdateSharePrefs(c *gin.Context) {
	var req UpdateSharePrefs
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.UpdateSharePrefs(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"share_prefs": res})
}

// ---------------------------------------------------------------------------
// Moderasi
// ---------------------------------------------------------------------------
//
// Seluruh handler di bawah ini memeriksa authz.PermissionModerate lebih dulu.
// Rutenya sudah dijaga AuthRanger, tetapi izinnya diperiksa lagi di sini —
// ranger yang tidak memegang izin moderasi tidak boleh lolos hanya karena
// perannya cocok.

func (h *handler) ListReports(c *gin.Context) {
	if !requireModerate(c) {
		return
	}
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.ListReports(c, c.Query("status"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"reports":  res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) ActionReport(c *gin.Context) {
	if !requireModerate(c) {
		return
	}
	var req DecideReport
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.ActionReport(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": res})
}

func (h *handler) DismissReport(c *gin.Context) {
	if !requireModerate(c) {
		return
	}
	var req DecideReport
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.DismissReport(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": res})
}

func (h *handler) ListThreadsForReview(c *gin.Context) {
	if !requireModerate(c) {
		return
	}
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.ListThreadsForReview(c, c.Query("status"), page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"threads":  res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) ModerateThread(c *gin.Context) {
	if !requireModerate(c) {
		return
	}
	var req DecideContent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.ModerateThread(c, c.Param("id"), c.Param("action"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"thread": res})
}

func (h *handler) ModerateComment(c *gin.Context) {
	if !requireModerate(c) {
		return
	}
	var req DecideContent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.ModerateComment(c, c.Param("id"), c.Param("action"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"comment": res})
}

func (h *handler) RestrictedAccounts(c *gin.Context) {
	if !requireModerate(c) {
		return
	}
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.RestrictedAccounts(c, page, perPage)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"accounts": res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}

func (h *handler) RestrictAccount(c *gin.Context) {
	if !requireModerate(c) {
		return
	}
	var req RestrictAccount
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.RestrictAccount(c, c.Param("public_id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"account": res})
}

func (h *handler) ModerationAudit(c *gin.Context) {
	if !requireModerate(c) {
		return
	}
	page, perPage := paginationParams(c)
	res, hasMore, err := h.Service.ModerationAudit(c, page, perPage)
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

func paginationParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	return gamification.NormalizePagination(page, perPage)
}

// requireModerate memastikan pemanggil memegang izin moderasi. Rutenya sudah
// dijaga AuthRanger; pemeriksaan ini yang memastikan izinnya benar-benar ada.
func requireModerate(c *gin.Context) bool {
	return auth.RequirePermission(c, authz.PermissionModerate)
}
