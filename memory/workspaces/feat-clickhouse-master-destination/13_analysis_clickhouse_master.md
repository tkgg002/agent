# 13 Analysis: Phân tích Kỹ thuật Nâng cấp Master Tier ClickHouse

> **Workspace**: `feat-clickhouse-master-destination`

---

## 1. Phân tích Bản chất Kiến trúc ClickHouse cho CDC

ClickHouse được thiết kế tối ưu cho các truy vấn phân tích quét khối lượng lớn (Analytical Columnar Engine). Khác với RDBMS như PostgreSQL, ClickHouse có các đặc tính kỹ thuật cần được xử lý đặc biệt trong CDC:

### 1.1 Vấn đề Mutation (UPDATE/DELETE)
- Trong PostgreSQL: `UPDATE` và `DELETE` diễn ra theo từng hàng (row-level) với ACID locks.
- Trong ClickHouse: `ALTER TABLE ... UPDATE/DELETE` là **Mutations** — nó viết lại toàn bộ data parts trên đĩa. Nếu chạy từng row hoặc batch nhỏ liên tục, ClickHouse sẽ sập vì quá tải I/O ("Too many mutations").
- **Giải pháp**: Bắt buộc dùng **`ReplacingMergeTree(_version, _deleted)`**:
  - Không bao giờ chạy mutation.
  - Toàn bộ thao tác INSERT, UPDATE, DELETE đều quy về **APPEND BATCH**.
  - Background merge tự động dọn dẹp các version cũ.

### 1.2 Hiệu năng Ghi (Batching Strategy)
- ClickHouse không thích hợp để insert từng dòng đơn lẻ (single-row insert).
- Chuẩn thiết kế: Gom các dòng từ Shadow Table thành batch từ 10,000 đến 50,000 dòng rồi gọi `batch.Send()`.
- Tốc độ insert có thể đạt hàng trăm nghìn dòng mỗi giây, giảm tải I/O tối đa.

---

## 2. Khả năng Mở rộng Dual-Master Fan-Out (Postgres + ClickHouse)
Do bảng `cdc_system.master_binding` trong CMS hỗ trợ 1 Source Object liên kết tới N Master Bindings:
- Khách hàng có thể cấu hình **Dual Master**:
  * Master 1 (`PostgreSQL`): Phục vụ các nghiệp vụ Transactional / OLTP / Tra cứu đơn lẻ theo ID.
  * Master 2 (`ClickHouse`): Phục vụ các nghiệp vụ Thống kê / Báo cáo / Dashboard Realtime / Data Science.
- Hai luồng Transmute chạy độc lập, bảo đảm tính sẵn sàng cao và phân lập tải hoàn toàn.
