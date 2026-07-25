package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"platform/backend/internal/middleware"
	"platform/backend/internal/models"
)

type ProfileService interface {
	UpdateName(ctx context.Context, userID any, firstName, lastName string) error
	GetUserProfile(ctx context.Context, userID any) (models.Profile, error)
}

func Profile(profileService ProfileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		profile, err := profileService.GetUserProfile(c.Request.Context(), user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load profile"})
			return
		}

		c.JSON(http.StatusOK, profile)
	}
}

func UpdateProfileName(profileService ProfileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		var m models.Profile
		if err := c.ShouldBindJSON(&m); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "first_name and last_name are required"})
			return
		}

		if err := profileService.UpdateName(c.Request.Context(), user.ID, m.FirstName, m.LastName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update profile"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"first_name": m.FirstName,
			"last_name":  m.LastName,
		})
	}
}

