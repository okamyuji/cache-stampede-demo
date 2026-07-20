import http from 'k6/http';
import { check } from 'k6';

const STRATEGY = __ENV.STRATEGY || 'naive';
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const PRODUCT_ID = __ENV.PRODUCT_ID || '1';
const ADMIN_KEY = __ENV.ADMIN_KEY || '';

// サーバー側でADMIN_KEYが設定されている場合、管理エンドポイントはX-Admin-Keyを要求する
const adminParams = ADMIN_KEY
  ? { headers: { 'X-Admin-Key': ADMIN_KEY } }
  : {};

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
  // 初期化に失敗したまま計測すると古いキャッシュと累積メトリクスで比較結果が汚染されるため、
  // 200以外は即座に試験全体を中断する
  const clearRes = http.post(`${BASE_URL}/cache/clear`, null, adminParams);
  if (clearRes.status !== 200) {
    throw new Error(
      `cache clear failed: status=${clearRes.status} (ADMIN_KEYの設定を確認してください)`
    );
  }

  const resetRes = http.post(`${BASE_URL}/metrics/reset`, null, adminParams);
  if (resetRes.status !== 200) {
    throw new Error(
      `metrics reset failed: status=${resetRes.status} (ADMIN_KEYの設定を確認してください)`
    );
  }
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
