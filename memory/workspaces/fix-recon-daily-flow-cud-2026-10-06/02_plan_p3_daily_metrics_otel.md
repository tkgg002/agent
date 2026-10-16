# Kế Hoạch Triển Khai (Roadmap) — Đẩy 2 Chỉ Số Kafka CDC Insert & Master Inserted Vào Metric Theo Từng Ngày

## Mục tiêu
Triển khai giải pháp tối ưu nhất (The Single Best Approach): Tích hợp OpenTelemetry Gauges chuẩn mực vào `cdc-cms-service` kèm Periodic Metric Collector chạy ngầm 24/7 để SigNoz và Grafana cào và vẽ biểu đồ realtime.

---

## Các Bước Triển Khai (Phases)

### Phase 1: Mở Rộng OTel Instruments trong `pkgs/metrics/cms_metrics.go`
1. Khai báo 2 biến Gauge singleton: `dailyFlowKafkaInserted` và `dailyFlowMasterInserted`.
2. Khởi tạo instruments trong `Init()` với đúng namespace `cms.daily_flow.*`, Description và Unit `records`.
3. Tạo hàm phát tán `RecordDailyFlowMetrics(ctx, shadowSchema, shadowTable, masterTable, sourceDB, entries)`.

### Phase 2: Kích Hoạt Metric Emission trên API Handler `internal/api/system/activity_log_handler.go`
1. Trong phương thức `DailyFlow(c *fiber.Ctx)`, sau khi gọi `h.reader.GetDailyFlow(...)` thành công:
2. Trích xuất danh sách `[]cmsmetrics.DailyFlowMetricEntry` từ `result.Data`.
3. Gọi `cmsmetrics.RecordDailyFlowMetrics` để cập nhật ngay lập tức các Gauges.

### Phase 3: Xây Dựng Background Metric Collector `internal/infra/observability/daily_flow_metric_collector.go`
1. Tạo struct `DailyFlowMetricCollector` nhận `gorm.DB`, `ActivityLogReader`, `zap.Logger`, `interval time.Duration`.
2. Trong hàm `Run(ctx context.Context)`:
   - Chạy định kỳ (mỗi 60s).
   - Truy vấn danh sách các pipeline đang hoạt động (`shadow_binding` có `is_active = true` join `master_binding`).
   - Với mỗi pipeline: gọi `reader.GetDailyFlow` với `Days: 7` (hoặc 14), timezone offset của server/client.
   - Gọi `RecordDailyFlowMetrics` để đẩy toàn bộ chuỗi ngày vào OTel SDK.
3. Quản lý Lifecycle trong `internal/server/server.go`:
   - Khởi tạo `dailyFlowCollector` trong `server.New`.
   - Khởi chạy trong `server.Start()` với cancelable context.
   - Dừng gracefully trong `server.Shutdown()`.

### Phase 4: Kiểm Thử & Nghiệm Thu (Verification)
1. Kiểm tra build và compile toàn bộ backend `cdc-cms-service`: `go build ./cmd/... ./internal/...`
2. Chạy `go vet ./...`
3. Viết unit test trong `pkgs/metrics/cms_metrics_test.go` xác nhận việc record metric an toàn, không panic, đúng attributes.
4. Báo cáo nghiệm thu đầy đủ.
