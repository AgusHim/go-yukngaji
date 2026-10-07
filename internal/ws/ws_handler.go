package ws

import (
	"log"
	"net/http"
	"time"

	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
)

// TokenVerifier memvalidasi token WebSocket dan memuat akun aktif.
type TokenVerifier interface {
	UserFromToken(ctx *gin.Context, token string) (*user.User, error)
}

type Handler struct {
	hub      *Hub
	verifier TokenVerifier
}

func NewHandler(h *Hub, verifier TokenVerifier) *Handler {
	return &Handler{
		hub:      h,
		verifier: verifier,
	}
}

// ConnectWS membuka koneksi WebSocket untuk sebuah event.
//
// Identitas hanya diambil dari token yang valid. Parameter query `user_id` dan
// `username` tidak lagi dipercaya: sebelumnya siapa pun bisa menyamar sebagai
// pengguna lain hanya dengan mengubah URL. Tanpa token valid, koneksi tetap
// dibuka sebagai tamu anonim.
func (h *Handler) ConnectWS(c *gin.Context) {
	roomID := c.Param("id")
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room id is required"})
		return
	}

	userID, username := "", "Anonim"
	if token := TokenFromRequest(c.Request); token != "" && h.verifier != nil {
		if u, err := h.verifier.UserFromToken(c, token); err == nil && u != nil {
			userID = u.ID
			username = u.Username
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Upgrade sudah menulis respons sendiri; jangan menulis lagi.
		log.Println("Error Upgrade Websocket", err)
		return
	}

	client := &Client{
		UserID:    userID,
		Username:  username,
		RoomID:    roomID,
		hub:       h.hub,
		conn:      conn,
		send:      make(chan *Message, 5),
		ConnectAt: time.Now(),
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
