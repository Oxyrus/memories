package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"

	"github.com/Oxyrus/memories/internal/http/render"
	"github.com/Oxyrus/memories/internal/storage"
	"github.com/Oxyrus/memories/web/pages"
)

type AlbumHandler struct {
	logger *slog.Logger
	albums storage.Albums
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func NewAlbumHandler(logger *slog.Logger, albums storage.Albums) *AlbumHandler {
	return &AlbumHandler{
		logger: logger,
		albums: albums,
	}
}

func (h *AlbumHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	albums, err := h.albums.List(ctx)
	if err != nil {
		h.logger.Error("failed to list albums", "error", err)
		c.String(http.StatusInternalServerError, "failed to load albums")
		return
	}

	items := make([]pages.AlbumListItem, 0, len(albums))
	for _, album := range albums {
		items = append(items, toAlbumListItem(album))
	}

	render.HTML(c, http.StatusOK, pages.AlbumsList(items))
}

func (h *AlbumHandler) New(c *gin.Context) {
	form := pages.AlbumForm{
		Errors: map[string]string{},
	}
	render.HTML(c, http.StatusOK, pages.AlbumNew(form))
}

func (h *AlbumHandler) Edit(c *gin.Context) {
	c.String(http.StatusNotImplemented, "/albums/:slug/edit not implemented")
}

func (h *AlbumHandler) View(c *gin.Context) {
	c.String(http.StatusNotImplemented, "/a/:slug not implemented")
}

func (h *AlbumHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	form := pages.AlbumForm{
		Title:       strings.TrimSpace(c.PostForm("title")),
		Slug:        strings.TrimSpace(c.PostForm("slug")),
		Description: strings.TrimSpace(c.PostForm("description")),
		Errors:      map[string]string{},
	}

	if form.Title == "" {
		form.Errors["title"] = "Title is required."
	}

	var slug string
	if form.Slug != "" {
		manual := form.Slug
		if !slugPattern.MatchString(strings.ToLower(manual)) {
			form.Errors["slug"] = "Slug may only contain letters, numbers, and hyphens."
		} else {
			slug = slugify(manual)
			form.Slug = slug
		}
	} else {
		slug = slugify(form.Title)
		form.Slug = slug
	}

	if slug == "" {
		if _, ok := form.Errors["slug"]; !ok {
			form.Errors["slug"] = "Slug may only contain letters, numbers, and hyphens."
		}
	}

	if len(form.Errors) > 0 {
		render.HTML(c, http.StatusUnprocessableEntity, pages.AlbumNew(form))
		return
	}

	album, err := h.albums.Create(ctx, storage.AlbumCreate{
		Slug:        slug,
		Title:       form.Title,
		Description: form.Description,
	})
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			form.Errors["slug"] = "An album with that slug already exists."
			render.HTML(c, http.StatusUnprocessableEntity, pages.AlbumNew(form))
			return
		}

		h.logger.Error("failed to create album", "error", err)
		c.String(http.StatusInternalServerError, "failed to create album")
		return
	}

	h.logger.Info("album created", "albumID", album.ID, "slug", album.Slug)
	c.Redirect(http.StatusSeeOther, "/albums")
}

func toAlbumListItem(album storage.Album) pages.AlbumListItem {
	meta := ""
	if !album.UpdatedAt.IsZero() {
		meta = fmt.Sprintf("Updated %s", album.UpdatedAt.UTC().Format(time.RFC1123))
	}

	return pages.AlbumListItem{
		Title:       album.Title,
		Description: album.Description,
		Href:        fmt.Sprintf("/a/%s", album.Slug),
		Meta:        meta,
	}
}

func slugify(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(value))

	prevHyphen := false

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			prevHyphen = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(unicode.ToLower(r))
			prevHyphen = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case r == '-' && !prevHyphen && b.Len() > 0:
			b.WriteRune('-')
			prevHyphen = true
		default:
			if !prevHyphen && b.Len() > 0 {
				b.WriteRune('-')
				prevHyphen = true
			}
		}
	}

	result := strings.Trim(b.String(), "-")
	return result
}
