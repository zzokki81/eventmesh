package response

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/zzokki81/eventmesh/order/domain"
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

// WriteValidationError writes a 400 Bad Request response with field-level
// details extracted from a validator.ValidationErrors value.
func WriteValidationError(w http.ResponseWriter, r *http.Request, err error) {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		WriteError(w, r, ErrInvalidJSON)
		return
	}

	fields := make([]string, 0, len(ve))
	for _, fe := range ve {
		fields = append(fields, fmt.Sprintf("%s: must satisfy '%s'", fe.Field(), fe.Tag()))
	}

	WriteJSON(w, r, http.StatusBadRequest, ErrorBody{
		Code:    "INVALID_REQUEST",
		Message: strings.Join(fields, "; "),
	})
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

	WriteJSON(w, r, status, body)
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

	case errors.Is(err, domain.ErrOrderNotFound):
		return http.StatusNotFound, ErrorBody{
			Code:    "ORDER_NOT_FOUND",
			Message: err.Error(),
		}

	case errors.Is(err, domain.ErrInvalidOrderID):
		return http.StatusBadRequest, ErrorBody{
			Code:    "INVALID_ORDER_ID",
			Message: err.Error(),
		}

	case errors.Is(err, domain.ErrInvalidOrderUserID):
		return http.StatusBadRequest, ErrorBody{
			Code:    "INVALID_ORDER_USER_ID",
			Message: err.Error(),
		}

	case errors.Is(err, domain.ErrInvalidOrderAmount):
		return http.StatusBadRequest, ErrorBody{
			Code:    "INVALID_ORDER_AMOUNT",
			Message: err.Error(),
		}

	case errors.Is(err, domain.ErrInvalidOrderStatus):
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
