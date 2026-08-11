package routes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecodearcher/limen"

	"platform/backend/internal/config"
	"platform/backend/internal/db"
	"platform/backend/internal/handlers"
	"platform/backend/internal/logger"
	"platform/backend/internal/middleware"
	miniorepo "platform/backend/internal/repository/minio"
	"platform/backend/internal/repository/postgres/avatar_repo"
	"platform/backend/internal/repository/postgres/qr_repo"
	"platform/backend/internal/repository/postgres/user_repo"
	"platform/backend/internal/service"
)

const (
	shutdownTimeout = 10 * time.Second
	logFlushTimeout = 5 * time.Second
	healthPath      = "/api/v1/health"

	avatarsBucket = "avatars"
	qrCodesBucket = "qr-codes"
)

func NewRouter(cfg *config.Config, log *slog.Logger, authInstance *limen.Limen, pool *pgxpool.Pool, minioClient *miniorepo.Client) *gin.Engine {
	r := gin.New()


	r.Use(
		gin.Logger(),
		middleware.RequestLogger(log, healthPath),
		middleware.Recovery(),
		middleware.CORS(cfg.FrontendURL),
	)

	r.StaticFile("/docs/api.html", "../docs/winforce-documentation.html")

	userRepo := user_repo.New(pool)

	profileService := service.NewProfile(userRepo)
	accessService := service.NewAccess(userRepo)
	avatarService := service.NewAvatar(avatar_repo.New(pool), minioClient, log)
	qrService := service.NewQR(qr_repo.New(pool), minioClient, log)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", handlers.Health)
		v1.Any("/auth/*path", gin.WrapH(authInstance.Handler()))
		v1.POST("/login", handlers.Login(authInstance))
		v1.GET("/profile", middleware.RequireAuth(authInstance), handlers.Profile(profileService))
		v1.PATCH("/profile", middleware.RequireAuth(authInstance), handlers.UpdateProfileName(profileService))
		v1.POST("/profile/avatar", middleware.RequireAuth(authInstance), handlers.UploadAvatar(avatarService, avatarsBucket))
		v1.GET("/qr", middleware.RequireAuth(authInstance), handlers.ListQRCodes(qrService))

		qr := v1.Group("/qr", middleware.RequireAuth(authInstance), middleware.RequireAdmin(accessService))
		{
			qr.POST("/create", handlers.UploadQRCode(qrService, qrCodesBucket))
			//qr.GET("", handlers.ListQRCodes(qrService))
		}
	}

	return r
}

func RunMigrate(args []string) {
	if err := runMigrate(args); err != nil {
		slog.Error("migrate failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func RunApp() {
	if err := run(); err != nil {
		slog.Error("application stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func runMigrate(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: app migrate <up [steps]|down <steps|all>|version|force <version>>")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx := context.Background()
	log, closeLogger, err := setupLogger(ctx, cfg)
	if err != nil {
		return err
	}
	defer flushLogs(closeLogger)

	if err := db.Run(ctx, cfg.DatabaseURL, args[0], args[1:]...); err != nil {
		return fmt.Errorf("migrate %s: %w", args[0], err)
	}

	log.Info("migrate completed", slog.String("action", args[0]))
	return nil
}


func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, closeLogger, err := setupLogger(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer flushLogs(closeLogger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewInstance(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer pool.Close()
	log.Info("connected to database")

	authInstance, authDB, err := service.NewAuth(cfg, pool)
	if err != nil {
		return fmt.Errorf("auth setup failed: %w", err)
	}
	defer authDB.Close()

	minioClient, err := miniorepo.New(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioUseSSL)
	if err != nil {
		return fmt.Errorf("minio setup failed: %w", err)
	}
	if err := minioClient.EnsureBucket(ctx, avatarsBucket, true); err != nil {
		return fmt.Errorf("minio bucket setup failed: %w", err)
	}
	if err := minioClient.EnsureBucket(ctx, qrCodesBucket, true); err != nil {
		return fmt.Errorf("minio bucket setup failed: %w", err)
	}
	log.Info("connected to minio")

	gin.SetMode(ginMode(cfg.Env))

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: NewRouter(cfg, log, authInstance, pool, minioClient),
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	log.Info("server stopped")
	return nil
}

func setupLogger(ctx context.Context, cfg *config.Config) (*slog.Logger, logger.CloseFunc, error) {
	log, closeLogger, err := logger.Setup(ctx, logger.Options{
		Level:       cfg.LogLevel,
		Format:      cfg.LogFormat,
		ServiceName: cfg.ServiceName,
		Env:         cfg.Env,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("setup logger: %w", err)
	}
	return log, closeLogger, nil
}



func flushLogs(closeLogger logger.CloseFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), logFlushTimeout)
	defer cancel()

	if err := closeLogger(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "flush logs: %v\n", err)
	}
}

func ginMode(env string) string {
	switch env {
	case "local", "development", "dev", "test":
		return gin.DebugMode
	default:
		return gin.ReleaseMode
	}
}
