# Tasks - Khôi phục Logic Scan Raw Data & Periodic Scan

## Danh sách Task chi tiết

- [x] Lập và duyệt Implementation Plan cho lỗi Scan Raw Data và Audit Handlers
- [x] Khôi phục logic `HandleScanRawData` trong `scan_handler.go`
  - [x] Quét fields/types bằng SQL query PostgreSQL (jsonb_object_keys)
  - [x] Query shadow binding
  - [x] Query mapping rules v2 hiện tại
  - [x] So sánh để tìm các field chưa mapped
  - [x] Insert pending mapping rules mới vào DB
  - [x] Publish result qua NATS với các info đúng dạng
- [x] Khôi phục logic `HandlePeriodicScan` trong `scan_handler.go`
  - [x] Lấy danh sách table configs từ metadata registry
  - [x] Quét và tự động tạo pending rules cho các tables đó
- [x] Build & Verify compile
- [x] Rà soát, audit logic của các handlers khác trong module `recon`
