package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/zzokki81/eventmesh/order/domain"
)

const validUUID = "550e8400-e29b-41d4-a716-446655440000"

func TestCreateRequestFrom(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		email   string
		amount  string
		wantErr error
	}{
		{
			name:   "valid input",
			userID: validUUID,
			email:  "buyer@example.com",
			amount: "42.00",
		},
		{
			name:    "invalid uuid",
			userID:  "not-a-uuid",
			email:   "buyer@example.com",
			amount:  "42.00",
			wantErr: domain.ErrInvalidOrderUserID,
		},
		{
			name:    "empty uuid",
			userID:  "",
			email:   "buyer@example.com",
			amount:  "42.00",
			wantErr: domain.ErrInvalidOrderUserID,
		},
		{
			name:    "unparseable amount",
			userID:  validUUID,
			email:   "buyer@example.com",
			amount:  "abc",
			wantErr: domain.ErrInvalidOrderAmount,
		},
		{
			name:    "zero amount",
			userID:  validUUID,
			email:   "buyer@example.com",
			amount:  "0",
			wantErr: domain.ErrInvalidOrderAmount,
		},
		{
			name:    "negative amount",
			userID:  validUUID,
			email:   "buyer@example.com",
			amount:  "-5.00",
			wantErr: domain.ErrInvalidOrderAmount,
		},
		{
			name:   "small positive amount",
			userID: validUUID,
			email:  "buyer@example.com",
			amount: "0.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := domain.CreateRequestFrom(tt.userID, tt.email, tt.amount)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: want %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr == nil && req == nil {
				t.Fatal("want a request, got nil")
			}
			if tt.wantErr != nil && req != nil {
				t.Errorf("want nil request on error, got %+v", req)
			}
		})
	}
}

// TestCreateRequestFrom_ParsesValues checks the raw strings are parsed onto the
// request, including the email which is passed through unvalidated at this layer.
func TestCreateRequestFrom_ParsesValues(t *testing.T) {
	req, err := domain.CreateRequestFrom(validUUID, "buyer@example.com", "42.50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.UserID != uuid.MustParse(validUUID) {
		t.Errorf("user id: want %s, got %s", validUUID, req.UserID)
	}
	if req.UserEmail != "buyer@example.com" {
		t.Errorf("email: want buyer@example.com, got %s", req.UserEmail)
	}
	if !req.Amount.Equal(decimal.RequireFromString("42.50")) {
		t.Errorf("amount: want 42.50, got %s", req.Amount)
	}
}
