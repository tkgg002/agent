# Plan: Điều tra lỗi Reconciliation Heal noop và khắc phục triệt để

## 1. Kiểm tra Trạng thái Hiện tại (Current State Check)
- **Hành động 1.1**: Chạy `go run scratch_find_missing.go` để xác định danh sách record lệch thực tế.
- **Hành động 1.2**: Phân tích log và cấu hình `timestamp_field` của bảng `payment_bills` và `export_jobs`.

## 2. Nghiên cứu Nguyên nhân Gốc rễ (Root Cause Investigation)
- **Hành động 2.1**: Xác định các vị trí trong source code của `centralized-data-service` đang hardcode trường `"updated_at"` khi thao tác với MongoDB documents.
- **Hành động 2.2**: Lên phương án trích xuất động `timestamp_field` từ TableRegistry/Shadow Object cấu hình thay vì hardcode.

## 3. Khắc phục & Sửa lỗi (Fixing)
- **Hành động 3.1**: Sửa `extractSourceTsFromDoc` ở `helpers.go` và `recon_heal_utils.go` để nhận và sử dụng `TimestampField`.
- **Hành động 3.2**: Sửa `fetchSourceTsMap` ở `backfill_source_ts.go` để truyền `TimestampField` và thêm nó vào MongoDB projection.
- **Hành động 3.3**: Sửa `recon_heal.go` và `dlq_worker.go` để truyền và sử dụng `TimestampField` động.
- **Hành động 3.4**: Sửa `recon_engine_segment_b.go`, `recon_smoke.go`, `recon_dest_query.go`, `recon_tier_a.go`, `recon_heal_v4.go` và `recon_handler_run.go` theo kế hoạch Segment B và đồng bộ Window.

## 4. Xác minh Kết quả (Verification)
- **Hành động 4.1**: Chạy unit tests:
  ```bash
  go test -v ./internal/service/recon/...
  go test -v ./internal/service/governance/...
  ```
- **Hành động 4.2**: Trigger manual heal qua API để đảm bảo sửa lỗi thành công và đồng bộ dữ liệu chính xác.
- **Hành động 4.3**: Cập nhật `05_progress.md` và `lessons.md`.
