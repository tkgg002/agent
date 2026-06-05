# 00_context — Fix /shadow Data Status & Transform "rỗng"

## Trigger
User báo tại `http://localhost:5173/shadow` 2 cột luôn rỗng:
- **Data Status** (`record.sync_status`)
- **Transform** (component `TransformProgress`)

## Stack
- FE: `data-hub/cdc-cms-web` (Vite + React 19 + AntD v6).
- BE: `data-hub/cdc-cms-service` (Fiber + GORM Postgres).
- Page: `src/pages/TableRegistry.tsx`, route `/shadow`.
- Fetch: `GET /api/v1/source-objects` → list rows
- Transform fetch per row:
  - V2: `GET /api/v1/source-objects/{id}/transform-status`
  - Legacy: `GET /api/v1/source-objects/registry/{id}/transform-status`

## Root cause
### Bug A — Transform luôn rỗng (REAL BUG)
- BE wire shape:
  ```json
  { "total_rows": N, "bridged_rows": M, "pending_bridge": K, ... }
  ```
- FE state shape:
  ```ts
  { total_rows: number, transformed_rows: number, pending_rows: number }
  ```
- `status.transformed_rows = undefined` → `pct = NaN` → AntD `<Progress>` clamp về 0 hoặc render trống.

### Bug B — Data Status hiển thị "Chưa kiểm" (data missing, not UI bug)
- SQL `source_object_read_repo_gorm.go` derive `sync_status` từ `cdc_reconciliation_report`:
  - status=source_error → `source_error`
  - target_table != NULL + diff != 0 → `drift`
  - target_table != NULL → `healthy`
  - ELSE → `unknown`
- Khi recon job chưa chạy cho table → CASE rơi vào `unknown` → FE label `"Chưa kiểm"`.
- Render đúng, không phải bug code. Vấn đề: user không biết "Chưa kiểm" do recon chưa chạy → tưởng rỗng.
