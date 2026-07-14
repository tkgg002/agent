# Kế hoạch Sửa đổi Logic Xác định Shadow Schema của SinkWorker (Sửa lỗi đứng lag)

## 1. Phân tích nguyên nhân gốc rễ
Hiện tại, trên môi trường DEV chung và local, luồng sink của `export_jobs` và `schedule_histories` bị báo đứng lag 5h vì:
1. SinkWorker tự derive tên shadow schema từ Kafka topic name (ví dụ topic `cdc.goopaylocal.centrallized-export-service.export-jobs` -> `"shadow_centrallized_export_service"`).
2. Tuy nhiên, trong database cấu hình (`cdc_system.shadow_binding`), schema shadow của bảng này lại được định nghĩa là `"shadow_testexp"`.
3. Do sự lệch nhau này, SinkWorker ghi dữ liệu mới vào schema `"shadow_centrallized_export_service"` (hoặc lỗi ghi do lệch schema), trong khi cdc-worker đối soát lại đọc từ `"shadow_testexp"`. Do `"shadow_testexp"` không có dữ liệu mới, lag đối soát bị đứng im không cập nhật.

---

## 2. Giải pháp kỹ thuật
Cập nhật hàm xác định shadow target trong SinkWorker để **ưu tiên tra cứu từ database** trước khi fallback về tự derive:
1. Trong `internal/sinkworker/worker.go` hoặc `utils.go`:
   Tra cứu `shadow_schema` và `shadow_table` từ `shadow_binding` JOIN `source_object_registry` bằng `source_database` và `source_object_name` trích xuất từ topic.
2. Thêm hàm `resolveShadowTarget(db *gorm.DB, topic string) (string, string)`:
   - Tách topic name ra `sourceDB` và `sourceTable`.
   - Query Postgres `cdc_system.shadow_binding` để lấy đúng schema và table.
   - Nếu không tìm thấy, fallback về hàm tự derive cũ `extractShadowTarget(topic)`.

---

## 3. Các file cần sửa đổi
- `internal/sinkworker/worker.go`:
  - Gọi `resolveShadowTarget` với connection DB và topic thay vì gọi trực tiếp `extractShadowTarget`.
- `internal/sinkworker/utils.go`:
  - Viết hàm `resolveShadowTarget(db *gorm.DB, topic string) (string, string)`.

## 4. Kế hoạch Thực thi của Muscle (Chi tiết)
1. **Chỉnh sửa file `internal/sinkworker/worker.go`**:
   - Thêm các packages import cần thiết.
   - Bổ sung field `activity` kiểu `*governance.ActivityLogger` vào struct `SinkWorker`.
   - Wire `activity` trong hàm `New` nếu `cfg.DB != nil`.
   - Triển khai method `resolveShadowTarget(ctx context.Context, topic string) (string, string, error)` thực hiện query trực tiếp Postgres để lấy `shadow_schema` và `shadow_table` (báo lỗi thẳng, không fallback).
   - Cập nhật hàm `HandleMessage` để gọi `resolveShadowTarget`, quản lý activity logging với `defer` để log `Complete`/`Fail`.
2. **Sửa đổi file test `test/internal/sinkworker/sinkworker_test.go`**:
   - Khắc phục các test cases bị fail sẵn bằng cách:
     - Thêm dummy key vào `source` map để tránh bị `unwrapAvroUnion` unwrap sai cách.
     - Cập nhật assert SQL strings cho khớp với logic thực tế của SQL builder (`EXCLUDED._source_ts >=` và lược bỏ unique index target `WHERE NOT _deleted` trên clause `ON CONFLICT`).
3. **Verify compile & test**:
   - Chạy lệnh `go build ./cmd/...`
   - Chạy lệnh `go test -v ./internal/sinkworker/... ./test/internal/sinkworker/...`
4. **Cập nhật log tiến độ và task list**:
   - Append log tiến độ vào `05_progress_recon_traces.md`.
   - Đánh dấu hoàn thành task tương ứng trong `08_tasks_recon_traces.md`.
5. **Chạy Linter Quy trình**:
   - Thực thi `python3 agent/tooling/verify_governance.py` để audit cuối phiên.

