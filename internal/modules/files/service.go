package files

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/mamed-gasimov/file-service/internal/modules/analysis"
	"github.com/mamed-gasimov/file-service/internal/storage"
)

const maxAnalysisContentLen = 100_000

type service interface {
	ListFiles(ctx context.Context) ([]File, error)
	UploadFile(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) (*File, error)
	DeleteFile(ctx context.Context, id int64) error
	AnalyzeFile(ctx context.Context, id int64) (*File, error)
}

var _ service = (*FileService)(nil)

type FileService struct {
	repo     repository
	storage  storage.Storage
	analyzer analysis.Provider
}

func NewFileService(repo repository, storage storage.Storage, analyzer analysis.Provider) *FileService {
	return &FileService{
		repo:     repo,
		storage:  storage,
		analyzer: analyzer,
	}
}

func (s *FileService) ListFiles(ctx context.Context) ([]File, error) {
	files, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	if files == nil {
		files = []File{}
	}

	return files, nil
}

func (s *FileService) UploadFile(ctx context.Context, filename string, reader io.Reader, size int64, contentType string) (*File, error) {
	objectKey := generateObjectKey(filename)

	if err := s.storage.Upload(ctx, objectKey, reader, size, contentType); err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	f := &File{
		Name:      filename,
		Size:      size,
		MimeType:  contentType,
		ObjectKey: objectKey,
	}

	if err := s.repo.Create(ctx, f); err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return nil, fmt.Errorf("save file record: %w", err)
	}

	return f, nil
}

func (s *FileService) DeleteFile(ctx context.Context, id int64) error {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if err := s.storage.Delete(ctx, file.ObjectKey); err != nil {
		return fmt.Errorf("delete from storage: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete file record: %w", err)
	}

	return nil
}

func (s *FileService) AnalyzeFile(ctx context.Context, id int64) (*File, error) {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	rc, err := s.storage.Download(ctx, file.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("download from storage: %w", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read file content: %w", err)
	}

	textContent := string(content)
	if len(textContent) > maxAnalysisContentLen {
		textContent = textContent[:maxAnalysisContentLen]
	}

	resume, err := s.analyzer.FileResume(ctx, textContent)
	if err != nil {
		return nil, fmt.Errorf("analyze file: %w", err)
	}

	updated, err := s.repo.UpdateResume(ctx, id, resume)
	if err != nil {
		return nil, fmt.Errorf("save analysis result: %w", err)
	}

	return updated, nil
}

func generateObjectKey(filename string) string {
	return fmt.Sprintf("%s/%s_%s",
		time.Now().Format("2006/01/02"),
		uuid.NewString(),
		filename,
	)
}
