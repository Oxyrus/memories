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
		Heading:      "Create a new album",
		Intro:        "Collect your photos under a memorable title. You can upload pictures after saving the basics.",
		Action:       "/albums",
		SubmitLabel:  "Create album",
		SlugEditable: true,
		Errors:       map[string]string{},
	}
	render.HTML(c, http.StatusOK, pages.AlbumNew(form))
}

func (h *AlbumHandler) Edit(c *gin.Context) {
	ctx := c.Request.Context()
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		c.String(http.StatusNotFound, "album not found")
		return
	}

	album, err := h.albums.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.String(http.StatusNotFound, "album not found")
			return
		}
		h.logger.Error("failed to load album for edit", "slug", slug, "error", err)
		c.String(http.StatusInternalServerError, "failed to load album")
		return
	}

	form := pages.AlbumForm{
		Heading:      "Edit album",
		Intro:        "Update the album details below.",
		Action:       fmt.Sprintf("/albums/%s/edit", album.Slug),
		SubmitLabel:  "Save changes",
		Title:        album.Title,
		Slug:         album.Slug,
		Description:  album.Description,
		Errors:       map[string]string{},
		SlugEditable: false,
	}

	render.HTML(c, http.StatusOK, pages.AlbumEdit(form))
}

func (h *AlbumHandler) View(c *gin.Context) {
	ctx := c.Request.Context()
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		c.String(http.StatusNotFound, "album not found")
		return
	}

	album, err := h.albums.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.String(http.StatusNotFound, "album not found")
			return
		}
		h.logger.Error("failed to load album", "slug", slug, "error", err)
		c.String(http.StatusInternalServerError, "failed to load album")
		return
	}

	data := pages.AlbumViewData{
		Title:       album.Title,
		Slug:        album.Slug,
		Description: album.Description,
		UpdatedAt:   formatTimestamp(album.UpdatedAt),
	}

	render.HTML(c, http.StatusOK, pages.AlbumView(data))
}

func (h *AlbumHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	form := pages.AlbumForm{
		Heading:      "Create a new album",
		Intro:        "Collect your photos under a memorable title. You can upload pictures after saving the basics.",
		Action:       "/albums",
		SubmitLabel:  "Create album",
		SlugEditable: true,
		Title:        strings.TrimSpace(c.PostForm("title")),
		Slug:         strings.TrimSpace(c.PostForm("slug")),
		Description:  strings.TrimSpace(c.PostForm("description")),
		Errors:       map[string]string{},
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

func (h *AlbumHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		c.String(http.StatusNotFound, "album not found")
		return
	}

	current, err := h.albums.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.String(http.StatusNotFound, "album not found")
			return
		}
		h.logger.Error("failed to load album for update", "slug", slug, "error", err)
		c.String(http.StatusInternalServerError, "failed to load album")
		return
	}

	form := pages.AlbumForm{
		Heading:      "Edit album",
		Intro:        "Update the album details below.",
		Action:       fmt.Sprintf("/albums/%s/edit", current.Slug),
		SubmitLabel:  "Save changes",
		Title:        strings.TrimSpace(c.PostForm("title")),
		Slug:         current.Slug,
		Description:  strings.TrimSpace(c.PostForm("description")),
		Errors:       map[string]string{},
		SlugEditable: false,
	}

	if form.Title == "" {
		form.Errors["title"] = "Title is required."
	}

	if len(form.Errors) > 0 {
		render.HTML(c, http.StatusUnprocessableEntity, pages.AlbumEdit(form))
		return
	}

	title := form.Title
	description := form.Description
	updateInput := storage.AlbumUpdate{
		Title:       &title,
		Description: &description,
	}

	updated, err := h.albums.Update(ctx, current.ID, updateInput)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			c.String(http.StatusNotFound, "album not found")
			return
		}

		h.logger.Error("failed to update album", "albumID", current.ID, "slug", current.Slug, "error", err)
		c.String(http.StatusInternalServerError, "failed to update album")
		return
	}

	h.logger.Info("album updated", "albumID", updated.ID, "slug", updated.Slug)
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/albums/%s", updated.Slug))
}

func toAlbumListItem(album storage.Album) pages.AlbumListItem {
	meta := ""
	if ts := formatTimestamp(album.UpdatedAt); ts != "" {
		meta = fmt.Sprintf("Updated %s", ts)
	}

	return pages.AlbumListItem{
		Title:       album.Title,
		Description: album.Description,
		Href:        fmt.Sprintf("/albums/%s", album.Slug),
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

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("Jan 2, 2006 15:04 MST")
}
