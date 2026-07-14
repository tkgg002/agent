# Kế hoạch Triển khai Chi tiết của AI - Phân tích các loại Recon

## 1. Nghiên cứu tài liệu & Source Code
- Đọc inventory `report_recon_actions_inventory_2026-06-12.md` để lấy danh sách 10 loại recon hiện có.
- Xem code `recon_tier_a.go` để phân tích sâu hơn về logic và luồng gọi của Tier 1 (chuyển đổi từ window-count sang bucket-aggregate), Tier 2 (xor hash + list IDTs), Tier 3 (256-bucket hash).
- Xem `recon_handler_run.go` để hiểu cách trigger các tier này thông qua message queue (NATS).

## 2. Viết tài liệu phân tích (`13_analysis_recon_analysis.md`)
- Cấu trúc tài liệu rõ ràng, chuyên nghiệp.
- Mô tả chi tiết 10 loại recon: Mục tiêu, Trigger, Luồng hoạt động, Chi phí và Trạng thái hiện tại.
- Phân tích sự chuyển dịch kiến trúc đối soát từ V4 (quá tải) sang V5 (tối ưu hóa phân tầng với Fast Path Tier 0).

## 3. Trả lời người dùng
- Xác nhận đã đọc `GEMINI.md` và `lessons.md`.
- Trình bày thông tin phân tích súc tích, mạch lạc.
- Liệt kê các skill đã sử dụng.
