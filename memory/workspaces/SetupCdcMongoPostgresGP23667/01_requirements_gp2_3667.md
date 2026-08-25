# 01_requirements_gp2_3667.md - Yêu cầu Chi tiết (Specs)

## 1. Yêu cầu Chức năng (Functional Requirements)
- **REQ-1:** Khảo sát và khai báo Source Connector MongoDB chứa dữ liệu lịch sử giao dịch (Collection `transaction_history` hoặc tương đương).
- **REQ-2:** Định nghĩa và đồng bộ Schema cho Shadow Table (MongoDB Raw BSON ExtJSON storage in PostgreSQL shadow schema).
- **REQ-3:** Thiết kế DDL cho Master Table `transaction_history` trên PostgreSQL target database với các trường chỉ mục (Indexes) tối ưu truy vấn theo `user_id`, `merchant_id`, `created_at`, `status`, `transaction_code`.
- **REQ-4:** Tạo các Mapping Rules (Chuyển đổi kiểu dữ liệu BSON/ExtJSON sang PostgreSQL Native SQL Types) trong `cdc-cms-service`.
- **REQ-5:** Kích hoạt luồng CDC đồng bộ thời gian thực (Realtime Oplog streaming via Debezium) và hỗ trợ Snapshot Runner V2 (nếu cần sync dữ liệu lịch sử ban đầu).
- **REQ-6:** Kiểm thử và xác minh tính đúng đắn dữ liệu (Data Integrity Audit - count matching, field-level correctness).

## 2. Yêu cầu Phi Chức năng (Non-Functional Requirements)
- **Truy vấn nhanh:** Master table trên PostgreSQL phải được indexed chuẩn hoá để đạt latency truy vấn < 50ms cho các truy vấn phân trang lịch sử giao dịch.
- **Tính sẵn sàng & Khắc phục sự cố:** Pipeline có khả năng resume tự động từ checkpoint/cursor mà không gây duplicate hoặc nát dữ liệu.
- **Tính toàn vẹn Metadata:** Mọi API request, query và mapping phải tuân thủ bộ ba `(connection_key, schema, table)`.
