package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/zzokki81/eventmesh/internal/entities/order"
	"github.com/zzokki81/eventmesh/internal/transports/http/dto"
	"github.com/zzokki81/eventmesh/internal/transports/http/response"

	orderSvc "github.com/zzokki81/eventmesh/internal/services/order"
)

// OrderHandler exposes order resources over HTTP.
type OrderHandler struct {
	// service performs the order use cases.
	service orderSvc.Service
}

// NewOrderHandler constructs an OrderHandler backed by the given service.
func NewOrderHandler(s orderSvc.Service) *OrderHandler {
	return &OrderHandler{service: s}
}

// Create handles POST /orders. It decodes the request body, delegates to
// the service, and writes the resulting order or an error response.
//
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var d dto.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		response.WriteError(w, r, response.ErrInvalidJSON)
		return
	}

	req, err := order.CreateRequestFrom(d.UserID, d.Amount)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	o, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.WriteJSON(w, r, http.StatusCreated, dto.OrderResponseFrom(o))
}
