package router

import (
	"log"
	"mainyuk/internal/auth"
	"mainyuk/internal/comment"
	"mainyuk/internal/divisi"
	"mainyuk/internal/event"
	"mainyuk/internal/feedback"
	"mainyuk/internal/like"
	"mainyuk/internal/presence"
	"mainyuk/internal/user"
	"mainyuk/internal/ws"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var r *gin.Engine

func InitRouter(authMiddleware auth.Middleware, userHandler user.Handler, eventHandler event.Handler, divisiHandler divisi.Handler, presenceHandler presence.Handler, commentHandler comment.Handler, likeHandler like.Handler, feedbackHandler feedback.Handler, wsHandler *ws.Handler) {
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
	api.POST("/register", userHandler.Register)
	api.POST("/login", userHandler.Login)

	api.POST("/events", authMiddleware.AuthAdmin, eventHandler.Create)
	api.GET("/events/:slug", eventHandler.Show)
	api.GET("/events/code/:code", eventHandler.ShowByCode)
	api.GET("/events", authMiddleware.AuthAdmin, eventHandler.Index)

	api.POST("/divisi", authMiddleware.AuthAdmin, divisiHandler.Create)
	api.GET("/divisi/:slug", authMiddleware.AuthAdmin, divisiHandler.Show)
	api.GET("/divisi", authMiddleware.AuthAdmin, divisiHandler.Index)

	api.POST("/presence", presenceHandler.Create)
	api.GET("/presence/:slug", presenceHandler.Show)
	api.GET("/presence", authMiddleware.AuthAdmin, presenceHandler.Index)

	api.POST("/comments", commentHandler.Create)
	api.GET("/comments", commentHandler.Index)

	api.GET("/comments/like", likeHandler.Index)
	api.POST("/comments/like", likeHandler.Create)
	api.DELETE("/comments/like/:id", likeHandler.Delete)

	api.GET("/feedback", authMiddleware.AuthAdmin, feedbackHandler.Index)
	api.POST("/feedback", feedbackHandler.Create)

	r.GET("/ws/events/:id", wsHandler.ConnectWS)
}

func Start(addr string) error {
	log.Printf("Server runing on %s", addr)
	return r.Run(addr)
}
