package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/mamed-gasimov/file-service/internal/config"
	"github.com/mamed-gasimov/file-service/internal/files"
	"github.com/mamed-gasimov/file-service/internal/server"
	miniostorage "github.com/mamed-gasimov/file-service/internal/storage/minio"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// --- Database -----------------------------------------------------------
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.PostgresDSN())
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	log.Println("connected to PostgreSQL")

	if err := runMigrations(cfg.PostgresDSN()); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// --- Object Storage (MinIO) ---------------------------------------------
	store, err := miniostorage.New(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
		cfg.MinioUseSSL,
	)
	if err != nil {
		return fmt.Errorf("init minio: %w", err)
	}

	if err := store.EnsureBucket(context.Background(), cfg.MinioBucket); err != nil {
		return fmt.Errorf("ensure bucket: %w", err)
	}
	log.Printf("MinIO bucket %q is ready\n", cfg.MinioBucket)

	// --- Layers -------------------------------------------------------------
	fileRepo := files.NewFileRepository(pool)
	fileHandler := files.NewFileHandler(fileRepo, store)

	e := server.New(fileHandler)

	// --- Graceful shutdown ---------------------------------------------------
	go func() {
		addr := ":" + cfg.ServerPort
		log.Printf("starting server on %s\n", addr)
		if err := e.Start(addr); err != nil {
			log.Printf("server stopped: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down …")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	return e.Shutdown(shutdownCtx)
}

func runMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	log.Println("migrations applied")
	return nil
}
