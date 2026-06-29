# Workspace: fe-data-integrity-empty-db-table
**Khởi tạo**: 2026-06-24T15:34:00+07:00 | **Agent**: Brain (Antigravity)
**Project**: data-hub / cdc-cms-web (Frontend React)
**Status**: 🟡 Planning

## Scope
Feature FE: Hiển thị cảnh báo rõ ràng trên trang **Data Integrity** khi bảng DB (shadow/master) **rỗng hoàn toàn** (`full_dest_count = 0` hoặc `full_source_count = 0`), giúp operator phân biệt:
- Bảng có dữ liệu nhưng recon khớp (ok)
- Bảng **thực sự rỗng** — không phải lỗi mà là chưa có data / pipeline chưa hoạt động

## Files liên quan
### Frontend (chính)
- `data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx` — Trang chính
- `data-hub/cdc-cms-web/src/components/ReconPipelineGrid.tsx` — Pipeline grid
- `data-hub/cdc-cms-web/src/hooks/useReconStatus.ts` — Types & hooks

### Backend (tham chiếu — KHÔNG sửa)
- `cdc-cms-service/internal/app/queries/recon/recon_read_models.go` — `LatestReportRow` với `FullSourceCount`, `FullDestCount`
- `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go` — SQL JOIN `reg.full_source_count, reg.full_dest_count`

## Context kỹ thuật
- Backend đã trả `full_source_count` / `full_dest_count` từ `cdc_table_registry` — đây là TỔNG RECORD thực (daily count), không phải window count.
- FE đã có column "Total Source" / "Total Dest" trong tab Tổng quan, nhưng **không có logic cảnh báo** khi cả 2 đều = 0.
- `ReconReport.full_source_count` kiểu `number | null` — nullable, nghĩa là chưa chạy full-count aggregator thì null.
- Status `ok_empty` đã tồn tại trong type `ReconStatus` nhưng chưa có visual treatment đặc biệt nào ngoài tag "Khớp (trống)".
