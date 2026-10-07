package user

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mainyuk/internal/apperr"
	"mainyuk/internal/authz"
	"mainyuk/internal/httperr"
	"mainyuk/internal/ratelimit"
	"mainyuk/utils"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	oauth2api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type handler struct {
	Service
}

var (
	googleOauthConfig *oauth2.Config
)

const oauthStateCookie = "oauthstate"

type customState struct {
	CSRFToken  string `json:"csrf_token"`
	RedirectTo string `json:"redirectTo"`
}

func NewHandler(s Service) Handler {
	// Scopes: OAuth 2.0 scopes provide a way to limit the amount of access that is granted to an access token.
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	return &handler{
		s,
	}
}

func (h *handler) Register(c *gin.Context) {
	var u CreateUser
	if err := c.ShouldBindJSON(&u); err != nil {
		if err.Error() == "EmailRegistered" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Email already registered",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	res, err := h.Service.Register(c, &u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, ToAccountResponse(res))
}

func (h *handler) Login(c *gin.Context) {
	var u Login
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	// Pembatas per IP: memperlambat percobaan tebak kata sandi.
	if !ratelimit.Login.Allow(ratelimit.Key(c, "")) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Terlalu banyak percobaan masuk. Coba lagi beberapa saat lagi.",
		})
		return
	}

	res, err := h.Service.Login(c, &u)
	if err != nil {
		if err.Error() == "EmailNotFound" || err.Error() == "PasswordNotMatch" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Wrong email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	token, err := utils.GenerateJWT(res.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":         ToAccountResponse(res),
		"access_token": token,
	})
}

// Me mengembalikan akun aktif sesuai identitas yang diverifikasi server.
func (h *handler) Me(c *gin.Context) {
	currentUser, ok := FromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": ToAccountResponse(currentUser),
	})
}

func (h *handler) UpdateByAdmin(c *gin.Context) {
	id := c.Param("id")
	var u CreateUser
	if err := c.ShouldBindJSON(&u); err != nil {
		if err.Error() == "EmailRegistered" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Email already registered",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	res, err := h.Service.Update(c, id, &u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user": ToAccountResponse(res),
	})
}

func (h *handler) Show(c *gin.Context) {
	id := c.Param("id")

	res, err := h.Service.Show(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}

	// Tidak ada access_token di sini: sebelumnya endpoint ini menerbitkan
	// token untuk akun yang dilihat, yang membuat pemanggil berizin bisa
	// memakai identitas akun tersebut.
	c.JSON(http.StatusOK, gin.H{
		"user": ToAccountResponse(res),
	})
}

func (h *handler) UpdateAuth(c *gin.Context) {
	currentUser, ok := FromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	var u CreateUser
	if err := c.ShouldBindJSON(&u); err != nil {
		if err.Error() == "EmailRegistered" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Email already registered",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid JSON",
			"message": err.Error(),
		})
		return
	}

	res, err := h.Service.Update(c, currentUser.ID, &u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user": ToAccountResponse(res),
	})
}

func (h *handler) AuthGoogleLogin(c *gin.Context) {
	redirectTo := c.DefaultQuery("redirectTo", "/events")
	oauthState := generateStateOauthCookie(redirectTo)

	u := googleOauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	c.SetCookie(oauthStateCookie, oauthState, 3600, "", "", isCookieSecure(), true)
	c.JSON(http.StatusOK, gin.H{
		"authUrl": u,
		"state":   oauthState,
	})
}

// isCookieSecure menentukan flag Secure pada cookie oauthstate.
// Default-nya false agar pengembangan di http://localhost tetap berjalan;
// produksi harus menyetel COOKIE_SECURE=true.
func isCookieSecure() bool {
	return os.Getenv("COOKIE_SECURE") == "true"
}

func (h *handler) AuthGoogleCallback(c *gin.Context) {
	stateQuery := c.DefaultQuery("state", "")
	if stateQuery == "" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid oauth state",
		})
		return
	}

	// Double-submit: state pada query harus sama dengan cookie yang kita
	// set saat memulai alur. Tanpa ini, penyerang bisa menyelesaikan alur
	// OAuth miliknya dan membuat korban login ke akun penyerang.
	cookieState, errCookie := c.Cookie(oauthStateCookie)
	if errCookie != nil || cookieState == "" ||
		subtle.ConstantTimeCompare([]byte(cookieState), []byte(stateQuery)) != 1 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid oauth state",
		})
		return
	}
	c.SetCookie(oauthStateCookie, "", -1, "", "", isCookieSecure(), true)

	stateDecoded, errDecode := base64.StdEncoding.DecodeString(stateQuery)
	if errDecode != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Error json unmarshal state",
		})
		return
	}

	var state customState
	if err := json.Unmarshal(stateDecoded, &state); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Error json unmarshal state",
		})
		return
	}

	code := c.DefaultQuery("code", "")
	if code == "" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Missing authorization code",
		})
		return
	}

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Could not get token",
		})
		return
	}

	client := googleOauthConfig.TokenSource(context.Background(), token)
	oauth2Service, err := oauth2api.NewService(context.Background(), option.WithTokenSource(client))

	if err != nil {
		log.Printf("could not create oauth2 service: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid oauth service",
		})
		return
	}

	userinfo, err := oauth2Service.Userinfo.Get().Do()
	if err != nil || userinfo == nil {
		log.Printf("could not get user info: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed get user info",
		})
		return
	}

	user, err := h.Service.AuthGoogleCallback(c, userinfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	jwt, err := utils.GenerateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":         ToAccountResponse(user),
		"access_token": jwt,
		"redirectTo":   state.RedirectTo,
	})

}

func generateStateOauthCookie(redirectTo string) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := customState{
		CSRFToken:  base64.URLEncoding.EncodeToString(b), // Replace with a generated token for security
		RedirectTo: redirectTo,
	}
	stateJSON, _ := json.Marshal(state)
	encodedState := base64.StdEncoding.EncodeToString(stateJSON)
	return encodedState
}

// requireUsersView memastikan pemanggil berhak membaca daftar akun.
//
// Bentuknya sama dengan auth.RequirePermission, tetapi tidak memanggilnya:
// paket auth sudah mengimpor paket ini, sehingga memakainya akan membuat
// impor melingkar. Pemeriksaannya karena itu dilakukan langsung lewat authz,
// yang tidak mengimpor apa pun dari sini.
func requireUsersView(c *gin.Context) bool {
	currentUser, ok := FromContext(c)
	if !ok {
		httperr.JSON(c, apperr.ErrUnauthorized)
		return false
	}
	if !authz.Can(currentUser.Role, authz.PermissionUsersView) {
		httperr.JSON(c, apperr.ErrForbidden)
		return false
	}
	return true
}

// List menampilkan akun untuk dashboard pengurus.
//
// Rutenya sudah dilindungi AuthPJ, yang meloloskan himpunan yang sama persis
// dengan pemegang users:view. Pemeriksaan izinnya tetap ada supaya haknya
// benar-benar dinyatakan, bukan hanya kebetulan cocok dengan peran yang
// diloloskan middleware. Daftar ini memang perlu izin tersendiri meski hanya
// membaca: isinya memuat email, telepon, dan alamat.
func (h *handler) List(c *gin.Context) {
	if !requireUsersView(c) {
		return
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	page, perPage = normalizePagination(page, perPage)

	res, hasMore, err := h.Service.List(c, c.Query("search"), c.Query("role"), page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"users":    res,
		"page":     page,
		"per_page": perPage,
		"has_more": hasMore,
	})
}
