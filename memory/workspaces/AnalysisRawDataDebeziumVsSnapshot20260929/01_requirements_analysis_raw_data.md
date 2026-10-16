# 01 Requirements: Phân tích so sánh _raw_data giữa Debezium và Snapshot:v2

## 1. Bối cảnh
Người dùng cung cấp 2 mẫu payload `_raw_data` thực tế từ collection `export-jobs` (service `centralized-export-service`):
- Mẫu 1 (Debezium CDC streaming):
  ```json
  {"__v": 0, "_id": "693f8ee6190816bcb7188bc7", "error": null, "jobId": "export-1765883013798-zgexblx4r", "params": {"dateFr": "2025-10-01T17:00:00.000Z", "dateTo": "2025-10-04T16:59:59.999Z", "exportType": "PaymentBillExport"}, "status": "6e780d556570839ba27442ff79bed8881de853ca1bc39bb063c5641c651cf34a", "userId": "61234c18f1fc05b7a79a71b9", "fileUrl": "1765773014929-15-12-2025_lich.xlsx", "progress": 100, "createdAt": 1765773013801, "exportType": "PaymentBillExport", "merchantId": "unknown", "lastUpdatedAt": 1765773014983}
  ```
- Mẫu 2 (Snapshot:v2 batch read):
  ```json
  {"__v": 0, "_id": {"$oid": "6940d414190816bcb7188c13"}, "error": null, "jobId": "export-1765856276732-7kzxp0yle", "params": {"dateFr": "2025-11-30T17:00:00.000Z", "dateTo": "2025-12-16T16:59:59.999Z", "exportType": "PaymentBillExport", "merchantEmail": "giang.nguyen@goopay.vn"}, "status": "6e780d556570839ba27442ff79bed8881de853ca1bc39bb063c5641c651cf34a", "userId": "61234c18f1fc05b7a79a71b9", "fileUrl": "1765856277932-16-12-2025_lich_su_don_hang.xlsx", "progress": 100, "createdAt": {"$date": "2025-12-16T03:37:56.735Z"}, "exportType": "PaymentBillExport", "merchantId": "unknown", "lastUpdatedAt": {"$date": "2025-12-16T03:37:57.992Z"}}
  ```

## 2. Mục tiêu phân tích
1. Phân tích sự khác biệt về mặt cấu trúc giữa 2 định dạng (BSON ExtJSON vs Debezium Ingestion Format).
2. Phân tích nguyên nhân gốc rễ (Root Cause) trong kiến trúc và mã nguồn sinh ra 2 định dạng này.
3. Phân tích tác động và cách hệ thống CDC Data Hub hiện tại xử lý tương thích (Dual-Stack Adapter / Polymorphic Handling) ở các tầng:
   - DynamicMapper & Typed Columns (`unwrapMongoTypes`, `convertType`)
   - Shadow SQL Builder & Casting (`BuildCastExpr`)
   - Transmuter sang Master DB (`unwrapMongoExtJSON`, `flatten.go`, `gjson`)
4. Đánh giá rủi ro, nguyên tắc bất biến về Raw Data (Ground Truth - Rule #12, Lesson 2026-09-16) và khuyến nghị vận hành.
