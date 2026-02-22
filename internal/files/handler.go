package files

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mamed-gasimov/file-service/internal/storage"
)

type FileHandler struct {
	repo    *FileRepository
	storage storage.Storage
}

func NewFileHandler(repo *FileRepository, storage storage.Storage) *FileHandler {
	return &FileHandler{
		repo:    repo,
		storage: storage,
	}
}

// ListFiles GET /api/files — returns a list of all uploaded files.
func (h *FileHandler) ListFiles(c echo.Context) error {
	files, err := h.repo.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("list files: %v", err))
	}

	if files == nil {
		files = []File{}
	}

	err = c.JSON(http.StatusOK, files)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("list files, json marshalling: %v", err))
	}

	return nil
}

// UploadFile POST /api/files — uploads a file to S3 via streaming and saves metadata to DB.
func (h *FileHandler) UploadFile(c echo.Context) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "field 'file' is required")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot open uploaded file")
	}
	defer src.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	objectKey := fmt.Sprintf("%s/%s_%s",
		time.Now().Format("2006/01/02"),
		uuid.NewString(),
		fileHeader.Filename,
	)

	// Stream the file body directly to MinIO (no temp file on disk).
	if err := h.storage.Upload(c.Request().Context(), objectKey, src, fileHeader.Size, contentType); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("upload to storage: %v", err))
	}

	f := &File{
		Name:      fileHeader.Filename,
		Size:      fileHeader.Size,
		MimeType:  contentType,
		ObjectKey: objectKey,
	}

	if err := h.repo.Create(c.Request().Context(), f); err != nil {
		// best-effort cleanup of the uploaded object
		_ = h.storage.Delete(c.Request().Context(), objectKey)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("save file record: %v", err))
	}

	return c.JSON(http.StatusCreated, f)
}

// DeleteFile DELETE /api/files/:id — removes file from S3 and database.
func (h *FileHandler) DeleteFile(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file id")
	}

	file, err := h.repo.GetByID(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("file not found: %v", err))
	}

	if err := h.storage.Delete(c.Request().Context(), file.ObjectKey); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("delete from storage: %v", err))
	}

	if err := h.repo.Delete(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("delete file record: %v", err))
	}

	return c.NoContent(http.StatusNoContent)
}
