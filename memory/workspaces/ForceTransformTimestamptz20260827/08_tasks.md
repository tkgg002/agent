# 08 Tasks — Force Transform + Timestamp Bug Fix

## Task 1 — BUG: FE transform in-progress sai trên child binding table

| # | Task | File | Status |
|---|------|------|--------|
| 1.1 | Sửa render child row: `activeJobId={activeTransformJobs[r.id] \|\| null}` | `TableRegistry.tsx` | ✅ |
| 1.2 | Đảm bảo không còn key collision giữa parent row và child row | `TableRegistry.tsx` | ✅ |

---

## Task 2 — FEATURE: Force Transform Mode (BE)

| # | Task | File | Status |
|---|------|------|--------|
| 2.1 | Thêm `Force bool`, `ForceFields []string` vào `BatchTransformPayload` | `batch_transform_handler.go` | ✅ |
| 2.2 | Validate: `force=true` + `force_fields` rỗng → return error sớm | `batch_transform_handler.go` | ✅ |
| 2.3 | Build SET clause: khi `force=true`, chỉ lấy rule có `TargetColumn ∈ ForceFields` | `batch_transform_handler.go` | ✅ |
| 2.4 | Build WHERE clause: khi `force=true`, không append `IS NULL`; `whereExpr = "TRUE"` | `batch_transform_handler.go` | ✅ |
| 2.5 | Khi `force=false`: giữ nguyên toàn bộ logic cũ (không đổi behavior) | `batch_transform_handler.go` | ✅ |

---

## Task 3 — FIX: BuildCastExpr ELSE branch cho timestamptz

| # | Task | File | Status |
|---|------|------|--------|
| 3.1 | Tách case `"timestamptz"`, `"timestamp with time zone"` thành nhánh riêng | `mapping_utils.go` | ✅ |
| 3.2 | Sửa ELSE branch của nhánh `timestamptz`: `::TIMESTAMP` → `::TIMESTAMPTZ` | `mapping_utils.go` | ✅ |
| 3.3 | Giữ nguyên nhánh `"timestamp"`, `"timestamp without time zone"` (không đổi) | `mapping_utils.go` | ✅ |

---

## Task 4 — BE HTTP handler: nhận force params

| # | Task | File | Status |
|---|------|------|--------|
| 4.1 | Định vị handler `POST /api/v1/source-objects/:id/transform` (`TransformV2`) | `cdc-cms-service/.../source_object_actions_handler.go` | ✅ |
| 4.2 | Đọc `force` + `force_fields` từ request body → forward vào `BatchTransformPayload` | `cdc-cms-service/.../source_object_actions_handler.go` | ✅ |

---

## Task 5 — FE: Trigger force transform từ Transform Modal (TableRegistry.tsx)

| # | Task | File | Status |
|---|------|------|--------|
| 5.1 | Giữ `MappingFieldsPage.tsx` thuần túy cấu hình mapping rules (không đặt nút trigger) | `MappingFieldsPage.tsx` | ✅ |
| 5.2 | Xây dựng `TransformModal` component trong `TableRegistry.tsx` với Switch Force mode & Multi-select field | `TableRegistry.tsx` | ✅ |
| 5.3 | Gắn Transform button & Modal cho cả parent table và child binding table | `TableRegistry.tsx` | ✅ |
| 5.4 | Build và typecheck toàn bộ frontend với `npm run build` | `cdc-cms-web` | ✅ |
