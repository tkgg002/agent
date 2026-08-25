# 01_requirements.md - Requirements & Mục tiêu Kiểm toán

## 1. Yêu cầu nghiệp vụ & Kỹ thuật
- **R1 - Giải trình bản chất kỹ thuật**: Phân tích rõ ràng, không suy diễn, bám sát từng dòng mã nguồn để trả lời câu hỏi của người dùng:
  1. Tại sao bảng Scan Fields lại xuất hiện các trường `requestData`, `responseData` mà không có `bankTransactionId`, `logs`?
  2. Tại sao các trường `createdAt`, `updatedAt`, `_id` lại bị gán kiểu `JSONB` thay vì `TIMESTAMPTZ` / `TEXT`?
- **R2 - Audit & Phản tỉnh toàn trình (Self-Improvement Loop)**:
  - Sử dụng tư duy phản biện để rà soát toàn bộ quy trình scan, phát hiện các điểm nghẽn kiến trúc (architectural gaps), giả định sai lầm (false assumptions) trong mã nguồn hiện tại.
  - Đối chiếu với Core Systems Architecture và Design Pattern của hệ thống CDC Data Hub.
  - Đảm bảo tính trung thực, không suy diễn, không báo cáo láo.
- **R3 - Đề xuất giải pháp khắc phục triệt để (The Single Best Approach)**:
  - Đưa ra giải pháp nâng cấp toàn diện cho bộ suy luận kiểu dữ liệu (`Type Inference Engine`) cho Extended JSON MongoDB (`$oid`, `$date`, `$numberDecimal`, ...).
  - Nâng cấp cơ chế lấy mẫu dữ liệu (`Sampling Engine`) để hỗ trợ Schema Polymorphic / Keyset Sampling / Reverse Sort thay vì chỉ lấy mẫu cố định 10 record đầu tiên hoặc LIMIT thô bạo.
- **R4 - Xuất báo cáo kiểm toán vật lý**: Lưu trữ đầy đủ toàn bộ kết quả audit vào `audit_report_mongo_scan_field.md` trong workspace.
