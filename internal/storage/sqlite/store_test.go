package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/Oxyrus/memories/internal/storage"
	"github.com/Oxyrus/memories/internal/storage/sqlite"
)

func TestOpenCreatesSchema(t *testing.T) {
	store := newStore(t)
	defer closeStore(t, store)

	ctx := context.Background()

	albums, err := store.Albums().List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(albums) != 0 {
		t.Fatalf("expected no albums, got %d", len(albums))
	}

	photos, err := store.Photos().ListByAlbum(ctx, 1)
	if err != nil {
		t.Fatalf("ListByAlbum returned error: %v", err)
	}
	if len(photos) != 0 {
		t.Fatalf("expected no photos, got %d", len(photos))
	}
}

func TestAlbumLifecycle(t *testing.T) {
	store := newStore(t)
	defer closeStore(t, store)
	ctx := context.Background()

	created, err := store.Albums().Create(ctx, storage.AlbumCreate{
		Slug:        "summer-roadtrip",
		Title:       "Summer Roadtrip",
		Description: "Driving along the coast.",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if created.ID == 0 {
		t.Fatalf("expected album ID to be set")
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps to be populated")
	}

	_, err = store.Albums().Create(ctx, storage.AlbumCreate{
		Slug:  "summer-roadtrip",
		Title: "Duplicate Slug",
	})
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate slug, got %v", err)
	}

	fetched, err := store.Albums().GetBySlug(ctx, "summer-roadtrip")
	if err != nil {
		t.Fatalf("GetBySlug returned error: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("expected fetched ID %d, got %d", created.ID, fetched.ID)
	}

	items, err := store.Albums().List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 album, got %d", len(items))
	}

	newTitle := "Summer Adventure"
	updated, err := store.Albums().Update(ctx, created.ID, storage.AlbumUpdate{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Title != newTitle {
		t.Fatalf("expected updated title %q, got %q", newTitle, updated.Title)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Fatalf("expected updated_at to be refreshed")
	}

	photo, err := store.Photos().Create(ctx, storage.PhotoCreate{
		AlbumID:  created.ID,
		Filename: "cover.jpg",
		Caption:  "Sunset over the ocean",
	})
	if err != nil {
		t.Fatalf("Photo create returned error: %v", err)
	}

	if err := store.Albums().SetCoverPhoto(ctx, created.ID, photo.ID); err != nil {
		t.Fatalf("SetCoverPhoto returned error: %v", err)
	}

	withCover, err := store.Albums().GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if withCover.CoverPhotoID == nil {
		t.Fatalf("expected cover photo to be set")
	}
	if *withCover.CoverPhotoID != photo.ID {
		t.Fatalf("expected cover photo ID %d, got %d", photo.ID, *withCover.CoverPhotoID)
	}

	if err := store.Albums().ClearCoverPhoto(ctx, created.ID); err != nil {
		t.Fatalf("ClearCoverPhoto returned error: %v", err)
	}

	cleared, err := store.Albums().GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if cleared.CoverPhotoID != nil {
		t.Fatalf("expected cover photo to be cleared")
	}

	if err := store.Albums().Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if _, err := store.Albums().GetByID(ctx, created.ID); err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestPhotosLifecycle(t *testing.T) {
	store := newStore(t)
	defer closeStore(t, store)
	ctx := context.Background()

	album, err := store.Albums().Create(ctx, storage.AlbumCreate{
		Slug:  "city-lights",
		Title: "City Lights",
	})
	if err != nil {
		t.Fatalf("Create album returned error: %v", err)
	}

	takenAt := time.Date(2024, 12, 24, 21, 15, 0, 0, time.UTC)

	first, err := store.Photos().Create(ctx, storage.PhotoCreate{
		AlbumID:  album.ID,
		Filename: "tower.jpg",
		Caption:  "Observation deck",
		TakenAt:  &takenAt,
	})
	if err != nil {
		t.Fatalf("Create photo returned error: %v", err)
	}

	second, err := store.Photos().Create(ctx, storage.PhotoCreate{
		AlbumID:  album.ID,
		Filename: "skyline.jpg",
		Caption:  "Downtown at night",
	})
	if err != nil {
		t.Fatalf("Create photo returned error: %v", err)
	}

	photos, err := store.Photos().ListByAlbum(ctx, album.ID)
	if err != nil {
		t.Fatalf("ListByAlbum returned error: %v", err)
	}
	if len(photos) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(photos))
	}
	if photos[0].ID != first.ID || photos[1].ID != second.ID {
		t.Fatalf("expected ordered photos [%d %d], got [%d %d]", first.ID, second.ID, photos[0].ID, photos[1].ID)
	}

	got, err := store.Photos().GetByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.TakenAt == nil || !got.TakenAt.Equal(takenAt) {
		t.Fatalf("expected TakenAt %v, got %v", takenAt, got.TakenAt)
	}

	if err := store.Photos().Delete(ctx, first.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if _, err := store.Photos().GetByID(ctx, first.ID); err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestSetCoverPhotoValidatesOwnership(t *testing.T) {
	store := newStore(t)
	defer closeStore(t, store)
	ctx := context.Background()

	albumA, err := store.Albums().Create(ctx, storage.AlbumCreate{
		Slug:  "album-a",
		Title: "Album A",
	})
	if err != nil {
		t.Fatalf("create album A: %v", err)
	}
	albumB, err := store.Albums().Create(ctx, storage.AlbumCreate{
		Slug:  "album-b",
		Title: "Album B",
	})
	if err != nil {
		t.Fatalf("create album B: %v", err)
	}

	photo, err := store.Photos().Create(ctx, storage.PhotoCreate{
		AlbumID:  albumB.ID,
		Filename: "photo.jpg",
		Caption:  "In album B",
	})
	if err != nil {
		t.Fatalf("create photo: %v", err)
	}

	if err := store.Albums().SetCoverPhoto(ctx, albumA.ID, photo.ID); err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound when using foreign photo, got %v", err)
	}
}

func newStore(t *testing.T) storage.Store {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "memories.db")

	store, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	return store
}

func closeStore(t *testing.T, store storage.Store) {
	t.Helper()
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}
