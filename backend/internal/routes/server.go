package routes

import (
	"github.com/gin-gonic/gin"

	"platform/backend/internal/handlers"
)

func NewRouter() *gin.Engine {
	r := gin.Default()


	r.StaticFile("/docs/api.html", "../docs/winforce-documentation.html")

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", handlers.Health)
	}

	return r
}
