package ws

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	hub *Hub
}

func NewHandler(h *Hub) *Handler {
	return &Handler{
		hub: h,
	}
}

func (h *Handler) ConnectWS(c *gin.Context) {
	userID := c.Query("user_id")
	username := c.Query("username")
	roomID := c.Param("id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error Upgrade Websocket", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	currentTime := time.Now()
	log.Printf("[User Join] %s join room %s", username, roomID)
	client := &Client{
		UserID:   userID,
		Username: username,
		RoomID:   roomID,
		hub:      h.hub, conn: conn, send: make(chan *Message, 5),
		ConnectAt: currentTime,
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
