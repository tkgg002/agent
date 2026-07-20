# 11_report_fix_audit_issues.md - Báo cáo thay đổi code

Dưới đây là thống kê chi tiết các tệp tin đã thay đổi trong đợt sửa lỗi Phase 0 & Phase 1:

## 1. Thống kê số lượng dòng code thay đổi (LOC)

| File | Trạng thái | Số dòng thêm | Số dòng xóa | Mô tả thay đổi |
|---|---|---|---|---|
| [cdc_event.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/model/shadow/cdc_event.go) | MODIFIED | +8 | -1 | Thêm các trường Kafka metadata vào CDCEvent và UpsertRecord |
| [event_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/event_handler.go) | MODIFIED | +11 | -0 | Map Kafka metadata sang UpsertRecord; thêm proxy SetOnCommitOffsets; thêm metrics cho Drop 3 & 4 |
| [batch_buffer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/batch_buffer.go) | MODIFIED | +44 | -13 | Thêm TopicPartition struct; onCommitOffsets callback; gọi callback sau flush; thêm writeFailedSyncLog helper check lỗi DLQ |
| [prometheus.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/metrics/prometheus.go) | MODIFIED | +8 | -0 | Định nghĩa Prometheus metric cdc_sink_events_dropped_total |
| [kafka_consumer.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/shadow/kafka_consumer.go) | MODIFIED | +74 | -37 | Đặt CommitInterval: 0; gán callback trong NewKafkaConsumer; thêm helper commitOffsets; DLQ-only commit; log/metrics Drop 1 & 2 |
| [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go) | MODIFIED | +16 | -4 | Bổ sung recover block và tự động cancel context khi panic trong background goroutine |
| [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go) | MODIFIED | +93 | -21 | Switch type cover int64/float64 trong dedup; bulkUpsertMaster retry backoff logic; isRetryableDBError helper; log rule skip; default value fallback |
| [server_setup.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/server/server_setup.go) | MODIFIED | +2 | -2 | Chuyển NATS Subscribe của transmute và transmute-shadow sang QueueSubscribe |

**Tổng cộng:** ~256 LOC thêm mới, ~78 LOC xóa/thay thế.

---

## 2. Đánh giá tính an toàn và Regression
- **Kafka Offset Commit:** Cơ chế CommitInterval: 0 kết hợp trì hoãn commit sau khi ghi DB thành công loại bỏ Timing Gap nguy cơ mất dữ liệu khi crash (đảm bảo At-Least-Once).
- **Type Assertions:** Switch type cover float64 loại bỏ hoàn toàn nguy cơ panic do cơ chế gjson/json unmarshal mặc định parse số sang float64.
- **DB Connection Resources:** recover block kết hợp cancel() context đảm bảo giải phóng mọi DB transactions/connections đang treo khi xảy ra panic, chặn rò rỉ connection pool.
- **NATS Load Balancing:** Chuyển sang QueueSubscribe giúp phân tải đều cho các transmute workers, tránh overload một worker duy nhất khi có lượng lớn trigger transmute.
- **DLQ Integrity:** DLQ write errors không còn bị swallow nữa, giúp SRE phát hiện sớm các vấn đề kết nối DB DLQ qua metric cdc_dlq_write_failures_total.
