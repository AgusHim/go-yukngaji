package router

import (
	"log"
	"mainyuk/internal/agenda"
	"mainyuk/internal/auth"
	"mainyuk/internal/comment"
	"mainyuk/internal/divisi"
	"mainyuk/internal/event"
	"mainyuk/internal/feedback"
	"mainyuk/internal/like"
	"mainyuk/internal/order"
	"mainyuk/internal/presence"
	"mainyuk/internal/ranger"
	"mainyuk/internal/ranger_presence"
	"mainyuk/internal/ticket"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"mainyuk/internal/ws"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var r *gin.Engine

func InitRouter(
	authMiddleware auth.Middleware,
	userHandler user.Handler,
	eventHandler event.Handler,
	divisiHandler divisi.Handler,
	presenceHandler presence.Handler,
	commentHandler comment.Handler,
	likeHandler like.Handler,
	feedbackHandler feedback.Handler,
	wsHandler *ws.Handler,
	agendaHandler agenda.Handler,
	rangerHandler ranger.Handler,
	rangerPresenceHandler ranger_presence.Handler,
	orderHandler order.Handler,
	ticketHandler ticket.Handler,
	userTicketPresenceHandler user_ticket.Handler,
) {
	mode := os.Getenv("GIN_MODE")
	gin.SetMode(mode)

	r = gin.Default()
	config := cors.DefaultConfig()
	config.AllowCredentials = true
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Authorization", "Content-Type"}
	r.Use(cors.New(config))

	api := r.Group("api")
	user_api := r.Group("user_api")
	ranger_api := r.Group("ranger_api")
	admin_api := r.Group("admin_api")

	api.POST("/register", userHandler.Register)
	api.POST("/login", userHandler.Login)
	user_api.PUT("/auth", authMiddleware.AuthUser, userHandler.UpdateAuth)
	admin_api.PUT("/users/:id", authMiddleware.AuthPJ, userHandler.UpdateByAdmin)
	admin_api.GET("/users/:id", authMiddleware.AuthPJ, userHandler.Show)

	api.POST("/events", authMiddleware.AuthAdmin, eventHandler.Create)
	api.GET("/events/:slug", eventHandler.Show)
	api.GET("/events/code/:code", eventHandler.ShowByCode)
	api.GET("/events", eventHandler.Index)
	api.PUT("/events/:id", authMiddleware.AuthAdmin, eventHandler.Create)

	admin_api.POST("/divisi", authMiddleware.AuthPJ, divisiHandler.Create)
	admin_api.GET("/divisi/:slug", authMiddleware.AuthPJ, divisiHandler.Show)
	admin_api.GET("/divisi", authMiddleware.AuthPJ, divisiHandler.Index)

	api.POST("/presence", presenceHandler.Create)
	api.GET("/presence/:slug", presenceHandler.Show)
	api.GET("/presence", authMiddleware.AuthAdmin, presenceHandler.Index)
	user_api.GET("/presence", authMiddleware.AuthUser, presenceHandler.Index)

	api.POST("/comments", commentHandler.Create)
	api.GET("/comments", commentHandler.Index)

	api.GET("/comments/like", likeHandler.Index)
	api.POST("/comments/like", likeHandler.Create)
	api.DELETE("/comments/like/:id", likeHandler.Delete)

	api.GET("/feedback", authMiddleware.AuthAdmin, feedbackHandler.Index)
	api.POST("/feedback", feedbackHandler.Create)

	admin_api.POST("/agenda", authMiddleware.AuthPJ, agendaHandler.Create)
	admin_api.GET("/agenda/:id", authMiddleware.AuthPJ, agendaHandler.Show)
	admin_api.GET("/agenda", authMiddleware.AuthPJ, agendaHandler.Index)
	admin_api.PUT("/agenda/:id", authMiddleware.AuthPJ, agendaHandler.Update)
	admin_api.DELETE("/agenda/:id", authMiddleware.AuthPJ, agendaHandler.Delete)

	admin_api.POST("/rangers", authMiddleware.AuthPJ, rangerHandler.Create)
	ranger_api.GET("/rangers/me", authMiddleware.AuthRanger, rangerHandler.Show)
	admin_api.GET("/rangers/:id", authMiddleware.AuthPJ, rangerHandler.Show)
	admin_api.GET("/rangers", authMiddleware.AuthPJ, rangerHandler.Index)
	admin_api.PUT("/rangers/:id", authMiddleware.AuthPJ, rangerHandler.Update)
	admin_api.DELETE("/rangers/:id", authMiddleware.AuthPJ, rangerHandler.Delete)

	admin_api.POST("/rangers/presence", authMiddleware.AuthPJ, rangerPresenceHandler.Create)
	admin_api.GET("/rangers/presence/:id", authMiddleware.AuthPJ, rangerPresenceHandler.Show)

	admin_api.GET("/rangers/presence", authMiddleware.AuthPJ, rangerPresenceHandler.Index)
	ranger_api.GET("/rangers/presence", authMiddleware.AuthRanger, rangerPresenceHandler.Index)

	/* Tickets */
	api.GET("/tickets", authMiddleware.AuthAdmin, ticketHandler.Index)
	api.POST("/tickets", authMiddleware.AuthAdmin, ticketHandler.Create)

	/* Orders */
	api.GET("/orders", authMiddleware.AuthUser, orderHandler.Index)
	api.POST("/orders", authMiddleware.AuthUser, orderHandler.Create)
	api.GET("/orders/:public_id", orderHandler.ShowByPublicID)
	admin_api.GET("/orders", authMiddleware.AuthAdmin, orderHandler.Index)
	admin_api.GET("/orders/:public_id", orderHandler.ShowByPublicID)

	r.GET("/ws/events/:id", wsHandler.ConnectWS)
}

func Start(addr string) error {
	log.Printf("Server runing on %s", addr)
	return r.Run(addr)
}
