# BÁO CÁO AUDIT KIẾN TRÚC TOÀN NGUỒN (CROSS-ENGINE AUDIT REPORT)
## RÀ SOÁT TẦM NHÌN OVERVIEW: POSTGRESQL, MARIADB, MYSQL, MONGODB & SFTP Ở TẦNG SOURCE

---
- **Thời gian thực hiện:** 2026-09-16T15:40:00+07:00
- **Thực thể thực hiện:** Brain (Chairman & Architect)
- **Workspace:** `/Users/trainguyen/Documents/work/agent/memory/workspaces/FixKafkaConsumerSourceRouteMultiConn20260916/`
- **Chủ đề:** Tầm nhìn kiến trúc mở rộng toàn bộ các hệ quản trị CSDL nguồn (PostgreSQL, MariaDB, MySQL, MongoDB, SFTP) đối với bài toán đa kết nối (Multi-Connection Source Isolation).

---

## I. PHẢN TỈNH & NGUYÊN NHÂN GỐC RỄ (ROOT CAUSE OF TUNNEL VISION)

Trước đó, khi giải quyết lỗi nhầm lẫn bảng Shadow giữa 2 kết nối MongoDB (`traitestmongodevct` vs `traitestctphs`), đội ngũ triển khai đã mắc **Bệnh Tầm Nhìn Hạn Hẹp (Single-Engine Tunnel Vision)**:
1. Chỉ chăm chăm kiểm tra và phân lập prefix cho MongoDB (`TOPIC_PREFIX_MONGODB.${name}`), trong khi **PostgreSQL và MySQL/MariaDB vẫn bị bỏ rơi với topic prefix tĩnh**.
2. Trong Kafka Consumer chỉ hardcode kiểm tra `sourceConnCode == "mongodb"`, **bỏ quên các kết nối nguồn PostgreSQL, MariaDB, MySQL**.
3. Bỏ qua sự khác biệt về cấu trúc namespace của PostgreSQL (`<schema>.<table>` thay vì `<db>.<collection>`), khiến việc bóc tách và phân giải route cho PostgreSQL bị lệch.

---

## II. SO SÁNH MA TRẬN ĐẶC TÍNH NGUỒN CỦA CÁC ENGINE (SOURCE ENGINE MATRIX)

| Tiêu chí | MongoDB | PostgreSQL | MySQL / MariaDB | SFTP |
|:---|:---|:---|:---|:---|
| **Cấu trúc Không gian tên** | `Database` + `Collection` | `Database` + `Schema` + `Table` | `Database` + `Table` (Schema = DB) | `Directory` + `File` (Table) |
| **Topic Debezium chuẩn** | `<prefix>.<db>.<collection>` | `<prefix>.<schema>.<table>` | `<prefix>.<db>.<table>` | `<prefix>.<filename>` |
| **Topic phân lập đa kết nối** | `cdc.goopay.<conn>.<db>.<collection>` | `cdc.goopay.<conn>.<schema>.<table>` | `cdc.goopay.<conn>.<db>.<table>` | `cdc.goopay.<conn>.<filename>` |
| **Số segments topic** | 5 parts | 5 parts | 5 parts | 4-5 parts |
| **Vị trí Connection trong Topic** | `parts[2]` | `parts[2]` | `parts[2]` | `parts[2]` |
| **Vị trí Table/Collection** | `parts[4]` (`len-1`) | `parts[4]` (`len-1`) | `parts[4]` (`len-1`) | `parts[len-1]` |
| **Vị trí DB / Schema trong Topic**| `parts[3]` (`len-2`) = DB | `parts[3]` (`len-2`) = **Schema** | `parts[3]` (`len-2`) = DB | N/A |
| **Trường Debezium Envelope** | `source.db`, `source.collection` | `source.db`, `source.schema`, `source.table` | `source.db`, `source.table` | Payload phẳng / wrapper |

---

## III. CHI TIẾT CÁC LỖ HỔNG HỆ THỐNG KHI CHẠY POSTGRESQL & MARIADB/MYSQL Ở SOURCE

### 🔴 1. Lỗ hổng Trộn Lẫn Kafka Topic trên Giao diện CMS (`SourceConnectors.tsx`)
- **Vị trí:** `cdc-cms-web/src/pages/SourceConnectors.tsx:L489-493`
- **Hiện trạng:**
  ```typescript
  } else if (dbKind === 'mongodb') {
    form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MONGODB}.${name}`);
  } else if (dbKind === 'mysql') {
    form.setFieldValue('topicPrefix', TOPIC_PREFIX_MYSQL); // ❌ LỖI
  } else if (dbKind === 'postgresql') {
    form.setFieldValue('topicPrefix', TOPIC_PREFIX_POSTGRESQL); // ❌ LỖI
  }
  ```
- **Hậu quả thực tế:**
  + Khi User tạo 2 connector PostgreSQL (`pg_conn_1` và `pg_conn_2`) hoặc 2 connector MariaDB/MySQL (`maria_conn_1` và `maria_conn_2`):
  + Form tự động gán chung `topicPrefix = "cdc.gpay"` (hoặc `"cdc.goopay"`).
  + Nếu 2 database nguồn có cùng tên schema và table (vd: `public.transactions` hoặc `payment.orders`):
  + Cả 2 connector trên Debezium Kafka Connect **sẽ sinh ra Kafka topic CÙNG TÊN** (`cdc.goopay.public.transactions`)!
  + Hai nguồn dữ liệu độc lập bị bắn chung vào 1 topic Kafka, gây xáo trộn dữ liệu (Data Corruption) ngay tại Broker!
- **Khắc phục chuẩn:**
  ```typescript
  } else if (dbKind === 'mysql') {
    form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MYSQL}.${name}`);
  } else if (dbKind === 'postgresql') {
    form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_POSTGRESQL}.${name}`);
  }
  ```

---

### 🔴 2. Lỗ hổng Hardcode "mongodb" trong Kafka Consumer (`kafka_consumer.go`)
- **Vị trí:** `centralized-data-service/internal/handler/shadow/kafka_consumer.go:L670`
- **Hiện trạng:**
  ```go
  sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.goopay.")
  sourceConnCode = strings.TrimPrefix(sourceConnCode, "cdc.")

  if sourceConnCode == "" || sourceConnCode == "mongodb" {
      parts := strings.Split(msg.Topic, ".")
      if len(parts) >= 5 && parts[0] == "cdc" {
          sourceConnCode = parts[2]
      }
  }
  ```
- **Hậu quả thực tế:**
  + Với PostgreSQL / MariaDB / MySQL cũ, nếu Debezium payload `sourceRaw["name"]` mang giá trị generic như `"postgres"`, `"postgresql"`, `"mysql"`, `"mariadb"`, `"gpay"`:
  + Biến `sourceConnCode` giữ nguyên `"postgres"` hoặc `"mysql"` (do điều kiện chỉ bắt `sourceConnCode == "mongodb"`).
  + Khi truyền `"postgres"` vào `ResolveSourceRoutes(sourceDB, sourceTable, "postgres")`, trong registry không có connection nào tên là `"postgres"` (tên connection thực tế là `traitest_pg_prod`).
  + Dẫn tới không tìm thấy route $\rightarrow$ Event bị bỏ rơi (Dropped)!
- **Khắc phục chuẩn:**
  Xóa bỏ hardcode `sourceConnCode == "mongodb"`, mở rộng kiểm tra tất cả generic engine names:
  ```go
  isGeneric := sourceConnCode == "" ||
      sourceConnCode == "mongodb" ||
      sourceConnCode == "postgres" ||
      sourceConnCode == "postgresql" ||
      sourceConnCode == "mysql" ||
      sourceConnCode == "mariadb" ||
      sourceConnCode == "gpay"
  if isGeneric {
      parts := strings.Split(msg.Topic, ".")
      if len(parts) >= 5 && parts[0] == "cdc" {
          sourceConnCode = parts[2]
      } else if len(parts) == 4 && parts[0] == "cdc" && parts[1] != "goopay" && parts[1] != "gpay" {
          sourceConnCode = parts[1]
      }
  }
  ```

---

### 🔴 3. Lỗ hổng Trích xuất Thiếu `schema` của PostgreSQL trong Kafka Consumer & Event Handler
- **Vị trí:**
  + `kafka_consumer.go:L653-665`
  + `event_handler.go:L188-208`
- **Hiện trạng:**
  Cả 2 file đều chỉ trích xuất `db` và `table` / `collection`:
  ```go
  if d, ok := sm["db"].(string); ok { sourceDB = strings.TrimSpace(d) }
  if t, ok := sm["table"].(string); ok { sourceTable = strings.TrimSpace(t) }
  ```
  Hoàn toàn **không đọc `sm["schema"]`**!
- **Hậu quả thực tế:**
  + Trong Debezium PostgreSQL:
    `source.db` = `"core_payment"` (Tên database Postgres).
    `source.schema` = `"public"` (Tên schema Postgres).
    `source.table` = `"orders"` (Tên bảng).
  + Nếu trong `source_object_registry`, người dùng đăng ký bảng là:
    `source_database = "core_payment"`, `source_schema = "public"`, `source_object_name = "orders"`.
    Registry sinh các key:
    `conn:core_payment|orders`
    `conn:core_payment|public.orders`
    `conn:public.orders`
    `conn:orders`
  + Nếu topic Kafka của PostgreSQL là `cdc.goopay.<conn>.public.orders`:
    Khi fallback theo topic: `table = parts[len-1] = "orders"`, `db = parts[len-2] = "public"`.
    Lúc này `sourceDB` bị gán là `"public"`, `sourceTable` là `"orders"`.
    Hàm `buildRouteLookupKeys` sinh key: `conn:public|orders`.
    Key này KHÔNG KHỚP với `conn:core_payment|public.orders` trong cache!
- **Khắc phục chuẩn:**
  1. Trích xuất cả `sourceSchema` từ Debezium payload (`sm["schema"]`).
  2. Bổ sung vào struct `CDCEvent`: trường `SourceSchema string `json:"source_schema,omitempty"``.
  3. Trong `buildRouteLookupKeys`: nếu `sourceDB` và `sourceTable` được truyền vào, bổ sung thêm key biến thể `conn:sourceDB.sourceTable` (tương ứng `conn:schema.table`). Khi đó `conn:public.orders` sẽ khớp 100%!

---

## IV. KẾ HOẠCH HÀNH ĐỘNG KHẮC PHỤC TRIỆT ĐỂ (FULL-ENGINE REMEDIATION PLAN)

1. **CMS Web Frontend (`SourceConnectors.tsx`):**
   - Phân lập topic prefix `${name}` cho toàn bộ: `mongodb`, `postgresql`, `mysql` (mariadb).
2. **Model CDCEvent (`cdc_event.go`):**
   - Bổ sung `SourceSchema string `json:"source_schema,omitempty"``.
3. **Kafka Consumer (`kafka_consumer.go`):**
   - Trích xuất `sourceSchema` từ `sm["schema"]`.
   - Mở rộng nhận diện connection code từ topic cho tất cả generic engine names (`postgres`, `postgresql`, `mysql`, `mariadb`, `mongodb`, `gpay`).
4. **Metadata Registry Utils (`metadata_registry_utils.go`):**
   - `buildRouteLookupKeys` bổ sung key `conn:sourceDB.sourceTable` để hỗ trợ tra cứu PostgreSQL khi schema nằm ở vị trí `sourceDB`.
5. **Metadata Registry Service (`metadata_registry_service.go`):**
   - Sửa dứt điểm Lỗ hổng 1: Khi `conn != ""` mà `len(filtered) == 0`, BẮT BUỘC `return nil`.
6. **Unit Test (`metadata_registry_service_test.go`):**
   - Viết test suite phân lập cho:
     * 2 kết nối PostgreSQL trùng DB, Schema, Table.
     * 2 kết nối MariaDB/MySQL trùng DB, Table.
     * Negative test: connection không tồn tại trả về `nil`.
