# Checklist thực hiện tối ưu hóa chỉ số đối soát Segment B

- [x] Chuẩn bị kế hoạch thực hiện và ghi nhận requirements.
- [x] Cập nhật file thiết kế chi tiết `09_tasks_solution_range_counts.md`.
- [x] Thực hiện sửa đổi trong `recon_tier_b.go`:
  - [x] Loại bỏ hàm `runCountCheckB` và các lời gọi tới nó.
  - [x] Loại bỏ tối ưu hóa count check trước khi hash.
  - [x] Cập nhật `RunHashWindowCheckB` để trả về `SourceCount` và `DestCount` dựa trên `totalShadow` và `totalMaster` trong dải thời gian quét.
  - [x] Cập nhật `RunDeepCheckB` tương tự để trả về số lượng quét được của các bucket thay vì của toàn bảng.
  - [x] Bỏ việc gán `TotalSourceCount` và `TotalDestCount`.
- [x] Biên dịch kiểm tra bằng `go build`.
- [ ] Chạy linter quy trình `verify_governance.py`.
