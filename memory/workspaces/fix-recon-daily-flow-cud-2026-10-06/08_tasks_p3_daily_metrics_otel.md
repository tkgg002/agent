# Danh Sách Nhiệm Vụ (Tasks) — Đẩy 2 Chỉ Số Kafka CDC Insert & Master Inserted Vào Metric Theo Từng Ngày

## Checklist Thực Thi

- [ ] **Task 1: Khai báo và Khởi tạo OTel Gauges trong `pkgs/metrics/cms_metrics.go`**
  - File: `cdc-cms-service/pkgs/metrics/cms_metrics.go`
  - Thêm `dailyFlowKafkaInserted` và `dailyFlowMasterInserted`.
  - Khởi tạo trong `Init()`.
  - Thêm struct `DailyFlowMetricEntry` và hàm `RecordDailyFlowMetrics`.

- [ ] **Task 2: Phát tán Metric tại API Handler `DailyFlow`**
  - File: `cdc-cms-service/internal/api/system/activity_log_handler.go`
  - Gọi `RecordDailyFlowMetrics` sau khi query `GetDailyFlow` thành công.

- [ ] **Task 3: Xây dựng Periodic Metric Collector**
  - File: `cdc-cms-service/internal/infra/observability/daily_flow_metric_collector.go`
  - Lấy danh sách active pipelines từ DB (`cdc_shadow_binding` + `cdc_master_binding`).
  - Lặp qua từng pipeline, query `GetDailyFlow` cho 7 ngày gần nhất và emit metric.

- [ ] **Task 4: Quản lý Vòng Đời Collector trong `Server`**
  - File: `cdc-cms-service/internal/server/server.go`
  - Thêm trường `dailyFlowCollector` và `dailyFlowCancel context.CancelFunc`.
  - Khởi động trong `Start()` và dừng an toàn trong `Shutdown()`.

- [ ] **Task 5: Viết Unit Test Kiểm Chứng**
  - File: `cdc-cms-service/pkgs/metrics/cms_metrics_test.go`
  - Kiểm tra `RecordDailyFlowMetrics` chạy mượt mà, không panic khi no-op meter hoặc OTel meter.

- [ ] **Task 6: Build & Verification**
  - Chạy `go build ./cmd/... ./internal/...`
  - Chạy `go test ./pkgs/metrics/...`
  - Báo cáo kết quả và tài liệu hướng dẫn PromQL query cho Grafana/SigNoz.
