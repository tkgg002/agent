# Plan: Investigate Master Sync New Field

## Mục tiêu
Kiểm tra xem codebase đã có chức năng đồng bộ dữ liệu cho các trường mới được thêm trên master từ shadow/raw_data chưa.

## Các bước thực hiện

### Phase 1: Research (Nghiên cứu Codebase)
* [ ] Bước 1.1: Quét codebase của `data-hub` (chủ yếu là `cdc-cms-service` hoặc các handler liên quan đến master/shadow) để tìm kiếm các API endpoint liên quan đến "sync", "transform", "reconcile", "mapping rule" của master.
* [ ] Bước 1.2: Phân tích cơ chế cập nhật từ shadow sang master. Khi chạy transform shadow, dữ liệu có tự động trigger đồng bộ sang master không?
* [ ] Bước 1.3: Tìm hiểu xem UI hoặc API có chức năng click "Đồng bộ" (Sync/Backfill/Re-sync) cho master mapping rules hoặc column registry hay không.

### Phase 2: Phân tích & Trả lời
* [ ] Bước 2.1: Tổng hợp kết quả tìm kiếm (các file code cụ thể, hành vi logic hiện tại).
* [ ] Bước 2.2: Soạn thảo câu trả lời chi tiết gửi đến người dùng.
