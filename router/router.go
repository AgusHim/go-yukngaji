package router

import (
	"mainyuk/internal/auth"
	"mainyuk/internal/divisi"
	"mainyuk/internal/event"
	"mainyuk/internal/presence"
	"mainyuk/internal/user"
	"mainyuk/internal/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var r *gin.Engine

func InitRouter(authMiddleware auth.Middleware, userHandler user.Handler, eventHandler event.Handler, divisiHandler divisi.Handler, presenceHandler presence.Handler, wsHandler *ws.Handler) {
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

	r.POST("/ws/createRoom", wsHandler.CrateRoom)
	r.GET("/ws/joinRoom/:roomId", wsHandler.JoinRoom)
	r.GET("/ws/getRooms", wsHandler.GetRooms)
	r.GET("/ws/getClients/:roomId", wsHandler.GetClients)
}

func Start(addr string) error {
	return r.Run(addr)
}
