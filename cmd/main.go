package main

import (
	"fmt"
	"log"
	"mainyuk/db"
	"mainyuk/internal/agenda"
	"mainyuk/internal/auth"
	"mainyuk/internal/comment"
	"mainyuk/internal/divisi"
	"mainyuk/internal/event"
	"mainyuk/internal/feedback"
	"mainyuk/internal/like"
	"mainyuk/internal/presence"
	"mainyuk/internal/ranger"
	"mainyuk/internal/ranger_presence"
	"mainyuk/internal/user"
	"mainyuk/internal/ws"
	"mainyuk/router"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	errEnv := godotenv.Load()
	if errEnv != nil {
		log.Fatalf("Error Load .env : %s", errEnv)
	}

	db, err := db.NewDatabase()
	if err != nil {
		go log.Fatalf("Could not initialize DB Connection: %s", err)
	}
	hub := ws.NewHub()
	wsHandler := ws.NewHandler(hub)

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

	commentRepository := comment.NewRepository(db)
	commentService := comment.NewService(commentRepository, userService, eventService, hub)
	commentHandler := comment.NewHandler(commentService)

	likeRepository := like.NewRepository(db)
	likeService := like.NewService(likeRepository, userService, commentService, hub)
	likeHandler := like.NewHandler(likeService)

	feedbackRepository := feedback.NewRepository(db)
	feedbackService := feedback.NewService(feedbackRepository, userService, eventService)
	feedbackHandler := feedback.NewHandler(feedbackService)

	agendaRepository := agenda.NewRepository(db)
	agendaService := agenda.NewService(agendaRepository)
	agendaHandler := agenda.NewHandler(agendaService)

	rangerRepository := ranger.NewRepository(db)
	rangerService := ranger.NewService(rangerRepository, userService, divisiService)
	rangerHandler := ranger.NewHandler(rangerService)

	rangerPresenceRepository := ranger_presence.NewRepository(db)
	rangerPresenceService := ranger_presence.NewService(rangerPresenceRepository, rangerService, agendaService, divisiService)
	rangerPresenceHandler := ranger_presence.NewHandler(rangerPresenceService)

	go hub.Run()

	router.InitRouter(authMiddleware, userHandler, eventHandler, divisiHandler, presenceHandler, commentHandler, likeHandler, feedbackHandler, wsHandler, agendaHandler, rangerHandler, rangerPresenceHandler)

	// output current time zone
	fmt.Print("Local time zone ")
	fmt.Println(time.Now().Zone())
	fmt.Println(time.Now().Format("2006-01-02T15:04:05.000 MST"))

	host := os.Getenv("HOST")
	router.Start(host)
}
