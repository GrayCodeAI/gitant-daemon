import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const apiLatency = new Trend('api_latency', true);

const BASE_URL = __ENV.GITANT_URL || 'http://localhost:7777';
const AUTH_TOKEN = __ENV.GITANT_TOKEN || '';

const headers = {
  'Content-Type': 'application/json',
  ...(AUTH_TOKEN ? { 'Authorization': `Bearer ${AUTH_TOKEN}` } : {}),
};

export const options = {
  stages: [
    { duration: '30s', target: 10 },  // ramp up
    { duration: '1m', target: 10 },   // steady state
    { duration: '30s', target: 50 },  // spike
    { duration: '1m', target: 50 },   // sustained spike
    { duration: '30s', target: 0 },   // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    errors: ['rate<0.1'],
  },
};

export default function () {
  // Health check
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health status 200': (r) => r.status === 200,
  });

  // List repos
  const reposRes = http.get(`${BASE_URL}/api/v1/repos`, { headers });
  check(reposRes, {
    'repos status 200': (r) => r.status === 200,
  });
  errorRate.add(reposRes.status !== 200);
  apiLatency.add(reposRes.timings.duration);

  // Status
  const statusRes = http.get(`${BASE_URL}/api/v1/status`);
  check(statusRes, {
    'status 200': (r) => r.status === 200,
  });

  // Create a repo (requires auth)
  if (AUTH_TOKEN) {
    const repoName = `loadtest-${Date.now()}-${Math.random().toString(36).slice(2)}`;
    const createRes = http.post(
      `${BASE_URL}/api/v1/repos`,
      JSON.stringify({ name: repoName, description: 'load test repo' }),
      { headers }
    );
    check(createRes, {
      'create repo 201': (r) => r.status === 201,
    });
    errorRate.add(createRes.status !== 201);

    // Get the created repo
    const getRes = http.get(`${BASE_URL}/api/v1/repos/${repoName}`, { headers });
    check(getRes, {
      'get repo 200': (r) => r.status === 200,
    });

    // Create an issue
    const issueRes = http.post(
      `${BASE_URL}/api/v1/repos/${repoName}/issues`,
      JSON.stringify({ title: 'Load test issue', body: 'Created during load testing' }),
      { headers }
    );
    check(issueRes, {
      'create issue 201': (r) => r.status === 201,
    });

    // List issues
    const issuesRes = http.get(`${BASE_URL}/api/v1/repos/${repoName}/issues`, { headers });
    check(issuesRes, {
      'list issues 200': (r) => r.status === 200,
    });
  }

  sleep(1);
}
