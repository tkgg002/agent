# Báo cáo Triển Khai: Tận Dụng Metrics Prometheus Cho 2 Chỉ Số Kafka CDC Insert & Master Inserted Theo Ngày

- **Ngày thực hiện:** 2026-10-12
- **Tác vụ:** Đẩy 2 thông tin (1) Kafka CDC bắn về Insert và (2) Ghi vào DB Master Inserted vào metrics theo từng ngày.
- **Quy tắc thiết kế:** Simplicity First, Minimal Impact. Tận dụng 100% Prometheus Counter chuẩn có sẵn, không chế thêm background worker hay metric dư thừa.

---

## 1. Tổng Quan Thay Đổi (Code Diff Overview)

- **Số file thay đổi:** 1 file
- **File:** `centralized-data-service/internal/handler/master/transmute_handler.go`
- **Số dòng thêm mới:** +24 dòng code logic metrics

### Chi tiết thay đổi:
Trong file `transmute_handler.go`, luồng Manual Batch trước đây đã ghi nhận `TransmuteOpsTotal`, nhưng luồng **CDC Realtime Debouncer** (`runDebouncedGroup` và `processSubBatch`) mới chỉ ghi nhật ký vào `cdc_activity_log` mà quên chưa gọi `metrics.TransmuteOpsTotal.WithLabelValues(...).Add(...)`.

Chúng ta đã bổ sung:
1. Tại nhánh thành công của `runDebouncedGroup`:
```go
triggerType := triggeredBy
if triggerType == "" {
    triggerType = "cdc-realtime"
}
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "inserted", triggerType).Add(float64(res.Inserted))
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "updated", triggerType).Add(float64(res.Updated))
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "skipped", triggerType).Add(float64(res.Skipped))
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "type_error", triggerType).Add(float64(res.TypeErrors))
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "rule_miss", triggerType).Add(float64(res.RuleMisses))
metrics.TransmuteDurationMs.WithLabelValues(masterTable).Observe(float64(res.DurationMs))
```
2. Tại nhánh thành công của `processSubBatch`:
```go
triggerType := triggeredBy
if triggerType == "" {
    triggerType = "cdc-realtime-split"
}
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "inserted", triggerType).Add(float64(res.Inserted))
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "updated", triggerType).Add(float64(res.Updated))
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "skipped", triggerType).Add(float64(res.Skipped))
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "type_error", triggerType).Add(float64(res.TypeErrors))
metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "rule_miss", triggerType).Add(float64(res.RuleMisses))
metrics.TransmuteDurationMs.WithLabelValues(masterTable).Observe(float64(res.DurationMs))
```

---

## 2. Thông Tin Đo Lường & Hai Metric Chuẩn Sẵn Có

| Chỉ số | Prometheus Metric Name | Labels | Vị trí Emit trong Code |
| :--- | :--- | :--- | :--- |
| **1. Kafka CDC Bắn Về (Insert)** | `cdc_sink_events_total` | `table="<source_table>"`, `op="c"` | `internal/handler/shadow/event_handler.go:504` |
| **2. Ghi Vào DB Master (Inserted)** | `cdc_transmute_ops_total` | `master_table="<master_table>"`, `op="inserted"` | `internal/handler/master/transmute_handler.go` |

*(Lưu ý: `op="c"` viết tắt của Create/Insert trong chuẩn Debezium CDC).*

---

## 3. Hướng Dẫn Truy Vấn Số Liệu Theo Ngày Trên Prometheus / Grafana / SigNoz

Do bản chất 2 metric trên là **Prometheus Counters** (giá trị tăng đơn điệu theo thời gian), việc xem số lượng theo từng ngày (hoặc từng giờ) được thực hiện cực kỳ chuẩn xác và nhẹ nhàng bằng các hàm PromQL `increase()`:

### 3.1. Số lượng Kafka CDC Insert bắn về hôm nay (hoặc 1 ngày qua):
```promql
increase(cdc_sink_events_total{table="payment_bills", op="c"}[1d])
```

### 3.2. Số lượng bản ghi mới thực sự được Insert vào DB Master trong 1 ngày:
```promql
sum(increase(cdc_transmute_ops_total{master_table="payment_bills", op="inserted"}[1d]))
```

### 3.3. So sánh độ lệch (Drift / Chênh lệch) giữa Kafka CDC Insert và Master Inserted:
```promql
increase(cdc_sink_events_total{table="payment_bills", op="c"}[1d])
- sum(increase(cdc_transmute_ops_total{master_table="payment_bills", op="inserted"}[1d]))
```

### 3.4. Xem biểu đồ Bar Chart theo từng ngày trên Grafana:
- Trong Grafana Panel, chọn visualization: **Time series** hoặc **Bar chart**.
- Query:
  - Metric A: `increase(cdc_sink_events_total{table=~"$table", op="c"}[1d])` (Legend: `Kafka CDC Insert`)
  - Metric B: `sum by (master_table) (increase(cdc_transmute_ops_total{master_table=~"$table", op="inserted"}[1d]))` (Legend: `Master Inserted`)
- Min step: `1d`

---

## 4. Kết Quả Kiểm Thử (Verification)
- `go build ./internal/handler/master/...` -> **PASS (Exit 0)**
- `go build ./cmd/...` -> **PASS (Exit 0)**
- Không ảnh hưởng đến performance, tận dụng in-memory atomic counter của thư viện `prometheus/client_golang` (`promauto`).
