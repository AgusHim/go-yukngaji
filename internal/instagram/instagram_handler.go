package instagram

import (
	"net/http"

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

// ListPublished adalah dinding publik di landing page.
func (h *handler) ListPublished(c *gin.Context) {
	res, err := h.Service.ListPublished(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"posts": res})
}

func (h *handler) ListAll(c *gin.Context) {
	if !requireInstagramManage(c) {
		return
	}
	res, err := h.Service.ListAll(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"posts": res})
}

func (h *handler) Create(c *gin.Context) {
	if !requireInstagramManage(c) {
		return
	}

	var req CreatePost
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.Create(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"post": res})
}

func (h *handler) Update(c *gin.Context) {
	if !requireInstagramManage(c) {
		return
	}

	var req UpdatePost
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.Update(c, c.Param("id"), &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"post": res})
}

func (h *handler) Delete(c *gin.Context) {
	if !requireInstagramManage(c) {
		return
	}
	if err := h.Service.Delete(c, c.Param("id")); err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Postingan dihapus"})
}

// Sync menarik postingan terbaru dari Graph API.
//
// Tanpa kredensial, service mengembalikan ErrSyncDisabled dan handler
// meneruskannya apa adanya, sehingga pengurus membaca pesan "belum
// dikonfigurasi" alih-alih kegagalan yang tidak jelas.
func (h *handler) Sync(c *gin.Context) {
	if !requireInstagramManage(c) {
		return
	}

	res, err := h.Service.Sync(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"sync": res})
}

// requireInstagramManage memastikan pemanggil punya izin mengelola dinding
// komunitas.
func requireInstagramManage(c *gin.Context) bool {
	return auth.RequirePermission(c, authz.PermissionInstagramManage)
}
