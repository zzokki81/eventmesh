package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zzokki81/eventmesh/internal/entities/order"
	"github.com/zzokki81/eventmesh/internal/pkg/errs"

	storage "github.com/zzokki81/eventmesh/internal/storage/orders"
)

// Storage persists Order aggregates in PostgreSQL.
//
// It uses pgxpool.Pool to execute queries against the orders table and maps
// pgx-level errors into domain errors defined in the entities/order package
// (e.g. pgx.ErrNoRows is translated into order.ErrNotFound).
//
// Storage satisfies the orders.OrderRepository contract and is safe for
// concurrent use; the underlying pool handles connection multiplexing.
type Storage struct {
	pool *pgxpool.Pool
}

// NewStorage returns a Storage backed by the given connection pool. The pool
// must be initialized and pinged before being passed in; callers are
// responsible for its lifecycle (typically closed during application shutdown).
func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

// CreateInTx inserts a new order into the orders table within the caller-owned transaction tx.
//
//	The transaction must be committed for the order to be persisted.
//	Returns an error if the insert fails; the caller is responsible for rolling back the transaction.
func (s *Storage) CreateInTx(ctx context.Context, tx pgx.Tx, o *order.Order) error {
	q := `INSERT INTO orders (id, user_id, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := tx.Exec(ctx, q,
		o.ID,
		o.UserID,
		o.Amount,
		o.Status,
		o.CreatedAt,
		o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// GetByID retrieves an order by its unique identifier.
// Returns order.ErrNotFound if no order matches.
func (s *Storage) GetByID(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	var o order.Order

	q := `SELECT id, user_id, amount, status, created_at, updated_at
		  FROM orders
		  WHERE id = $1`

	err := s.pool.QueryRow(ctx, q, id).Scan(
		&o.ID,
		&o.UserID,
		&o.Amount,
		&o.Status,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrOrderNotFound
		}
		return nil, fmt.Errorf("postgres: get order by id: %w", err)
	}
	return &o, nil
}

// Compile-time check that *Storage implements orders.OrderRepository.
var _ storage.OrderRepository = (*Storage)(nil)
