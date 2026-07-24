package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"platform/backend/internal/middleware"
)


func Profile(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"email": user.Email,
	})
}
