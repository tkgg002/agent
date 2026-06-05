# 01_requirements

## Functional
- F-1: Nút "Sửa" chỉ xuất hiện 1 lần/source object trong table — không lặp lại trên từng binding row.
- F-2: Nút "Quét field" disabled nếu binding `Trạng thái table` đang OFF (binding-level is_active = false).
- F-3: Nút "Snapshot" disabled nếu source-level `is_active` = false (chưa bật "Kích hoạt debezium Sync").
- F-4: Nút "Manage Masters" disabled cùng điều kiện F-3.
- F-5: Tooltip hint hiện rõ lý do disabled để user biết phải bật gì.

## Non-functional
- NF-1: FE-only, KHÔNG đụng BE handler.
- NF-2: Min impact: 1 file `TableRegistry.tsx`.
- NF-3: Build pass `npm run build`, không thêm dependency.
- NF-4: KHÔNG break inline Switch ở column "Trạng thái table" (per-binding toggle vẫn route đúng `PATCH /shadow-bindings/:id`).

## Out of scope
- BE cascade behavior của `update_source_object_v2.go` giữ nguyên — đúng semantic source-level.
- Refactor `bindingColumns` tab "Shadow Bindings" — không liên quan bug này.
- Bỏ inline Switch "Trạng thái table" — vẫn cần per-binding control.
