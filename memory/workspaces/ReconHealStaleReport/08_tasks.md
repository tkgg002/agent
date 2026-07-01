# Tasks: Sửa lỗi healSegmentA/healSegmentB lặp lại do lấy stale report

## Danh sách công việc thực thi:
- [x] Tạo workspace `ReconHealStaleReport` và tài liệu context, plan.
- [x] Khai báo hằng số `healReportMaxAge = 5 * time.Minute` trong `recon_heal_v4.go`.
- [x] Triển khai kiểm tra `isStale` cho `healSegmentB` để bypass stale report.
- [x] Triển khai kiểm tra `isStale` cho `healSegmentA` để bypass stale report.
- [x] Tạo file unit test `recon_heal_v4_test.go` với các mock SQL và NATS phù hợp.
- [x] Bổ sung test case `TestHealSegmentA_StaleReportFallback` nhằm kiểm chứng stale bypass.
- [x] Chạy unit tests để verify logic hoạt động chính xác.
- [x] Đưa toàn bộ cấu trúc thư mục workspace tuân thủ 100% quy trình Governance V3.
