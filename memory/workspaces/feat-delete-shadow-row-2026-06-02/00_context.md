# 00_context — Feature: Xoá 1 row trên page /shadow

## Trigger
User: "tôi muốn xoá 1 http://localhost:5173/shadow"

## Stack & path
- FE: `data-hub/cdc-cms-web` — page `src/pages/TableRegistry.tsx`, route `/shadow`.
- BE: `data-hub/cdc-cms-service` — Fiber router `internal/router/router.go`, handler `internal/api/source_objects_handler.go`.
- DB: cdc_system.{source_object_registry, shadow_binding, mapping_rule_v2, recon_report, bridge_status, worker_schedule}.

## Hiện trạng
- Page liệt kê: V2 source-objects (`GET /api/v1/source-objects`) + shadow-bindings (`GET /api/v1/shadow-bindings`).
- **Chưa có DELETE endpoint** cho source-object hoặc shadow-binding.
- Đã có pattern destructive sẵn: middleware chain JWT → OpsAdmin → Idempotency → Audit (`reason ≥ 10 chars` body). Áp dụng cho `apiGroup.Delete("/v1/system/connectors/:name", deleteHandlers...)` và `admin.Delete("/v1/sensitive-fields/:id", ...)`.

## 3 mức xoá khả thi
| Mức | Behavior | Reversible |
|---|---|---|
| A | Soft: `is_active=false` | Yes |
| B | Hard delete metadata + cascade FK (mapping_rule, recon_report, bridge_status, schedule, binding) | Metadata: no |
| C | B + DROP TABLE shadow vật lý + remove Debezium signal | All data: no |

## Ràng buộc
- "Simplicity First, minimal impact" — không thay đổi config, không cheat DB.
- Pattern existing: destructive chain + FE modal confirm + body `reason ≥ 10 chars` + Idempotency-Key.
- Reference: `pages/SourceConnectors.tsx` (deleteMut) + `pages/MasterRegistry.tsx` (toggle pattern).

## Chờ user chốt
1. Xoá source_object vs shadow_binding vs both.
2. Mức A / B / C.
3. Có drop schema vật lý không.
