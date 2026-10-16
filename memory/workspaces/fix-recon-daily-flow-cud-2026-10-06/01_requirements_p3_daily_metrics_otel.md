# Yêu Cầu Kỹ Thuật (Requirements) — Đẩy 2 Chỉ Số Kafka CDC Insert & Master Inserted Vào Metric Theo Từng Ngày (OTel / Prometheus)

## 1. Bối cảnh & Mục tiêu
- **Bối cảnh:** 
  Hệ thống CDC đa nguồn đẩy dữ liệu từ Database nguồn qua Debezium/Kafka CDC vào Shadow DB, sau đó Transmute Worker tổng hợp sang Master DB.
  Người vận hành cần giám sát trực tiếp trên Grafana / SigNoz thông số:
  1. Mỗi ngày Debezium/Kafka CDC bắn về bao nhiêu bản ghi Insert (`kafka.c`).
  2. Mỗi ngày Master DB thực sự ghi nhận bao nhiêu bản ghi mới Inserted (`master.inserted`).
  Độ lệch giữa 2 chỉ số này là tín hiệu cảnh báo quan trọng nhất về tính toàn vẹn của luồng CDC hàng ngày.
- **Mục tiêu:**
  Đẩy 2 thông tin này vào OpenTelemetry / Prometheus Metrics tại Backend `cdc-cms-service` để Grafana và SigNoz thu thập (scrape / export) liên tục 24/7 theo từng ngày và theo từng pipeline.

---

## 2. Đặc tả Metrics (OpenTelemetry & Prometheus Specification)

### Metric 1: Số bản ghi Kafka CDC Insert theo ngày
- **Tên metric (OTel):** `cms.daily_flow.kafka_inserted`
- **Tên khi query PromQL (Grafana/SigNoz):** `cms_daily_flow_kafka_inserted`
- **Loại:** `Int64Gauge` (Giá trị đo lường số lượng bản ghi phát sinh trong một ngày cụ thể)
- **Đơn vị (Unit):** `records`
- **Mô tả:** `Total Kafka CDC insert (op=c) events received per calendar day for a given pipeline`
- **Labels / Dimensions:**
  - `shadow_schema`: Schema của bảng Shadow (ví dụ: `shadow_traitestpbs`)
  - `shadow_table`: Tên bảng Shadow (ví dụ: `payment_bills`)
  - `master_table`: Tên bảng Master (ví dụ: `payment_bills`)
  - `source_database`: Tên Database nguồn (ví dụ: `payment-bill-service`)
  - `date`: Ngày theo lịch địa phương `YYYY-MM-DD` (ví dụ: `2026-10-12`)

### Metric 2: Số bản ghi Master DB Inserted theo ngày
- **Tên metric (OTel):** `cms.daily_flow.master_inserted`
- **Tên khi query PromQL (Grafana/SigNoz):** `cms_daily_flow_master_inserted`
- **Loại:** `Int64Gauge`
- **Đơn vị (Unit):** `records`
- **Mô tả:** `Total records inserted into Master DB (transmute) per calendar day for a given pipeline`
- **Labels / Dimensions:** (Tương tự Metric 1)
  - `shadow_schema`, `shadow_table`, `master_table`, `source_database`, `date`

---

## 3. Cơ chế Thu thập & Cập nhật (Collection & Emission Flow)

### Chặng A: Cập nhật Tức thời khi có Truy vấn (On-Demand Emission)
- Khi API `/api/reconciliation/daily-flow` được gọi (từ CMS Web hoặc hệ thống ngoài), kết quả tổng hợp ngày từ `GetDailyFlow` sẽ lập tức được phát tán vào OTel Instruments.

### Chặng B: Worker Thu thập Nền Tự Động (Periodic Background Collector)
- Khởi chạy một goroutine nền trong `cdc-cms-service` (chu kỳ mặc định: 60 giây).
- Tự động lấy danh sách active pipelines từ DB (`shadow_binding` + `master_binding`).
- Chạy `GetDailyFlow` cho 7 ngày gần nhất (sử dụng Partition Pruning cực nhanh <10ms).
- Ghi đè Gauge mới nhất cho từng ngày vào OTel SDK.
- OTel Periodic Reader (30s) sẽ tự động push các metrics này về SigNoz / Prometheus.

---

## 4. Definition of Done (DoD)
- [ ] Metric instruments được đăng ký chuẩn trong `pkgs/metrics/cms_metrics.go`.
- [ ] Không crash khi OTel disabled hoặc collector unreachable.
- [ ] Labels cách ly triệt để đa connector (`shadow_schema` + `shadow_table` + `source_database`).
- [ ] Biên dịch `go build ./cmd/... ./internal/...` không lỗi (Exit code 0).
- [ ] Chạy unit test kiểm chứng phát tán metric thành công.
