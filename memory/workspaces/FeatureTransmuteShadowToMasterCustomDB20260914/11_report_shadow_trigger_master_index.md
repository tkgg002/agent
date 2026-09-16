# 11_report_shadow_trigger_master_index.md
# BÁO CÁO TỔNG HỢP KỸ THUẬT VÀ THAY ĐỔI MÃ NGUỒN (CHANGE LOG & IMPLEMENTATION REPORT)

---

## I. TỔNG QUAN THỰC THI (EXECUTIVE SUMMARY)
Hệ thống Agent Role **Muscle (Chief Engineer)** đã hoàn tất triển khai toàn trình bộ giải pháp kỹ thuật theo kế hoạch đã được phê duyệt bởi User:
1. **Phần 1: Phục hồi Sonyflake Trigger Shadow**: Đảm bảo 100% Shadow Table (kể cả khi tạo tự động qua CDC event V1 hay bấm nút `Create` qua NATS command `cdc.cmd.create-default-columns`) đều sở hữu Sequence `fencing_token_seq` và Trigger BEFORE INSERT `trg_<tableName>_sonyflake_fallback`, ngăn chặn triệt để hiện tượng `_gpay_id = NULL` khi drop/create table hoặc snapshot ingest.
2. **Phần 2: Master Index Multi-Connection Suite**: Thông suốt tham số `connectionKey` qua 3 tầng (Frontend `cdc-cms-web` -> CMS API `cdc-cms-service` -> CDS Worker Engine `centralized-data-service`). Phân giải chính xác target database qua dynamic connection pool kèm fallback tra cứu `cdc_system.master_binding`. Tháo bỏ khối chặn cứng `if (plane === 'shadow')` trên UI, mở rộng đề xuất index cho Master Table (bổ sung `_updated_at`, loại bỏ `_source_id`).

---

## II. DANH SÁCH CÁC TỆP TIN THAY ĐỔI & CHI TIẾT DÒNG CODE

### 1. `centralized-data-service/internal/service/shadow/schema_adapter.go`
- **Số dòng thay đổi**: +73 dòng.
- **Chi tiết thay đổi**:
  * Thêm method `EnsureSonyflakeTrigger(ctx context.Context, schemaName, tableName string) error`: tạo sequence `fencing_token_seq`, function `gen_sonyflake_id()`, trigger function `tg_sonyflake_fallback()`, và trigger BEFORE INSERT `trg_<tableName>_sonyflake_fallback` với đầy đủ sanitize identifier bằng `sqlutil.QuoteIdent`.
  * Tại cuối hàm `EnsureCDCColumnsInSchema` (dòng 1030): Gọi `EnsureSonyflakeTrigger` kèm kiểm tra lỗi nghiêm ngặt (`err != nil`) thay vì nuốt lỗi.
  * Tại hàm `createShadowTableV1WithCols` (dòng 316): Gọi `EnsureSonyflakeTrigger` bảo vệ luồng tự động tạo bảng CDC V1.

### 2. `centralized-data-service/internal/service/governance/index_manager.go`
- **Số dòng thay đổi**: +24 dòng, sửa 2 dòng.
- **Chi tiết thay đổi**:
  * Cập nhật signature `GetRecommendations`: nhận thêm tham số `plane string`.
  * Điều kiện hóa đề xuất index `_source_id`: chỉ đề xuất khi `!isMaster` (vì Master Table không có cột `_source_id`).
  * Giữ nguyên partial index `_deleted = true` cho cả Shadow và Master Table.
  * Bổ sung đề xuất index `_updated_at` cho Master Table (`isMaster`) nếu chưa có index để tối ưu hóa delta scan.

### 3. `centralized-data-service/internal/handler/governance/index_handler.go`
- **Số dòng thay đổi**: +42 dòng, sửa 15 dòng.
- **Chi tiết thay đổi**:
  * Thêm trường `ConnectionKey string \`json:"connection_key"\`` vào struct payload của cả 3 handlers: `HandleIntrospectIndexes`, `HandleCreateIndex`, `HandleDropIndex`.
  * Thêm helper method `resolveDB(ctx context.Context, plane, schema, table, connKey string) (*gorm.DB, error)`: phân giải target database qua dynamic connection pool `h.connMgr.GetMasterDB(ctx, key)` với cơ chế fallback tra cứu `cdc_system.master_binding` theo `table` / `schema.table`.
  * Thay thế hardcode `"default"` trong 3 handlers bằng lời gọi `h.resolveDB(...)`.
  * Truyền `payload.Plane` vào `h.indexManager.GetRecommendations`.

### 4. `centralized-data-service/internal/service/governance/index_manager_test.go`
- **Số dòng thay đổi**: +20 dòng, sửa 2 dòng.
- **Chi tiết thay đổi**:
  * Cập nhật các lời gọi `GetRecommendations` truyền thêm tham số `plane` ("shadow").
  * Bổ sung Case C: Kiểm thử Master plane recommendations xác nhận `_source_id` không bị đề xuất và `_updated_at` được đề xuất chính xác khi chưa tồn tại.

### 5. `cdc-cms-service/internal/api/system/introspection_handler.go`
- **Số dòng thay đổi**: +9 dòng, sửa 3 dòng.
- **Chi tiết thay đổi**:
  * Trong `ListIndexes`: Đọc query param `connectionKey := c.Query("connection_key")`, đóng gói `"connection_key": connectionKey` vào NATS payload `cdc.cmd.introspect-indexes`.
  * Trong `CreateIndex`: Thêm `ConnectionKey string \`json:"connection_key"\`` vào body struct, đóng gói `"connection_key": body.ConnectionKey` vào NATS payload `cdc.cmd.create-index`.
  * Trong `DropIndex`: Đọc query param `connectionKey := c.Query("connection_key")`, đóng gói `"connection_key": connectionKey` vào NATS payload `cdc.cmd.drop-index`.

### 6. `cdc-cms-web/src/components/TableIndexManager.tsx`
- **Số dòng thay đổi**: +25 dòng, sửa 10 dòng.
- **Chi tiết thay đổi**:
  * Thêm `connectionKey?: string` vào interface `TableIndexManagerProps`.
  * Truyền `connection_key: connectionKey` trong params của `fetchIndexes` và `handleDropIndex`.
  * Truyền `connection_key: connectionKey` trong request body của `handleCreateIndex` và `handleCreateRecommended`.
  * Cập nhật dependency array của `useEffect` thành `[schema, table, plane, connectionKey]`.
  * Tháo bỏ khối chặn cứng `if (plane === 'shadow')`, mở rộng logic đề xuất index:
    - `_deleted = true` (partial): đề xuất cho cả Shadow và Master.
    - `_source_ts`: đề xuất cho cả Shadow và Master.
    - `_updated_at`: đề xuất riêng cho Master Table.
    - `_source_id`: đề xuất riêng cho Shadow Table.
    - `backendRecommendations`: tích hợp cho cả Shadow và Master Table.

### 7. `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx`
- **Số dòng thay đổi**: +2 dòng.
- **Chi tiết thay đổi**:
  * Thêm `master_connection_code?: string;` vào interface `MasterBinding`.
  * Truyền `connectionKey={binding.master_connection_code}` vào component `<TableIndexManager />`.

---

## III. TỔNG KẾT VÀ ĐỐI SOÁT GOVERNANCE
- **Tổng số tệp tin mã nguồn thay đổi**: 7 tệp tin (4 backend CDS, 1 backend CMS, 2 frontend Web).
- **Tuân thủ bài học kinh nghiệm**:
  * `#system-wide-audit-blindspot`: Thông suốt multi-connection cho toàn bộ flow Index từ Web UI đến Worker Engine, không bỏ sót subsystem quản trị index.
  * `#accidental-variable-obliteration`: Phạm vi thay thế minimal chunk, đối chiếu 1-1 từng biến, không làm mất bất kỳ biến lân cận nào.
  * `#simplicity-first`: Tái sử dụng `ConnectionManager` và trigger pattern PostgreSQL có sẵn, không over-engineer.
  * `#cheat-db-violation`: Hoàn toàn giải quyết trong mã nguồn, không thao tác sửa data DB trực tiếp.
