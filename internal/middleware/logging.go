package middleware

import (
	"cmp"
	"crypto/rand"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"slices"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"platform/backend/internal/logger"
)

const (
	ContextLoggerKey = "logger"
	RequestIDHeader  = "X-Request-Id"
)


func RequestLogger(base *slog.Logger, quietPaths ...string) gin.HandlerFunc {
	quiet := slices.Clone(quietPaths)

	return func(c *gin.Context) {
		start := time.Now()

		requestID := cmp.Or(c.GetHeader(RequestIDHeader), rand.Text())
		c.Header(RequestIDHeader, requestID)

		reqLogger := base.With(
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)
		c.Set(ContextLoggerKey, reqLogger)
		c.Request = c.Request.WithContext(logger.NewContext(c.Request.Context(), reqLogger))

		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			slog.Int("bytes", max(c.Writer.Size(), 0)),
			slog.String("client_ip", c.ClientIP()),
		}
		if query := c.Request.URL.RawQuery; query != "" {
			attrs = append(attrs, slog.String("query", query))
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("gin_errors", c.Errors.String()))
		}

		const msg = "request"
		switch {
		case status >= http.StatusInternalServerError:
			reqLogger.Error(msg, attrs...)
		case status >= http.StatusBadRequest:
			reqLogger.Warn(msg, attrs...)
		case isQuiet(c, quiet):
			reqLogger.Debug(msg, attrs...)
		default:
			reqLogger.Info(msg, attrs...)
		}
	}
}


func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			log := Logger(c)
			if isBrokenPipe(rec) {
	
				log.Warn("client closed connection", slog.Any("panic", rec))
				c.Abort()
				return
			}

			log.Error("panic recovered",
				slog.Any("panic", rec),
				slog.String("stack", string(debug.Stack())),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		}()

		c.Next()
	}
}

func Logger(c *gin.Context) *slog.Logger {
	if value, exists := c.Get(ContextLoggerKey); exists {
		if log, ok := value.(*slog.Logger); ok {
			return log
		}
	}
	if c.Request != nil {
		return logger.FromContext(c.Request.Context())
	}
	return slog.Default()
}

func isQuiet(c *gin.Context, quiet []string) bool {
	return slices.Contains(quiet, c.FullPath()) || slices.Contains(quiet, c.Request.URL.Path)
}

func isBrokenPipe(rec any) bool {
	err, ok := rec.(error)
	if !ok {
		return false
	}

	opErr, ok := errors.AsType[*net.OpError](err)
	if !ok {
		return false
	}
	syscallErr, ok := errors.AsType[*os.SyscallError](opErr.Err)
	if !ok {
		return false
	}
	return errors.Is(syscallErr, syscall.EPIPE) || errors.Is(syscallErr, syscall.ECONNRESET)
}
