package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zzokki81/eventmesh/order/domain"
	"github.com/zzokki81/eventmesh/order/repository"
)

// OrdersRepository persists Order aggregates in PostgreSQL.
//
// It uses pgxpool.Pool to execute queries against the orders table and maps
// pgx-level errors into domain errors defined in the domain package
// (e.g. pgx.ErrNoRows is translated into domain.ErrOrderNotFound).
//
// OrdersRepository satisfies the repository.Orders contract and is safe for
// concurrent use; the underlying pool handles connection multiplexing.
type OrdersRepository struct {
	pool *pgxpool.Pool
}

// NewOrders returns an OrdersRepository backed by the given connection pool.
// The pool must be initialized and pinged before being passed in; callers are
// responsible for its lifecycle (typically closed during application shutdown).
func NewOrders(pool *pgxpool.Pool) *OrdersRepository {
	return &OrdersRepository{pool: pool}
}

// Create inserts a new order into the orders table within the caller-owned transaction tx.
//
// The transaction must be committed for the order to be persisted.
// Returns an error if the insert fails; the caller is responsible for rolling back the transaction.
func (s *OrdersRepository) Create(ctx context.Context, tx pgx.Tx, o *domain.Order) error {
	q := `INSERT INTO orders (id, user_id, user_email, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := tx.Exec(ctx, q,
		o.ID,
		o.UserID,
		o.UserEmail,
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
// Returns domain.ErrOrderNotFound if no order matches.
func (s *OrdersRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	var o domain.Order
	q := `SELECT id, user_id, user_email, amount, status, created_at, updated_at
		  FROM orders
		  WHERE id = $1`
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&o.ID,
		&o.UserID,
		&o.UserEmail,
		&o.Amount,
		&o.Status,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("postgres: get order by id: %w", err)
	}
	return &o, nil
}

// Compile-time check that *OrdersRepository implements repository.Orders.
var _ repository.Orders = (*OrdersRepository)(nil)
