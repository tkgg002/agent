# 02_plan — Roadmap & Phương án thực hiện

## 1. Phương án kỹ thuật (The Single Best Approach)

### Backend: `source_repo_gorm.go`
- Khi `updates["is_active"] == true`:
  1. Lấy `sourceObjectID` từ `source_object_registry` bằng `(source_object_name, source_database/source_schema)`.
  2. Truy vấn trực tiếp `shadow_binding` theo `source_object_id = sourceObjectID AND shadow_table = entry.TargetTable AND is_active = true LIMIT 1` để lấy `shadowBindingID`.
  3. Chỉ UPDATE `mapping_rule_v2` với điều kiện `source_object_id = ? AND shadow_binding_id = ? AND status != 'approved'`.

### Frontend: `MappingFieldsPage.tsx`
- Sửa `handleToggleActive`:
  - `const nextStatus = rule.is_active ? 'rejected' : 'approved'`
  - Gửi API PATCH `/api/mapping-rules/${rule.id}` với `{ status: nextStatus }`
  - Cập nhật local state: `setRules(prev => prev.map(r => r.id === rule.id ? { ...r, is_active: !r.is_active, status: nextStatus } : r))`

## 2. Kế hoạch kiểm thử (Verification Plan)
- Run `go build` trên `cdc-cms-service`.
- Run `npm run build` trên `cdc-cms-web`.
- Thực hiện QC Audit gắt gao với tư duy phản biện.
