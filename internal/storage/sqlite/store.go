package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/Oxyrus/memories/internal/storage"
)

// Store is a SQLite-backed implementation of the storage.Store interface.
type Store struct {
	db     *sql.DB
	albums *albumRepository
	photos *photoRepository
}

// Open initialises (or opens) a SQLite database located at the provided path.
// The directory is created if it does not already exist. The returned Store is
// safe for concurrent use.
func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite: path must not be empty")
	}

	if err := ensureDir(path); err != nil {
		return nil, fmt.Errorf("sqlite: ensure directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := configure(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := bootstrap(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{
		db:     db,
		albums: &albumRepository{db: db},
		photos: &photoRepository{db: db},
	}, nil
}

// Albums returns the album repository.
func (s *Store) Albums() storage.Albums {
	return s.albums
}

// Photos returns the photo repository.
func (s *Store) Photos() storage.Photos {
	return s.photos
}

// Ping verifies the database connection is still alive.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func configure(db *sql.DB) error {
	stmts := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA journal_mode = WAL;",
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("sqlite: configure: %w", err)
		}
	}

	return nil
}

func bootstrap(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS albums (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			cover_photo_id INTEGER,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS photos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			album_id INTEGER NOT NULL,
			filename TEXT NOT NULL,
			caption TEXT NOT NULL DEFAULT '',
			taken_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY(album_id) REFERENCES albums(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_photos_album_id ON photos(album_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_photos_album_filename ON photos(album_id, filename);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("sqlite: bootstrap: %w", err)
		}
	}

	return nil
}

var _ storage.Store = (*Store)(nil)
