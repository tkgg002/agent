# Yêu cầu Chi tiết - Tính năng Index Manager UI & Backend (Quản lý Index qua CMS)

## 1. Bối cảnh & Mục tiêu
Để khắc phục triệt để lỗi lock contention (`SQLSTATE 55P03`) và tối ưu hóa hiệu năng các query chậm trong reconciliation (như `BucketCounts` và `CountDeletedRows`), hệ thống cần có cơ chế quản lý index linh hoạt từ giao diện vận hành (CMS). 
Tính năng này cho phép operator:
- Kiểm tra danh sách index hiện tại trên cả Shadow DB và Master DB cho một bảng tương ứng.
- Tạo index mới một cách an toàn (sử dụng `CREATE INDEX CONCURRENTLY` không block read/write).
- Drop các index không cần thiết (ngoại trừ primary key/unique constraint) sử dụng `DROP INDEX CONCURRENTLY`.

## 2. Kiến trúc & Ràng buộc Kỹ thuật
- **Kiến trúc Phân cấp (Core Systems)**:
  - `cdc-cms-web` (Frontend): Hiển thị component `TableIndexManager` ở 2 màn hình mapping: Shadow Mapping và Master Mapping.
  - `cdc-cms-service` (CMS API): Nhận HTTP request từ Web, đóng gói và chuyển tiếp (Proxy) thành NATS RPC request sang Worker.
  - `centralized-data-service` (Worker): Nơi duy nhất kết nối với cả Shadow DB và Master DB, chịu trách nhiệm query metadata và thực thi câu lệnh DDL.
- **Ràng buộc An toàn (Safety Guards)**:
  - Mọi thao tác DDL (CREATE/DROP) **BẮT BUỘC** phải chạy ngoài transaction (vì Postgres không cho phép `CONCURRENTLY` trong transaction block).
  - Whitelist nghiêm ngặt các ký tự đầu vào của Table Name, Column Name, Index Name để tránh SQL Injection (chỉ chấp nhận `[a-zA-Z0-9_]`).
  - Cấm drop các index bắt đầu bằng `pk_` hoặc `ux_` nhằm bảo vệ tính toàn vẹn của primary key và unique constraint của hệ thống.
  - Timeout cho các command NATS RPC tạo index phải lớn (ví dụ 60s) để tránh lỗi kẹt/timeout NATS Request trên các bảng lớn.

## 3. Definition of Done (DoD)
- **FE**: Component `TableIndexManager` hiển thị chính xác danh sách index (tên, cột, size, lượt scan, valid status). Cho phép add index và drop index.
- **CMS API**: 3 REST API endpoints (`GET /v1/introspection/indexes/:table`, `POST /v1/introspection/indexes`, `DELETE /v1/introspection/indexes/:name`) proxy chính xác qua NATS.
- **Worker**: 3 NATS handlers (`cdc.cmd.introspect-indexes`, `cdc.cmd.create-index`, `cdc.cmd.drop-index`) hoạt động tốt, thực hiện DDL concurrency an toàn, không block DB.
- **Verify**: Các tests chạy qua, không lỗi build, deploy local chạy thử OK, không bị race.
