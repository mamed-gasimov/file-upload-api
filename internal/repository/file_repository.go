package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mamed-gasimov/file-service/internal/model"
)

type FileRepository struct {
	pool *pgxpool.Pool
}

func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

func (r *FileRepository) Create(ctx context.Context, f *model.File) error {
	query := `
		INSERT INTO files (name, size, mime_type, object_key)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		f.Name, f.Size, f.MimeType, f.ObjectKey,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *FileRepository) List(ctx context.Context) ([]model.File, error) {
	query := `SELECT id, name, size, mime_type, object_key, created_at, updated_at
	           FROM files ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(&f.ID, &f.Name, &f.Size, &f.MimeType, &f.ObjectKey, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}

	return files, rows.Err()
}

func (r *FileRepository) GetByID(ctx context.Context, id int64) (*model.File, error) {
	query := `SELECT id, name, size, mime_type, object_key, created_at, updated_at
	           FROM files WHERE id = $1`

	var f model.File
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.Name, &f.Size, &f.MimeType, &f.ObjectKey, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get file by id: %w", err)
	}

	return &f, nil
}

func (r *FileRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM files WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("file with id %d not found", id)
	}

	return nil
}
