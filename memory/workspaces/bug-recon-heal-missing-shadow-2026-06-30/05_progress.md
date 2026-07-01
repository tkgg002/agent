# Progress: Điều tra lỗi không đồng bộ record thiếu sau khi bắn Debezium Signal

## Metadata Integrity
- **2026-06-30 11:45:00 +0700 [Agent:Gemini 3.5 Flash (High)]** Action: Khởi tạo workspace `bug-recon-heal-missing-shadow-2026-06-30`.
- **2026-06-30 11:50:00 +0700 [Agent:Gemini 3.5 Flash (High)]** Action: Bắt đầu kiểm tra cấu hình Debezium Connector.
- **2026-06-30 12:00:00 +0700 [Agent:Gemini 3.5 Flash (High)]** Action: Phát hiện lỗi kiểu dữ liệu Primary Key và cơ chế sinh filter của buildSnapshotIDFilter.

## Root Cause Analysis (Governance & Configuration)
- **Vấn đề**: Record bị thiếu `41063` không xuất hiện ở Shadow DB mặc dù Debezium signal đã bắn thành công.
- **Gốc rễ (Root Cause)**: 
  1. Trong `cdc_table_registry`, `primary_key_type` của table `payment_bills` cấu hình sai là `VARCHAR(24)`. Thực tế trong MongoDB, tất cả 40,055 bản ghi đều có `_id` kiểu `int` (bigint ở Shadow DB).
  2. Hàm `buildSnapshotIDFilter` khi sinh filter cho MongoDB mặc định bọc ngoặc kép `""` quanh ID (VD: `{"_id": {"$in": ["41063"]}}`). MongoDB so sánh kiểu dữ liệu nghiêm ngặt nên query trả về empty cho Debezium Connect, khiến snapshot không trigger dữ liệu thật.
- **Hậu quả**: Dữ liệu đối soát bị lệch và tự động chữa lành không hoàn thành nhiệm vụ.
- **Bài học & Biện pháp khắc phục**: 
  1. Cập nhật registry của `payment_bills` thành `BIGINT` trong database `cdc_dw` (đã thực hiện).
  2. Sửa hàm `buildSnapshotIDFilter` trong `recon_heal_v4.go` để hỗ trợ numeric IDs cho MongoDB (không bọc `""` nếu pkType là numeric).

## Phân tích Gốc rễ (Root Cause) Vi phạm Quy trình Governance
- **Lỗi vi phạm**: Không có vi phạm quy trình Governance nào.
- **Nguyên nhân gốc rễ**: N/A.
- **Hành động khắc phục**: N/A.

## Tiến độ thực hiện
- [x] Khởi tạo workspace và lập kế hoạch điều tra.
- [x] Kiểm tra connector config của Debezium (đã cập nhật signal.data.collection ở bước trước).
- [x] Kiểm tra Kafka topics (đã kiểm tra qua scratch_read_signal.go).
- [x] Xác định nguyên nhân gốc rễ.
- [/] Sửa lỗi sinh filter trong `recon_heal_v4.go`.
- [ ] Xác minh kết quả và sync record 41063 về shadow DB.
