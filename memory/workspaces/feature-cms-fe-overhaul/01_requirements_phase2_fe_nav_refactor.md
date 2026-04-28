# Requirements — Phase 2 FE Nav Refactor

## Mục tiêu

- Bỏ `CDCInternalRegistry` khỏi luồng điều hướng chính.
- Tổ chức lại sidebar theo lifecycle vận hành V2:
  - `Setup`
  - `Operate`
  - `Advanced`
- Đổi các label/operator text mang semantics legacy sang ngôn ngữ V2.

## Phạm vi

- `src/App.tsx`
- `src/pages/SourceToMasterWizard.tsx`
- `src/pages/TableRegistry.tsx`
- `src/pages/SourceConnectors.tsx`
- `src/pages/MasterRegistry.tsx`
- `src/pages/ActivityManager.tsx`

## Điều kiện hoàn thành

1. Không còn menu entry cho `CDCInternalRegistry`.
2. Route cũ `/cdc-internal` không vỡ bookmark, phải redirect về `/registry`.
3. Sidebar phản ánh đúng nhóm chức năng mới.
4. Build FE pass.
5. String `cdc_internal` không còn xuất hiện trong runtime navigation/text chính; nếu còn thì chỉ được nằm trong page legacy chưa route hoặc artifact lịch sử.
