package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"platform/backend/internal/filelink"
	"platform/backend/internal/middleware"
	miniorepo "platform/backend/internal/repository/minio"
	"platform/backend/internal/service"
)

const defaultFileContentType = "application/octet-stream"

type FileService interface {
	Open(ctx context.Context, req service.FileRequest) (io.ReadCloser, miniorepo.ObjectInfo, service.FileAccess, error)
	CacheControl(access service.FileAccess) string
}

func ServeFile(fileService FileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := service.FileRequest{
			Bucket:     c.Param("bucket"),
			ObjectName: strings.TrimPrefix(c.Param("object"), "/"),
			Expires:    c.Query(filelink.QueryExpires),
			Signature:  c.Query(filelink.QuerySignature),
		}

		body, info, access, err := fileService.Open(c.Request.Context(), req)
		if err != nil {
			writeFileError(c, err, req)
			return
		}
		defer body.Close()

		contentType := info.ContentType
		if contentType == "" {
			contentType = defaultFileContentType
		}

		headers := map[string]string{"Cache-Control": fileService.CacheControl(access)}
		if info.ETag != "" {
			headers["ETag"] = `"` + strings.Trim(info.ETag, `"`) + `"`
		}
		if !info.LastModified.IsZero() {
			headers["Last-Modified"] = info.LastModified.UTC().Format(http.TimeFormat)
		}
		if match := c.GetHeader("If-None-Match"); match != "" && headers["ETag"] != "" && strings.Contains(match, headers["ETag"]) {
			for key, value := range headers {
				c.Header(key, value)
			}
			c.Status(http.StatusNotModified)
			return
		}

		c.DataFromReader(http.StatusOK, info.Size, contentType, body, headers)
	}
}

func writeFileError(c *gin.Context, err error, req service.FileRequest) {
	switch {
	case errors.Is(err, service.ErrFileNotFound), errors.Is(err, service.ErrBadObjectName):
		c.JSON(http.StatusNotFound, gin.H{"message": "file not found"})
	case errors.Is(err, service.ErrFileForbidden):
		c.JSON(http.StatusForbidden, gin.H{"message": "file link is invalid or has expired"})
	default:
		middleware.Logger(c).Error("serve file failed",
			slog.Any("error", err),
			slog.String("bucket", req.Bucket),
			slog.String("object_name", req.ObjectName),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to serve file"})
	}
}
