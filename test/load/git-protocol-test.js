import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.GITANT_URL || 'http://localhost:7777';
const AUTH_TOKEN = __ENV.GITANT_TOKEN || '';

export const options = {
  stages: [
    { duration: '30s', target: 5 },
    { duration: '2m', target: 5 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
  },
};

export default function () {
  // info/refs discovery
  const refsRes = http.get(`${BASE_URL}/api/v1/repos/test/info/refs?service=git-upload-pack`);
  check(refsRes, {
    'info/refs accessible': (r) => r.status === 200 || r.status === 404,
  });

  // clone (download pack)
  const cloneRes = http.get(`${BASE_URL}/api/v1/repos/test/clone`);
  check(cloneRes, {
    'clone accessible': (r) => r.status === 200 || r.status === 404,
  });

  sleep(2);
}
