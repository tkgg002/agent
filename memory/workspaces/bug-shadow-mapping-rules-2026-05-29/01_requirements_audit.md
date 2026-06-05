# 01 — Requirements (Audit + Fix)

## R1 — Mapping Rules per-binding (Bug 1)
- `/shadow/:id/mappings` của binding B chỉ hiển thị mapping rules thuộc binding B (`mapping_rule_v2.shadow_binding_id = B.id`).
- API `GET /api/mapping-rules` đã có filter `?binding_id=` ✅. **Requirement còn lại**: FE phải xác định ĐÚNG `binding_id` của route hiện tại trước khi gọi API.
- Acceptance:
  - Click row binding `wallet_capsets` → `/shadow/<so_id>/mappings?binding_id=<b1>` → list rules WHERE `shadow_binding_id=b1`.
  - Click row binding `wallet_capsets_1` → `/shadow/<so_id>/mappings?binding_id=<b2>` → list rules WHERE `shadow_binding_id=b2`.
  - Khi DB rule có `shadow_binding_id IS NULL` (legacy/system_default) → KHÔNG leak vào view binding cụ thể. Filter sẽ loại bỏ NULL rule, hoặc hiển thị nhãn "Legacy unscoped" tuỳ quyết định ở `02_plan`.

## R2 — Source Data Type + Status/InShadow split (Bug 2)
- DB: cột `mapping_rule_v2.source_data_type VARCHAR(100)` ĐÃ tồn tại (migration `067_add_mapping_rule_v2_binding_and_source_type.sql`) ✅.
- Scan logic: `centralized-data-service/internal/handler/command_handler.go:1852,1973` ĐÃ ghi `SourceDataType: &sourceType` khi scan ✅.
- BE DTO/repo: ĐÃ plumb `SourceDataType *string` ở `mapping_rule_dto.go`, `mapping_rule_repo_gorm.go`, `domain/mapping/rule.go` ✅.
- **Gap còn lại**:
  - FE column "Data Type source" (UI render `source_data_type`).
  - Status logic split: cột "Status" = `mapping_rule_v2.status` (pending/approved/rejected), cột "In Shadow" = audit khớp shadow schema (probe runtime). Đảm bảo 2 cột render từ 2 nguồn dữ liệu khác nhau, không tái sử dụng cùng field.
- Acceptance:
  - Sau scan binding mới, mỗi row có `source_data_type` non-null trong API response + UI render.
  - 2 cột "Status" và "In Shadow" độc lập: switch status không đổi In Shadow và ngược lại.

## R3 — Hide Preview + Backfill action (Bug 3)
- FE `cdc-cms-web/src/pages/MappingFieldsPage.tsx` cột Action có 2 button "Preview" + "Backfill".
- Yêu cầu: ẩn hiển thị (comment JSX hoặc flag điều kiện), GIỮ NGUYÊN hàm handler + API call → dễ revert.
- Acceptance:
  - Build PASS, không error TS unused-import (giữ import nếu logic có thể bật lại).
  - Render 0 Preview/Backfill button. Console no warning.

## R4 — Snapshot V2 binding route lookup (Bug 4)
- Mục tiêu: snapshot v2 cho binding B (vừa active) phải tìm thấy route B trong registry cache → KHÔNG báo `shadow_binding_id=B not in active registry routes`.
- Acceptance:
  - Khi user activate binding B (DB: `shadow_binding.is_active=true`) → trigger snapshot v2 với `binding_id=B` → worker `ResolveSourceRoutes(srcDB, srcColl)` chứa cả binding cũ + B.
  - Khi cache stale (binding vừa insert, signal chưa tới) → snapshot runner pre-flight `ReloadAll(ctx)` đảm bảo load. Nếu vẫn miss → log `err_type=registry_stale` + `markProgressError` rõ ràng (phân biệt với `binding_inactive`).
  - `routeBySourceID[src.ID]` không còn bị overwrite per binding (đang là `map[int64]*ResolvedSourceRoute` → cần đổi sang `map[int64][]*ResolvedSourceRoute` hoặc keyed (sourceID, bindingID)).

## Cross-cut acceptance
- Build PASS cả 3 repo: `cdc-cms-service`, `centralized-data-service`, `cdc-cms-web`.
- Test PASS các test tracked (skip pre-existing fail trên untracked file đã ghi nhận ở workspace `snapshot_v2_multi_binding/05_progress.md`).
- Activity log + SigNoz: thêm `shadow_binding_id` field cho action liên quan (đã có sẵn từ fix snapshot v2 multi-binding).
