# Báo Cáo Thay Đổi (Change Report) - Phase 3

Báo cáo này tổng hợp các thay đổi về mã nguồn và tài liệu trong Phase 3.

## 1. Danh sách các file đã sửa đổi

| Đường dẫn file | Số dòng thay đổi | Mô tả tóm tắt thay đổi |
| :--- | :--- | :--- |
| `pkgs/metrics/prometheus.go` | ~9 dòng | Định nghĩa Prometheus metric `cdc_transmute_rule_dropped_total`. |
| `internal/service/master/transmuter.go` | ~40 dòng | Import `pgconn`, warning log & increment metric khi drop rules, chuẩn hóa hàm `isRetryableDBError` bằng `errors.As`. |
| `internal/handler/shadow/batch_buffer.go` | ~45 dòng | Import `pgconn`, bổ sung hàm `isRetryableDBError`, cập nhật sequential fallback để return early khi gặp transient DB error. |
| `13_analysis_risks_phase_3.md` | Mới (180 dòng) | Nghiên cứu và phân tích thiết kế giải pháp cho OCC clock skew và FNV-1a collision. |
| `08_tasks_phase_3.md` | ~15 dòng | Đánh dấu hoàn thành các task của Phase 3. |
| `05_progress_phase_3.md` | ~2 dòng | Ghi nhận log tiến độ hoàn thành của tác nhân `Muscle`. |

---

## 2. Chi tiết các thay đổi chính

### 2.1. Cấu hình Metric Theo Dõi Rule Bị Drop (TX-C3)
- Đăng ký metric `cdc_transmute_rule_dropped_total` với các nhãn có độ phân giải thấp (low cardinality) nhằm tránh lock/saturation trên Prometheus TSDB: `master_table`, `source_field`, `target_column`, `reason`.
- Cập nhật `loadRules()` trong transmuter.go để in Warn log chi tiết và tăng counter tương ứng khi rule bị bỏ qua do:
  - Transform function không nằm trong whitelist.
  - Kiểu dữ liệu không hợp lệ.
  - Identifier của PostgreSQL không hợp lệ (chứa ký tự `$`).

### 2.2. Phân Tách Lỗi Transient và Permanent DB (SINK-H5)
- Tích hợp thư viện `github.com/jackc/pgx/v5/pgconn` để cast lỗi sang `*pgconn.PgError` bằng `errors.As` giúp phân loại chính xác mã lỗi SQLSTATE của PostgreSQL.
- **Lỗi Transient (Retryable):** connection exception (`08xxx`), serialization failure (`40001`), deadlock (`40P01`), admin shutdown (`57P01`/`57P02`/`57P03`), và object in use (`55000`).
- **Lỗi Permanent (Non-retryable):** unique constraint violation (`23505`) và foreign key violation (`23503`).
- **Xử lý tại Batch Buffer sequential fallback:**
  - Nếu gặp lỗi permanent, tiếp tục ghi DLQ và tiếp tục xử lý các bản ghi tiếp theo trong batch (bảo toàn luồng realtime).
  - Nếu gặp lỗi transient, lập tức abort vòng lặp, từ chối ghi DLQ cho bản ghi đó, dừng tiến trình và trả lỗi về hàm `Flush()`. Điều này khiến worker Kafka từ chối commit offset Kafka, kích hoạt cơ chế retry/restart pod ở mức hạ tầng (K8s/Supervisor) để tự khôi phục kết nối.

### 2.3. Báo Cáo Phân Tích Rủi Ro Cơ Sở Hạ Tầng (TX-H3 & TX-H6)
- Phân tích rủi ro Clock Skew khi sử dụng `_source_ts` để OCC cập nhật Master DB, đề xuất thiết kế Vector Clocks cho multi-source merge hoặc dùng Logical Clocks (MongoDB Optime / Postgres LSN).
- Phân tích rủi ro va chạm hash FNV-1a dùng cho đối soát, đề xuất chuyển sang XXHash64/XXH3 hoặc SHA-256 đối với các dữ liệu yêu cầu độ an toàn tuyệt đối.
