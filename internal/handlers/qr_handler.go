package handlers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"platform/backend/internal/middleware"
	"platform/backend/internal/models"
)

const (
	maxQRCodeSize   = 2 << 20
	maxTargetLinkLn = 2048
)

var allowedQRCodeTypes = map[string]bool{
	"image/png":     true,
	"image/jpeg":    true,
	"image/webp":    true,
	"image/svg+xml": true,
}

type QRService interface {
	SaveQRCode(ctx context.Context, userID any, targetLink, bucket, objectName string, reader io.Reader, size int64, contentType string) (models.QRCode, error)
	ListQRCodes(ctx context.Context, userID any, bucket string) ([]models.QRCode, error)
}

func UploadQRCode(qrService QRService, bucket string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxQRCodeSize+1<<10)

		targetLink := strings.TrimSpace(c.PostForm("link"))
		if !validTargetLink(targetLink) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "link is required and must be a valid http(s) url"})
			return
		}

		fileHeader, err := c.FormFile("qr")
		if err != nil {
			middleware.Logger(c).Warn("qr form file missing or invalid", slog.Any("error", err))
			c.JSON(http.StatusBadRequest, gin.H{"message": "qr file is required (multipart/form-data, field name \"qr\") and must not exceed 2MB"})
			return
		}

		contentType := fileHeader.Header.Get("Content-Type")
		if !allowedQRCodeTypes[contentType] {
			c.JSON(http.StatusBadRequest, gin.H{"message": "unsupported qr file type"})
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			middleware.Logger(c).Error("open qr file failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read qr file"})
			return
		}
		defer file.Close()

		objectName := fmt.Sprintf("%s%s", uuid.NewString(), strings.ToLower(filepath.Ext(fileHeader.Filename)))

		qr, err := qrService.SaveQRCode(c.Request.Context(), user.ID, targetLink, bucket, objectName, file, fileHeader.Size, contentType)
		if err != nil {
			middleware.Logger(c).Error("save qr code failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save qr code"})
			return
		}

		c.JSON(http.StatusCreated, qr)
	}
}

func ListQRCodes(qrService QRService, bucket string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		codes, err := qrService.ListQRCodes(c.Request.Context(), user.ID, bucket)
		if err != nil {
			middleware.Logger(c).Error("list qr codes failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load qr codes"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"qr_codes": codes})
	}
}

func validTargetLink(link string) bool {
	if link == "" || len(link) > maxTargetLinkLn {
		return false
	}

	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
