# 13_analysis_gp2_3667.md - Phân tích Kỹ thuật của AI

## Phân tích Chi tiết Yêu cầu Setup CDC MongoDB -> PostgreSQL (GP2-3667)

### 1. Phân tích Luồng Dữ liệu (Data Flow Analysis)
- **MongoDB Collection:** `transaction_history` (chứa các document giao dịch: nạp tiền, rút tiền, thanh toán, chuyển tiền).
- **ExtJSON Representation:** Trong Kafka topic và Shadow Table, các giá trị được biểu diễn theo định dạng ExtJSON:
  - Timestamp: `{"$date": "2026-08-24T08:00:00Z"}`
  - BigInt / Long: `{"$numberLong": "100000"}`
  - ObjectId: `{"$oid": "66bc8d..."}`
- **PostgreSQL Master Table Requirements:**
  - `id`: Primary key (Trích xuất từ `_id.$oid` hoặc string `_id`).
  - `trans_code`: Unique Index phục vụ tìm kiếm chính xác giao dịch.
  - `user_id` & `created_at`: Composite Index `(user_id, created_at DESC)` cho trang Lịch sử giao dịch cá nhân.
  - `merchant_id` & `created_at`: Composite Index `(merchant_id, created_at DESC)` cho trang Lịch sử giao dịch đối tác.

### 2. Phân tích Các Rủi ro & Bẫy Kỹ thuật (Tripwires & Safety Mitigation)
- **Bẫy 1 (Metadata Triplet Integrity):** Cần đảm bảo luôn truyền đủ bộ ba `(connection_key, schema, table)` khi thao tác với CMS & Transmuter, tránh rớt schema dẫn đến ép nhầm schema `public` hay sai physical DB.
- **Bẫy 2 (No-Shadow-Files):** Mọi tài liệu và báo cáo phải được lưu thành file vật lý trong workspace `SetupCdcMongoPostgresGP23667`.
- **Bẫy 3 (Brain Code Prohibition):** Brain lập Kế hoạch và thiết kế DDL / Solution trong `09_tasks_solution_gp2_3667.md` trước khi thực thi code.
