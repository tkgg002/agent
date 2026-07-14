# Checklist thực hiện sửa lỗi recon_tier_b.go

- [x] Phân tích các lỗi biên dịch trong `recon_tier_b.go`.
- [x] Tìm hiểu định nghĩa của `MasterBindingRef` để xem trường nào thay thế cho `SourceDB`.
- [x] Tìm hiểu định nghĩa của `ReconciliationReport` để xem trường nào thay thế cho `TargetSchema`.
- [x] Tìm hiểu định nghĩa của `ReconCore` để xem phương thức nào thay thế cho `RunSegmentB` hoặc `stampB`.
- [x] Lập kế hoạch chi tiết trong `12_implementation_plan_tier_b_fix.md`.
- [x] Thực hiện sửa đổi các lỗi compile thông qua Muscle/Sub-agent.
- [x] Chạy `go build` để verify file biên dịch thành công.
- [x] Chạy linter quy trình `verify_governance.py`.
