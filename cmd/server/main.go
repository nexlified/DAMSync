package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/nexlified/dam/config"
	apphttp "github.com/nexlified/dam/internal/infrastructure/http"
	"github.com/nexlified/dam/internal/infrastructure/http/middleware"
	"github.com/nexlified/dam/internal/infrastructure/postgres"
	redisinfra "github.com/nexlified/dam/internal/infrastructure/redis"
	"github.com/nexlified/dam/internal/infrastructure/storage"
	"github.com/nexlified/dam/internal/infrastructure/transform"
	"github.com/nexlified/dam/internal/application/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	setupLogger(cfg)
	log.Info().Str("env", cfg.Env).Msg("starting DAM server")

	// ─── Infrastructure: Database ─────────────────────────────────────────────
	db, err := postgres.NewDB(cfg.Database.DSN, cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	log.Info().Msg("postgres connected")

	// Run migrations
	if err := runMigrations(cfg.Database.DSN, cfg.Migrations.Path); err != nil {
		log.Fatal().Err(err).Msg("migration failed")
	}
	log.Info().Msg("migrations applied")

	// ─── Infrastructure: Redis ────────────────────────────────────────────────
	redisClient, err := redisinfra.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	log.Info().Msg("redis connected")

	rateLimiter := redisinfra.NewRateLimiter(redisClient)
	eventPublisher := redisinfra.NewEventPublisher(redisClient)

	// ─── Infrastructure: Object Storage ──────────────────────────────────────
	s3Storage, err := storage.NewS3Storage(storage.S3Config{
		Endpoint:      cfg.Storage.Endpoint,
		Region:        cfg.Storage.Region,
		Bucket:        cfg.Storage.Bucket,
		AccessKey:     cfg.Storage.AccessKey,
		SecretKey:     cfg.Storage.SecretKey,
		UsePathStyle:  cfg.Storage.UsePathStyle,
		PublicBaseURL: cfg.Storage.PublicBaseURL,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize storage")
	}
	log.Info().Msg("storage initialized")

	// ─── Infrastructure: Image Transformer ────────────────────────────────────
	transformer := transform.NewBimgTransformer()

	// ─── Infrastructure: Repositories ─────────────────────────────────────────
	orgRepo := postgres.NewOrgRepository(db)
	userRepo := postgres.NewUserRepository(db)
	apiKeyRepo := postgres.NewAPIKeyRepository(db)
	assetRepo := postgres.NewAssetRepository(db)
	folderRepo := postgres.NewFolderRepository(db)
	tagRepo := postgres.NewTagRepository(db)
	collectionRepo := postgres.NewCollectionRepository(db)
	domainRepo := postgres.NewDomainRepository(db)
	styleRepo := postgres.NewStyleRepository(db)
	transformCacheRepo := postgres.NewTransformCacheRepository(db)
	webhookRepo := postgres.NewWebhookRepository(db)
	auditRepo := postgres.NewAuditLogRepository(db)

	// ─── Application: Services ────────────────────────────────────────────────
	authSvc := services.NewAuthService(
		userRepo,
		apiKeyRepo,
		redisClient,
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	orgSvc := services.NewOrgService(orgRepo, userRepo)

	assetSvc := services.NewAssetService(
		assetRepo,
		orgRepo,
		s3Storage,
		transformer,
		eventPublisher,
		auditRepo,
	)

	folderSvc := services.NewFolderService(folderRepo)
	tagSvc := services.NewTagService(tagRepo, assetRepo)
	collectionSvc := services.NewCollectionService(collectionRepo)
	searchSvc := services.NewSearchService(assetRepo)

	styleSvc := services.NewStyleService(
		styleRepo,
		transformCacheRepo,
		assetRepo,
		s3Storage,
		transformer,
		redisClient,
		eventPublisher,
	)

	domainSvc := services.NewDomainService(domainRepo, orgRepo, redisClient)

	webhookSvc := services.NewWebhookService(webhookRepo)

	// Register event dispatcher — wire webhooks to domain events
	_ = webhookSvc // TODO: subscribe to event stream for async dispatch

	// ─── HTTP Server ──────────────────────────────────────────────────────────
	svcs := &apphttp.Services{
		Auth:       authSvc,
		Org:        orgSvc,
		Asset:      assetSvc,
		Folder:     folderSvc,
		Tag:        tagSvc,
		Collection: collectionSvc,
		Style:      styleSvc,
		Domain:     domainSvc,
		Webhook:    webhookSvc,
		Search:     searchSvc,
	}

	// Audit logger middleware
	_ = middleware.NewAuditLogger(auditRepo) // wired in server middleware

	server := apphttp.NewServer(
		svcs,
		redisClient,
		rateLimiter,
		s3Storage,
		cfg.Env == "development",
	)

	addr := apphttp.Addr(cfg.Server.Host, cfg.Server.Port)
	log.Info().Str("addr", addr).Msg("listening")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("shutting down gracefully...")
		if err := server.App().Shutdown(); err != nil {
			log.Error().Err(err).Msg("shutdown error")
		}
	}()

	if err := server.Listen(addr); err != nil {
		log.Info().Msg("server stopped")
	}
}

func runMigrations(dsn, migrationsPath string) error {
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		dsn,
	)
	if err != nil {
		return fmt.Errorf("create migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func setupLogger(cfg *config.Config) {
	if cfg.Log.Format == "pretty" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
	switch cfg.Log.Level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
