// Package postgres provides factories for database connection pools.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zzokki81/eventmesh/internal/config"
)

// NewPool constructs a new pgxpool.Pool configured from cfg.
// It verifies connectivity by pinging the database before returning.
//
// The caller is responsible for calling Close() on the returned pool
// when the application shuts down.
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	// Apply tuning parameters from application config.
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnIdleTime = cfg.ConnMaxIdleTime
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	// Verify the pool is actually usable. Without this, errors only surface
	// on the first query — much harder to debug than failing at startup.
	pingCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
