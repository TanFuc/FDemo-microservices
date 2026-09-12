import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 100 },  // 100 concurrent browsers
    { duration: '1m', target: 500 },   // Sustained 500 VUs
    { duration: '30s', target: 1000 },  // Peak traffic 1000 VUs
    { duration: '30s', target: 0 },    // Ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<50', 'p(99)<100'], // High-speed cache target < 50ms p95
    http_req_failed: ['rate<0.005'],              // Under 0.5% failure
  },
};

const BASE_URL = __ENV.API_GATEWAY_URL || 'http://localhost:8080';

const SEARCH_TERMS = ['laptop', 'iphone', 'shoes', 'headphones', 'keyboard', 'monitor'];

export default function () {
  // 1. Browse Catalog with Pagination & Category Filters
  const catalogRes = http.get(`${BASE_URL}/api/v1/catalog/products?page=1&limit=20&category=electronics`);
  check(catalogRes, {
    'catalog returned status 200': (r) => r.status === 200,
  });

  // 2. Full-Text Search with Fuzzy Term
  const randomTerm = SEARCH_TERMS[Math.floor(Math.random() * SEARCH_TERMS.length)];
  const searchRes = http.get(`${BASE_URL}/api/v1/search/products?q=${randomTerm}`);
  check(searchRes, {
    'search returned status 200': (r) => r.status === 200,
  });

  sleep(0.5);
}
