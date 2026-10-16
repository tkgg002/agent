# 08 Danh Sách Task: Sửa Lỗi Dropdown "Fields (Shadow)"

- [x] Task 1: Cập nhật hàm `fetchShadowRules` trong `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx`
  - Thêm fallback query qua `shadow_schema` + `shadow_table` khi `shadow_binding_id` rỗng hoặc null.
  - Bổ sung `page_size: 200`.
  - Nạp `fetchShadowRules` vào `useEffect` phụ thuộc `binding`.
- [x] Task 2: Tạo computed memo `shadowFieldOptions` trong `MasterMappingFieldsPage.tsx`
  - Hợp nhất field từ `shadowRules` (`source_field`, `source_path`, `target_column`) và `shadowColumns`.
  - Deduplicate và loại bỏ cột hệ thống (`_gpay_id`, `_raw_data`, v.v.).
  - Sắp xếp alphabet cho danh sách gợi ý.
- [x] Task 3: Gắn `shadowFieldOptions` vào Ant Design `<Select mode="tags">` của mục `Fields (Shadow):`
  - Thêm thuộc tính `showSearch`, `optionFilterProp="label"`, `allowClear`.
- [x] Task 4: Sanitize và trim danh sách field trong `handleCreateMapping` trước khi gửi lên API
  - Đảm bảo không dính khoảng trắng gây lỗi trích xuất GJSON ở backend.
- [x] Task 5: Kiểm tra TypeScript compile và xác nhận không có lỗi regression.
