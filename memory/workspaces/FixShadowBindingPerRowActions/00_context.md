# 00_context — Fix Shadow Binding Per-Row Actions

## Background
Sau khi fix list-display bug (LATERAL LIMIT 1 → LEFT JOIN trực tiếp ở
`source_object_read_repo_gorm.go`), 1 source object có N shadow_binding
hiện đúng N row trên UI. Tuy nhiên các per-row action vẫn được code với
giả định "1 source = 1 row":

1. **Switch is_active (Bug 1 — user gọi "lặp")**:
   - rowKey="object_code" trùng cho N row → React reconciliation lẫn.
   - Cột `is_active` đọc `so.is_active` (1 giá trị cho cả N row).
   - PATCH `/api/v1/source-objects/:id` cascade `is_active` xuống TẤT
     CẢ binding (`update_source_object_v2.go:122-135`).
   - Hệ quả: nhấn 1 Switch → backend đổi cả 2 binding → list refetch
     → 2 row đều flip cùng lúc.

2. **Scan-fields trỏ sai binding (Bug 2)**:
   - User nhấn scan-fields ở row `sd_export_jobs_dev_1` (sb.id=4)
     nhưng activity log ghi `sd_export_jobs_dev` (sb.id=1).
   - Lý do: handler `ScanFieldsV2` resolve dispatch scope chỉ qua
     `source_object_id`. Backend dispatchScopeQuery (LEFT JOIN sb với
     ORDER BY sb.updated_at DESC NULLS LAST) trả "binding gần nhất"
     — không phải binding user chỉ định.

Cùng pattern lỗi áp dụng cho mọi per-row action:
`CreateDefaultColumnsV2`, `ScanFieldsV2`, `StandardizeV2`,
`DetectTimestampFieldV2`, snapshot, transform, ...

## DB state (verify trên gpay-postgres-cdc)
- so.id=1 (object_code=src_mongodb_goopay_dev_centrallized_export_service_export_jobs)
  - sb.id=1 (sd_export_jobs_dev, is_active=t, ddl=created)
  - sb.id=4 (sd_export_jobs_dev_1, is_active=f, ddl=pending)

## Code locations đã đọc
- `cdc-cms-service/internal/api/source_object_actions_handler.go:47`
  `resolveDispatchScopeBySourceObjectID` (6 callsite: 5 dispatch + 1 read)
- `cdc-cms-service/internal/infra/persistence/bridge_status_repo_gorm.go:76`
  `dispatchScopeQuery` — LEFT JOIN sb ORDER BY sb.updated_at DESC LIMIT 2
- `cdc-cms-service/internal/app/commands/update_source_object_v2.go:122-135`
  Cascade is_active xuống all sb của source.
- `cdc-cms-service/internal/router/router.go:361-365` per-row admin routes.
- `cdc-cms-web/src/pages/TableRegistry.tsx:670-672` Switch render.
  `:825-839` Table rowKey="object_code". `:382-408` updateEntry calls
  PATCH /api/v1/source-objects/:sourceObjectId.

## CLAUDE.md guardrails
- Minimal impact, không over-engineer.
- Verify-before-done: build + test + DB SQL + UI manual click.
- Đối với migration semantic, GIỮ behavior cũ cho row v2_source_only
  (không có binding) — chỉ thay đổi cho row có shadow_binding_id.
