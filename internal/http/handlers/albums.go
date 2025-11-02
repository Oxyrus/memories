package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Oxyrus/memories/internal/http/render"
	"github.com/Oxyrus/memories/internal/storage"
	"github.com/Oxyrus/memories/web/pages"
)

type AlbumHandler struct {
	logger *slog.Logger
	albums storage.Albums
}

func NewAlbumHandler(logger *slog.Logger, albums storage.Albums) *AlbumHandler {
	return &AlbumHandler{
		logger: logger,
		albums: albums,
	}
}

func (h *AlbumHandler) List(c *gin.Context) {
	render.HTML(c, http.StatusOK, pages.AlbumsList())
}

func (h *AlbumHandler) New(c *gin.Context) {
	c.String(http.StatusNotImplemented, "/albums/new not implemented")
}

func (h *AlbumHandler) Edit(c *gin.Context) {
	c.String(http.StatusNotImplemented, "/albums/:albumId/edit not implemented")
}

func (h *AlbumHandler) View(c *gin.Context) {
	c.String(http.StatusNotImplemented, "/a/:albumId not implemented")
}
