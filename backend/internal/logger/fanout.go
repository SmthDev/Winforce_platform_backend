package logger

import (
	"context"
	"log/slog"
)

// levelFilter enforces a minimum level on a handler that has none of its own,
// namely otelslog.Handler.
type levelFilter struct {
	slog.Handler
	level slog.Level
}

func withMinLevel(h slog.Handler, level slog.Level) slog.Handler {
	return levelFilter{Handler: h, level: level}
}

func (f levelFilter) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= f.level && f.Handler.Enabled(ctx, level)
}

func (f levelFilter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return levelFilter{Handler: f.Handler.WithAttrs(attrs), level: f.level}
}

func (f levelFilter) WithGroup(name string) slog.Handler {
	return levelFilter{Handler: f.Handler.WithGroup(name), level: f.level}
}

type contextKey struct{}



func NewContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
