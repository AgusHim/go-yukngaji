package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mainyuk/utils"
	"net/http"
	"os"

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
	c.JSON(http.StatusOK, res)
}

func (h *handler) Login(c *gin.Context) {
	var u Login
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
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
		"user":         res,
		"access_token": token,
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
	c.JSON(http.StatusOK, res)
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
	token, err := utils.GenerateJWT(res.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":         res,
		"access_token": token,
	})
}

func (h *handler) UpdateAuth(c *gin.Context) {
	authUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not Authorized",
		})
		return
	}
	currentUser, ok := authUser.(User)

	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Error parsing current user",
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
			"error": "Invalid JSON",
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
	c.JSON(http.StatusOK, res)
}

func (h *handler) AuthGoogleLogin(c *gin.Context) {
	redirectTo := c.DefaultQuery("redirectTo", "/events")
	oauthState := generateStateOauthCookie(redirectTo)

	u := googleOauthConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	c.SetCookie("oauthstate", oauthState, 3600, "", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"authUrl": u,
		"state":   oauthState,
	})
}

func (h *handler) AuthGoogleCallback(c *gin.Context) {
	stateQuery := c.DefaultQuery("state", "")
	if stateQuery == "" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid oauth state",
		})
		return
	}
	stateDecoded, _ := base64.StdEncoding.DecodeString(stateQuery)

	var state customState
	if err := json.Unmarshal(stateDecoded, &state); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Error json unmarshal state",
		})
		return
	}

	code := c.DefaultQuery("code", "")
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
		"user":         user,
		"access_token": jwt,
		"redirectTo":   state.RedirectTo,
	})

}

func (h *handler) AuthGoogleVerify(c *gin.Context) {
	state := c.DefaultQuery("state", "")
	if state == "" {
		log.Println("invalid oauth state")
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid oauth state",
		})
		return
	}

	code := c.DefaultQuery("code", "")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("could not get token: %v\n", err)
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
		"user":         user,
		"access_token": jwt,
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
