# Danh sách Task chi tiết - Khắc phục logic tự sửa đổi index trong Transmuter & Bổ sung đề xuất trên UI

- [x] Task 1: Nghiên cứu cấu trúc code và thiết kế giải pháp
  - [x] Phân tích cấu trúc `transmuter.go` và cách `ensureShadowSourceIDIndex` hoạt động.
  - [x] Thiết kế mô hình đề xuất index (Index Recommendations) trong `index_manager.go`.
- [x] Task 2: Viết kế hoạch triển khai chi tiết (`implementation_plan.md`) và gửi đề xuất cho User
- [x] Task 3: Ủy quyền Muscle thực thi thay đổi code
  - [x] Sửa đổi `transmuter.go`: Loại bỏ DDL `CREATE/DROP INDEX` trong `ensureShadowSourceIDIndex`, chuyển sang log cảnh báo `Warn`.
  - [x] Sửa đổi `index_manager.go`: Bổ sung hàm đề xuất index và trả về kèm theo kết quả introspect.
  - [x] Sửa đổi `index_handler.go`: Truyền recommendations về NATS payload của `introspect-indexes`.
  - [x] Chạy `go test` biên dịch thành công.
- [x] Task 4: Chạy linter quy trình và hoàn thành DoD
  - [x] Báo cáo thay đổi trong `11_report_shadow_source_id_index.md`
  - [x] Tạo walkthrough báo cáo kết quả `14_walkthrough_shadow_source_id_index.md`
  - [x] Chạy `python3 tooling/verify_governance.py`
