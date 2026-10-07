package auth

import (
	"errors"
	"mainyuk/internal/apperr"
	"mainyuk/internal/authz"
	"mainyuk/internal/httperr"
	"mainyuk/internal/user"
	"mainyuk/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type middleware struct {
	UserService user.Service
}

func NewMiddleware(us user.Service) Middleware {
	return &middleware{
		UserService: us,
	}
}

var errUnauthorized = errors.New("unauthorized")

// currentUserFromRequest membaca identitas dari header Authorization.
//
// Tanpa header Bearer sama sekali -> (nil, nil): pemanggil memutuskan apakah
// itu tamu yang sah (AuthOptionalUser) atau ditolak (Auth*).
// Header ada tetapi tidak bisa dipakai -> (nil, errUnauthorized).
func (m *middleware) currentUserFromRequest(c *gin.Context) (*user.User, error) {
	authHeader := c.GetHeader("Authorization")
	if !strings.Contains(authHeader, "Bearer") {
		return nil, nil
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return nil, errUnauthorized
	}
	tokenString := strings.TrimSpace(parts[1])
	if tokenString == "" {
		return nil, errUnauthorized
	}
	return m.userFromToken(c, tokenString)
}

// UserFromToken memvalidasi token JWT yang datang bukan dari header
// Authorization (mis. query WebSocket) dan memuat akun aktifnya.
func (m *middleware) UserFromToken(c *gin.Context, tokenString string) (*user.User, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, errUnauthorized
	}
	return m.userFromToken(c, tokenString)
}

func (m *middleware) userFromToken(c *gin.Context, tokenString string) (*user.User, error) {
	token, err := utils.ValidateJWT(tokenString)
	if err != nil {
		return nil, errUnauthorized
	}

	claim, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errUnauthorized
	}

	userID, ok := claim["user_id"].(string)
	if !ok || userID == "" {
		return nil, errUnauthorized
	}

	u, err := m.UserService.Show(c, userID)
	if err != nil {
		return nil, errUnauthorized
	}
	return u, nil
}

// guard menolak request tanpa identitas valid atau dengan role yang tidak
// diizinkan. Handler berikutnya hanya jalan bila guard lolos.
func (m *middleware) guard(c *gin.Context, allow func(role string) bool) {
	u, err := m.currentUserFromRequest(c)
	if err != nil || u == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}
	if !allow(u.Role) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized role",
		})
		return
	}
	c.Set(user.ContextKey, *u)
}

func (m *middleware) AuthAdmin(c *gin.Context) {
	m.guard(c, func(role string) bool {
		return role == authz.RoleAdmin
	})
}

func (m *middleware) AuthPJ(c *gin.Context) {
	m.guard(c, authz.IsStaff)
}

func (m *middleware) AuthRanger(c *gin.Context) {
	m.guard(c, authz.IsRanger)
}

// AuthUser menerima semua pengguna yang terautentikasi, apa pun role-nya
// (anggota, ranger, pj, maupun admin).
func (m *middleware) AuthUser(c *gin.Context) {
	m.guard(c, func(role string) bool {
		return role != ""
	})
}

// AuthOptionalUser tidak menolak request. Bila Bearer valid, user aktif
// dipasang di context; bila tidak, request lanjut sebagai tamu.
//
// Middleware ini dipakai pada route publik yang perilakunya berbeda untuk
// pengguna login (mis. komentar, presence, poll). Handler harus tetap
// mengambil identitas lewat user.FromContext, bukan dari body request.
func (m *middleware) AuthOptionalUser(c *gin.Context) {
	u, err := m.currentUserFromRequest(c)
	if err == nil && u != nil {
		c.Set(user.ContextKey, *u)
	}
	c.Next()
}

// RequirePermission memeriksa izin bertipe milik pengguna aktif dan menulis
// responsnya sendiri bila gagal. Mengembalikan false berarti handler harus
// berhenti.
//
// Middleware AuthPJ hanya membatasi ke pengurus; fungsi ini menutup perbedaan
// izin antar-role pengurus (mis. ranger tidak boleh mengelola misi).
func RequirePermission(c *gin.Context, p authz.Permission) bool {
	u, ok := user.FromContext(c)
	if !ok {
		httperr.JSON(c, apperr.ErrUnauthorized)
		return false
	}
	if !authz.Can(u.Role, p) {
		httperr.JSON(c, apperr.ErrForbidden)
		return false
	}
	return true
}
