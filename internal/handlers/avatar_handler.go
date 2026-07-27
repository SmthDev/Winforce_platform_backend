package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"platform/backend/internal/middleware"
)

const maxAvatarSize = 5 << 20 

var allowedAvatarTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

type AvatarService interface {
	UploadAvatar(ctx context.Context, userID any, bucket, objectName string, reader io.Reader, size int64, contentType string) (string, error)
}

func UploadAvatar(avatarService AvatarService, bucket string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarSize+1<<10)

		fileHeader, err := c.FormFile("avatar")
		if err != nil {
			middleware.Logger(c).Warn("avatar form file missing or invalid", slog.Any("error", err))
			c.JSON(http.StatusBadRequest, gin.H{"message": "avatar file is required (multipart/form-data, field name \"avatar\") and must not exceed 5MB"})
			return
		}

		contentType := fileHeader.Header.Get("Content-Type")
		if !allowedAvatarTypes[contentType] {
			c.JSON(http.StatusBadRequest, gin.H{"message": "unsupported avatar file type"})
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			middleware.Logger(c).Error("open avatar file failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read avatar"})
			return
		}
		defer file.Close()

		objectName := fmt.Sprintf("%s%s", uuid.NewString(), strings.ToLower(filepath.Ext(fileHeader.Filename)))

		link, err := avatarService.UploadAvatar(c.Request.Context(), user.ID, bucket, objectName, file, fileHeader.Size, contentType)
		if err != nil {
			middleware.Logger(c).Error("upload avatar failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to upload avatar"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"avatar_link": link})
	}
}
