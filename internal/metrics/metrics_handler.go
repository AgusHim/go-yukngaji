package metrics

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
	return &handler{Service: s}
}

// Show mengembalikan satu laporan angka untuk pengurus.
//
// Laporan ini hanya berisi agregat, tanpa satu pun identitas akun. Karena itu
// ia tidak memerlukan penyaring privasi seperti endpoint komunitas — yang
// perlu dijaga justru siapa yang boleh melihatnya.
func (h *handler) Show(c *gin.Context) {
	if !requireMetricsView(c) {
		return
	}

	res, err := h.Service.Show(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"metrics": res})
}

// requireMetricsView memastikan pemanggil berhak membaca laporan agregat.
//
// Rutenya sudah dilindungi AuthPJ, yang meloloskan himpunan yang sama persis
// dengan pemegang metrics:view. Pemeriksaan ini tetap ada supaya izinnya
// benar-benar dinyatakan, bukan hanya kebetulan cocok dengan peran yang
// diloloskan middleware.
func requireMetricsView(c *gin.Context) bool {
	return auth.RequirePermission(c, authz.PermissionMetricsView)
}
