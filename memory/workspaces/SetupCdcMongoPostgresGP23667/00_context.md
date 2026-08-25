# 00_context.md - Scope & Context for GP2-3667

## 1. Tổng quan Epic & Task
- **Epic:** Optimize Transaction History (Tối ưu hiệu năng và cấu trúc xử lý lịch sử giao dịch, đảm bảo truy vấn nhanh, ổn định và dễ mở rộng).
- **Task ID:** GP2-3667
- **Tên Task:** Setup CDC MongoDB - PostgresSQL
- **Người thực hiện:** Lâm Văn Cảnh / Agentic Core
- **Mục tiêu:** Thiết lập hoàn chỉnh đường ống CDC từ MongoDB (Source Collection chứa dữ liệu Transaction History) sang PostgreSQL (Master Database chuẩn hoá cho truy vấn hiệu năng cao).

## 2. Các thành phần liên quan trong Hệ thống CDC (`data-hub`)
- **Source Database:** MongoDB (Chứa collection `transaction_history` / `transactions`).
- **Kafka Connect / Debezium MongoDB Connector:** Capture Change Data Capture (Oplog) từ MongoDB đẩy vào Kafka topic.
- **cdc-cms-service / cdc-cms-web:** Quản lý cấu hình System Connector, Source Objects, Shadow Table Schema, Master Table Registry và Mapping Rules.
- **centralized-data-service (CDS):** Engine xử lý CDC (Batch Buffer, Snapshot Runner V2, Sink Worker, Transmuter Module) từ Shadow Table sang Master Table PostgreSQL.
- **Master Database:** PostgreSQL (Nơi lưu trữ bảng `transaction_history` đã được transform chuẩn hoá với đầy đủ index và kiểu dữ liệu chuẩn SQL).
- **Consumer Service:** `core-trans-his-v2` (Task GP2-3706) sẽ truy vấn trực tiếp từ PostgreSQL Master Database.

## 3. Ranh giới & Ràng buộc Kỹ thuật
- Tuân thủ bộ ba định danh Metadata: `(connection_key, schema, table/collection)`.
- Tuyệt đối không tự ý dùng giá trị mặc định `public` hay tự ý hardcode localhost fallback.
- Đảm bảo tính nhất quán giữa ExtJSON MongoDB BSON Types (ObjectID, ISODate, NumberLong/Decimal128) và Postgres Types (UUID/VARCHAR, TIMESTAMPTZ, BIGINT/NUMERIC).
