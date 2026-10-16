# Hồ Sơ Toàn Bộ Code Cần Thực Hiện — Tận Dụng Metrics Prometheus Gốc Đã Có Sẵn

## 1. Bản Chất Kỹ Thuật
- **Hạ tầng đã có:** Worker (`centralized-data-service`) đã có sẵn endpoint Prometheus `/metrics` ([`server_setup.go:578`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go#L578)) và 2 Counters chuẩn:
  1. `cdc_sink_events_total` / `cdc_events_processed_total`: Đo số lượng event Kafka CDC bắn về theo `op` ("c", "u", "d") và `table`. Đã được gọi đầy đủ tại `event_handler.go:500-504`.
  2. `cdc_transmute_ops_total`: Đo số lượng record ghi vào Master DB theo `master_table` và `op` ("inserted", "updated", "skipped").
- **Điểm sót duy nhất trong toàn bộ hệ thống:** Trong file [`internal/handler/master/transmute_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go), metric `TransmuteOpsTotal` mới chỉ được cộng ở luồng Manual Batch (`HandleTransmute:318`), nhưng ở luồng **CDC Realtime Debouncer** (`runDebouncedGroup` và `runSubBatch`) thì chỉ ghi DB `cdc_activity_log` mà quên không gọi `TransmuteOpsTotal.Add(res.Inserted)`.
- **Mục tiêu can thiệp:** Chỉ cần bổ sung lệnh gọi `metrics.TransmuteOpsTotal` vào đúng 2 vị trí thành công của luồng Debouncer trong `transmute_handler.go`. Toàn bộ hệ thống sẽ kích hoạt đo lường tự nhiên 100% không cần chế thêm bất kỳ worker hay metric mới nào!

---

## 2. Chi Tiết Thay Đổi Code Duy Nhất (Single File Diff)

### File mục tiêu: [`centralized-data-service/internal/handler/master/transmute_handler.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)

#### Vị trí 1: Trong hàm `runDebouncedGroup` (Luồng mẻ Debounce thành công - dòng 547)

```diff
--- a/internal/handler/master/transmute_handler.go
+++ b/internal/handler/master/transmute_handler.go
@@ -546,6 +546,15 @@ func (h *TransmuteHandler) runDebouncedGroup(ctx context.Context, batch []NatsMsgTask) {
 					"master_binding_id": masterBindingID,
 				})
 			}
+			triggerType := triggeredBy
+			if triggerType == "" {
+				triggerType = "cdc-realtime"
+			}
+			metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "inserted", triggerType).Add(float64(res.Inserted))
+			metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "updated", triggerType).Add(float64(res.Updated))
+			metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "skipped", triggerType).Add(float64(res.Skipped))
+			metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "type_error", triggerType).Add(float64(res.TypeErrors))
+			metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "rule_miss", triggerType).Add(float64(res.RuleMisses))
+			metrics.TransmuteDurationMs.WithLabelValues(masterTable).Observe(float64(res.DurationMs))
 		}
 		// Phản hồi thành công cho toàn bộ tasks trong mẻ
 		for _, t := range batch {
```

#### Vị trí 2: Trong hàm `binarySearchSplit` (Luồng mẻ con sau khi split thành công - dòng 657)

```diff
--- a/internal/handler/master/transmute_handler.go
+++ b/internal/handler/master/transmute_handler.go
@@ -656,6 +656,13 @@ func (h *TransmuteHandler) binarySearchSplit(ctx context.Context, masterTable string, batch []NatsMsgTask) {
 					"master_binding_id": masterBindingID,
 				})
 			}
+			triggerType := triggeredBy
+			if triggerType == "" {
+				triggerType = "cdc-realtime-split"
+			}
+			metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "inserted", triggerType).Add(float64(res.Inserted))
+			metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "updated", triggerType).Add(float64(res.Updated))
+			metrics.TransmuteOpsTotal.WithLabelValues(masterTable, "skipped", triggerType).Add(float64(res.Skipped))
+			metrics.TransmuteDurationMs.WithLabelValues(masterTable).Observe(float64(res.DurationMs))
 		}
 		// Nửa này thành công -> reply OK cho tất cả các tasks trong nửa này
 		for _, t := range subBatch {
```

---

## 3. Cách Thức Truy Vấn Số Liệu Theo Ngày Trên Grafana / Prometheus

Không cần tạo thêm gauge ngày hay lưu string ngày vào label (chống Cardinality explosion cho TSDB), ta sử dụng câu lệnh PromQL chuẩn mực của Prometheus cho Counter:

### 1. Số bản ghi KAFKA CDC BẮN VỀ Insert theo từng ngày:
```promql
increase(cdc_sink_events_total{table="payment_bills", op="c"}[1d])
```
*(Hoặc query qua `cdc_events_processed_total{operation="c", table="payment_bills"}[1d]`)*

### 2. Số bản ghi GHI VÀO DB MASTER Inserted theo từng ngày:
```promql
increase(cdc_transmute_ops_total{master_table="payment_bills", op="inserted"}[1d])
```

### 3. Đối soát chênh lệch Insert hàng ngày (CDC Insert Drift):
```promql
increase(cdc_sink_events_total{table="payment_bills", op="c"}[1d]) 
- 
increase(cdc_transmute_ops_total{master_table="payment_bills", op="inserted"}[1d])
```
- Nếu $= 0$: Khớp 100%.
- Nếu $> 0$: Master chưa nạp kịp hoặc có thất thoát.
