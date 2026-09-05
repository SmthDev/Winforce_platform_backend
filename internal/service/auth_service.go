
package service

import (
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/thecodearcher/limen"
	sqladapter "github.com/thecodearcher/limen/adapters/sql"
	credentialpassword "github.com/thecodearcher/limen/plugins/credential-password"

	"platform/backend/internal/config"
)


const AuthBasePath = "/api/v1/auth"


func NewAuth(cfg *config.Config, pool *pgxpool.Pool) (*limen.Limen, *sql.DB, error) {
	sqlDB := stdlib.OpenDBFromPool(pool)

	instance, err := limen.New(&limen.Config{
		BaseURL:  cfg.BaseURL,
		Database: sqladapter.NewPostgreSQL(sqlDB),
		Secret:   []byte(cfg.LimenSecret),
		Plugins: []limen.Plugin{
			credentialpassword.New(),
		},
		HTTP: limen.NewDefaultHTTPConfig(
			limen.WithHTTPBasePath(AuthBasePath),
			limen.WithHTTPTrustedOrigins(cfg.AllowedOrigins),
			limen.WithHTTPCookieSecure(cfg.CookieSecure),
		),

		Email: limen.NewDefaultEmailConfig(
			limen.WithEmailVerification(limen.WithDisableEmailVerification()),
		),
	})
	if err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("init limen: %w", err)
	}

	return instance, sqlDB, nil
}
