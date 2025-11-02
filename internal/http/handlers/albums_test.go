package handlers_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Oxyrus/memories/internal/http/handlers"
	"github.com/Oxyrus/memories/internal/storage"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAlbumHandlerListSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	req, err := http.NewRequest(http.MethodGet, "/albums", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	ctx.Request = req

	albums := &stubAlbums{
		list: []storage.Album{
			{
				ID:          1,
				Title:       "Summer Roadtrip",
				Description: "Sunset drives along the coast.",
				Slug:        "summer-roadtrip",
				UpdatedAt:   time.Date(2025, 2, 15, 10, 30, 0, 0, time.UTC),
			},
		},
	}

	handler := handlers.NewAlbumHandler(newTestLogger(), albums)

	handler.List(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Summer Roadtrip") {
		t.Fatalf("response body missing album title: %s", body)
	}
	if !strings.Contains(body, "/a/summer-roadtrip") {
		t.Fatalf("response body missing album link: %s", body)
	}
}

func TestAlbumHandlerListError(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	req, err := http.NewRequest(http.MethodGet, "/albums", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	ctx.Request = req

	albums := &stubAlbums{
		listErr: errors.New("boom"),
	}

	handler := handlers.NewAlbumHandler(newTestLogger(), albums)

	handler.List(ctx)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "failed to load albums" {
		t.Fatalf("unexpected response body: %q", rec.Body.String())
	}
}

func TestAlbumHandlerNew(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	req, err := http.NewRequest(http.MethodGet, "/albums/new", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	ctx.Request = req

	handler := handlers.NewAlbumHandler(newTestLogger(), &stubAlbums{})
	handler.New(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `form method="post" action="/albums"`) {
		t.Fatalf("expected form action, got %s", body)
	}
}

func TestAlbumHandlerCreateSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	form := make(url.Values)
	form.Set("title", "Summer Roadtrip")
	form.Set("description", "Sunset drives.")

	req := httptest.NewRequest(http.MethodPost, "/albums", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx.Request = req

	albums := &stubAlbums{
		createResp: storage.Album{
			ID:    42,
			Slug:  "summer-roadtrip",
			Title: "Summer Roadtrip",
		},
	}

	handler := handlers.NewAlbumHandler(newTestLogger(), albums)
	handler.Create(ctx)

	ctx.Writer.WriteHeaderNow()

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status, got %d (location=%q body=%q, createCalled=%v)", rec.Code, rec.Header().Get("Location"), rec.Body.String(), albums.createCalled)
	}
	if location := rec.Header().Get("Location"); location != "/albums" {
		t.Fatalf("expected redirect to /albums, got %q", location)
	}
	if !albums.createCalled {
		t.Fatalf("expected Create to be called")
	}
	if albums.lastCreate.Slug != "summer-roadtrip" {
		t.Fatalf("expected slug summer-roadtrip, got %q", albums.lastCreate.Slug)
	}
}

func TestAlbumHandlerCreateValidationError(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	form := make(url.Values)
	form.Set("title", "")
	form.Set("slug", "Invalid Slug!!")

	req := httptest.NewRequest(http.MethodPost, "/albums", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx.Request = req

	albums := &stubAlbums{}
	handler := handlers.NewAlbumHandler(newTestLogger(), albums)
	handler.Create(ctx)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Title is required.") {
		t.Fatalf("expected title error, got %s", body)
	}
	if !strings.Contains(body, "Slug may only contain letters, numbers, and hyphens.") {
		t.Fatalf("expected slug error, got %s", body)
	}
	if albums.createCalled {
		t.Fatalf("Create should not have been called on validation failure")
	}
}

func TestAlbumHandlerCreateConflict(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)

	form := make(url.Values)
	form.Set("title", "Summer Roadtrip")
	form.Set("slug", "summer-roadtrip")

	req := httptest.NewRequest(http.MethodPost, "/albums", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx.Request = req

	albums := &stubAlbums{
		createErr: storage.ErrConflict,
	}

	handler := handlers.NewAlbumHandler(newTestLogger(), albums)
	handler.Create(ctx)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "An album with that slug already exists.") {
		t.Fatalf("expected conflict message, got %s", body)
	}
	if !albums.createCalled {
		t.Fatalf("expected Create to be called")
	}
}

type stubAlbums struct {
	list         []storage.Album
	listErr      error
	createResp   storage.Album
	createErr    error
	createCalled bool
	lastCreate   storage.AlbumCreate
}

func (s *stubAlbums) Create(_ context.Context, input storage.AlbumCreate) (storage.Album, error) {
	s.createCalled = true
	s.lastCreate = input
	if s.createErr != nil {
		return storage.Album{}, s.createErr
	}
	return s.createResp, nil
}

func (s *stubAlbums) GetByID(context.Context, int64) (storage.Album, error) {
	panic("unexpected call to GetByID")
}

func (s *stubAlbums) GetBySlug(context.Context, string) (storage.Album, error) {
	panic("unexpected call to GetBySlug")
}

func (s *stubAlbums) List(context.Context) ([]storage.Album, error) {
	return s.list, s.listErr
}

func (s *stubAlbums) Update(context.Context, int64, storage.AlbumUpdate) (storage.Album, error) {
	panic("unexpected call to Update")
}

func (s *stubAlbums) Delete(context.Context, int64) error {
	panic("unexpected call to Delete")
}

func (s *stubAlbums) SetCoverPhoto(context.Context, int64, int64) error {
	panic("unexpected call to SetCoverPhoto")
}

func (s *stubAlbums) ClearCoverPhoto(context.Context, int64) error {
	panic("unexpected call to ClearCoverPhoto")
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}
