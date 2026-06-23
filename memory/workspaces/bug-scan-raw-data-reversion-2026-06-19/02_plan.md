# Plan: Sửa lỗi Scan Raw Data và Audit Logic Handlers

## Phase 1: Research & Audit
1. So sánh chi tiết logic của `HandleScanRawData` và `HandlePeriodicScan` giữa `scan_handler.go` hiện tại và `c439b9c:internal/handler/command_handler.go`.
2. Phân tích các thay đổi trong cách import, sử dụng repository mới thay vì gọi DB trực tiếp của gorm (hoặc giữ nguyên `h.DB` nếu repository chưa hỗ trợ).
3. Rà soát từng dòng logic của các function khác trong `recon_handler.go`, `recon_heal_v4.go`, `dlq_handler.go`, và `discover_handler.go`.

## Phase 2: Implementation (Sửa lỗi Scan)
1. Cập nhật `HandleScanRawData` trong `scan_handler.go` để:
   - Truy vấn shadow binding tương ứng của target table.
   - Truy vấn tất cả mapping rules V2 đã tồn tại của source object đó.
   - Phân tích keys trong `_raw_data` (lấy mẫu `Limit` bản ghi).
   - Xác định các key chưa được map (không tồn tại trong rules V2 hiện có) và loại trừ các system keys.
   - Thêm các record `MappingRuleV2` với `Status: "pending"`, `IsActive: false` vào DB (bảng `cdc_system.mapping_rule_v2`).
   - Trả về kết quả NATS theo format cũ.
2. Cập nhật `HandlePeriodicScan` để thực hiện quét tất cả active table configs (hoặc danh sách tables được config), tự động insert pending rules tương tự.

## Phase 3: Verification
1. Viết unit test hoặc chạy smoke test để kiểm tra.
2. Kiểm tra log của scheduler và handler khi chạy scan.
3. Chạy kiểm tra bảo mật `/security-agent`.
