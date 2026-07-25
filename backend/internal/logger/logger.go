
package logger

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

const (
	FormatJSON = "json"
	FormatText = "text"
)

const (
	defaultServiceName = "backend"
	defaultEnv         = "local"
	tenantHeader       = "X-Scope-OrgID"
)

type Options struct {

	Level string
	Format      string
	ServiceName string
	Env         string
	AddSource   bool
	OTLPEndpoint string
	TenantID string
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

	if opts.OTLPEndpoint == "" {
		base := slog.New(identified)
		slog.SetDefault(base)
		return base, func(context.Context) error { return nil }, nil
	}

	provider, err := newLoggerProvider(ctx, opts, serviceName, env)
	if err != nil {
		return nil, nil, err
	}


	stdoutOnly := slog.New(identified)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		stdoutOnly.Error("otlp log export failed", slog.Any("error", err))
	}))

	otelHandler := withMinLevel(otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(provider)), level)
	base := slog.New(otelHandler)
	slog.SetDefault(base)

	return base, provider.Shutdown, nil
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

func newLoggerProvider(ctx context.Context, opts Options, serviceName, env string) (*sdklog.LoggerProvider, error) {
	endpoint, err := url.Parse(opts.OTLPEndpoint)
	if err != nil {
		return nil, fmt.Errorf("parse otlp endpoint %q: %w", opts.OTLPEndpoint, err)
	}
	if endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, fmt.Errorf("otlp endpoint %q must include scheme and host", opts.OTLPEndpoint)
	}

	exporterOpts := []otlploghttp.Option{otlploghttp.WithEndpointURL(opts.OTLPEndpoint)}
	if endpoint.Scheme == "http" {
		exporterOpts = append(exporterOpts, otlploghttp.WithInsecure())
	}
	if opts.TenantID != "" {
		exporterOpts = append(exporterOpts, otlploghttp.WithHeaders(map[string]string{
			tenantHeader: opts.TenantID,
		}))
	}

	exporter, err := otlploghttp.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("init otlp log exporter: %w", err)
	}


	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		semconv.ServiceName(serviceName),
		semconv.DeploymentEnvironmentNameKey.String(env),
	))
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	), nil
}
