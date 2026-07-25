package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecodearcher/limen"

	"platform/backend/internal/middleware"
	"platform/backend/internal/service"
)


const maxLoginBodyBytes = 1 << 16


func Login(instance *limen.Limen) gin.HandlerFunc {
	handler := instance.Handler()
	target := service.AuthBasePath + "/signin/credential"

	return func(c *gin.Context) {
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxLoginBodyBytes))
		if err != nil {
			middleware.Logger(c).Warn("read login body failed", slog.Any("error", err))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}
		body = withCredentialAlias(body)

		req := c.Request.Clone(c.Request.Context())
		req.URL.Path = target
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))

		handler.ServeHTTP(c.Writer, req)
	}
}

func withCredentialAlias(body []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}

	if _, ok := payload["credential"]; ok {
		return body
	}
	email, ok := payload["email"]
	if !ok {
		return body
	}
	payload["credential"] = email

	patched, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return patched
}
