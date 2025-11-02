package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Oxyrus/memories/internal/storage"
)

type albumRepository struct {
	db *sql.DB
}

func (r *albumRepository) Create(ctx context.Context, input storage.AlbumCreate) (storage.Album, error) {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO albums (slug, title, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		input.Slug,
		input.Title,
		input.Description,
		now,
		now,
	)
	if err != nil {
		return storage.Album{}, fmt.Errorf("sqlite: create album: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return storage.Album{}, fmt.Errorf("sqlite: create album: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *albumRepository) GetByID(ctx context.Context, id int64) (storage.Album, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, slug, title, description, cover_photo_id, created_at, updated_at
		FROM albums
		WHERE id = ?`,
		id,
	)
	return scanAlbum(row)
}

func (r *albumRepository) GetBySlug(ctx context.Context, slug string) (storage.Album, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, slug, title, description, cover_photo_id, created_at, updated_at
		FROM albums
		WHERE slug = ?`,
		slug,
	)
	return scanAlbum(row)
}

func (r *albumRepository) List(ctx context.Context) ([]storage.Album, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, slug, title, description, cover_photo_id, created_at, updated_at
		FROM albums
		ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list albums: %w", err)
	}
	defer rows.Close()

	var result []storage.Album
	for rows.Next() {
		album, err := scanAlbum(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, album)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: list albums: %w", err)
	}

	return result, nil
}

func (r *albumRepository) Update(ctx context.Context, id int64, input storage.AlbumUpdate) (storage.Album, error) {
	setClauses := make([]string, 0, 3)
	args := make([]any, 0, 4)

	if input.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *input.Title)
	}

	if input.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *input.Description)
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now().UTC())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE albums SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return storage.Album{}, fmt.Errorf("sqlite: update album: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return storage.Album{}, fmt.Errorf("sqlite: update album: %w", err)
	}

	if rowsAffected == 0 {
		return storage.Album{}, storage.ErrNotFound
	}

	return r.GetByID(ctx, id)
}

func (r *albumRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM albums WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("sqlite: delete album: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: delete album: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *albumRepository) SetCoverPhoto(ctx context.Context, albumID, photoID int64) error {
	var exists int
	err := r.db.QueryRowContext(ctx, `
		SELECT 1
		FROM photos
		WHERE id = ? AND album_id = ?`,
		photoID,
		albumID,
	).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return storage.ErrNotFound
		}
		return fmt.Errorf("sqlite: set cover photo: %w", err)
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE albums
		SET cover_photo_id = ?, updated_at = ?
		WHERE id = ?`,
		photoID,
		time.Now().UTC(),
		albumID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: set cover photo: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: set cover photo: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *albumRepository) ClearCoverPhoto(ctx context.Context, albumID int64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE albums
		SET cover_photo_id = NULL, updated_at = ?
		WHERE id = ?`,
		time.Now().UTC(),
		albumID,
	)
	if err != nil {
		return fmt.Errorf("sqlite: clear cover photo: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: clear cover photo: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

type albumScanner interface {
	Scan(dest ...any) error
}

func scanAlbum(s albumScanner) (storage.Album, error) {
	var (
		album        storage.Album
		coverPhotoID sql.NullInt64
		createdAtRaw time.Time
		updatedAtRaw time.Time
	)

	err := s.Scan(
		&album.ID,
		&album.Slug,
		&album.Title,
		&album.Description,
		&coverPhotoID,
		&createdAtRaw,
		&updatedAtRaw,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return storage.Album{}, storage.ErrNotFound
		}
		return storage.Album{}, fmt.Errorf("sqlite: scan album: %w", err)
	}

	if coverPhotoID.Valid {
		v := coverPhotoID.Int64
		album.CoverPhotoID = &v
	}

	album.CreatedAt = createdAtRaw.UTC()
	album.UpdatedAt = updatedAtRaw.UTC()

	return album, nil
}
