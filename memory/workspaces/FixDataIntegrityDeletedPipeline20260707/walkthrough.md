# Walkthrough: Hiding Deleted/Inactive Pipelines Completed

Chúng ta đã sửa đổi thành công truy vấn báo cáo đối soát ở tầng đọc (`read side`) của `cdc-cms-service` để loại bỏ các pipeline không còn hoạt động (đã xóa connector) khỏi giao diện Data Integrity Dashboard.

## Thay đổi đã thực hiện

### Component: cdc-cms-service (Read Side Persistence)
- **File**: [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go)
- **Giải pháp**: 
  - Thay thế `LEFT JOIN cdc_table_registry` bằng `INNER JOIN cdc_table_registry reg ON ... AND reg.is_active = TRUE` trong biến hằng số truy vấn `listLatestPrimary`.
  - Thay thế tương tự trong biến hằng số truy vấn fallback `listLatestLegacy`.
  - Từ đây, bất kỳ bảng nào không có cấu hình hoạt động trong `cdc_table_registry` sẽ bị loại trực tiếp tại tầng DB và không được trả về phía frontend.

---

## Kết quả kiểm thử & Quy trình

### 1. Unit Tests (cdc-cms-service)
Đã thực thi kiểm thử và pass 100% tất cả các query và API tests liên quan mà không sử dụng cache:
```bash
go test -count=1 ./test/internal/app/queries/...
```
**Kết quả**: `ok  cdc-cms-service/test/internal/app/queries  0.458s` ➔ **PASS 🟢**

### 2. Linter Quy trình (verify_governance.py)
Đã chạy linter trên workspace hiện tại:
```bash
python3 agent/tooling/verify_governance.py --workspace FixDataIntegrityDeletedPipeline20260707
```
**Kết quả**: `⛳ GOVERNANCE AUDIT PASSED 🟢`

---

## Đồng bộ Workspace
Các tài liệu quy trình đã được tạo và lưu giữ đầy đủ tại `/Users/trainguyen/Documents/work/agent/memory/workspaces/FixDataIntegrityDeletedPipeline20260707/`:
- `01_requirements_delete_pipeline.md` (Đặc tả yêu cầu)
- `05_progress_delete_pipeline.md` (Nhật ký tiến độ - 5 entries)
- `08_tasks_delete_pipeline.md` (Checklist công việc - 100% hoàn thành)
- `09_tasks_solution_delete_pipeline.md` (Giải pháp kỹ thuật chi tiết)
- `12_implementation_plan_delete_pipeline.md` (Kế hoạch AI của Muscle)
- `13_analysis_delete_pipeline.md` (Tài liệu phân tích kỹ thuật của Muscle)
- `14_walkthrough_delete_pipeline.md` (Tài liệu walkthrough của Muscle)
- `implementation_plan.md` (Bản đồng bộ plan đã duyệt)
- `walkthrough.md` (Báo cáo kết quả nghiệm thu này)
