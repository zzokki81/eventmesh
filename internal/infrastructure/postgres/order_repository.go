package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zzokki81/eventmesh/internal/domain/order"
)

// OrderRepository persists Order entities in Postgres.
// It implements the order.Repository interface defined in the domain layer.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository constructs a new OrderRepository with the given Postgres connection pool.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// Create inserts a new order into the database.
// Returns an error if the operation fails.
func (r *OrderRepository) Create(ctx context.Context, o *order.Order) error {
	q := `INSERT INTO orders (id, user_id, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, q,
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
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	var o order.Order

	q := `SELECT id, user_id, amount, status, created_at, updated_at
		  FROM orders
		  WHERE id = $1`

	err := r.pool.QueryRow(ctx, q, id).Scan(
		&o.ID,
		&o.UserID,
		&o.Amount,
		&o.Status,
		&o.CreatedAt,
		&o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, order.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: get order by id: %w", err)
	}
	return &o, nil
}

// Compile-time check that *OrderRepository implements order.Repository.
var _ order.Repository = (*OrderRepository)(nil)
