
// k6 load test for the order service's POST /orders endpoint.
//
// Each request creates an order, which writes the order row and an outbox row
// in a single Postgres transaction. The relay then drains the outbox in the
// background. Run it against a locally running order service (make run-order)
// with the stack up (make up).
//
// Usage:
//   k6 run loadtest/orders.js
//   BASE_URL=http://localhost:8080 k6 run loadtest/orders.js
//
// While it runs, capture a CPU profile from the dev pprof server to see where
// the service spends its time (see loadtest/README.md).

import http from 'k6/http';
import { check } from 'k6';
import exec from 'k6/execution';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  // Ramp up to 50 concurrent virtual users, hold, then ramp down. With
  // DB_MAX_CONNS=25 the hold phase deliberately pushes past the pool size so
  // connection contention shows up under load.
  stages: [
    { duration: '30s', target: 50 },
    { duration: '1m', target: 50 },
    { duration: '10s', target: 0 },
  ],
  // Targets, not guarantees: a failed threshold is the signal to go profile.
  thresholds: {
    http_req_failed: ['rate<0.01'], // fewer than 1% non-2xx/errored requests
    http_req_duration: ['p(95)<300'], // 95% of requests under 300ms
  },
};

// uuidv4 returns a random RFC-4122 v4 UUID, matching the endpoint's
// `uuid4` validation on user_id.
function uuidv4() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

export default function () {
  const payload = JSON.stringify({
    user_id: uuidv4(),
    // VU number + this VU's iteration keeps each email unique across the run,
    // so the data stays valid even if email uniqueness is enforced later.
    user_email: `user${__VU}_${exec.vu.iterationInInstance}@example.com`,
    amount: (Math.random() * 1000).toFixed(2),
  });

  const res = http.post(`${BASE_URL}/orders`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 201': (r) => r.status === 201,
  });
}
