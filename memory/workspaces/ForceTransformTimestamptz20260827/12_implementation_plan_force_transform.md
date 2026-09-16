# 12 AI Implementation Plan (Archived)

## Kế hoạch thực hiện chi tiết
1. **Phân tích nguyên nhân gốc rễ (Root Cause Analysis):**
   - Bug 1: Key collision trên state FE `activeTransformJobs` do dùng `source_object_id`.
   - Bug 2: `BuildCastExpr` fallback sang `::TIMESTAMP` thay vì `::TIMESTAMPTZ`.
   - Feature: Bổ sung Force mode vào Batch Transform pipeline.
2. **Kế hoạch triển khai (Execution Plan):**
   - Step 1: Fix `mapping_utils.go` (`BuildCastExpr`).
   - Step 2: Update `batch_transform_handler.go` (Payload & Worker logic).
   - Step 3: Update `source_object_actions_handler.go` (CMS API & NATS dispatch).
   - Step 4: Fix `TableRegistry.tsx` (Child table job key).
   - Step 5: Update `MappingFieldsPage.tsx` (UI Force Transform button).
   - Step 6: Viết unit tests và verify toàn bộ test/build suites.
