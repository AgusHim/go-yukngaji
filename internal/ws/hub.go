// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ws

import (
	"fmt"
	"log"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Room struct {
	ID      string
	Clients map[string]*Client
}
type Hub struct {
	// Connected room
	Rooms map[string]*Room

	// Inbound messages from the clients.
	Broadcast chan *Message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan *Message, 5),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		Rooms:      make(map[string]*Room),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			data := make(map[string]interface{})
			data["type"] = "user.join"
			data["data"] = fmt.Sprintf("%s connect", client.Username)
			client.hub.Broadcast <- &Message{
				RoomID:   client.RoomID,
				Username: client.Username,
				Message:  data,
			}
			// Check room is exist and update room
			if _, ok := h.Rooms[client.RoomID]; ok {
				// Get room
				r := h.Rooms[client.RoomID]
				if _, ok := r.Clients[client.UserID]; !ok {
					r.Clients[client.UserID] = client
				}
			} else {
				// Add new client to room
				clients := make(map[string]*Client)
				clients[client.UserID] = client

				// Create new room
				h.Rooms[client.RoomID] = &Room{
					ID:      client.RoomID,
					Clients: clients,
				}
			}

		case client := <-h.unregister:

			data := make(map[string]interface{})
			data["type"] = "user.out"
			data["data"] = fmt.Sprintf("%s out", client.Username)
			client.hub.Broadcast <- &Message{
				RoomID:   client.RoomID,
				Username: client.Username,
				Message:  data,
			}
			client.hub.Broadcast <- &Message{
				RoomID:   client.RoomID,
				Username: client.Username,
				Message:  data,
			}
			// Check room is exist
			if _, ok := h.Rooms[client.RoomID].Clients[client.UserID]; ok {
				delete(h.Rooms[client.RoomID].Clients, client.UserID)
				close(client.send)
			}
		case msg := <-h.Broadcast:
			if _, ok := h.Rooms[msg.RoomID]; ok {
				for _, cl := range h.Rooms[msg.RoomID].Clients {
					cl.send <- msg
				}
			} else {
				log.Print("[HubBroadcast] Error")
			}
		}
	}
}
