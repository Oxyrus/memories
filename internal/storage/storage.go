package storage

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound indicates that the requested entity does not exist in the
// underlying storage.
var ErrNotFound = errors.New("storage: not found")

// Store exposes the persistence primitives required by the application. It is
// expected to be safe for concurrent use.
type Store interface {
	Albums() Albums
	Photos() Photos
	Ping(ctx context.Context) error
	Close() error
}

// Album represents a logical collection of photos.
type Album struct {
	ID           int64
	Slug         string
	Title        string
	Description  string
	CoverPhotoID *int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AlbumCreate captures the data required to create a new album.
type AlbumCreate struct {
	Slug        string
	Title       string
	Description string
}

// AlbumUpdate describes the mutable fields for an album. A nil field indicates
// that no update should be applied for that attribute.
type AlbumUpdate struct {
	Title       *string
	Description *string
}

// Albums defines the operations supported for managing albums.
type Albums interface {
	Create(ctx context.Context, input AlbumCreate) (Album, error)
	GetByID(ctx context.Context, id int64) (Album, error)
	GetBySlug(ctx context.Context, slug string) (Album, error)
	List(ctx context.Context) ([]Album, error)
	Update(ctx context.Context, id int64, input AlbumUpdate) (Album, error)
	Delete(ctx context.Context, id int64) error
	SetCoverPhoto(ctx context.Context, albumID, photoID int64) error
	ClearCoverPhoto(ctx context.Context, albumID int64) error
}

// Photo is a single image that belongs to an album.
type Photo struct {
	ID        int64
	AlbumID   int64
	Filename  string
	Caption   string
	TakenAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PhotoCreate contains the data required to insert a new photo.
type PhotoCreate struct {
	AlbumID  int64
	Filename string
	Caption  string
	TakenAt  *time.Time
}

// Photos defines the operations supported for managing photos.
type Photos interface {
	Create(ctx context.Context, input PhotoCreate) (Photo, error)
	GetByID(ctx context.Context, id int64) (Photo, error)
	ListByAlbum(ctx context.Context, albumID int64) ([]Photo, error)
	Delete(ctx context.Context, id int64) error
}
