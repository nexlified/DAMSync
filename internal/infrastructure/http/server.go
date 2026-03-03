package http

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/rs/zerolog/log"
	"github.com/nexlified/dam/internal/application/ports/inbound"
	"github.com/nexlified/dam/internal/application/ports/outbound"
	"github.com/nexlified/dam/internal/application/services"
	"github.com/nexlified/dam/internal/infrastructure/http/middleware"
	"github.com/nexlified/dam/internal/infrastructure/http/serve"
	v1 "github.com/nexlified/dam/internal/infrastructure/http/v1"
)

// Services bundles all application service dependencies for the HTTP server.
type Services struct {
	Auth       *services.AuthServiceImpl
	Org        inbound.OrgService
	Asset      inbound.AssetService
	Folder     inbound.FolderService
	Tag        inbound.TagService
	Collection inbound.CollectionService
	Style      inbound.StyleService
	Domain     inbound.DomainService
	Webhook    inbound.WebhookService
	Search     inbound.SearchService
}

type Server struct {
	app        *fiber.App
	services   *Services
	cache      outbound.CachePort
	rl         outbound.RateLimiterPort
	storage    outbound.StoragePort
	cdnBaseURL string
}

func NewServer(
	svcs *Services,
	cache outbound.CachePort,
	rl outbound.RateLimiterPort,
	storage outbound.StoragePort,
	isDev bool,
	allowOrigins string,
	cdnBaseURL string,
) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		ErrorHandler:          errorHandler,
		DisableStartupMessage: !isDev,
		BodyLimit:             500 * 1024 * 1024, // 500 MiB max body
	})

	s := &Server{app: app, services: svcs, cache: cache, rl: rl, storage: storage, cdnBaseURL: cdnBaseURL}
	s.registerMiddleware(isDev, allowOrigins)
	s.registerRoutes()
	return s
}

func (s *Server) registerMiddleware(isDev bool, allowOrigins string) {
	s.app.Use(recover.New())

	if allowOrigins != "" {
		s.app.Use(cors.New(cors.Config{
			AllowOrigins:     allowOrigins,
			AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
			AllowHeaders:     "Origin,Content-Type,Authorization",
			AllowCredentials: true,
		}))
	}

	s.app.Use(requestid.New())
	s.app.Use(securityHeaders())

	if isDev {
		s.app.Use(logger.New(logger.Config{
			Format: "${time} | ${status} | ${latency} | ${method} ${path} | id=${locals:requestid}\n",
		}))
	}

	s.app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	s.app.Use(middleware.NewDomainResolver(s.services.Domain, s.cache))
}

func (s *Server) registerRoutes() {
	// Health check
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "time": time.Now().UTC()})
	})

	// CDN asset serving (public, no auth needed)
	serveHandler := serve.NewHandler(s.services.Asset, s.services.Style, s.storage)
	s.app.Get("/files/*", serveHandler.ServePublic)
	s.app.Get("/styles/:style/*", serveHandler.ServeStyled)
	s.app.Get("/secure/:token/:expires/*", serveHandler.ServeSigned)

	// API v1
	api := s.app.Group("/api/v1")
	api.Use(middleware.NewRateLimiter(s.rl, 200, 60))

	// Auth (unauthenticated)
	authHandler := v1.NewAuthHandler(s.services.Auth, s.services.Org)
	api.Post("/auth/register", authHandler.Register)
	api.Post("/auth/login", authHandler.Login)
	api.Post("/auth/refresh", authHandler.Refresh)
	api.Post("/auth/logout", middleware.RequireAuth(s.services.Auth), authHandler.Logout)

	// Protected routes
	protected := api.Group("", middleware.RequireAuth(s.services.Auth))

	// Orgs / users / API keys
	orgsHandler := v1.NewOrgsHandler(s.services.Org, s.services.Auth)
	protected.Get("/orgs/me", orgsHandler.GetCurrentOrg)
	protected.Put("/orgs/me", middleware.RequireRole("owner", "admin"), orgsHandler.UpdateOrg)
	protected.Get("/orgs/me/storage", orgsHandler.GetStorageUsage)
	protected.Post("/users", middleware.RequireRole("owner", "admin"), orgsHandler.CreateUser)
	protected.Get("/users", orgsHandler.ListUsers)
	protected.Get("/users/:id", orgsHandler.GetUser)
	protected.Put("/users/:id", middleware.RequireRole("owner", "admin"), orgsHandler.UpdateUser)
	protected.Delete("/users/:id", middleware.RequireRole("owner", "admin"), orgsHandler.DeleteUser)
	protected.Post("/api-keys", orgsHandler.CreateAPIKey)
	protected.Get("/api-keys", orgsHandler.ListAPIKeys)
	protected.Delete("/api-keys/:id", orgsHandler.RevokeAPIKey)

	// Assets
	assetsHandler := v1.NewAssetsHandler(s.services.Asset, s.storage, s.cdnBaseURL)
	api.Post("/assets", middleware.RequireAuth(s.services.Auth), middleware.RequireScope("assets:write"), assetsHandler.Upload)
	api.Post("/assets/bulk", middleware.RequireAuth(s.services.Auth), middleware.RequireScope("assets:write"), assetsHandler.BulkUpload)
	protected.Get("/assets", assetsHandler.List)
	protected.Get("/assets/:id", assetsHandler.Get)
	protected.Put("/assets/:id", middleware.RequireScope("assets:write"), assetsHandler.UpdateMetadata)
	protected.Delete("/assets/:id", middleware.RequireScope("assets:delete"), assetsHandler.Delete)
	protected.Post("/assets/:id/move", middleware.RequireScope("assets:write"), assetsHandler.Move)
	protected.Get("/assets/:id/signed-url", assetsHandler.GetSignedURL)

	// Folders
	foldersHandler := v1.NewFoldersHandler(s.services.Folder)
	protected.Post("/folders", foldersHandler.Create)
	protected.Get("/folders", foldersHandler.List)
	protected.Get("/folders/tree", foldersHandler.Tree)
	protected.Get("/folders/:id", foldersHandler.Get)
	protected.Put("/folders/:id", foldersHandler.Update)
	protected.Delete("/folders/:id", foldersHandler.Delete)

	// Tags
	tagsHandler := v1.NewTagsHandler(s.services.Tag)
	protected.Post("/tags", tagsHandler.Create)
	protected.Get("/tags", tagsHandler.List)
	protected.Delete("/tags/:id", tagsHandler.Delete)
	protected.Post("/assets/:id/tags", tagsHandler.TagAsset)
	protected.Delete("/assets/:id/tags", tagsHandler.UntagAsset)

	// Collections
	collectionsHandler := v1.NewCollectionsHandler(s.services.Collection)
	protected.Post("/collections", collectionsHandler.Create)
	protected.Get("/collections", collectionsHandler.List)
	protected.Get("/collections/:id", collectionsHandler.Get)
	protected.Put("/collections/:id", collectionsHandler.Update)
	protected.Delete("/collections/:id", collectionsHandler.Delete)
	protected.Post("/collections/:id/assets/:assetId", collectionsHandler.AddAsset)
	protected.Delete("/collections/:id/assets/:assetId", collectionsHandler.RemoveAsset)
	protected.Get("/collections/:id/assets", collectionsHandler.ListAssets)

	// Image Styles
	stylesHandler := v1.NewStylesHandler(s.services.Style)
	protected.Post("/styles", middleware.RequireRole("owner", "admin"), stylesHandler.Create)
	protected.Post("/styles/import-defaults", middleware.RequireRole("owner", "admin"), stylesHandler.ImportDefaults)
	protected.Get("/styles", stylesHandler.List)
	protected.Get("/styles/:id", stylesHandler.Get)
	protected.Put("/styles/:id", middleware.RequireRole("owner", "admin"), stylesHandler.Update)
	protected.Delete("/styles/:id", middleware.RequireRole("owner", "admin"), stylesHandler.Delete)

	// Domains
	domainsHandler := v1.NewDomainsHandler(s.services.Domain)
	protected.Post("/domains", middleware.RequireRole("owner", "admin"), domainsHandler.Add)
	protected.Get("/domains", domainsHandler.List)
	protected.Post("/domains/:id/verify", middleware.RequireRole("owner", "admin"), domainsHandler.Verify)
	protected.Delete("/domains/:id", middleware.RequireRole("owner", "admin"), domainsHandler.Remove)

	// Webhooks
	webhooksHandler := v1.NewWebhooksHandler(s.services.Webhook)
	protected.Post("/webhooks", middleware.RequireRole("owner", "admin"), webhooksHandler.Create)
	protected.Get("/webhooks", webhooksHandler.List)
	protected.Get("/webhooks/:id", webhooksHandler.Get)
	protected.Put("/webhooks/:id", middleware.RequireRole("owner", "admin"), webhooksHandler.Update)
	protected.Delete("/webhooks/:id", middleware.RequireRole("owner", "admin"), webhooksHandler.Delete)
	protected.Post("/webhooks/:id/test", webhooksHandler.Test)
	protected.Get("/webhooks/:id/deliveries", webhooksHandler.ListDeliveries)

	// Search
	searchHandler := v1.NewSearchHandler(s.services.Search)
	protected.Get("/search", searchHandler.Search)
}

func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

func (s *Server) App() *fiber.App {
	return s.app
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal server error"

	if fe, ok := err.(*fiber.Error); ok {
		code = fe.Code
		msg = fe.Message
		if code >= 500 {
			log.Error().Err(err).Str("method", c.Method()).Str("path", c.Path()).Str("request_id", c.GetRespHeader("X-Request-Id")).Msg("internal error")
		}
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}

	if err != nil {
		switch {
		case containsStr(err.Error(), "not found"):
			code = fiber.StatusNotFound
			msg = err.Error()
		case containsStr(err.Error(), "unauthorized"):
			code = fiber.StatusUnauthorized
			msg = "unauthorized"
		case containsStr(err.Error(), "forbidden"):
			code = fiber.StatusForbidden
			msg = "forbidden"
		case containsStr(err.Error(), "invalid"):
			code = fiber.StatusBadRequest
			msg = err.Error()
		case containsStr(err.Error(), "quota"):
			code = fiber.StatusPaymentRequired
			msg = err.Error()
		case containsStr(err.Error(), "already exists"):
			code = fiber.StatusConflict
			msg = err.Error()
		}
	}

	if code >= 500 {
		log.Error().Err(err).Str("method", c.Method()).Str("path", c.Path()).Str("request_id", c.GetRespHeader("X-Request-Id")).Msg("internal error")
	}

	return c.Status(code).JSON(fiber.Map{"error": msg})
}

func containsStr(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func securityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		return c.Next()
	}
}

func Addr(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}
