# سرویس تخصیص هوشمند انبار

این پروژه یک میکروسرویس Go برای چالش تخصیص انبار ایکامرس است. سرویس بعد از دریافت موقعیت مشتری، SKU، تعداد کالا، هزینه‌های حمل‌ونقل و محدودیت‌های لجستیکی، بهترین انبار را انتخاب می‌کند و موجودی همان انبار را به‌شکل اتمیک رزرو می‌کند.

## اجرای محلی با Docker Compose

```bash
docker compose up --build
```

بررسی سلامت پردازه:

```bash
curl -i http://localhost:8080/healthz
```

بررسی آماده‌بودن سرویس و اتصال دیتابیس:

```bash
curl -i http://localhost:8080/readyz
```

نمونه درخواست مسیر:

```bash
curl -s http://localhost:8080/route \
  -H 'Content-Type: application/json' \
  -d '{
    "customer_location": {"lat": 35.7219, "lon": 51.3347},
    "sku": "SKU-1",
    "quantity": 2,
    "expected_delivery_time": 60,
    "transportation_costs": [
      {"warehouse_id": "tehran-west", "cost": 100000, "eta_minutes": 45},
      {"warehouse_id": "tehran-east", "cost": 85000, "eta_minutes": 70},
      {"warehouse_id": "karaj", "cost": 70000, "eta_minutes": 95}
    ],
    "logistics_constraints": [
      {"warehouse_id": "tehran-west", "traffic_coefficient": 1.2, "fleet_priority_factor": 1.1},
      {"warehouse_id": "tehran-east", "traffic_coefficient": 1.1, "fleet_priority_factor": 1},
      {"warehouse_id": "karaj", "traffic_coefficient": 1, "fleet_priority_factor": 0.9}
    ]
  }'
```

نمونه پاسخ:

```json
{
  "WarehouseID": "tehran-west",
  "EstimatedDeliveryTime": 50,
  "TotalCost": 114090.91,
  "RouteOptimizationScore": 46.71
}
```

## API

`POST /route`

ورودی‌ها:

- `customer_location.lat` و `customer_location.lon`
- `sku`
- `quantity`
- `expected_delivery_time` اختیاری، به دقیقه
- `transportation_costs[]`
- `logistics_constraints[]`

خروجی‌ها:

- `WarehouseID`
- `EstimatedDeliveryTime`
- `TotalCost`
- `RouteOptimizationScore`

## منطق الگوریتم

سرویس ابتدا انبارهایی را پیدا می‌کند که برای SKU درخواستی موجودی کافی دارند. سپس برای هر انبار کاندید، هزینه کل را با ترکیب هزینه حمل، فاصله جغرافیایی، ضریب ترافیک، اولویت ناوگان و جریمه زمان تحویل محاسبه می‌کند. انباری که کمترین هزینه کل را داشته باشد انتخاب می‌شود.

در این نسخه از امتیازدهی وزن‌دار استفاده شده، چون ورودی چالش گراف جاده‌ای واقعی ندارد. اگر داده یال‌های مسیر و زمان واقعی بین نقاط اضافه شود، پیاده‌سازی Dijkstra یا A* انتخاب دقیق‌تری خواهد بود.

## سازگاری موجودی

رزرو موجودی با یک `UPDATE` شرطی در PostgreSQL انجام می‌شود:

- فقط وقتی `available_quantity >= quantity` باشد رزرو انجام می‌شود.
- هم‌زمانی درخواست‌ها نمی‌تواند موجودی منفی بسازد.
- سرویس stateless می‌ماند و وضعیت فقط در دیتابیس نگهداری می‌شود.

## تست‌ها

تست‌های واحد:

```bash
go test ./...
go vet ./...
```

تست پذیرش مطابق PDF روی سرویس اجراشده:

```bash
python3 scripts/acceptance_test.py http://localhost:8080
```

تست یکپارچه لایه دیتابیس:

```bash
TEST_DATABASE_URL='postgres://routing:routing@localhost:5432/routing?sslmode=disable' go test ./internal/store
```

## CI/CD

فایل `.github/workflows/ci.yml` این مراحل را اجرا می‌کند:

- `go test ./...`
- `go vet ./...`
- اسکن SonarQube در صورت وجود `SONAR_TOKEN`
- ساخت Docker image بدون انتشار

## Kubernetes

مانفیست‌های پایه در مسیر `k8s/` قرار دارند:

- `Deployment`
- `Service`
- `HorizontalPodAutoscaler`

مقدار `DATABASE_URL` باید از Secret با نام `warehouse-routing` و کلید `database-url` تامین شود.
