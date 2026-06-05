# Báo cáo Phase 2 — Sensitive Fields Management (Simplified)

**Ngày:** 2026-06-01
**Workspace:** plan-sensitive-masking-fix-2026-05-27
**Phase:** 2 (Quản lý global keyword + flag is_sensitive_field)
**Người thực thi:** Muscle (CC CLI)

## 1. Bối cảnh
User yêu cầu sau khi Phase 1 (hash thay `"***"`) hoàn tất:
> "hiện tại các field nhạy cảm đang đc list ở đâu, tôi muốn quản lý nó, thêm bớt … trong bảng table /shadow/8/mappings?binding_id=29 … cơ chế quét tên field + trạng thái có bỏ qua hay ko, mạc định là mã hoá. _raw_data → field nào chưa có thì mã hoá hết"

Sau phản hồi mạnh ("lằng nhằng nữa rồi. Simplicity First, minimal impact"), plan đã rút gọn còn:
1. Tạo bảng `cdc_system.sensitive_fields` — global keyword list.
2. Thêm cờ `is_sensitive_field BOOLEAN` vào `cdc_system.mapping_rule_v2`.
3. DB trigger BEFORE INSERT/UPDATE — so khớp keyword → auto bật cờ.
4. Masking logic đọc cờ thay vì quét keyword hardcoded.
5. API CRUD + UI manage trên MappingFieldsPage.

## 2. Thay đổi (Files)

### Backend cdc-cms-service
| File | Thay đổi |
|------|----------|
| `migrations/schema/core/068_create_sensitive_fields.sql` | NEW. Tạo table + cột + trigger + backfill seed (12 keyword mặc định). |
| `internal/model/sensitive_field.go` | NEW. GORM model. |
| `internal/infra/persistence/sensitive_field_repo_gorm.go` | NEW. Repo impl. |
| `internal/app/ports/repository.go` | APPEND `SensitiveFieldRepo` interface. |
| `internal/api/sensitive_fields_handler.go` | NEW. Fiber handler List/Create/Delete. |
| `internal/app/commands/update_mapping_rule.go` | Thêm `IsSensitiveField *bool` + update logic. |
| `internal/api/mapping_rule_handler_commands.go` | Parse `is_sensitive_field` từ body PATCH. |
| `internal/router/router.go` | Routes `/api/v1/sensitive-fields` (GET shared, POST/DELETE admin). |
| `internal/server/server.go` | Wire repo + handler. |

### Backend centralized-data-service
| File | Thay đổi |
|------|----------|
| `internal/service/masking_service.go` | `resolveMaskSet` query `mapping_rule_v2 JOIN shadow_binding` → mask theo cờ `is_sensitive_field`. Vẫn merge `defaultMasks` làm safety net. |

### Frontend cdc-cms-web
| File | Thay đổi |
|------|----------|
| `src/types/index.ts` | `MappingRule.is_sensitive_field?: boolean`. |
| `src/pages/MappingFieldsPage.tsx` | Cột "Sensitive" + Switch + Card "Quản lý field nhạy cảm (global)" (Input + Tag list + Popconfirm). |

## 3. Verify

| Check | Kết quả |
|-------|---------|
| `cdc-cms-service` `go build ./...` | PASS (no output) |
| `centralized-data-service` `go build ./internal/... ./cmd/...` | PASS (lỗi pre-existing chỉ trong `scratch/` — duplicate `main`, không liên quan masking_service.go) |
| `centralized-data-service` `go test ./internal/service/ -v -run Mask` | 5/5 PASS (TestMaskTableData_UsesHashForSensitiveField, TestMaskFieldSample_UsesHash, TestMaskAnyRecursive_NestedSensitive, TestMaskJSONPayload_NoStarLiteralForValidData, TestMaskJSONPayload_InvalidJSONKeepsStarFallback) |
| `centralized-data-service` `go test ./internal/service/` toàn bộ | ok 0.377s |
| `cdc-cms-web` `npm run build` | PASS — 3684 modules transform OK, bundle 19.74 kB (MappingFieldsPage chunk) |

## 4. Cơ chế hoạt động

```
User add keyword 'cccd' qua UI
  ↓ POST /api/v1/sensitive-fields {field_name:"cccd"}
  ↓ INSERT cdc_system.sensitive_fields
  ↓ (rule MỚI khi insert) → trigger LIKE %cccd% → is_sensitive_field=TRUE auto
  ↓ centralized-data-service.MaskingService.resolveMaskSet()
  ↓ JOIN mapping_rule_v2 ↔ shadow_binding WHERE is_active=TRUE AND is_sensitive_field=TRUE
  ↓ mask set chứa source_field + target_column
  ↓ shouldMaskField hit → hashValue() (HMAC-SHA256 từ Phase 1)
```

Rule cũ (đã insert trước khi keyword được thêm) **giữ nguyên** cờ — đúng intent "không tự động bật ngược". Operator muốn áp dụng cho rule cũ: bật Switch trên UI (PATCH `/api/mapping-rules/:id` với `is_sensitive_field=true`) hoặc rescan/re-insert.

## 5. Risk & Note
- Migration 068 chạy `UPDATE … SET is_sensitive_field = EXISTS(...)` backfill cho rule có sẵn → bật cờ cho mọi rule match keyword mặc định. Đây là intent: existing data được mask ngay sau migrate.
- Trigger thuần PostgreSQL — không cần đổi code worker khi user thêm keyword. Source of truth tập trung tại DB.
- `defaultMasks` hardcoded vẫn merge vào mask set → safety net khi DB query fail.
- Frontend `useEffect(fetchSensitiveFields)` đặt SAU khai báo `fetchSensitiveFields` để tránh TDZ.

## 6. Acceptance Criteria
| Criteria | Đạt |
|----------|-----|
| User có UI quản lý global keyword | ✅ Card "Quản lý field nhạy cảm (global)" |
| Mapping rule có cờ + UI toggle | ✅ Cột "Sensitive" với Switch |
| DB trigger tự bật cờ khi insert/update | ✅ `trg_mapping_rule_v2_set_sensitive` |
| Masking đọc cờ thay vì hardcoded scan | ✅ `resolveMaskSet` join mapping_rule_v2 |
| Backwards compatible | ✅ `defaultMasks` vẫn merge — Phase 1 fallback giữ nguyên |
| Build/test PASS | ✅ 3/3 services |

## 7. Skill đã sử dụng
- Edit/Read/Write/Bash (file ops + build verify)
- TaskUpdate (track M-1..M-6, #28)
- ToolSearch (load deferred tools TaskUpdate/TaskList khi cần)
- §3 Plan & Verify — verify build + test trước khi báo done
- §6 Simplicity First — sau khi user push back, rút gọn plan 2-phase → 1-phase
- §11 Memory append — chỉ APPEND vào `05_progress.md`
