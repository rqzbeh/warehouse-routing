#!/usr/bin/env python3
import json
import statistics
import sys
import time
import urllib.error
import urllib.request


BASE_URL = sys.argv[1].rstrip("/") if len(sys.argv) > 1 else "http://localhost:8080"


def request(path, payload=None, timeout=10):
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(
        BASE_URL + path,
        data=data,
        headers={"Content-Type": "application/json"},
        method="GET" if payload is None else "POST",
    )
    start = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as res:
            body = res.read().decode()
            return res.status, body, (time.perf_counter() - start) * 1000
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read().decode(), (time.perf_counter() - start) * 1000


valid_payload = {
    "customer_location": {"lat": 35.7219, "lon": 51.3347},
    "sku": "SKU-1",
    "quantity": 1,
    "requested_at": "2026-07-22T12:00:00Z",
    "expected_delivery_time": 60,
    "transportation_costs": [
        {"warehouse_id": "tehran-west", "cost": 100000, "eta_minutes": 45},
        {"warehouse_id": "tehran-east", "cost": 85000, "eta_minutes": 70},
        {"warehouse_id": "karaj", "cost": 70000, "eta_minutes": 95},
    ] + [
        {"warehouse_id": warehouse_id, "cost": 2000000, "eta_minutes": 1000}
        for warehouse_id in (
            "mashhad",
            "isfahan",
            "shiraz",
            "tabriz",
            "ahvaz",
            "qom",
            "kermanshah",
            "urmia",
            "rasht",
            "zahedan",
            "hamedan",
            "kerman",
            "yazd",
            "ardabil",
            "bandar-abbas",
            "arak",
            "zanjan",
            "sanandaj",
            "qazvin",
            "gorgan",
            "sari",
            "khorramabad",
            "bushehr",
            "birjand",
            "bojnurd",
            "shahrekord",
            "ilam",
            "semnan",
            "yasuj",
            "kish",
        )
    ],
    "logistics_constraints": [
        {
            "warehouse_id": "tehran-west",
            "traffic_coefficient": 1.2,
            "fleet_priority_factor": 1.1,
            "start_time": "2026-07-22T11:00:00Z",
            "end_time": "2026-07-22T13:00:00Z",
        },
        {
            "warehouse_id": "tehran-east",
            "traffic_coefficient": 1.1,
            "fleet_priority_factor": 1,
            "start_time": "2026-07-22T11:00:00Z",
            "end_time": "2026-07-22T13:00:00Z",
        },
        {
            "warehouse_id": "karaj",
            "traffic_coefficient": 1,
            "fleet_priority_factor": 0.9,
            "start_time": "2026-07-22T11:00:00Z",
            "end_time": "2026-07-22T13:00:00Z",
        },
    ],
}


def assert_true(ok, message):
    if not ok:
        raise SystemExit(message)


status, _, _ = request("/readyz")
assert_true(status == 204, f"/readyz status={status}, want 204")

status, body, latency = request("/route", valid_payload)
assert_true(status == 200, f"/route status={status}, body={body}")
decision = json.loads(body)
for field in ("WarehouseID", "EstimatedDeliveryTime", "TotalCost", "RouteOptimizationScore"):
    assert_true(field in decision, f"missing output field {field}: {decision}")
assert_true(decision["WarehouseID"] == "tehran-west", f"WarehouseID={decision['WarehouseID']}, want tehran-west")
assert_true(decision["EstimatedDeliveryTime"] == 50, f"EstimatedDeliveryTime={decision['EstimatedDeliveryTime']}, want 50")
assert_true(decision["TotalCost"] == 114090.91, f"TotalCost={decision['TotalCost']}, want 114090.91")
assert_true(decision["RouteOptimizationScore"] == 46.71, f"RouteOptimizationScore={decision['RouteOptimizationScore']}, want 46.71")

bad_payload = dict(valid_payload)
bad_payload["customer_location"] = {"lat": 200, "lon": 51}
status, _, _ = request("/route", bad_payload)
assert_true(status == 400, f"bad request status={status}, want 400")

missing_stock = dict(valid_payload)
missing_stock["sku"] = "MISSING-SKU"
status, _, _ = request("/route", missing_stock)
assert_true(status == 404, f"missing stock status={status}, want 404")

missing_costs = dict(valid_payload)
missing_costs["transportation_costs"] = []
status, _, _ = request("/route", missing_costs)
assert_true(status == 400, f"missing transportation costs status={status}, want 400")

latencies = []
for _ in range(20):
    status, _, ms = request("/route", valid_payload)
    assert_true(status == 200, f"latency sample status={status}, want 200")
    latencies.append(ms)

latencies.sort()
p95 = latencies[int(len(latencies) * 0.95) - 1]
assert_true(p95 < 200, f"p95 latency={p95:.2f}ms, want <200ms")
assert_true(latencies[-1] < 200, f"max latency={latencies[-1]:.2f}ms, want <200ms")
print(json.dumps({
    "base_url": BASE_URL,
    "decision": decision,
    "single_request_ms": round(latency, 2),
    "latency_samples": len(latencies),
    "avg_ms": round(statistics.mean(latencies), 2),
    "p95_ms": round(p95, 2),
    "max_ms": round(latencies[-1], 2),
    "pdf_latency_target_ms": 200,
}, ensure_ascii=False, indent=2))
