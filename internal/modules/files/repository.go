package files

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repository interface {
	Create(ctx context.Context, f *File) error
	List(ctx context.Context) ([]File, error)
	GetByID(ctx context.Context, id int64) (*File, error)
	UpdateResume(ctx context.Context, id int64, resume string) (*File, error)
	Delete(ctx context.Context, id int64) error
	UpdateTranslationSummary(ctx context.Context, id int64, summary string) error
}

var _ repository = (*FileRepository)(nil)

type FileRepository struct {
	pool *pgxpool.Pool
}

func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

func (r *FileRepository) Create(ctx context.Context, f *File) error {
	query := `
		INSERT INTO files (name, size, mime_type, object_key, resume)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		f.Name, f.Size, f.MimeType, f.ObjectKey, f.Resume,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *FileRepository) List(ctx context.Context) ([]File, error) {
	query := `SELECT id, name, size, mime_type, object_key, created_at, updated_at, resume, translation_summary
	           FROM files ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Name, &f.Size, &f.MimeType, &f.ObjectKey, &f.CreatedAt, &f.UpdatedAt, &f.Resume, &f.TranslationSummary); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		files = append(files, f)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (r *FileRepository) GetByID(ctx context.Context, id int64) (*File, error) {
	query := `SELECT id, name, size, mime_type, object_key, created_at, updated_at, resume, translation_summary
	           FROM files WHERE id = $1`

	var f File
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.Name, &f.Size, &f.MimeType, &f.ObjectKey, &f.CreatedAt, &f.UpdatedAt, &f.Resume, &f.TranslationSummary,
	)
	if err != nil {
		return nil, fmt.Errorf("get file by id: %w", err)
	}

	return &f, nil
}

func (r *FileRepository) UpdateResume(ctx context.Context, id int64, resume string) (*File, error) {
	query := `UPDATE files SET resume = $1, updated_at = NOW()
	           WHERE id = $2
	           RETURNING id, name, size, mime_type, object_key, created_at, updated_at, resume, translation_summary`

	var f File
	err := r.pool.QueryRow(ctx, query, resume, id).Scan(
		&f.ID, &f.Name, &f.Size, &f.MimeType, &f.ObjectKey, &f.CreatedAt, &f.UpdatedAt, &f.Resume, &f.TranslationSummary,
	)
	if err != nil {
		return nil, fmt.Errorf("update resume: %w", err)
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

func (r *FileRepository) UpdateTranslationSummary(ctx context.Context, id int64, summary string) error {
	query := `UPDATE files SET translation_summary = $1, updated_at = NOW() WHERE id = $2`

	ct, err := r.pool.Exec(ctx, query, summary, id)
	if err != nil {
		return fmt.Errorf("update translation summary: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("file with id %d not found", id)
	}

	return nil
}
