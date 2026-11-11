// Live demo load test for devops-app (brief section 3.2, step 5).
// Run against the k3d load balancer, which spreads load across both pods:
//   k6 run load-tests/test.js
// A short check run overrides the stages:
//   k6 run --stage 20s:10 --stage 10s:0 load-tests/test.js
import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Top-level stages run the ramping-vus executor. About 6 minutes in total.
export const options = {
  stages: [
    { duration: '1m', target: 20 },
    { duration: '2m', target: 100 },
    { duration: '2m', target: 100 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    // Deliberate 500s from fail= are expected responses (see below), so this
    // counts only real failures: refused connections, timeouts, other statuses.
    http_req_failed: ['rate<0.01'],
    'http_req_duration{expected_response:true}': ['p(95)<500'],
    checks: ['rate>0.99'],
  },
};

// Each mix moves a different dashboard panel: fast requests drive the rate,
// slow ones raise p95/p99, and fail=0.2 shows up in the 5xx ratio.
const MIX = [
  { weight: 0.75, name: 'fast', query: 'ms=50', statuses: [200] },
  { weight: 0.15, name: 'slow', query: 'ms=300', statuses: [200] },
  { weight: 0.1, name: 'flaky', query: 'ms=50&fail=0.2', statuses: [200, 500] },
];

const flakyCallback = http.expectedStatuses(200, 500);

function pick() {
  let r = Math.random();
  for (const m of MIX) {
    if (r < m.weight) return m;
    r -= m.weight;
  }
  return MIX[0];
}

export default function () {
  const m = pick();
  const params = { tags: { name: m.name } };
  if (m.statuses.includes(500)) params.responseCallback = flakyCallback;
  const res = http.get(`${BASE_URL}/api/work?${m.query}`, params);
  check(res, {
    'expected status': (r) => m.statuses.includes(r.status),
    'body has configmap message': (r) => typeof r.body === 'string' && r.body.includes('hello from configmap'),
  });
}
