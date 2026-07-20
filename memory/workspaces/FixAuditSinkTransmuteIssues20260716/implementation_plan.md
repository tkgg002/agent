# Implementation Plan - Phase 3: Khắc Phục Rủi Ro Còn Tồn Đọng (Remaining Risks)

Kế hoạch này giải quyết triệt để 4 rủi ro lớn còn tồn đọng chặng Sink & Transmute được ghi nhận sau khi tự kiểm toán.

## User Review Required
> [!IMPORTANT]
> - **SINK-H5 (Fallback Protection):** Việc phân tách lỗi DB transient khi fallback ghi từng dòng sẽ thay đổi hành vi: Khi gặp lỗi mất mạng hoặc timeout DB, worker sẽ dừng và không commit offset (bảo toàn at-least-once) thay vì tự động đẩy hết vào DLQ như trước. Điều này đảm bảo an toàn dữ liệu tuyệt đối nhưng đòi hỏi hệ thống tự động restart worker (Supervisor/K8s pod restart) hoạt động đúng đắn.
> - **Chẩn đoán Lỗi Chuẩn PostgreSQL (SQLSTATE):** Chúng tôi tích hợp thư viện `"github.com/jackc/pgx/v5/pgconn"` để trích xuất mã lỗi SQLSTATE của PostgreSQL (ví dụ `08xxx`, `40001`, `40P01`, `57P01`) thay vì chỉ check chuỗi tin nhắn lỏng lẻo. Các lỗi nghiệp vụ (như `23505` unique, `23503` fk violation) được xác định là Permanent Errors và đẩy vô DLQ để bảo vệ luồng realtime không bị kẹt (Stalled Pipeline).
> - **Phòng ngừa Prometheus Cardinality Leak:** Đảm bảo metrics `cdc_transmute_rule_dropped_total` chỉ sử dụng các nhãn tĩnh giới hạn bởi schema (`master_table`, `source_field`, `target_column`, `reason`). Tuyệt đối không đưa các nhãn động (như `rule_id` tự sinh, hoặc nội dung của field bị lỗi) vào Prometheus labels để tránh bùng nổ timeseries (High Cardinality).
> - **TX-H3 & TX-H6:** Hai rủi ro này liên quan đến rủi ro cơ sở hạ tầng (Clock Skew và Hash Collision). Chúng tôi sẽ thực hiện phân tích chuyên sâu và đề xuất thiết kế trong tệp tin nghiên cứu, chưa thay đổi logic code ở phase này để tránh over-engineering.

---

## Proposed Changes

### Component: Prometheus Metrics

#### [MODIFY] [prometheus.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go)
- Khai báo metric `RulesDropped`: `cdc_transmute_rule_dropped_total` với các labels `master_table`, `source_field`, `target_column`, `reason`.

```go
var RulesDropped = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "cdc_transmute_rule_dropped_total",
		Help: "Total transmute rules dropped by reason (limited bounds by schema)",
	},
	[]string{"master_table", "source_field", "target_column", "reason"},
)
```

---

### Component: Transmute Module

#### [MODIFY] [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)
- Cập nhật hàm `loadRules()` để in Warn log và tăng metric `cdc_transmute_rule_dropped_total` khi rule bị drop do DataType hoặc TransformFn không hợp lệ.
- Cập nhật hàm helper `isRetryableDBError` trong transmuter.go để cast sang `*pgconn.PgError` và check mã SQLSTATE chuẩn của Postgres.

```go
	valid := rules[:0]
	for _, r := range rules {
		if r.TransformFn != nil && !IsTransformWhitelisted(*r.TransformFn) {
			t.logger.Warn("loadRules: skipping rule with non-whitelisted TransformFn",
				zap.String("master", row.MasterTable),
				zap.String("source_field", r.SourceField),
				zap.String("target_column", r.TargetColumn),
				zap.String("transform_fn", *r.TransformFn),
			)
			metrics.RulesDropped.WithLabelValues(row.MasterTable, r.SourceField, r.TargetColumn, "non_whitelisted_transform_fn").Inc()
			continue
		}
		if !t.typeRes.Validate(r.DataType) {
			t.logger.Warn("loadRules: skipping rule with invalid DataType",
				zap.String("master", row.MasterTable),
				zap.String("source_field", r.SourceField),
				zap.String("target_column", r.TargetColumn),
				zap.String("data_type", r.DataType),
			)
			metrics.RulesDropped.WithLabelValues(row.MasterTable, r.SourceField, r.TargetColumn, "invalid_data_type").Inc()
			continue
		}
		if strings.ContainsRune(r.TargetColumn, '$') {
			t.logger.Warn("loadRules: skipping rule with invalid PG identifier", zap.String("target_column", r.TargetColumn))
			metrics.RulesDropped.WithLabelValues(row.MasterTable, r.SourceField, r.TargetColumn, "invalid_pg_identifier").Inc()
			continue
		}
		valid = append(valid, r)
	}
```

---

### Component: Sink Batch Buffer

#### [MODIFY] [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go)
- Import `"strings"` ở đầu file.
- Bổ sung helper `isRetryableDBError` kiểm tra lỗi transient (lock timeout, connection refused, network disconnect).
- Trong vòng lặp sequential fallback: Nếu `res.Error` hoặc `err` là lỗi transient DB, lập tức trả về lỗi (return error) ra ngoài `Flush()`, dừng vòng lặp và ngăn commit offset Kafka.

---

## Verification Plan

### Automated Tests
- Chạy unit tests hiện có để verify không phát sinh regression:
  - `go test -v ./internal/handler/shadow/...`
  - `go test -v ./internal/service/master/...`
  - `go build -o /dev/null ./cmd/...`
