# 08 — Danh sách Tasks: Audit Recon payment_bills

> Gắn với: 01_requirements_audit.md | Trạng thái: DONE (phân tích)

## Tasks Phân tích

- [x] **T1** Đọc và map trace log sang code flow
- [x] **T2** Đọc `recon_tier_a.go` — hàm `RunHashWindowCheck`, `pickScanRangeWithLag`, `buildWindows`
- [x] **T3** Đọc `recon_dest_hash.go` — hàm `HashWindow` (nhánh TIMESTAMP vs _source_ts)
- [x] **T4** Đọc `recon_dest_query.go` — hàm `ListIDTsInWindow`, `MaxWindowTs`
- [x] **T5** Đọc `recon_engine.go` — config defaults (WindowSize=15min, HotWindowLookback=2h)
- [x] **T6** Phân tích root cause timezone drift trên TIMESTAMP column
- [x] **T7** Phân tích MongoDB index missing (MaxWindowTs 2.46s, ListIDTsInWindow 5.3s)
- [x] **T8** Phân tích mismatch granularity hash(ms) ↔ diff(giây)
- [x] **T9** Lập bảng tổng hợp P1/P2/P3/P4 issues
- [x] **T10** Đề xuất fix cụ thể + ước tính hiệu năng sau fix

## Tasks Action (Đã thực hiện)

- [x] **A1** [P1-CRITICAL] Sửa lỗi timezone drift trên Shadow DB:
  - [x] Chuyển đổi tLo/tHi sang DB local timezone trong `HashWindow` (`recon_dest_hash.go`).
  - [x] Chuyển đổi tLo/tHi tương tự trong các hàm query `CountInWindow`, `CountRecentDeletedRows`, `BucketCounts`, `ListIDTsInWindow` (`recon_dest_query.go`).
  - [x] Cập nhật test cases sang explicit UTC và mock `SHOW TIMEZONE` (`recon_dest_agent_test.go`).
- [ ] **A2** [P2-HIGH] Tạo MongoDB index `{ lastUpdatedAt: 1 }` trên collection `payment_bills` (được ghi nhận trong analysis, không trực tiếp thay đổi code vì source DB readonly).
- [ ] **A3** [P3-HIGH] Tạo compound index `{ lastUpdatedAt: 1, _id: 1 }` trên MongoDB (được ghi nhận trong analysis).
- [ ] **A4** [P4-MEDIUM] Review `diffIDTsSegmentA` — align granularity hash ↔ diff.
- [x] **A5** Chạy `verify_governance.py` cuối phiên.

