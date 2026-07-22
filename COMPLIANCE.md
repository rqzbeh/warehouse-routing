# تطبیق با نیازمندی‌های PDF

## صورت مسئله

سرویس `Warehouse Routing Service` بعد از دریافت سفارش، انبار مناسب را با هدف کاهش هزینه لجستیک انتخاب می‌کند. پیاده‌سازی در `cmd/server`، منطق انتخاب در `internal/routing` و دسترسی به موجودی در `internal/store` قرار دارد.

## ورودی‌های API

| نیاز PDF | وضعیت پیاده‌سازی |
| --- | --- |
| مختصات جغرافیایی مشتری | `customer_location.lat` و `customer_location.lon` |
| ماتریس هزینه حمل از هر انبار | `transportation_costs[]` با `warehouse_id`، `cost` و `eta_minutes` |
| ضرایب ترافیک جاده‌ای | `logistics_constraints[].traffic_coefficient` |
| اولویت ناوگان توزیع سنگین | `logistics_constraints[].fleet_priority_factor` |
| بازه زمانی محدودیت لجستیکی | `requested_at` در درخواست و `start_time`/`end_time` در هر constraint |
| بررسی SKU و موجودی | `sku` و `quantity`، سپس query روی جدول `inventory` |
| ۳۳ هاب انبارداری | seed دیتابیس شامل ۳۳ انبار نمونه است و سرویس به تعداد ثابت وابسته نیست |

## خروجی‌های API

خروجی موفق `POST /route` دقیقا این فیلدها را دارد:

- `WarehouseID`
- `EstimatedDeliveryTime`
- `TotalCost`
- `RouteOptimizationScore`

نمونه خروجی در `README.md` و schema در `openapi.yaml` ثبت شده است.

## الزامات فنی

| الزام PDF | وضعیت |
| --- | --- |
| Microservice | یک سرویس HTTP مستقل با Go |
| Stateless | state در حافظه برنامه نگهداری نمی‌شود؛ موجودی در PostgreSQL است |
| Scale-out / Kubernetes | مانفیست‌های `k8s/deployment.yaml`، `k8s/service.yaml` و `k8s/hpa.yaml` |
| Search & Persistence | PostgreSQL با ایندکس `inventory(sku, available_quantity)` |
| Latency زیر ۲۰۰ms | تست پذیرش و k6 روی VPS-local مقدار p95 و max کمتر از ۲۰۰ms را enforce می‌کنند |
| CI/CD | `.github/workflows/ci.yml` شامل test، vet، SonarQube و Docker build |
| Docker | `Dockerfile` چندمرحله‌ای و `docker-compose.yml` |

## معیارهای ارزیابی

| معیار | پوشش پروژه |
| --- | --- |
| Optimization | امتیازدهی وزن‌دار با هزینه حمل، فاصله، ترافیک، ناوگان و زمان تحویل |
| Performance | pool دیتابیس، timeout درخواست و تست پذیرش/بار |
| Code Quality | تفکیک HTTP، routing و store؛ تست واحد و یکپارچه |
| Innovation | تحلیل محدودیت‌های ترافیکی با بازه زمانی و ضریب اولویت ناوگان |

## ساختار ارسال

- لینک GitHub: `https://github.com/rqzbeh/warehouse-routing.git`
- راه‌اندازی Docker Compose در `README.md`
- تحلیل الگوریتم و چالش‌ها در `TECHNICAL_ANALYSIS.md`
- تست پذیرش در `scripts/acceptance_test.py`
- تست بار در `scripts/k6-route.js`
