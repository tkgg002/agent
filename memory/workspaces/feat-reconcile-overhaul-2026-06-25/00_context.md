# Context: Reconcile Component Overhaul (2026-06-25)

## Bối cảnh & Vấn đề hiện tại
- **Mô tả**: Thành phần Reconciliation (Đối soát) hiện tại đang ghi nhận lịch sử đối soát thông qua bảng `cdc_system.cdc_reconciliation_report`.
- **Vấn đề cấu trúc table**:
  - Bảng `cdc_reconciliation_report` đã trải qua nhiều đợt tiến hóa schema (từ migration 008 ban đầu, rồi bổ sung thêm các cột qua các đợt 081, 082, 083, 084, 085).
  - Tên các cột và cấu trúc bị chắp vá (ví dụ: `target_table` ở dạng bare-name dẫn đến ambiguous, sau đó phải chắp vá thêm `shadow_schema`, `shadow_table` và `run_id` ở migration 085).
  - Bản ghi đối soát phình to nhanh chóng do ghi đè/chèn liên tiếp hàng ngàn bản ghi `ok` lặp đi lặp lại sau mỗi chu kỳ quét tự động (cron/poller), gây rác cơ sở dữ liệu ("không khác gì đống rác").
- **Vấn đề Logic**:
  - Sự tách biệt giữa Segment A (Source ↔ Shadow) và Segment B (Shadow ↔ Master) chưa được chuẩn hóa rõ ràng ở mức cấu trúc lưu trữ và cách biểu diễn pipeline.
  - Logic ghi log đối soát còn nặng nề, thiếu cơ chế nén hoặc dọn dẹp các bản ghi thành công trùng lặp (success log retention/pruning).

## Mục tiêu Overhaul
1. **Review toàn diện** cấu trúc bảng đối soát và logic xử lý của cụm reconcile.
2. **Thiết kế lại schema** sạch sẽ, nhất quán, tối ưu hóa kích thước lưu trữ và tốc độ truy vấn cho dashboard.
3. **Chuẩn hóa logic ghi report**: Chỉ lưu vết chi tiết khi có biến động/drift hoặc lỗi, thu gọn các bản ghi `ok` thành các phiên chạy (runs) tổng quát, hoặc thiết kế cơ chế lưu trữ watermark/checkpoint thông minh thay vì chèn mới vô tội vạ.
4. **Bàn giao bộ tài liệu thiết kế chi tiết (DESIGN)** cho User duyệt trước khi thực thi bất kỳ thay đổi nào lên mã nguồn (Source Code).
