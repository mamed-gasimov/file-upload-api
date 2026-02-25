package files

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/mamed-gasimov/file-service/internal/modules/analysis"
	"github.com/mamed-gasimov/file-service/internal/storage"
)

const maxAnalysisContentLen = 100_000

type FileHandler struct {
	repo     *FileRepository
	storage  storage.Storage
	analyzer analysis.Provider
}

func NewFileHandler(repo *FileRepository, storage storage.Storage, analyzer analysis.Provider) *FileHandler {
	return &FileHandler{
		repo:     repo,
		storage:  storage,
		analyzer: analyzer,
	}
}

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
		_ = h.storage.Delete(c.Request().Context(), objectKey)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("save file record: %v", err))
	}

	return c.JSON(http.StatusCreated, f)
}

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

// AnalyzeFile POST /api/files/analyze — uploads a file, generates an OpenAI overview, and stores both.
func (h *FileHandler) AnalyzeFile(c echo.Context) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "field 'file' is required")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot open uploaded file")
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "cannot read uploaded file")
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	objectKey := fmt.Sprintf("%s/%s_%s",
		time.Now().Format("2006/01/02"),
		uuid.NewString(),
		fileHeader.Filename,
	)

	ctx := c.Request().Context()

	if err := h.storage.Upload(ctx, objectKey, bytes.NewReader(content), fileHeader.Size, contentType); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("upload to storage: %v", err))
	}

	textContent := string(content)
	if len(textContent) > maxAnalysisContentLen {
		textContent = textContent[:maxAnalysisContentLen]
	}

	resume, err := h.analyzer.FileResume(ctx, textContent)
	if err != nil {
		_ = h.storage.Delete(ctx, objectKey)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("analyze file: %v", err))
	}

	f := &File{
		Name:      fileHeader.Filename,
		Size:      fileHeader.Size,
		MimeType:  contentType,
		ObjectKey: objectKey,
		Resume:    &resume,
	}

	if err := h.repo.Create(ctx, f); err != nil {
		_ = h.storage.Delete(ctx, objectKey)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("save file record: %v", err))
	}

	return c.JSON(http.StatusCreated, f)
}
