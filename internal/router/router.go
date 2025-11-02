package router

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Oxyrus/memories/internal/config"
	"github.com/Oxyrus/memories/internal/http/handlers"
	"github.com/Oxyrus/memories/internal/http/middleware"
	"github.com/Oxyrus/memories/internal/storage"
)

func New(cfg *config.Config, logger *slog.Logger, store storage.Store) *gin.Engine {
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.Logging(logger))

	albumHandler := handlers.NewAlbumHandler(logger, store.Albums())
	authHandler := handlers.NewAuthHandler(logger, cfg.AdminPassword, cfg.AdminCookie)

	protected := r.Group("/")
	protected.Use(middleware.RequireAdmin(cfg.AdminCookie))
	protected.GET("/albums", albumHandler.List)
	protected.GET("/albums/new", albumHandler.New)
	protected.POST("/albums", albumHandler.Create)
	protected.GET("/albums/:slug/edit", albumHandler.Edit)
	protected.GET("/a/:slug", albumHandler.View)

	r.GET("/login", authHandler.ShowLogin)
	r.POST("/login", authHandler.SubmitLogin)

	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})

	return r
}
