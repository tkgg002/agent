# Requirements - Khôi phục Logic Scan Raw Data & Periodic Scan

## 1. Yêu cầu nghiệp vụ
Khôi phục hành vi gốc của hệ thống quét schema tự động từ commit `c439b9c` để giải quyết các vấn đề sau:
1. Khi scan `_raw_data` phát hiện ra trường JSONB mới chưa được định nghĩa mapping rule, hệ thống phải tự động tạo mapping rule ở trạng thái `pending` trong database (`cdc_system.mapping_rule_v2`).
2. Scheduler quét định kỳ (`HandlePeriodicScan`) phải quét đúng danh sách bảng CDC đang hoạt động cấu hình trong metadata registry, thay vì quét toàn bộ các bảng trong schema public của database.

## 2. Tiêu chí hoàn thành (Definition of Done)
- [x] Hàm `HandleScanRawData` so sánh các trường quét được với mapping rules hiện có trong `mapping_rule_v2` thay vì so với các cột vật lý trong database.
- [x] Tự động tạo rule mới với `Status: "pending"`, `IsActive: false`, `SourceFormat: "raw"`, `CreatedBy: "system-scan"` trong database.
- [x] Hàm `HandlePeriodicScan` lấy danh sách active table configs từ `h.metadataRegistry.ListTableConfigs()`, kiểm tra sự tồn tại của cột `_raw_data` và gọi `HandleScanRawData`.
- [x] Không có lỗi biên dịch (compile error).
- [x] Tất cả các test cases liên quan trong `go test ./internal/handler/recon/...` và `go test ./test/internal/handler/...` đều pass 100%.
- [x] Báo cáo đầy đủ danh sách thay đổi và số dòng code thay đổi trong workspace.
