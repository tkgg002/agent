# Requirements: CDC Integration (Developer Tasks)

> **Source**: `/Users/trainguyen/Documents/work-desc/feature/cdc.md`
> **Date**: 2026-03-16
> **Strategy**: Hybrid Approach (Debezium + NATS + Airbyte)

## 🎯 Objective

Xây dựng hệ thống CDC하이brid cho GooPay với 2 luồng dữ liệu song song:
- **Real-time Stream** (Debezium → NATS → Go Worker → PostgreSQL)
- **Batch Sync** (Airbyte → PostgreSQL)

## 📋 Developer Tasks (Chi tiết từ Requirements + Update)

> **Updated**: 2026-03-16 với Dynamic Mapping Engine, JSONB Landing Zone, CMS Approval Workflow

### Task 1: Xây dựng Go CDC Worker (Luồng NATS)

**Mục tiêu**: Viết code Go xử lý dữ liệu từ Debezium để thực hiện các logic nghiệp vụ tức thời và **Data Enrichment**.

**Yêu cầu**:
- Subscribe vào NATS JetStream topics từ Debezium
- Parse CDC events (INSERT/UPDATE/DELETE operations)
- Thực hiện data enrichment (thêm metadata, computed fields)
- Xử lý business logic real-time (ví dụ: validate transaction, calculate balances)
- Write enriched data vào PostgreSQL
- Error handling và retry mechanism
- Logging và metrics cho monitoring

**Input**: NATS events từ Debezium (format: Debezium CloudEvents hoặc Avro)
**Output**: Enriched records trong PostgreSQL tables

---

### Task 2: Quản lý Schema Mapping tại PostgreSQL

**Mục tiêu**: Thiết kế cấu hình bảng để nhận dữ liệu từ cả Airbyte và Go Worker. Đảm bảo không có xung đột về Primary Key hoặc Constraints.

**Yêu cầu**:
- Thiết kế schema tương thích cả 2 nguồn write (Airbyte + Go Worker)
- Xử lý conflict resolution:
  - Upsert strategy (INSERT ON CONFLICT UPDATE)
  - Timestamp-based versioning để detect stale data
  - Metadata columns: `source` (airbyte/debezium), `synced_at`, `modified_at`
- Partition strategy cho large tables
- Indexes optimization cho query performance

**Ràng buộc**:
- Primary Key phải consistent giữa source MongoDB/MySQL và target PostgreSQL
- Airbyte writes có thể chậm hơn CDC Worker → cần conflict resolution
- Schema changes phải backward compatible

---

### Task 3: Viết Event Bridge (Postgres → NATS)

**Mục tiêu**: Với các bảng mà Airbyte ghi trực tiếp vào Postgres, bạn cần viết logic (ví dụ: Postgres Trigger hoặc Go Listener) để phát ra NATS event nếu các service khác trong hệ thống Moleculer cần biết.

**Yêu cầu**:
- **Option A**: PostgreSQL Trigger + NOTIFY/LISTEN mechanism
  - Trigger phát event khi có INSERT/UPDATE/DELETE
  - Go Listener service lắng nghe NOTIFY và publish vào NATS
- **Option B**: Polling-based approach
  - Go service poll PostgreSQL changelog table
  - Publish changes vào NATS với debounce/batching
- Format NATS events tương thích với Moleculer event format
- Filter events theo business rules (không phải tất cả changes đều cần publish)

**Output**: NATS events cho Moleculer services (format chuẩn với action, params, meta)

---

### Task 4: Xử lý Data Reconciliation (Đối soát)

**Mục tiêu**: Viết script kiểm tra xem dữ liệu giữa "luồng nhanh" (Debezium) và "luồng chậm" (Airbyte) có khớp nhau không, tránh tình trạng lệch dữ liệu (Data Drift).

**Yêu cầu**:
- So sánh record counts giữa Source DB và PostgreSQL
- Checksum-based validation cho data integrity
- Detect missing records (có trong source nhưng không có trong target)
- Detect stale records (target data cũ hơn source)
- Report discrepancies với details (table, primary key, fields mismatch)
- Auto-repair mechanism (optional) hoặc alert cho manual intervention

**Frequency**:
- Critical tables: Mỗi 5-15 phút
- Non-critical tables: Mỗi 1-4 giờ

**Output**:
- Reconciliation report (JSON/CSV)
- Metrics gửi lên monitoring system (Prometheus/Grafana)
- Alerts khi phát hiện drift > threshold

---

### Task 5: Schema Drift Detection Module (NEW)

**Mục tiêu**: Xây dựng module Go tự động phát hiện sự thay đổi schema (field mới) trong CDC events và trigger approval workflow.

**Yêu cầu**:
- **Schema Inspector**: So sánh JSON payload từ NATS/Airbyte với `information_schema.columns` của PostgreSQL
- **Drift Detection Logic**:
  - Parse JSON event, extract tất cả field names
  - Query PostgreSQL để lấy danh sách columns hiện tại của bảng
  - Detect fields mới (có trong JSON nhưng không có trong DB)
- **Pending Fields Tracking**:
  - Lưu fields mới vào bảng `pending_fields` với metadata (table_name, field_name, sample_value, detected_at)
  - Suy đoán data type dựa trên sample value (String, Number, Boolean, JSON)
- **Alert Mechanism**:
  - Publish event `schema.drift.detected` lên NATS
  - Gửi notification tới CMS Service để hiển thị trên UI
- **JSONB Fallback**: Khi detect field mới, tự động lưu toàn bộ JSON vào cột `_raw_data` để đảm bảo zero data loss

**Output**:
- Bảng `pending_fields` được populate
- CMS nhận được alert
- Dữ liệu KHÔNG bị mất (lưu trong `_raw_data`)

---

### Task 6: CMS Integration & Approval Workflow (NEW)

**Mục tiêu**: Xây dựng CMS service với UI cho DevOps/Dev review và approve schema changes.

**Yêu cầu**:

#### 6.1 CMS Backend API (Go/Node.js)
- **GET /api/schema-changes/pending**: List tất cả pending schema changes
- **POST /api/schema-changes/{id}/approve**: Approve schema change
  - Execute `ALTER TABLE ADD COLUMN` trên PostgreSQL
  - Update `cdc_mapping_rules` table với field mapping mới
  - Publish `schema.config.reload` event lên NATS
  - Mark pending_field as `approved`
- **POST /api/schema-changes/{id}/reject**: Reject schema change
  - Mark as rejected với reason
- **GET /api/mapping-rules**: List tất cả mapping rules
- **POST /api/mapping-rules**: Create/Update mapping rule manually

#### 6.2 CMS Frontend UI (React/Vue)
- **Pending Changes View**: Hiển thị danh sách field mới cần approve
  - Table name, Field name, Detected at, Sample values
  - Suggested data type (có thể edit)
  - Target column name (có thể customize)
  - Approve/Reject buttons
- **Mapping Rules Manager**: CRUD interface cho `cdc_mapping_rules`
  - Filter by source table
  - Toggle is_active
  - Mark fields as "enriched" (cần Go logic xử lý)

#### 6.3 Audit Trail
- Log tất cả schema changes vào `schema_changes_log` table
- Bao gồm: who approved, when, original schema vs new schema

**Output**:
- CMS service hoạt động với UI đầy đủ
- DevOps có thể approve schema changes qua web UI

---

### Task 7: Dynamic Mapping Engine (NEW)

**Mục tiêu**: Chuyển Go CDC Worker từ hard-coded struct sang generic processor sử dụng mapping rules từ CMS.

**Yêu cầu**:

#### 7.1 Rule-based Mapping
- **Load Mapping Rules**:
  - Khi startup, load tất cả rules từ `cdc_mapping_rules` vào memory cache (Redis hoặc in-memory map)
  - Subscribe NATS topic `schema.config.reload` để reload rules khi có changes
- **Dynamic Query Builder**:
  - Thay vì fixed `INSERT ... VALUES (...)`, build query động dựa trên mapping rules
  - Example:
    ```go
    func buildUpsertQuery(tableName string, rules []MappingRule, data map[string]interface{}) string {
        // Loop through rules
        // Extract data from JSON using source_field
        // Build INSERT ... ON CONFLICT ... UPDATE query
    }
    ```
- **Type Conversion**: Convert JSON types → PostgreSQL types theo `data_type` trong rule
  - Handle special types: DECIMAL, TIMESTAMP, JSONB

#### 7.2 Enrichment Pipeline
- Fields marked với `is_enriched = true` trong mapping rules sẽ đi qua enrichment logic
- Enrichment functions có thể:
  - Calculate derived fields (balance_after = balance_before + amount)
  - Lookup related data từ cache/DB
  - Validate business rules

#### 7.3 Config Reload (Zero Downtime)
- Khi nhận event `schema.config.reload`, reload mapping rules từ DB
- KHÔNG cần restart service
- Latency < 5 giây

**Output**:
- CDC Worker có thể handle bất kỳ table/field nào mà không cần code changes
- Thêm field mới = chỉ cần approve trên CMS

---

### Task 8: Migration Automation & CI/CD Integration (NEW)

**Mục tiêu**: Tự động hóa việc ALTER TABLE và trigger Airbyte refresh khi có schema changes.

**Yêu cầu**:

#### 8.1 Migration Script Generator
- When CMS approves schema change:
  - Generate SQL migration file: `ALTER TABLE {table} ADD COLUMN {column} {type};`
  - Store in migration history table
  - Optionally commit to Git repo (infrastructure as code)

#### 8.2 Airbyte API Integration
- CMS backend gọi Airbyte API để:
  - **Refresh Source Schema**: `POST /v1/sources/{sourceId}/discover_schema`
  - **Update Connection**: Enable field mới trong sync configuration
  - **Trigger Sync**: `POST /v1/connections/{connectionId}/sync` (optional, nếu cần sync immediately)
- Handle API errors gracefully (rollback ALTER TABLE nếu Airbyte fail)

#### 8.3 CI/CD Pipeline Integration
- Trigger migration pipeline khi có Git commit mới
- Run migrations trên PostgreSQL (staging → production)
- Automated testing sau migration:
  - Verify column exists
  - Verify Airbyte can sync
  - Verify Go Worker can map correctly

**Output**:
- Schema changes hoàn toàn automated (approval → ALTER → Airbyte refresh)
- Zero manual SQL execution

---

## 🔄 Workflow Phối hợp (Dev & DevOps)

### Initial Setup
1. **DevOps** bàn giao cho **Dev** danh sách các bảng nào đi qua đường Airbyte, bảng nào đi qua đường Debezium
2. **Dev** triển khai Service Go để "hứng" luồng Debezium với Dynamic Mapping Engine (Task 1 + 7)
3. **Dev** thiết kế PostgreSQL schema mapping với JSONB Landing Zone (Task 2)
4. **Dev** triển khai CMS Service cho schema approval workflow (Task 6)
5. **DevOps** cấu hình Airbyte chạy định kỳ cho các bảng còn lại
6. **Dev** triển khai Event Bridge cho Airbyte-synced tables (Task 3)
7. **Dev** triển khai Schema Drift Detection Module (Task 5)
8. **Dev** triển khai Migration Automation (Task 8)
9. **Dev** triển khai Reconciliation job (Task 4)
10. **Cả hai** cùng thực hiện Integration Test để đảm bảo PostgreSQL luôn là "Single Source of Truth"

### Schema Change Workflow (Automated)
Khi source DB (MySQL/Mongo) thêm field mới:

| Bước | Thành phần | Hành động |
|------|-----------|-----------|
| **1. Detect** | Schema Inspector (Go) | Phát hiện field mới trong CDC event, lưu vào `pending_fields` |
| **2. Alert** | Schema Inspector | Publish NATS event `schema.drift.detected` → CMS Service |
| **3. Fallback** | CDC Worker | Lưu toàn bộ JSON vào `_raw_data` (JSONB) → Zero data loss |
| **4. Review** | CMS UI | DevOps/Dev xem field mới, suggest data type, tên cột |
| **5. Approve** | CMS Backend | Execute `ALTER TABLE ADD COLUMN` trên PostgreSQL |
| **6. Update Rules** | CMS Backend | Insert vào `cdc_mapping_rules` |
| **7. Notify** | CMS Backend | Publish `schema.config.reload` event qua NATS |
| **8. Reload** | CDC Worker | Reload mapping rules từ DB (< 5s), không restart |
| **9. Airbyte Sync** | Migration Orchestrator | Gọi Airbyte API: refresh schema + enable field + trigger sync |
| **10. Extract** | CDC Worker | Bắt đầu extract field mới từ `_raw_data` ra cột riêng |

**Kết quả**: Field mới được thêm vào PostgreSQL hoàn toàn tự động sau khi approve trên CMS, không mất dữ liệu, không cần restart services.

---

## 💡 Lợi ích Mong đợi

### Performance & Cost
- Tối ưu chi phí: Không chạy Debezium cho bảng ít quan trọng
- Giảm latency: Real-time CDC cho critical business data (< 100ms)
- Flexibility: Batch sync cho analytics tables không cần real-time
- Resilience: 2 luồng độc lập, 1 luồng fail không ảnh hưởng luồng kia

### Data Safety
- **Zero Data Loss**: JSONB Landing Zone đảm bảo mọi data được lưu ngay cả khi schema chưa ready
- **Audit Trail**: Mọi schema change đều có log, traceability 100%
- **Rollback Support**: Có thể revert schema changes thông qua migration history

### Developer Experience
- **No Code Changes**: Thêm field mới không cần push code, rebuild, restart
- **Self-Service**: DevOps tự approve schema changes qua CMS UI
- **Config-Driven**: Mapping rules được quản lý tập trung, dễ maintain
- **Fast Iteration**: Schema change approval → live trong < 2 phút

### Operational Excellence
- **Automated Schema Evolution**: Từ detect → approve → ALTER → Airbyte sync hoàn toàn tự động
- **Change Management**: CMS approval workflow đảm bảo kiểm soát chặt chẽ
- **Monitoring**: Schema drift được detect và alert trong < 1 phút

---

## ⚠️ Constraints & Assumptions

- PostgreSQL là **Single Source of Truth** cho downstream consumers
- Debezium và Airbyte không conflict về table ownership (được phân chia rõ ràng)
- NATS JetStream đã được setup sẵn bởi DevOps
- Source databases (MongoDB/MySQL) đã enable CDC (binlog/oplog)
- Go services chạy trên Kubernetes với auto-scaling capability

---

## 📚 References

- Debezium Documentation: https://debezium.io/documentation/
- NATS JetStream Guide: https://docs.nats.io/nats-concepts/jetstream
- Airbyte CDC Connectors: https://docs.airbyte.com/understanding-airbyte/cdc

