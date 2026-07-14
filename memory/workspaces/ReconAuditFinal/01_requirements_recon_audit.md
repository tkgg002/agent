# Yêu cầu Audit và Đánh giá Parity Hệ thống Reconciliation mới

## 1. Yêu cầu chi tiết
- Phân tích tính đúng đắn của logic đối soát (Reconciliation) mới nằm trong `internal/handler/recon` so với logic cũ `internal/handler/recon_bk`.
- Xác định sự tương đương của 2 luồng chính:
  1. Luồng Smoke Test (`recon_smoke.go`).
  2. Luồng Reconciliation Check (`recon check`) với các trường hợp: Lookback (hot/cold), Full Search (full_diff) và Deep Check.
- Xác nhận các điều kiện ràng buộc giữa 3 tuỳ chọn trên UI (không chọn cùng nhau) và tính hợp lệ của validator max 30 ngày.
- Mô tả chi tiết flow đi qua các hàm (từ `cms` -> `api` -> `cdc`) cho từng trường hợp.

## 2. Tiêu chuẩn Hoàn thành (DoD)
- Báo cáo phân tích chi tiết được lưu trữ tại `13_analysis_recon_audit.md`.
- File check-list tiến độ `05_progress_recon_audit.md` được ghi nhận.
- Trình bày câu trả lời rõ ràng, súc tích bằng tiếng Việt cho User.
