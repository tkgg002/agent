# Kế hoạch Triển khai Task GP2-3667: Setup CDC MongoDB - PostgreSQL (Optimize Transaction History)

## 1. Mô tả Tổng quan
Task **GP2-3667** thuộc Epic **Optimize Transaction History** nhằm tối ưu hiệu năng và cấu trúc xử lý lịch sử giao dịch. 
Mục tiêu là thiết lập đường ống CDC (Change Data Capture) đồng bộ dữ liệu lịch sử giao dịch từ **MongoDB Source Collection** (`transaction_history`) sang **PostgreSQL Master Database** (`transaction_history`) được thiết kế với DDL chuẩn hoá và các chỉ mục (Indexes) tối ưu cho truy vấn phân trang hiệu năng cao (< 50ms).

---

## 2. Các điểm cần User Review / Duyệt (User Review Required)

> [!IMPORTANT]
> **Quyết định Mô hình Kiến trúc 2 Tầng (Source -> Shadow -> Master):**
> 1. **Shadow Tier (Tier 1):** Dữ liệu Change Events ExtJSON BSON từ MongoDB Kafka Change Stream được nạp nguyên văn vào Shadow Table `shadow_transaction_history` trên PostgreSQL (Shadow Schema).
> 2. **Master Tier (Tier 2):** Engine Transmuter trong `centralized-data-service` đọc từ Shadow Table, tự động biến đổi (parse ExtJSON BSON Date, NumberLong, ObjectId, JSONB) và ghi vào Master Table `transaction_history` trên PostgreSQL.
>
> **DDL Master Table & Indexing Strategy:**
> - `id` (VARCHAR 64) - Primary Key
> - `trans_code` (VARCHAR 64 UNIQUE) - Mã giao dịch
> - `user_id` (VARCHAR 64) & `created_at` (TIMESTAMPTZ) - Composite Index `(user_id, created_at DESC)`
> - `merchant_id` (VARCHAR 64) & `created_at` (TIMESTAMPTZ) - Composite Index `(merchant_id, created_at DESC)`
> - `amount` (BIGINT), `fee` (BIGINT), `status` (VARCHAR 32), `trans_type` (VARCHAR 32)
> - `extra_data` (JSONB) - Chứa các trường mở rộng linh hoạt

---

## 3. Các bước Triển khai Chi tiết (Proposed Changes)

### Phase 1: Tạo Workspace Memory & Kiểm tra Hạ tầng CDC
- [x] Tạo đầy đủ Workspace memory: `agent/memory/workspaces/SetupCdcMongoPostgresGP23667/` với bộ tài liệu quy chuẩn (00..13).
- [ ] Kiểm tra kết nối MongoDB Source Connector và PostgreSQL Target Database trên `cdc-cms-service` và Kafka Connect (`http://localhost:8084`).

### Phase 2: Khai báo Master DDL & Register CMS Bindings
- [ ] Thực thi DDL tạo bảng `transaction_history` trên PostgreSQL Target Database.
- [ ] Khai báo Source Object `transaction_history` (Collection MongoDB) qua `cdc-cms-service` API.
- [ ] Khai báo Shadow Table Binding & Master Table Binding (`master_transaction_history`).
- [ ] Đăng ký danh mục Sync Mapping Rules (BSON ExtJSON -> Postgres Native Types) trên CMS.

### Phase 3: Kích hoạt CDC Pipeline & Verification
- [ ] Khởi chạy MongoDB Debezium Connector / Snapshot Runner V2 để backfill và stream dữ liệu thời gian thực.
- [ ] Kiểm thử End-to-End từ Mongo Source -> Change Stream -> Shadow PG -> Transmuter -> Master PG.
- [ ] Đối soát dữ liệu (Count Source vs Count Target, Field accuracy).

---

## 4. Kế hoạch Kiểm thử & Xác minh (Verification Plan)

### Automated Verification
- Chạy linter quy trình: `python3 agent/tooling/verify_governance.py --workspace SetupCdcMongoPostgresGP23667`
- Chạy integration tests trong `centralized-data-service` và `cdc-cms-service`.

### Manual Verification
- Thực hiện insert 1 document giao dịch mẫu trên MongoDB.
- Kiểm tra dữ liệu được biến đổi tức tính và hiển thị đúng kiểu dữ liệu (TIMESTAMPTZ, BIGINT, JSONB) trên bảng PostgreSQL Master `transaction_history`.
- Kiểm tra latency truy vấn SQL lọc theo `user_id` và `created_at`.
