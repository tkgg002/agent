# Phân Tích Kiến Trúc Đa Hình (Polymorphic Source Object Model) & Cơ Chế Đếm Dữ Liệu Theo Thời Gian

- **Ngày cập nhật:** 2026-10-12
- **Tác giả:** Brain (Chairman & Architect)
- **Cơ sở chấn chỉnh từ User:**
  > *"hệ thống ko chỉ có 2 loại này, có thể là sftp, mongo, postgress, mysql ..."*

---

## I. TẠI SAO CẦN MÔ HÌNH ĐA HÌNH TOÀN DIỆN (POLYMORPHIC ARCHITECTURE)?

Hệ thống Data Hub của Goopay là nền tảng tích hợp dữ liệu đa nguồn (Omni-channel Integration & CDC Platform). Dữ liệu nghiệp vụ không chỉ gói gọn trong 2 CSDL Postgres và Mongo, mà trải dài trên rất nhiều công nghệ hạ tầng:

| Engine Type | Đặc thù dữ liệu | Cấp Namespace / Container | Tên đối tượng (Object Name) | Loại đối tượng (Object Type) | Cơ chế xác định ngày tạo |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **PostgreSQL** | Relational DB | `Schema` (vd: `master_payment_bill_service`) | `Table Name` (vd: `payment_bills`) | `table`, `partition`, `view` | Cột timestamp (vd: `created_at`, `create_time`) |
| **MySQL / MariaDB** | Relational DB | `Database` (vd: `goopay_prod`, `payment_db`) | `Table Name` (vd: `orders`, `transactions`) | `table`, `view` | Cột datetime (vd: `created_at`) |
| **MongoDB** | Document NoSQL | `Database` (vd: `traitest_ctphs`) | `Collection Name` (vd: `trans_his`, `events`) | `collection` | Field ISO date / timestamp (vd: `createdAt`) |
| **SFTP / File** | File đối soát định kỳ (CSV / Excel / XML) | `Directory Path` (vd: `/reconciliation/inbox/`) | `File Name / Pattern` (vd: `clearing_*.csv`) | `file`, `file_batch` | Ngày trong tên file (`clearing_YYYYMMDD.csv`) hoặc `file_mtime` |
| **ClickHouse** | Columnar Analytics | `Database` (vd: `analytics_dw`) | `Table Name` (vd: `audit_events`) | `table` | Cột `event_date`, `created_at` |
| **Kafka / NATS** | Event Streaming | `Cluster / Namespace` | `Topic Name` (vd: `core.payment.trans.events`) | `topic` | Kafka record timestamp |

Nếu DDL và giao diện chỉ tư duy nhị phân ("table hoặc collection", "schema_or_db") thì:
1. Hoàn toàn không thể cấu hình đối soát cho các nguồn **SFTP file** từ ngân hàng/đối tác đẩy về hàng ngày.
2. Không thể mở rộng sang **MySQL**, **MariaDB**, hay **ClickHouse**.
3. Ràng buộc `CHECK` bị lỗi cú pháp logic khi gặp đối tượng dạng file hoặc topic.

---

## II. THIẾT KẾ BẢNG `cdc_system.daily_source_counts` ĐA HÌNH CHUẨN XÁC

Để phản ánh chính xác bản chất đa hình của toàn bộ các engine trong hệ sinh thái Data Hub:

```sql
-- Migration 082_create_polymorphic_daily_source_counts.sql
BEGIN;

CREATE TABLE IF NOT EXISTS cdc_system.daily_source_counts (
    id                      BIGSERIAL PRIMARY KEY,
    schedule_id             INT REFERENCES cdc_system.cdc_worker_schedule(id) ON DELETE SET NULL,
    
    -- 1. Phân định Trạm & Kết nối
    layer_type              VARCHAR(32) NOT NULL,               -- 'source', 'shadow', 'master'
    connection_code         VARCHAR(128) NOT NULL,              -- Mã định danh kết nối từ cdc_system.connection_registry
    engine_type             VARCHAR(32) NOT NULL,               -- 'postgresql', 'mongodb', 'mysql', 'mariadb', 'sftp', 'clickhouse', 'kafka', 'file'
    
    -- 2. Định vị Đối tượng Dữ liệu (Polymorphic Namespace & Object)
    container_namespace     VARCHAR(255) NOT NULL DEFAULT '',   -- Postgres: schema | MySQL/Mongo: database | SFTP: remote directory path
    object_name             VARCHAR(255) NOT NULL,              -- SQL: table | Mongo: collection | SFTP: filename pattern | Kafka: topic
    object_type             VARCHAR(32) NOT NULL,               -- 'table', 'collection', 'file', 'file_batch', 'topic', 'view'
    
    -- 3. Chiến lược & Trường Ngày Tạo
    date_strategy           VARCHAR(64) NOT NULL DEFAULT 'column', -- 'column' (CSDL), 'filename_regex' (SFTP), 'file_mtime' (SFTP), 'message_timestamp' (Kafka)
    date_field              VARCHAR(128) NOT NULL,              -- CSDL: tên cột ('createdAt', 'created_at') | SFTP: pattern/cột ('clearing_(\d{8}).csv')
    
    -- 4. Kết quả Kiểm đếm
    report_date             DATE NOT NULL,                      -- Ngày đối soát (vd: 2026-10-11)
    creation_count          BIGINT NOT NULL DEFAULT 0,          -- Số lượng bản ghi / số dòng dữ liệu sinh ra trong ngày
    
    -- 5. Giám sát & Đo lường
    calculated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Thời điểm Worker thực thi lệnh đếm
    execution_duration_ms   INT NOT NULL DEFAULT 0,             -- Thời gian thực thi (ms)
    metadata_json           JSONB DEFAULT '{}'::jsonb,          -- Chi tiết: danh sách file SFTP, query SQL, line count breakdown, checksum
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ràng buộc duy nhất đảm bảo tính idempotent
    UNIQUE (layer_type, connection_code, container_namespace, object_name, report_date)
);

-- Index tra cứu siêu tốc cho CMS Reconciliation Grid và Confluence Bot
CREATE INDEX IF NOT EXISTS idx_daily_source_counts_lookup 
  ON cdc_system.daily_source_counts(layer_type, container_namespace, object_name, report_date);

CREATE INDEX IF NOT EXISTS idx_daily_source_counts_engine 
  ON cdc_system.daily_source_counts(engine_type, object_type);

CREATE INDEX IF NOT EXISTS idx_daily_source_counts_report_date 
  ON cdc_system.daily_source_counts(report_date);

COMMENT ON TABLE cdc_system.daily_source_counts IS 
  'Bảng lưu trữ tập trung kết quả đếm số lượng bản ghi/dòng dữ liệu nghiệp vụ sinh ra trong ngày theo từng nguồn (PostgreSQL, MongoDB, MySQL, SFTP, ClickHouse).';

COMMIT;
```

---

## III. GIAO DIỆN MODAL ĐA HÌNH TRÊN `/activity-manager`

Form Modal tạo/chỉnh sửa tác vụ `source-count` trên `ActivityManager.tsx` được thiết kế phản ứng thông minh (Reactive UI) theo `engine_type` của Connection được chọn:

```
[BƯỚC 1] Chọn Layer Type: [ source ▼ ]
            │
            ▼
[BƯỚC 2] Chọn Connection: [ sftp_bank_clearing (SFTP) ▼ ]
            │
            ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ Giao diện tự động thích ứng với Engine SFTP:                                          │
│ • Container Path:    [ /reconciliation/inbox/vcb/                                    ] │
│ • Object Name:        [ clearing_vcb_*.csv                                           ] │
│ • Loại đối tượng:     [ File Batch (Quét nhiều file khớp pattern)                     ] │
│ • Chiến lược ngày:    [ Trích xuất ngày từ tên file (Regex: clearing_vcb_(\d{8})\.csv)] │
└────────────────────────────────────────────────────────────────────────────────────────┘

(Hoặc nếu chọn Connection Mongo):
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ Giao diện tự động thích ứng với Engine MONGODB:                                       │
│ • Database:           [ traitest_ctphs                                               ] │
│ • Collection:         [ trans_his                                                    ] │
│ • Loại đối tượng:     [ Collection                                                   ] │
│ • Trường ngày tạo:    [ createdAt                                                    ] │
└────────────────────────────────────────────────────────────────────────────────────────┘

(Hoặc nếu chọn Connection PostgreSQL):
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ Giao diện tự động thích ứng với Engine POSTGRESQL:                                    │
│ • Schema:             [ master_payment_bill_service                                  ] │
│ • Table:              [ payment_bills                                                ] │
│ • Loại đối tượng:     [ Table                                                        ] │
│ • Trường ngày tạo:    [ created_at                                                   ] │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## IV. CƠ CHẾ THỰC THI KIỂM ĐẾM CỦA WORKER RUNNER THEO TỪNG ENGINE

Trong `centralized-data-service`, runner `source-count` ủy quyền xử lý cho các driver chuyên biệt:

1. **Driver SFTP (`engine_type == 'sftp'`):**
   - Mở kết nối SSH/SFTP tới host cấu hình trong `connection_registry`.
   - Liệt kê các file trong thư mục `container_namespace`.
   - Lọc các file khớp với `object_name` (vd: `clearing_vcb_*.csv`) và có ngày tương ứng với `report_date` ($D-1$).
   - Đọc và đếm số dòng dữ liệu (loại trừ header/trailer nếu có).
   - Ghi tổng số dòng vào `daily_source_counts`, đồng thời lưu danh sách file đã đếm vào `metadata_json`.

2. **Driver PostgreSQL / MySQL / ClickHouse (`engine_type IN ('postgresql', 'mysql', 'mariadb', 'clickhouse')`):**
   - Chạy truy vấn SQL đếm có điều kiện thời gian:
     ```sql
     SELECT COUNT(*) FROM "container_namespace"."object_name"
     WHERE "date_field" >= :startOfDay AND "date_field" <= :endOfDay;
     ```

3. **Driver MongoDB (`engine_type == 'mongodb'`):**
   - Chạy lệnh đếm documents:
     ```javascript
     db.getSiblingDB(container_namespace).getCollection(object_name).countDocuments({
       [date_field]: { $gte: ISODate(startOfDay), $lte: ISODate(endOfDay) }
     });
     ```
