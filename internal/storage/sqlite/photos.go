package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Oxyrus/memories/internal/storage"
)

type photoRepository struct {
	db *sql.DB
}

func (r *photoRepository) Create(ctx context.Context, input storage.PhotoCreate) (storage.Photo, error) {
	now := time.Now().UTC()

	var takenAt sql.NullTime
	if input.TakenAt != nil {
		utc := input.TakenAt.UTC()
		takenAt = sql.NullTime{Time: utc, Valid: true}
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO photos (album_id, filename, caption, taken_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		input.AlbumID,
		input.Filename,
		input.Caption,
		takenAt,
		now,
		now,
	)
	if err != nil {
		return storage.Photo{}, fmt.Errorf("sqlite: create photo: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return storage.Photo{}, fmt.Errorf("sqlite: create photo: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *photoRepository) GetByID(ctx context.Context, id int64) (storage.Photo, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, album_id, filename, caption, taken_at, created_at, updated_at
		FROM photos
		WHERE id = ?`,
		id,
	)
	return scanPhoto(row)
}

func (r *photoRepository) ListByAlbum(ctx context.Context, albumID int64) ([]storage.Photo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, album_id, filename, caption, taken_at, created_at, updated_at
		FROM photos
		WHERE album_id = ?
		ORDER BY taken_at IS NULL, taken_at, created_at, id`,
		albumID,
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list photos: %w", err)
	}
	defer rows.Close()

	var result []storage.Photo
	for rows.Next() {
		photo, err := scanPhoto(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, photo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: list photos: %w", err)
	}

	return result, nil
}

func (r *photoRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM photos WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("sqlite: delete photo: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: delete photo: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

type photoScanner interface {
	Scan(dest ...any) error
}

func scanPhoto(s photoScanner) (storage.Photo, error) {
	var (
		photo        storage.Photo
		takenAtRaw   sql.NullTime
		createdAtRaw time.Time
		updatedAtRaw time.Time
	)

	err := s.Scan(
		&photo.ID,
		&photo.AlbumID,
		&photo.Filename,
		&photo.Caption,
		&takenAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return storage.Photo{}, storage.ErrNotFound
		}
		return storage.Photo{}, fmt.Errorf("sqlite: scan photo: %w", err)
	}

	if takenAtRaw.Valid {
		t := takenAtRaw.Time.UTC()
		photo.TakenAt = &t
	}

	photo.CreatedAt = createdAtRaw.UTC()
	photo.UpdatedAt = updatedAtRaw.UTC()

	return photo, nil
}
