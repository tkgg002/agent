# 01_requirements.md — Yêu cầu Chuẩn hóa Phân tách Schema (Schema Isolation)

## 1. Yêu cầu Chức năng (Functional Specs)
- **R1 (Explicit Identification):** Mọi NATS Event, REST Payload và hàm xử lý Recon/SysOps phải truyền và sử dụng `shadow_schema` kết hợp `target_table` (hoặc `qualified key: shadow_schema.target_table`).
- **R2 (No Fallback Guessing):** Loại bỏ toàn bộ logic fallback tự đoán sang `pureTable` khi lookup thất bại. Nếu không tìm thấy cấu hình với `shadow_schema.target_table`, trả về lỗi tường minh ngay lập tức.
- **R3 (DB Query Isolation):** Mọi câu truy vấn Database (`Where("target_table = ?")`) trong repo/service phải có thêm điều kiện `Where("shadow_schema = ?")` khi có schema.
- **R4 (Backward Compatibility):** Nếu một bảng thuộc schema đơn lẻ (không trùng lặp), lookup `target_table` vẫn hoạt động khi không có tiền tố schema, nhưng cảnh báo log.

## 2. Tiêu chuẩn Kiểm định (DoD)
- Không có bất kỳ hiện tượng chạy chéo giữa `testbidv` và `testbvb` trên bảng `bank_requests`.
- Full test suite `go test ./internal/...` PASS 100%.
- Binary `cmd/worker` build pass (Exit code 0).
