package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"kirta-backend-api/internal/api"
	"kirta-backend-api/internal/api/routes"
	"kirta-backend-api/internal/config"
	"kirta-backend-api/internal/persistance/db"
	"kirta-backend-api/internal/service"
	"kirta-backend-api/internal/service/exploitability"
	"kirta-backend-api/internal/service/sca"
	"kirta-backend-api/internal/storage"
	"kirta-backend-api/migrations"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type App struct {
	logFile *os.File
	pool    *pgxpool.Pool
	s       *http.Server
}

type Runner interface {
	Run() error
	Shutdown(ctx context.Context) error
}

func (a *App) Run() error {
	err := a.s.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	serverErr := a.s.Shutdown(ctx)
	if a.pool != nil {
		a.pool.Close()
	}
	var logErr error
	if a.logFile != nil {
		logErr = a.logFile.Close()
	}
	return errors.Join(serverErr, logErr)
}

func runMigrations(pool *pgxpool.Pool) error {
	srcDriver, err := iofs.New(migrations.FS, "migrations")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	dbDriver, err := pgxmigrate.WithInstance(sqlDB, &pgxmigrate.Config{})
	if err != nil {
		return fmt.Errorf("migrations db driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", srcDriver, "pgx5", dbDriver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func newLogger(cfg config.LoggerConfig) (*slog.Logger, *os.File, error) {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(cfg.Level)) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	var writer io.Writer = os.Stdout
	var logFile *os.File
	if strings.TrimSpace(cfg.LogFile) != "" {
		file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file: %w", err)
		}
		logFile = file
		writer = io.MultiWriter(os.Stdout, file)
	}

	logger := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		AddSource: cfg.EnableCaller,
		Level:     level,
	}))
	return logger, logFile, nil
}

func newPostgresPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	host := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.Username, cfg.Password),
		Host:   host,
		Path:   cfg.Database,
	}
	query := dsn.Query()
	query.Set("sslmode", cfg.SSL)
	dsn.RawQuery = query.Encode()

	poolCfg, err := pgxpool.ParseConfig(dsn.String())
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	if cfg.PoolConfig.MaxConnections > 0 {
		poolCfg.MaxConns = cfg.PoolConfig.MaxConnections
	}
	if cfg.PoolConfig.MinConnections > 0 {
		poolCfg.MinConns = cfg.PoolConfig.MinConnections
	}
	if cfg.PoolConfig.MaxConnectionLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.PoolConfig.MaxConnectionLifetime
	}
	if cfg.PoolConfig.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.PoolConfig.MaxConnIdleTime
	}
	if cfg.PoolConfig.HealthCheckPeriod > 0 {
		poolCfg.HealthCheckPeriod = cfg.PoolConfig.HealthCheckPeriod
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	pingCtx := ctx
	cancel := func() {}
	if cfg.PoolConfig.ConnectTimeout > 0 {
		pingCtx, cancel = context.WithTimeout(ctx, cfg.PoolConfig.ConnectTimeout)
	}
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func newMinioClient(cfg config.MinioConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			cfg.Credentials.AccessKey,
			cfg.Credentials.SecretKey,
			"",
		),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	return client, nil
}

func New(ctx context.Context, cfg *config.Config) (Runner, error) {
	if err := validateOpenRouterConfig(cfg); err != nil {
		return nil, err
	}

	log, logFile, err := newLogger(cfg.Logger)
	if err != nil {
		return nil, err
	}
	cleanupLog := true
	defer func() {
		if cleanupLog && logFile != nil {
			_ = logFile.Close()
		}
	}()

	pool, err := newPostgresPool(ctx, cfg.Postgres)
	if err != nil {
		return nil, err
	}
	cleanupPool := true
	defer func() {
		if cleanupPool {
			pool.Close()
		}
	}()

	if err := runMigrations(pool); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	s3Client, err := newMinioClient(cfg.Minio)
	if err != nil {
		return nil, err
	}
	exists, err := s3Client.BucketExists(ctx, cfg.App.BucketName)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := s3Client.MakeBucket(ctx, cfg.App.BucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
	}
	s3Store := storage.New(s3Client, cfg.App.BucketName)

	gin.SetMode(gin.ReleaseMode)
	g := gin.New()
	g.Use(api.RequestLogMiddleware(log), gin.Recovery())

	scanRepo := db.NewScanRepository(pool)
	scaScanner := sca.NewScanner(cfg.App.GrypePath, cfg.App.ScaPath, cfg.App.GraphPath)
	exploitabilityEnricher := exploitability.New(exploitability.Config{
		APIKey:       cfg.App.OpenRouterAPIKey,
		Model:        cfg.App.OpenRouterModel,
		BaseURL:      cfg.App.OpenRouterBaseURL,
		Timeout:      cfg.App.OpenRouterTimeout,
		CallMapFiles: cfg.App.OpenRouterCallMapFiles,
		CallMapCalls: cfg.App.OpenRouterCallMapCalls,
	})
	scanner := service.NewScanner(scaScanner, cfg.App.SyftPath, scanRepo, s3Store, exploitabilityEnricher)
	scanAPIHandler := api.NewScanHandler(log, scanner)
	routes.RegisterGinRoutes(g, scanAPIHandler)

	server := api.NewServer(g, cfg.Http)
	cleanupLog = false
	cleanupPool = false
	return &App{
		logFile: logFile,
		pool:    pool,
		s:       server,
	}, nil
}

func validateOpenRouterConfig(cfg *config.Config) error {
	if strings.TrimSpace(cfg.App.OpenRouterAPIKey) == "" {
		return fmt.Errorf("app.openrouter_api_key is required")
	}
	if strings.TrimSpace(cfg.App.OpenRouterModel) == "" {
		return fmt.Errorf("app.openrouter_model is required")
	}
	return nil
}
