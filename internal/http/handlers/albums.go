package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
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
	logger     *slog.Logger
	albums     storage.Albums
	photos     storage.Photos
	uploadsDir string
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const formDateTimeLayout = "2006-01-02T15:04"

func NewAlbumHandler(logger *slog.Logger, albums storage.Albums, photos storage.Photos, uploadsDir string) *AlbumHandler {
	return &AlbumHandler{
		logger:     logger,
		albums:     albums,
		photos:     photos,
		uploadsDir: uploadsDir,
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

	photoRecords, err := h.photos.ListByAlbum(ctx, album.ID)
	if err != nil {
		h.logger.Error("failed to load album photos", "slug", slug, "error", err)
		c.String(http.StatusInternalServerError, "failed to load album photos")
		return
	}

	photos := make([]pages.AlbumPhoto, 0, len(photoRecords))
	for _, photo := range photoRecords {
		photos = append(photos, toAlbumPhoto(photo))
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
		UploadAction: fmt.Sprintf("/albums/%s/photos", album.Slug),
		Photos:       photos,
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

	photoRecords, err := h.photos.ListByAlbum(ctx, album.ID)
	if err != nil {
		h.logger.Error("failed to load album photos", "slug", slug, "error", err)
		c.String(http.StatusInternalServerError, "failed to load album photos")
		return
	}

	viewPhotos := make([]pages.AlbumPhoto, 0, len(photoRecords))
	for _, photo := range photoRecords {
		viewPhotos = append(viewPhotos, toAlbumPhoto(photo))
	}

	data := pages.AlbumViewData{
		Title:       album.Title,
		Slug:        album.Slug,
		Description: album.Description,
		UpdatedAt:   formatTimestamp(album.UpdatedAt),
		Photos:      viewPhotos,
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

func (h *AlbumHandler) UploadPhoto(c *gin.Context) {
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

		h.logger.Error("failed to load album for photo upload", "slug", slug, "error", err)
		c.String(http.StatusInternalServerError, "failed to load album")
		return
	}

	fileHeader, err := c.FormFile("photo")
	if err != nil {
		c.String(http.StatusBadRequest, "photo file is required")
		return
	}

	filename, err := generatePhotoFilename(fileHeader.Filename)
	if err != nil {
		h.logger.Error("failed to generate photo filename", "error", err)
		c.String(http.StatusInternalServerError, "failed to save photo")
		return
	}

	albumDir := filepath.Join(h.uploadsDir, album.Slug)
	if err := os.MkdirAll(albumDir, 0o755); err != nil {
		h.logger.Error("failed to ensure album upload directory", "dir", albumDir, "error", err)
		c.String(http.StatusInternalServerError, "failed to save photo")
		return
	}

	diskPath := filepath.Join(albumDir, filename)
	if err := c.SaveUploadedFile(fileHeader, diskPath); err != nil {
		h.logger.Error("failed to save uploaded file", "path", diskPath, "error", err)
		c.String(http.StatusInternalServerError, "failed to save photo")
		return
	}

	caption := strings.TrimSpace(c.PostForm("caption"))
	takenAtValue := strings.TrimSpace(c.PostForm("taken_at"))
	var takenAt *time.Time
	if takenAtValue != "" {
		parsed, parseErr := time.Parse(formDateTimeLayout, takenAtValue)
		if parseErr != nil {
			_ = os.Remove(diskPath)
			c.String(http.StatusBadRequest, "invalid taken_at format")
			return
		}
		utc := parsed.UTC()
		takenAt = &utc
	}

	storedPath := path.Join(album.Slug, filename)

	_, err = h.photos.Create(ctx, storage.PhotoCreate{
		AlbumID:  album.ID,
		Filename: storedPath,
		Caption:  caption,
		TakenAt:  takenAt,
	})
	if err != nil {
		_ = os.Remove(diskPath)
		h.logger.Error("failed to persist photo metadata", "albumID", album.ID, "error", err)
		c.String(http.StatusInternalServerError, "failed to save photo")
		return
	}

	h.logger.Info("photo uploaded", "albumID", album.ID, "slug", album.Slug, "filename", storedPath)
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/albums/%s/edit", album.Slug))
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

func toAlbumPhoto(photo storage.Photo) pages.AlbumPhoto {
	caption := strings.TrimSpace(photo.Caption)
	if caption == "" {
		caption = path.Base(strings.ReplaceAll(photo.Filename, "\\", "/"))
	}
	item := pages.AlbumPhoto{
		ID:       photo.ID,
		Filename: path.Base(strings.ReplaceAll(photo.Filename, "\\", "/")),
		Caption:  caption,
		URL:      photoURL(photo.Filename),
	}
	if photo.TakenAt != nil {
		item.TakenAt = formatTimestamp(*photo.TakenAt)
	}
	return item
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

func generatePhotoFilename(original string) (string, error) {
	ext := strings.ToLower(filepath.Ext(original))
	const tokenSize = 12
	buf := make([]byte, tokenSize)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	timestamp := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("%s-%s%s", timestamp, token, ext), nil
}

func photoURL(rel string) string {
	clean := strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(rel, "\\", "/")), "/")
	return "/uploads/" + clean
}
