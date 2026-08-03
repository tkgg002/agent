# 08 — Checklist Tasks: Khắc phục Timezone Drift

- [ ] `[ ]` Phân tích kỹ thuật & Thiết kế chi tiết logic (`13_analysis_timezone_drift.md`)
- [ ] `[ ]` Tạo Kế hoạch Triển khai (`12_implementation_plan_timezone_drift.md`)
- [ ] `[ ]` Thêm map cache `colTypes` và mutex `mu` vào `ReconDestAgent` (`recon_dest_agent.go`)
- [ ] `[ ]` Implement `IsColTimestamptz` trong `recon_dest_query.go` để query và cache kiểu dữ liệu cột
- [ ] `[ ]` Thêm hàm `parsePostgresTimestampWithLocationAndType` vào `recon_query.go` hỗ trợ parse động theo kiểu cột
- [ ] `[ ]` Cập nhật `HashWindow` trong `recon_dest_hash.go` để sử dụng logic parse động
- [ ] `[ ]` Cập nhật `ListIDTsInWindow` trong `recon_dest_query.go` để sử dụng logic parse động
- [ ] `[ ]` Đồng bộ hóa mock và logic test trong `recon_dest_agent_test.go`
- [ ] `[ ]` Biên dịch và chạy Unit Tests local
- [ ] `[ ]` Viết báo cáo hoàn thành (`11_report_timezone_drift.md`)
