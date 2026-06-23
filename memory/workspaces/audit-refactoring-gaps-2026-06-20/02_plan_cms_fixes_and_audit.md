# Kế hoạch Thực thi: Phase cms_fixes_and_audit

Phase này được chia thành các bước thực thi tuần tự để đảm bảo tính an toàn và chất lượng đầu ra:

## 1. Thiết kế và Lập giải pháp kỹ thuật (Brain)
- Phân tích chi tiết code, chuẩn bị diff sửa đổi chính xác.
- Tạo tài liệu giải pháp kỹ thuật `09_tasks_solution_cms_fixes_and_audit.md`.
- Trình User phê duyệt giải pháp trước khi thực hiện code changes.

## 2. Thực thi sửa lỗi cdc-cms-service (Muscle)
- Thực hiện áp dụng thay đổi code cho `master_mapping_rule_repo_gorm.go`.
- Thực hiện áp dụng thay đổi code cho `drop_column.go`.
- Tiến hành chạy biên dịch thử: `go build ./...` trong `cdc-cms-service`.

## 3. So khớp chi tiết Reconciliation Engine (Brain & Muscle)
- Tiến hành diff so sánh `recon_core.go` gốc (từ backup `data-hub-bf`) với bộ ba file của `centralized-data-service` mới (`recon_engine.go`, `recon_tier_a.go`, `recon_tier_b.go`).
- Ghi nhận báo cáo audit chi tiết. Nếu phát hiện gap logic, bổ sung yêu cầu sửa đổi.

## 4. Kiểm thử và Xác nhận (Quality Gate)
- Chạy toàn bộ Unit Tests của `cdc-cms-service` và `centralized-data-service` để đảm bảo không có lỗi regression.
- Ghi nhận kết quả chạy test và chứng minh chất lượng vào `06_test_cases.md` hoặc `06_validation.md`.
