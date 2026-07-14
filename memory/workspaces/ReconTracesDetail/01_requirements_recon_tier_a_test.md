# Yêu cầu kiểm thử logic RunHashWindowCheck

Nhiệm vụ: Viết unit tests bổ sung để kiểm thử logic của `RunHashWindowCheck` với các kịch bản thực tế nhằm xác nhận giải pháp hoạt động hoàn hảo.

## 1. Các kịch bản cần kiểm thử
* **TestRunHashWindowCheck_GlobalMatch_NoDrift:**
  - Khoảng thời gian kiểm tra: trong vòng 7 ngày (ví dụ 3 ngày).
  - Kết quả: Source Agent và Dest Agent trả về khớp `Count` và `XorHash`.
  - Kỳ vọng: Hệ thống trả về kết quả OK ngay lập tức bằng Global Hash Check nhanh, không chạy qua bất kỳ window loop con nào.
* **TestRunHashWindowCheck_GlobalMismatch_FallbackToLoop:**
  - Khoảng thời gian kiểm tra: trong vòng 7 ngày (ví dụ 1 giờ, WindowSize = 15 phút).
  - Kết quả: Source Agent và Dest Agent trả về lệch `Count` hoặc `XorHash` (drift).
  - Kỳ vọng: Hệ thống tự động fallback về window loop 15-phút để định vị và drill down chữa lành (gọi các phương thức list IDs/TSs trong window bị lệch).
* **TestRunHashWindowCheck_BlockPartitioning:**
  - Khoảng thời gian kiểm tra: lớn hơn 7 ngày (ví dụ 10 ngày).
  - Kết quả: Chia làm 2 block (7 ngày + 3 ngày). Cả 2 block đều khớp Global Hash.
  - Kỳ vọng: Hệ thống phân chia thành các block để kiểm tra Global Hash trước khi quyết định fallback. Kết quả trả về OK mà không chạy fallback loop con.

## 2. Ràng buộc kỹ thuật
- Sử dụng `sqlmock` để mock tất cả các database (Postgres source, Postgres dest/shadow, và Core DB).
- Vị trí file test: `internal/service/recon/recon_tier_a_test.go`.
- Chạy thử bằng lệnh: `go test -v -run TestRunHashWindowCheck ./internal/service/recon/...`.
- Cập nhật dấu vết logs và kết quả vào `06_validation_recon_traces.md` và `05_progress_recon_traces.md`.
- TUYỆT ĐỐI không thay đổi code git ở repo. Chỉ tạo file test và chạy test.
