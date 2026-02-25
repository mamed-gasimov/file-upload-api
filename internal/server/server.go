package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mamed-gasimov/file-service/internal/modules/files"
)

func New(fileHandler *files.FileHandler) *echo.Echo {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	api := e.Group("/api")
	{
		api.GET("/files", fileHandler.ListFiles)
		api.POST("/files", fileHandler.UploadFile)
		api.POST("/files/analyze", fileHandler.AnalyzeFile)
		api.DELETE("/files/:id", fileHandler.DeleteFile)
	}

	return e
}
