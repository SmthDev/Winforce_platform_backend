package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgxdriver "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const migrationsDir = "migrations"

//go:embed migrations/*.sql
var migrationsFS embed.FS


func Run(ctx context.Context, dsn, action string, args ...string) error {
	if dsn == "" {
		return errors.New("database dsn is empty")
	}

	src, err := iofs.New(migrationsFS, migrationsDir)
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer conn.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	drv, err := pgxdriver.WithInstance(conn, &pgxdriver.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", drv)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			slog.Warn("close migrator", slog.Any("source_error", srcErr), slog.Any("db_error", dbErr))
		}
	}()

 var n int
	if len(args) > 0 && args[0] != "all" {
		if n, err = strconv.Atoi(args[0]); err != nil {
			return fmt.Errorf("invalid argument %q: want a number", args[0])
		}
	}

	switch action {
	case "up":
		if n < 0 {
			return fmt.Errorf("invalid step count %d: want a positive number", n)
		}
		if n > 0 {
			err = m.Steps(n)
		} else {
			err = m.Up()
		}
	case "down":
		switch {
		case len(args) == 0:
			return errors.New(`down requires a step count or "all"`)
		case args[0] == "all":
			err = m.Down()
		case n > 0:
			err = m.Steps(-n)
		default:
			return fmt.Errorf(`invalid step count %q: want a positive number or "all"`, args[0])
		}
	case "version":
		version, dirty, verErr := m.Version()
		if errors.Is(verErr, migrate.ErrNilVersion) {
			slog.Info("no migrations applied")
			return nil
		}
		if verErr != nil {
			return fmt.Errorf("version: %w", verErr)
		}
		slog.Info("migration version", slog.Uint64("version", uint64(version)), slog.Bool("dirty", dirty))
		return nil
	case "force":
		if len(args) == 0 {
			return errors.New("force requires a version argument")
		}
		err = m.Force(n)
	default:
		return fmt.Errorf("unknown action %q: want up, down, version or force", action)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("%s: %w", action, err)
	}
	return nil
}
