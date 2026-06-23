-- trace_context carries the W3C trace-context propagation fields (traceparent,
-- tracestate) captured when the event was created, so the relay can restore the
-- originating request's trace before publishing. Nullable: rows created without
-- an active trace simply leave it empty.
ALTER TABLE outbox_events ADD COLUMN trace_context JSONB;
