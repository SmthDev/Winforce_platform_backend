package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/saneechka/Winforce_platform/backend/internal/handlers"
)

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/health", handler.Health)
	return r
}
