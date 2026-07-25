package routes

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecodearcher/limen"

	"platform/backend/internal/config"
	"platform/backend/internal/db"
	"platform/backend/internal/handlers"
	"platform/backend/internal/middleware"
	"platform/backend/internal/repository/postgres/user_repo"
	"platform/backend/internal/service"
)

func NewRouter(cfg *config.Config, authInstance *limen.Limen, pool *pgxpool.Pool) *gin.Engine {
	r := gin.Default()

	r.Use(middleware.CORS(cfg.FrontendURL))

	r.StaticFile("/docs/api.html", "../docs/winforce-documentation.html")

	profileService := service.NewProfile(user_repo.New(pool))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", handlers.Health)
		v1.Any("/auth/*path", gin.WrapH(authInstance.Handler()))
		v1.POST("/login", handlers.Login(authInstance))
		v1.GET("/profile", middleware.RequireAuth(authInstance), handlers.Profile(profileService))
		v1.PATCH("/profile", middleware.RequireAuth(authInstance), handlers.UpdateProfileName(profileService))
	}

	return r
}

func RunMigrate(args []string) {
	if len(args) == 0 {
		log.Fatalf("usage: app migrate <up [steps]|down <steps|all>|version|force <version>>")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := db.Run(context.Background(), cfg.DatabaseURL, args[0], args[1:]...); err != nil {
		log.Fatalf("migrate %s: %v", args[0], err)
	}
	log.Printf("migrate %s: ok", args[0])
}

func RunApp(){
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := db.NewInstance(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()
	log.Printf("connected to database")

	authInstance, authDB, err := service.NewAuth(cfg, pool)
	if err != nil {
		log.Fatalf("auth setup failed: %v", err)
	}
	defer authDB.Close()

	NewRouter(cfg, authInstance, pool).Run(":" + cfg.Port)
}
