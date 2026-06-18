package domain

import "errors"

// ErrInProgress indicates another attempt currently holds the event. It is
// transient: the caller should retry rather than skip or fail permanently.
var ErrInProgress = errors.New("event processing in progress")
