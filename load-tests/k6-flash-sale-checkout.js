import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  stages: [
    { duration: '30s', target: 50 },  // Ramp-up to 50 concurrent virtual users
    { duration: '1m', target: 200 },   // Stress test at 200 VUs
    { duration: '30s', target: 500 },  // Flash spike to 500 VUs
    { duration: '30s', target: 0 },    // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<250', 'p(99)<500'], // 95% of requests must complete below 250ms
    http_req_failed: ['rate<0.01'],                 // Under 1% failures tolerated under stress
  },
};

const BASE_URL = __ENV.API_GATEWAY_URL || 'http://localhost:8080';

export default function () {
  const customerId = `cust-${__VU}-${__ITER}`;
  const idempotencyKey = uuidv4();

  const payload = JSON.stringify({
    customer_id: customerId,
    shipping_address: {
      street: '123 Tech Boulevard',
      city: 'Ho Chi Minh City',
      country: 'VN',
    },
    items: [
      {
        product_id: 'prod-iphone-15-pro',
        sku: 'IPHONE15-TITAN-256',
        quantity: 1,
        unit_price: 1199.0,
      },
    ],
    payment_method: 'STRIPE_CARD',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': idempotencyKey,
      'X-Correlation-ID': uuidv4(),
    },
  };

  // 1. Submit Checkout Request with unique Idempotency Key
  const res = http.post(`${BASE_URL}/api/v1/orders`, payload, params);

  check(res, {
    'checkout status is 200 or 201': (r) => r.status === 200 || r.status === 201,
  });

  // 2. Immediate Retry with identical Idempotency Key (Verify Idempotency Resiliency)
  const retryRes = http.post(`${BASE_URL}/api/v1/orders`, payload, params);

  check(retryRes, {
    'retry returns idempotent deterministic response': (r) =>
      r.status === 200 || r.status === 201 || r.status === 409,
  });

  sleep(1);
}
