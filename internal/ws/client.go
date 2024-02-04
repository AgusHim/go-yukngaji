// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	RoomID   string      `json:"room_id"`
	Username string      `json:"username"`
	Message  interface{} `json:"message"`
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	UserID   string
	RoomID   string
	Username string
	hub      *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan *Message

	// User connect at
	ConnectAt time.Time
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, msg, err := c.conn.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ReadPumpError]: %v", err)
			}
			log.Printf("[ReadPumpError]: %v", err)
			break
		}

		var data map[string]interface{}
		json.Unmarshal(msg, &data)

		message := &Message{
			RoomID:   c.RoomID,
			Username: c.Username,
			Message:  data,
		}

		if c.conn != nil {
			c.hub.Broadcast <- message
		} else {
			break
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				// The hub closed the channel.
				log.Printf("[WritePump] Hub close the channel %s", c.Username)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Marshal the struct into JSON bytes
			jsonBytes, err := json.Marshal(message)
			if err != nil {
				log.Fatalln("Error:", err)
				return
			}

			err = c.conn.WriteMessage(websocket.TextMessage, jsonBytes)
			if err != nil {
				log.Println("[WritePump] error ", err)
				return
			}
			c.conn.SetWriteDeadline(time.Time{})

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			c.conn.SetWriteDeadline(time.Time{})
		}
	}
}

// serveWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	userID := queryParams.Get("user_id")
	username := queryParams.Get("username")
	roomID := queryParams.Get("room_id")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	connectedAt := time.Now()

	client := &Client{
		UserID:   userID,
		Username: username,
		RoomID:   roomID,
		hub:      hub, conn: conn, send: make(chan *Message, 5),
		ConnectAt: connectedAt,
	}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
