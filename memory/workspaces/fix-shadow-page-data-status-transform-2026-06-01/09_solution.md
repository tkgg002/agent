# 09_solution — Fix /shadow Data Status & Transform

## Fix A — Transform field name alignment
**File**: `data-hub/cdc-cms-web/src/pages/TableRegistry.tsx`
**Component**: `TransformProgress`
**Diff (logic)**:
- State type: `transformed_rows` → `bridged_rows`, `pending_rows` → `pending_bridge`.
- `pct = Math.round((bridged / total_rows) * 100)` với `bridged = status.bridged_rows ?? 0`.
- Tooltip dùng `bridged.toLocaleString()` và `status.total_rows.toLocaleString()`.
- Header comment ghi rõ wire shape thực tế của BE.

## Fix B — Data Status tooltip (informational, không đổi logic)
**File**: cùng file.
**Column**: `Data Status` (dataIndex `sync_status`).
- Thêm map `tooltips`: giải thích từng giá trị (unknown / healthy / drift / source_error).
- `unknown` tooltip: "Reconciliation chưa chạy cho table này. Trigger recon hoặc đợi tier 2/3 schedule."
- Bọc Tag bằng AntD Tooltip để user biết nguyên do "Chưa kiểm".

## Verify
- `npx tsc --noEmit -p tsconfig.app.json` EXIT=0
- `git diff --stat src/pages/TableRegistry.tsx` → 33 LOC (+27/-6) (my edits = 28 LOC, còn lại pre-existing).
- ESLint: 3 errors pre-existing (setState-in-effect line 155, no-empty line 222), không phát sinh từ patch.

## Out-of-scope (đề xuất follow-up nếu user muốn)
- Trigger recon thủ công cho 1 table từ UI (button + POST /api/v1/recon/run).
- Backend test wire-shape của transform-status để chặn regression sau này:
  ```go
  // test/internal/api/transform_status_shape_test.go
  // ensure response contains keys: total_rows, bridged_rows, pending_bridge
  ```
