package logger

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

const (
	FormatJSON = "json"
	FormatText = "text"
)

const (
	defaultServiceName = "backend"
	defaultEnv         = "local"
)

type Options struct {
	Level       string
	Format      string
	ServiceName string
	Env         string
	AddSource   bool
}

type CloseFunc func(context.Context) error

func Setup(ctx context.Context, opts Options) (*slog.Logger, CloseFunc, error) {
	level, err := ParseLevel(opts.Level)
	if err != nil {
		return nil, nil, err
	}

	serviceName := cmp.Or(opts.ServiceName, defaultServiceName)
	env := cmp.Or(opts.Env, defaultEnv)

	stdout, err := newStdoutHandler(opts.Format, level, opts.AddSource)
	if err != nil {
		return nil, nil, err
	}

	identified := stdout.WithAttrs([]slog.Attr{
		slog.String("service", serviceName),
		slog.String("env", env),
	})

	base := slog.New(identified)
	slog.SetDefault(base)

	return base, func(context.Context) error { return nil }, nil
}

func ParseLevel(name string) (slog.Level, error) {
	if name == "" {
		return slog.LevelInfo, nil
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(name)); err != nil {
		return 0, fmt.Errorf("parse log level %q: %w", name, err)
	}
	return level, nil
}

func newStdoutHandler(format string, level slog.Level, addSource bool) (slog.Handler, error) {
	opts := &slog.HandlerOptions{Level: level, AddSource: addSource}

	switch strings.ToLower(cmp.Or(format, FormatJSON)) {
	case FormatJSON:
		return slog.NewJSONHandler(os.Stdout, opts), nil
	case FormatText:
		return slog.NewTextHandler(os.Stdout, opts), nil
	default:
		return nil, fmt.Errorf("unknown log format %q: want %q or %q", format, FormatJSON, FormatText)
	}
}
