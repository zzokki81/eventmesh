// Package broker defines the messaging contracts of the event mesh.
//
// All transport-neutral contracts (Publisher, Subscriber, MessageHandler)
// live in interface.go. Per-provider implementations live in subpackages
// such as jetstream/. Concrete handlers that act on decoded envelopes
// live in handlers/.
//
//	broker/
//	├── interface.go    contracts: Publisher, Subscriber, MessageHandler
//	├── jetstream/      NATS JetStream implementations of the contracts
//	└── handlers/       concrete MessageHandler implementations
package broker
