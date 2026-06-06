import http from 'k6/http';
import { check } from 'k6';

const STRATEGY = __ENV.STRATEGY || 'naive';
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const PRODUCT_ID = __ENV.PRODUCT_ID || '1';

export const options = {
  scenarios: {
    stampede: {
      executor: 'shared-iterations',
      vus: 100,
      iterations: 5000,
      maxDuration: '60s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

export function setup() {
  const clearRes = http.post(`${BASE_URL}/cache/clear`);
  check(clearRes, { 'cache cleared': (r) => r.status === 200 });

  const resetRes = http.post(`${BASE_URL}/metrics/reset`);
  check(resetRes, { 'metrics reset': (r) => r.status === 200 });
}

export default function () {
  const res = http.get(
    `${BASE_URL}/products/${PRODUCT_ID}?strategy=${STRATEGY}`
  );
  check(res, {
    'status is 200': (r) => r.status === 200,
    'has body': (r) => r.body.length > 0,
  });
}

export function teardown() {
  const res = http.get(`${BASE_URL}/metrics`);
  if (res.status === 200) {
    console.log(`\n=== ${STRATEGY} metrics ===`);
    console.log(res.body);
  }
}
