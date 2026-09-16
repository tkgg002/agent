# 01_requirements — Yêu cầu kỹ thuật & Definition of Done

## 1. Yêu cầu Backend (Scope Isolation for Auto-Approve)
- **R1.1**: Khi kích hoạt (`is_active = true`) một table registry (`entry`):
  - Hệ thống chỉ được phép auto-approve các mapping rules thuộc đúng `shadow_binding` của registry entry đó.
  - Tuyệt đối KHÔNG được approve lan sang các `shadow_binding` khác có cùng `source_object_id`.
  - Phải resolve `shadow_binding_id` bằng cặp `(source_object_id, shadow_table)` với `shadow_table = entry.TargetTable`.
  - Nếu không tìm thấy `shadow_binding` hợp lệ (`shadowBindingID <= 0`), phải skip an toàn, không được approve bừa toàn bộ source object.

## 2. Yêu cầu Frontend (Mapping Rule Active Toggle)
- **R2.1**: Khi toggle switch Active trên từng dòng mapping rule:
  - Nếu rule đang active (`is_active = true`) -> gửi `{ status: 'rejected' }` để deactivate.
  - Nếu rule đang inactive (`is_active = false`) -> gửi `{ status: 'approved' }` để activate.
- **R2.2**: Optimistic update trên giao diện phải đồng bộ cả 2 trường `is_active` và `status` trong state `rules`, tránh tình trạng Switch bật xanh nhưng Tag Status vẫn hiển thị `pending` hoặc `rejected`.

## 3. Definition of Done (DoD)
- [x] Backend biên dịch thành công (`go build`).
- [x] Frontend biên dịch và typecheck thành công (`npm run build`).
- [x] Kiểm tra quan hệ schema thực tế (PostgreSQL migrations) để không có column/path giả định.
- [x] Có tài liệu audit phản biện đầy đủ.
