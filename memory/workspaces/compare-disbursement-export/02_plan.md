# Plan: So sánh logic DisbursementTicketExport

## 1. Research & Discovery
- [/] Tìm kiếm vị trí các file liên quan đến DisbursementTicket trong `centralized-export-service`.
- [/] Tìm kiếm vị trí logic export gốc trong `work-feat/export/stepplaning/ExportDisbursementTicket`.
- [/] Xác định các file logic chính: Handler, Query, Model và DTO.

## 2. Analysis
- [/] Phân tích `GetAllDisbursementTicketExportHandler.ts` để hiểu logic fetch dữ liệu hiện tại.
- [/] Phân tích `module_ticket.go`, `entity_ticket.go` và `info.md` (NodeJS mockup) để hiểu logic gốc.
- [/] So sánh mapping giữa các trường database (MongoDB) và các cột trong Excel report.

## 3. Comparison & Evaluation
- [/] Đối soát danh sách các filter được hỗ trợ.
- [/] Kiểm tra tính tương đồng của field mapping.
- [/] Đánh giá các cải tiến (như cursor-based pagination).

## 4. Documentation
- [/] Tạo báo cáo so sánh chi tiết (`walkthrough.md`).
- [/] Cập nhật tiến độ và bài học kinh nghiệm (`05_progress.md`, `lessons.md`).
