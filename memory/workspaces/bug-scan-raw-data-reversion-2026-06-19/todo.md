# Todo List: Sửa lỗi Scan Raw Data và Audit Logic Handlers

- [x] Phase 1: Research & Audit
  - [x] So sánh `scan_handler.go` hiện tại và `c439b9c:internal/handler/command_handler.go` để lập danh sách sai lệch logic.
  - [x] Rà soát logic `HandlePeriodicScan` cũ vs mới.
  - [x] Rà soát logic các function đối soát và heal trong `recon_handler.go`, `recon_heal_v4.go`.
- [x] Phase 2: Implementation (Sửa lỗi Scan)
  - [x] Khôi phục logic `HandleScanRawData` lưu pending rules vào DB.
  - [x] Khôi phục logic `HandlePeriodicScan` tự động quét và lưu pending rules vào DB.
  - [x] Đảm bảo code compile thành công.
- [x] Phase 3: Verification & Security Gate
  - [x] Chạy local tests và build để xác nhận tính đúng đắn.
  - [x] Rà soát an ninh mã nguồn và hoàn tất báo cáo audit.

