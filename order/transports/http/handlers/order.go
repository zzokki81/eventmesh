package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/zzokki81/eventmesh/order/domain"
	"github.com/zzokki81/eventmesh/order/transports/http/dto"
	"github.com/zzokki81/eventmesh/order/transports/http/response"

	orderSvc "github.com/zzokki81/eventmesh/order/service"
)

// OrderHandler exposes order resources over HTTP.
type OrderHandler struct {
	service  orderSvc.Orders
	validate *validator.Validate
}

// NewOrderHandler constructs an OrderHandler backed by the given service.
func NewOrderHandler(s orderSvc.Orders) *OrderHandler {
	return &OrderHandler{
		service:  s,
		validate: validator.New(),
	}
}

// Create handles POST /orders. It decodes the request body, delegates to
// the service, and writes the resulting order or an error response.
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var d dto.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		response.WriteError(w, r, response.ErrInvalidJSON)
		return
	}

	if err := h.validate.Struct(d); err != nil {
		response.WriteValidationError(w, r, err)
		return
	}

	req, err := domain.CreateRequestFrom(d.UserID, d.UserEmail, d.Amount)
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

// Get handles GET /orders/{id}. It parses the order ID from the path,
// delegates to the service, and writes the resulting order or an error
// response.
func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseOrderID(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	o, err := h.service.Get(r.Context(), id)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	response.WriteJSON(w, r, http.StatusOK, dto.OrderResponseFrom(o))
}
