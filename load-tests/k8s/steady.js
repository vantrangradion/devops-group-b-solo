// Steady background load for the live demo (#140). Runs inside the cluster,
// so the dashboard is never empty and a fault toggle flips against a flat
// baseline. No fail= in the mix: every 5xx during the demo comes from a fault
// toggle or a real problem.
//
// About 5 requests per second in total: 2 VUs, each iteration is one request
// (about 100 ms on average) and a 300 ms pause.
import http from 'k6/http';
import { sleep } from 'k6';
import exec from 'k6/execution';

const BASE_URL = __ENV.BASE_URL || 'http://devops-app:8080';

export const options = {
  scenarios: {
    steady: {
      executor: 'constant-vus',
      vus: 2,
      duration: '24h',
    },
  },
  // No thresholds: a demo fault must not stop the run.
  thresholds: {},
  // kube-proxy balances per connection, not per request. With kept-alive
  // connections, both VUs can stick to one pod and the other gets nothing.
  noConnectionReuse: true,
  // Only the end-of-test summary would use these; skip the work.
  summaryTrendStats: ['avg', 'p(95)'],
};

// The User-Agent tells this traffic apart in Tempo
// ({ span.user_agent.original = "k6-steady" }) and in Loki records.
const params = {
  headers: { 'User-Agent': 'k6-steady' },
  tags: { load: 'steady' },
};

export default function () {
  const ms = Math.random() < 0.8 ? 50 : 300;
  const res = http.get(`${BASE_URL}/api/work?ms=${ms}`, params);
  // k6 runs with --quiet, so this is the only sign of life in kubectl logs:
  // one line per VU about every 2 minutes.
  if (exec.vu.iterationInScenario % 300 === 0) {
    console.log(`vu ${exec.vu.idInTest}: iteration ${exec.vu.iterationInScenario}, last status ${res.status}`);
  }
  sleep(0.3);
}
