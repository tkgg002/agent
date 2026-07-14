# Báo cáo Thay đổi (Report) - Audit Reconciliation & Test Clean up

Tài liệu này ghi nhận chi tiết danh sách tệp tin, số lượng dòng code thay đổi (additions/deletions) và lý do thay đổi trong quá trình thực hiện sửa test compile và decommission `recon_bk`.

## Tóm tắt Dòng thay đổi (Line Stats Summary)

- **Số lượng file thay đổi:** 11 file kiểm thử và mã nguồn (không bao gồm tài liệu workspace).
- **Thư mục bị loại bỏ:** `internal/handler/recon_bk/` (15 files, ~3000 dòng code legacy đã decommission).
- **Thư mục lưu trữ tài liệu đối chiếu cũ:** `docs/recon_legacy/` (13 files `.go.bak` được phục hồi từ lịch sử Git).
- **Tổng số dòng thêm mới (Additions):** 467 dòng.
- **Tổng số dòng xóa bỏ (Deletions):** 456 dòng (chỉ tính các file đang track).

---

## Chi tiết các tệp tin thay đổi (File-by-file Diff Detail)

### 1. Phục hồi và Cập nhật Test Suite (`handler/recon`)
- **[MODIFY] [recon_heal_v4_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4_test.go)**
  - **Thay đổi:** +36 additions, -0 deletions.
  - **Lý do:** Thêm kiểu tương thích ngược `ReconHandler` cục bộ cho test, constructor compatibility, và helper mock `WithBackfill` để chạy thành công 6 tests cũ bằng `HealHandler` mới.

### 2. Di chuyển các tệp tin kiểm thử quét (`scan`)
- **[NEW] [scan_array_path_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/scan/scan_array_path_test.go)**
  - **Thay đổi:** +143 additions, -0 deletions.
  - **Lý do:** Di chuyển từ `internal/handler/recon/` sang `internal/handler/scan/` để kiểm thử đúng gói `scan`.
- **[NEW] [scan_handler_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/scan/scan_handler_test.go)**
  - **Thay đổi:** +238 additions, -0 deletions.
  - **Lý do:** Di chuyển về package `scan`.
- **[DELETE] [scan_array_path_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_array_path_test.go)**
  - **Thay đổi:** +0 additions, -142 deletions.
- **[DELETE] [scan_handler_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/scan_handler_test.go)**
  - **Thay đổi:** +0 additions, -237 deletions.

### 3. Cập nhật Gói Nghiệp vụ (`service/recon`)
- **[MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)**
  - **Thay đổi:** +46 additions, -37 deletions.
  - **Lý do:** Refactor hàm `pickScanRangeWithLag` để hỗ trợ phân tách `srcTS` và `dstTS`.
- **[MODIFY] [recon_engine_segment_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_segment_b.go)**
  - **Thay đổi:** +4 additions, -0 deletions.
  - **Lý do:** Thêm wrapper method `StampA` phục vụ ghi report cho Segment A.
- **[MODIFY] [recon_fallback_test.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_fallback_test.go)**
  - **Thay đổi:** +3 additions, -2 deletions.
  - **Lý do:** Cập nhật lệnh gọi test của hàm `resolveSourceAndDestTSFields`.
- **[MODIFY] [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go)**
  - **Thay đổi:** +1 addition, -1 deletion.
  - **Lý do:** Sửa signature gọi `pickScanRangeWithLag`.

### 4. Lưu trữ mã nguồn Legacy phục vụ Audit
- **[NEW] [docs/recon_legacy/](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/docs/recon_legacy)**
  - **Thay đổi:** Khôi phục 13 file từ lịch sử Git của `internal/handler/recon/` trước thời điểm refactor, đổi phần mở rộng thành `.go.bak`.
  - **Lý do:** Lưu trữ mã nguồn cũ trực tiếp trong workspace để tiện đối chiếu (diff/audit) thủ công bằng IDE của lập trình viên, đồng thời hậu tố `.go.bak` tránh việc trình biên dịch Go quét qua gây lỗi biên dịch trùng lặp symbol.
