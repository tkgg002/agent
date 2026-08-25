# 05_progress.md — Audit Log (Append ONLY)

## [2026-08-24T13:43] [Agent:Brain] PHÁT HIỆN BUG NGHIÊM TRỌNG

### Trigger
User phản ánh: RunNow `bank_requests` báo `rows_updated=2000` nhưng Master table đúng rỗng. Data thực sự ghi vào `bank_requests` ở schema PostgreSQL khác — sai DB hoàn toàn.

### Root Cause Confirmed (Code-first audit)
File: `centralized-data-service/internal/service/source/connection_manager.go`

```go
// Line 51-53 — key BỊ DISCARD HOÀN TOÀN
func (m *ConnectionManager) GetMasterDB(ctx context.Context, key string) (*gorm.DB, error) {
    _ = ctx
    return m.reg.GetDB(database.RoleDestination) // HARDCODE 1 DB duy nhất
}

// Line 46-48 — key BỊ DISCARD HOÀN TOÀN  
func (m *ConnectionManager) GetShadowDB(ctx context.Context, key string) (*gorm.DB, error) {
    _ = ctx
    return m.reg.GetDB(database.RoleShadow) // HARDCODE 1 DB duy nhất
}
```

### Impact
- Mọi `master_connection_key` (dù là "bidv", "goopay", "default"...) → write vào CÙNG 1 DB `RoleDestination`
- `RoleDestination` = config `masterDb` = `CDS_MASTER_DB_*` env
- Nếu `CDS_MASTER_DB_*` trỏ sai DB → toàn bộ transmute ghi nhầm
- `rows_updated` có thể dương nhưng data vào sai bảng ở sai DB

### Status
- [x] Bug phát hiện & ghi lesson
- [x] Tạo workspace documents đầy đủ
- [x] Lập plan fix schema-qualified master table
- [x] Triển khai Fix Round 1 (8 files/functions)
- [x] Tiến hành QC gắt gao & phát hiện 3 gap logic (Null safety + API DTO)
- [x] Lập báo cáo 11_report_audit_qc.md
- [x] Ghi nhận bài học chống Cheat DB vào lessons.md

## [2026-08-24T14:20] [Agent:Brain] HOÀN TẤT TRIỂN KHAI ROUND 2 & FULL DOC SET
- Đã khởi tạo đầy đủ 18 file tài liệu chuẩn Governance trong thư mục Workspace.
- Đã sửa triệt để 4 task Round 2:
  1. `transmute_schedule_handler.go`: Thêm `MasterSchema` vào DTO và validate controller.
  2. `transmute_scheduler.go`: Nối chuỗi FQN bằng `COALESCE(NULLIF(mb.master_schema, ''), 'public')`.
  3. `master_binding_repo.go`: Cập nhật 2 method query với `COALESCE` NULL-safe.
  4. `transmute_schedule_repository_gorm.go`: Sửa `GetHeaderByID` và `Save()` NULL-safe.
- Verify: Compile sạch `go build ./internal/... ./cmd/...` trên cả 2 repository (Exit Code 0).

## [2026-08-24T14:32] [Agent:Brain] HOÀN TẤT BÁO CÁO AUDIT & PHẢN BIỆN CHUYÊN SÂU
- Đã thực hiện kiểm toán line-by-line toàn bộ 9 file sửa đổi.
- Bổ sung NULL-safe guard cho `loadMaster()` trong `transmuter.go` và `loadBinding()` trong `master_ddl_generator.go`.
- Chạy toàn bộ Unit Test Suites trên cả 2 repository: PASS 100%.
- Báo cáo chi tiết: `agent/memory/workspaces/fix-getmasterdb-connection-key/audit_report_adversarial_qc.md`.

## [2026-08-24T15:12] [Agent:Brain] FIX SỰ CỐ PAYLOAD POST /api/v1/schedules THIẾU MASTER_SCHEMA
- **Vấn đề:** Khi UI/Client gửi payload `{master_table: "bank_requests"}` không có `master_schema`, logic `Save()` trước đó ép `master_schema` thành `'public'` làm rớt các binding có schema khác như `master_bidv_connector_service`.
- **Khắc phục:**
  1. `transmute_schedule_repository_gorm.go`: Nếu `masterSchema == ""`, tìm kiếm theo `master_table` để tự động khớp binding duy nhất; nếu `masterSchema != ""`, tìm theo cả hai.
  2. `transmute_schedule_handler.go`: Tự động parse FQN nếu `req.MasterTable` chứa `schema.table` (ví dụ `master_bidv_connector_service.bank_requests`).
- **Verify:** `go build` & `go test` pass 100% trên cả 2 repository.
