package event

// Subjects are the broker subjects on which events are published. They form
// part of the public messaging contract shared between producers and consumers,
// so both sides route on the same constant rather than a duplicated string.
const (
	// SubjectOrderCreated is published when a new order is created.
	SubjectOrderCreated = "orders.created"

	// SubjectOrdersAll is the wildcard matching every order subject. It is used
	// to bind the stream to the whole orders.* family, not to publish on.
	SubjectOrdersAll = "orders.>"
)

// Schema versions identify the payload shape carried under a subject. Bump the
// relevant version on any breaking change so consumers can branch on it.
const (
	// VersionOrderCreated is the schema version of the orders.created payload.
	VersionOrderCreated = "1"
)
