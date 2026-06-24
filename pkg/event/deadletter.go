package event

import "time"

// DeadLetter wraps an event that could not be processed — either it failed
// unrecoverably or it exhausted its retries — together with the metadata
// needed to diagnose the failure and, later, redrive the original event back
// onto its subject.
type DeadLetter struct {
	// Event is the original envelope that failed processing.
	Event Envelope `json:"event"`

	// Subject is the subject the event was originally delivered on, so a
	// redrive can republish it to the same destination.
	Subject string `json:"subject"`

	// Reason is the processing error that sent the event to the dead-letter
	// queue.
	Reason string `json:"reason"`

	// Deliveries is how many times delivery was attempted before the event
	// was dead-lettered.
	Deliveries uint64 `json:"deliveries"`

	// FailedAt is the UTC timestamp when the event was dead-lettered.
	FailedAt time.Time `json:"failed_at"`
}

// NewDeadLetter wraps a failed envelope with its failure metadata, stamping
// FailedAt with the current UTC time.
func NewDeadLetter(env Envelope, subject, reason string, deliveries uint64) DeadLetter {
	return DeadLetter{
		Event:      env,
		Subject:    subject,
		Reason:     reason,
		Deliveries: deliveries,
		FailedAt:   time.Now().UTC(),
	}
}
