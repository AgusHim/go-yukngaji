package main

import (
	"log"
	"mainyuk/db"
	"mainyuk/internal/auth"
	"mainyuk/internal/divisi"
	"mainyuk/internal/event"
	"mainyuk/internal/presence"
	"mainyuk/internal/user"
	"mainyuk/internal/ws"
	"mainyuk/router"
)

func main() {
	db, err := db.NewDatabase()
	if err != nil {
		log.Fatalf("Could not initialize DB Connection: %s", err)
	}

	userRepository := user.NewRepository(db)
	userService := user.NewService(userRepository)
	userHandler := user.NewHandler(userService)

	eventRepository := event.NewRepository(db)
	eventService := event.NewService(eventRepository)
	eventHandler := event.NewHandler(eventService)

	divisiRepository := divisi.NewRepository(db)
	divisiService := divisi.NewService(divisiRepository)
	divisiHandler := divisi.NewHandler(divisiService)

	presenceRepository := presence.NewRepository(db)
	presenceService := presence.NewService(presenceRepository, userService, eventService)
	presenceHandler := presence.NewHandler(presenceService)

	authMiddleware := auth.NewMiddleware(userService)

	hub := ws.NewHub()
	wsHandler := ws.NewHandler(hub)

	go hub.Run()

	router.InitRouter(authMiddleware, userHandler, eventHandler, divisiHandler, presenceHandler, wsHandler)
	router.Start("0.0.0.0:8000")
}
