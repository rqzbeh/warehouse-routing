import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 10,
  duration: '30s',
  thresholds: {
    checks: ['rate>0.99'],
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200', 'max<200'],
  },
};

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';

const distantCosts = [
  'mashhad',
  'isfahan',
  'shiraz',
  'tabriz',
  'ahvaz',
  'qom',
  'kermanshah',
  'urmia',
  'rasht',
  'zahedan',
  'hamedan',
  'kerman',
  'yazd',
  'ardabil',
  'bandar-abbas',
  'arak',
  'zanjan',
  'sanandaj',
  'qazvin',
  'gorgan',
  'sari',
  'khorramabad',
  'bushehr',
  'birjand',
  'bojnurd',
  'shahrekord',
  'ilam',
  'semnan',
  'yasuj',
  'kish',
].map((warehouseId) => ({
  warehouse_id: warehouseId,
  cost: 2000000,
  eta_minutes: 1000,
}));

const payload = JSON.stringify({
  customer_location: { lat: 35.7219, lon: 51.3347 },
  sku: __ENV.SKU || 'SKU-1',
  quantity: 1,
  requested_at: '2026-07-22T12:00:00Z',
  expected_delivery_time: 60,
  transportation_costs: [
    { warehouse_id: 'tehran-west', cost: 100000, eta_minutes: 45 },
    { warehouse_id: 'tehran-east', cost: 85000, eta_minutes: 70 },
    { warehouse_id: 'karaj', cost: 70000, eta_minutes: 95 },
  ].concat(distantCosts),
  logistics_constraints: [
    {
      warehouse_id: 'tehran-west',
      traffic_coefficient: 1.2,
      fleet_priority_factor: 1.1,
      start_time: '2026-07-22T11:00:00Z',
      end_time: '2026-07-22T13:00:00Z',
    },
    {
      warehouse_id: 'tehran-east',
      traffic_coefficient: 1.1,
      fleet_priority_factor: 1,
      start_time: '2026-07-22T11:00:00Z',
      end_time: '2026-07-22T13:00:00Z',
    },
    {
      warehouse_id: 'karaj',
      traffic_coefficient: 1,
      fleet_priority_factor: 0.9,
      start_time: '2026-07-22T11:00:00Z',
      end_time: '2026-07-22T13:00:00Z',
    },
  ],
});

export default function () {
  const res = http.post(`${baseURL}/route`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'returns WarehouseID': (r) => r.json('WarehouseID') === 'tehran-west',
  });
}
