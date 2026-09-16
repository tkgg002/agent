# 12_implementation_plan_shadow_trigger_master_index.md
# KẾ HOẠCH TRIỂN KHAI CHI TIẾT (IMPLEMENTATION PLAN)

---

## I. MỤC TIÊU
Triển khai trọn vẹn 2 gói tính năng kỹ thuật:
1. Tự động gắn trigger Sonyflake ID trên bảng Shadow khi tạo hoặc chuẩn hóa cột CDC.
2. Nâng cấp Master Index & Recommendation Suite hỗ trợ đa kết nối (Multi-Connection) trên toàn bộ 3 tầng (Web Frontend -> CMS Service -> CDS Worker Engine).

---

## II. DANH SÁCH FILE THAY ĐỔI THEO TỪNG COMPONENT

### Component 1: CDS Worker Engine (`centralized-data-service`)
1. `internal/service/shadow/schema_adapter.go`:
   - [NEW METHOD] `EnsureSonyflakeTrigger(ctx context.Context, schemaName, tableName string) error`
   - [MODIFY] `EnsureCDCColumnsInSchema`: Gọi `EnsureSonyflakeTrigger` với error handling đầy đủ.
   - [MODIFY] `createShadowTableV1WithCols`: Gọi `EnsureSonyflakeTrigger` cho luồng auto-create V1.
2. `internal/handler/governance/index_handler.go`:
   - [NEW METHOD] `resolveDB(ctx context.Context, plane, schema, table, connKey string) (*gorm.DB, error)`
   - [MODIFY] `HandleIntrospectIndexes`: Nhận `connection_key`, lấy DB động qua `resolveDB`, truyền `plane` vào `GetRecommendations`.
   - [MODIFY] `HandleCreateIndex`: Nhận `connection_key`, lấy DB động qua `resolveDB`.
   - [MODIFY] `HandleDropIndex`: Nhận `connection_key`, lấy DB động qua `resolveDB`.
3. `internal/service/governance/index_manager.go`:
   - [MODIFY] `GetRecommendations`: Nhận thêm `plane string`. Không đề xuất `_source_id` cho Master. Bổ sung đề xuất `_updated_at` cho Master.

### Component 2: CMS Backend Service (`cdc-cms-service`)
1. `internal/api/system/introspection_handler.go`:
   - [MODIFY] `ListIndexes`: Đọc query param `connection_key`, đưa vào payload NATS `cdc.cmd.introspect-indexes`.
   - [MODIFY] `CreateIndex`: Nhận `connection_key` từ JSON body, đưa vào payload NATS `cdc.cmd.create-index`.
   - [MODIFY] `DropIndex`: Đọc query param `connection_key`, đưa vào payload NATS `cdc.cmd.drop-index`.

### Component 3: CMS Web Frontend (`cdc-cms-web`)
1. `src/components/TableIndexManager.tsx`:
   - [MODIFY] Bổ sung `connectionKey?: string` vào props interface.
   - [MODIFY] Gửi `connection_key` trong `fetchIndexes`, `handleCreateIndex`, `handleDropIndex`, `handleCreateRecommended`.
   - [MODIFY] Tháo bỏ `if (plane === 'shadow')`. Cung cấp đề xuất chuyên biệt cho Master (`_updated_at`, `_source_ts`, `_deleted`). Giữ `_source_id` chỉ cho Shadow. Tích hợp `backendRecommendations` cho cả hai.
2. `src/pages/MasterMappingFieldsPage.tsx`:
   - [MODIFY] Mở rộng `interface MasterBinding` có `master_connection_code?: string`.
   - [MODIFY] Truyền `connectionKey={binding.master_connection_code}` vào component `<TableIndexManager>`.

---

## III. KẾ HOẠCH KIỂM THỬ VÀ VERIFICATION (TEST PLAN)

### 1. Build Verification
- Chạy `cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go build ./cmd/worker` -> Pass.
- Chạy `cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-service && go build ./cmd/server` -> Pass.
- Chạy `cd /Users/trainguyen/Documents/work/data-hub/cdc-cms-web && npm run build` -> Pass.

### 2. Unit Tests
- Chạy `cd /Users/trainguyen/Documents/work/data-hub/centralized-data-service && go test -v ./internal/handler/shadow/... ./internal/service/governance/...` -> Pass.

### 3. Verification trên Hệ thống Sống (End-to-End Live Check)
- Tạo thử một bảng shadow mới qua API `create-default-columns`: Verify sequence `fencing_token_seq`, trigger `trg_*_sonyflake_fallback` được tạo ngay lập tức.
- Mở trang Master Mapping Fields của Master 2:
  * Verify danh sách index của Master 2 được tải đúng từ container DB Master 2 (không bị 0 index do query nhầm default DB).
  * Verify hiển thị các khuyến nghị index phù hợp: `_updated_at`, `_source_ts`, `_deleted` (không còn đề xuất nhầm `_source_id`).
  * Bấm thử tạo một index khuyến nghị: Verify index được tạo thành công trên DB đích.
