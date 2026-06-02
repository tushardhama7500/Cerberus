package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"entgo.io/ent/dialect"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"cerberus/config"
	"cerberus/ent"
	"cerberus/graph"
	"cerberus/graph/resolver"
	"cerberus/internal/authz"
	"cerberus/internal/middleware"
	"cerberus/internal/repository"
	"cerberus/internal/service"
	"cerberus/pkg/s3"
)

func main() {
	// ── 1. Load Config ────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// ── 2. Connect to Database ────────────────────────────────────────────────
	entClient, err := ent.Open(dialect.MySQL, cfg.Database.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	log.Printf("Connected to database %s successfully", cfg.Database.Name)
	defer entClient.Close()

	// ── 3. Run Auto-Migration ─────────────────────────────────────────────────
	// In production, use versioned migrations instead of AutoMigrate.
	// For local dev, this is fine.
	if err := entClient.Schema.Create(
		context.Background(),
		// migrate.WithDropIndex(true),  // Uncomment to drop old indexes
		// migrate.WithDropColumn(true), // Uncomment with caution in prod
	); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Println("Database migration complete")

	// ── 4. Initialize S3 Client ───────────────────────────────────────────────
	s3Client, err := s3.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize S3 client: %v", err)
	}
	log.Printf("Connected to S3 bucket %s successfully", cfg.AWS.S3Bucket)

	// ── 5. Wire Repositories ──────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(entClient)
	log.Printf("UserRepository initialized")
	requestRepo := repository.NewAccessRequestRepository(entClient)
	log.Printf("AccessRequestRepository initialized")
	auditRepo := repository.NewAuditRepository(entClient)
	log.Printf("AuditRepository initialized")

	// ── 6. Initialize OPA client ──────────────────────────────────────────────
	// NOTE: Using default localhost URL to avoid requiring config changes here.
	// Replace with cfg.OPA.BaseURL once you add OPA configuration to `config`.
	opaBaseURL := "http://localhost:8181"
	opaClient := authz.NewClient(opaBaseURL)

	// Non-fatal health check: warn if OPA is down (fail-close is enforced in service)
	if err := opaClient.Health(context.Background()); err != nil {
		log.Printf("WARNING: OPA health check failed: %v", err)
		log.Printf("Start OPA with: docker-compose -f docker-compose.opa.yml up -d")
	} else {
		log.Printf("OPA client ready at %s", opaBaseURL)
	}

	// ── 7. Wire Services ──────────────────────────────────────────────────────
	authSvc := service.NewAuthService(userRepo, cfg)
	log.Printf("AuthService initialized")
	requestSvc := service.NewAccessRequestService(
		requestRepo,
		auditRepo,
		userRepo,
		s3Client,
		opaClient, // pass OPA client here
	)
	log.Printf("AccessRequestService initialized")

	// ── 8. Build GraphQL Server ───────────────────────────────────────────────
	rootResolver := &resolver.Resolver{
		AuthService:          authSvc,
		AccessRequestService: requestSvc,
	}

	schema := graph.NewExecutableSchema(graph.Config{
		Resolvers: rootResolver,
	})

	gqlHandler := handler.NewDefaultServer(schema)

	// Add query complexity limit — prevents deeply nested malicious queries
	gqlHandler.Use(extension.FixedComplexityLimit(300))

	// Disable introspection in production — prevents schema discovery by attackers
	if cfg.Server.Env == "production" {
		gqlHandler.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
			graphql.GetOperationContext(ctx).DisableIntrospection = true
			return next(ctx)
		})
	}

	// ── 9. Setup Gin ──────────────────────────────────────────────────────────
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New() // gin.New() not gin.Default() — we add our own middleware

	// CORS — configure allowed origins for production
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://your-blaze-frontend.com"},
		AllowMethods:     []string{"POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.Use(middleware.Recovery())
	router.Use(gin.Logger())
	router.Use(middleware.JWTAuth(cfg))

	// GraphQL endpoint
	router.POST("/query", func(c *gin.Context) {
		log.Printf("Received GraphQL request: %s %s", c.Request.Method, c.Request.URL.Path)
		gqlHandler.ServeHTTP(c.Writer, c.Request)
	})

	// GraphQL playground — only in development
	if cfg.Server.Env != "production" {
		router.GET("/playground", func(c *gin.Context) {
			playground.Handler("Cerberus GraphQL", "/query").ServeHTTP(c.Writer, c.Request)
		})
		log.Printf("GraphQL Playground: http://localhost:%s/playground", cfg.Server.Port)
	}

	// Health check — used by load balancers
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "cerberus"})
	})

	// ── 10. Start Server with Graceful Shutdown ────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine so we can listen for shutdown signals
	go func() {
		log.Printf("Cerberus server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait for interrupt signal (Ctrl+C or SIGTERM from process manager)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give in-flight requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("Server stopped cleanly")
}
