// Envious mixed-workload load test (manual, not part of go test).
//
// Run:  API_BASE=http://127.0.0.1:8080 API_KEY=<key> k6 run web/test/load/k6.js
//
// Exercises the enterprise path: auth + writes (retried under contention) +
// reads, asserting availability and latency. Tune RATE_LIMIT_* server-side
// above the target RPS or the 429s will (correctly) fail the run.
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '1m', target: 20 },  // warm up
    { duration: '3m', target: 50 },  // sustained mixed load
    { duration: '30s', target: 0 },  // drain
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

const BASE = __ENV.API_BASE || 'http://127.0.0.1:8080';
const KEY = __ENV.API_KEY || '';

export function setup() {
  if (!KEY) throw new Error('API_KEY env is required');
  const res = http.post(
    `${BASE}/api/apps`,
    JSON.stringify({ name: `load-${Date.now()}` }),
    { headers: { 'Content-Type': 'application/json', 'X-API-Key': KEY } },
  );
  check(res, { 'setup app created': (r) => r.status === 201 });
  return { appID: res.json().id };
}

export default function (data) {
  const H = { 'Content-Type': 'application/json', 'X-API-Key': KEY };
  const envName = `env-${__VU}-${__ITER}`;

  let res = http.post(`${BASE}/api/apps/${data.appID}/envs`, JSON.stringify({ name: envName }), { headers: H });
  check(res, { 'env created': (r) => r.status === 201 });
  if (res.status !== 201) return;
  const envID = res.json().id;

  res = http.post(`${BASE}/api/envs/${envID}/vars`, JSON.stringify({ key: 'LOAD', value: 'x'.repeat(64) }), { headers: H });
  check(res, { 'var set': (r) => r.status === 200 });

  res = http.get(`${BASE}/api/envs/${envID}/vars`, { headers: { 'X-API-Key': KEY } });
  check(res, { 'vars listed': (r) => r.status === 200 });

  sleep(0.2);
}
