package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/zzokki81/eventmesh/internal/pkg/errs"
)

// ErrInvalidJSON is returned when a request body cannot be parsed as JSON.
var ErrInvalidJSON = errors.New("invalid json")

// ErrorBody is the public shape of an error payload.
type ErrorBody struct {
	// Code is a stable machine-readable error identifier.
	Code string `json:"code"`

	// Message is a human-readable description of the error.
	Message string `json:"message"`
}

// errorResponse is the JSON envelope used for error responses.
type errorResponse struct {
	// Error carries the error details for the client.
	Error ErrorBody `json:"error"`
}

// WriteError serializes err into a JSON error response with an appropriate
// HTTP status. Unknown errors are reported as 500 Internal Server Error
// and logged so they remain visible after being hidden from the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	status, body := mapError(err)

	if status >= 500 {
		slog.ErrorContext(r.Context(), "http server error",
			"err", err,
			"path", r.URL.Path,
			"method", r.Method,
		)
	}

	WriteJSON(w, r, status, errorResponse{Error: body})
}

// mapError translates a domain or transport error into the HTTP status
// and ErrorBody returned to the client.
func mapError(err error) (int, ErrorBody) {
	switch {
	case errors.Is(err, ErrInvalidJSON):
		return http.StatusBadRequest, ErrorBody{
			Code:    "INVALID_JSON",
			Message: err.Error(),
		}

	case errors.Is(err, errs.ErrOrderNotFound):
		return http.StatusNotFound, ErrorBody{
			Code:    "ORDER_NOT_FOUND",
			Message: err.Error(),
		}

	case errors.Is(err, errs.ErrInvalidOrderUserID):
		return http.StatusBadRequest, ErrorBody{
			Code:    "INVALID_ORDER_USER_ID",
			Message: err.Error(),
		}

	case errors.Is(err, errs.ErrInvalidOrderAmount):
		return http.StatusBadRequest, ErrorBody{
			Code:    "INVALID_ORDER_AMOUNT",
			Message: err.Error(),
		}

	case errors.Is(err, errs.ErrInvalidOrderStatus):
		return http.StatusBadRequest, ErrorBody{
			Code:    "INVALID_ORDER_STATUS",
			Message: err.Error(),
		}

	default:
		return http.StatusInternalServerError, ErrorBody{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "An unexpected error occurred",
		}
	}
}
