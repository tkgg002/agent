# Phân tích Kỹ thuật: Xoá Shadow & Xoá Master

## 1. Cơ chế Cascade Delete
- Một Shadow binding có quan hệ 1-N với Master binding thông qua khoá ngoại `shadow_binding_id` trong bảng `cdc_system.master_binding`.
- Một Master binding có quan hệ 1-N với Mapping rules qua khoá ngoại `master_binding_id` trong bảng `cdc_system.mapping_rule_master`.
- Do đó, khi xoá Shadow binding:
  1. Quét tìm tất cả `master_binding.id` liên kết với `shadow_binding.id`.
  2. Với mỗi `master_binding.id`, gọi `DeleteClonedRules` để dọn dẹp các rule ánh xạ.
  3. Gọi `DeleteMasterBinding` để xoá master binding.
  4. Xoá dòng shadow binding chính.
- Khi xoá Master binding:
  1. Chỉ gọi `DeleteClonedRules` và `DeleteMasterBinding` cho master binding đó.
  2. Bỏ qua shadow binding (vẫn giữ nguyên).

## 2. Tích hợp Audit Log
- Sử dụng middleware `destructive.Audit` chặn các route `DELETE`.
- Middleware này tự động trích xuất trường `reason` từ JSON body của request.
- Để đảm bảo Audit log lưu lại lý do hợp lệ, frontend truyền lý do từ `ConfirmDestructiveModal` vào trong body request của Axios: `{ data: { reason } }`.
