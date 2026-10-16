# 09 Hồ Sơ Giải Pháp Kỹ Thuật: Sửa Lỗi Dropdown "Fields (Shadow)" Trong Modal Tạo Master Mapping Rule Thủ Công

## 1. Bối cảnh & Vấn đề
Trên giao diện CMS tại trang quản lý Master Mapping Rules (`/masters/payment_bills/mappings?binding_id=<id>`), khi mở modal **"Tạo Master Mapping Rule thủ công"** và chuyển sang chế độ **"JSON Subset"** (cả 2 mode: Value và Type/Presence), người dùng gặp lỗi tại mục chọn **"Fields (Shadow)"**:
- Danh sách options bị rỗng hoặc không tải được dữ liệu trường từ shadow.
- Nếu tải được thì bị lỗi duplicate options (trùng key/value), gây crash render hoặc lỗi tương tác chọn tag của Ant Design `<Select mode="tags">`.
- Dropdown không hiển thị đủ các cột của Shadow table và thiếu các trường nested/renamed.
- Chưa có tính năng search/filter trường nhanh khi bảng có nhiều thuộc tính.

## 2. Phân tích Nguyên nhân Gốc rễ (Root Cause)
1. **Lifecycle Hook bị khuyết**: `fetchShadowRules` không được gọi trong `useEffect` khi trang khởi tạo hoặc khi `binding` thay đổi. Khi user click nút "Create Mapping", `fetchShadowRules()` được gọi bất đồng bộ nhưng không được `await`, modal mở ngay lập tức trong khi state `shadowRules` vẫn là mảng rỗng `[]`.
2. **Điều kiện chặn cứng `shadow_binding_id`**: Hàm `fetchShadowRules` có guard: `if (!binding?.shadow_binding_id) return;`. Với các master binding không có hoặc chưa gán `shadow_binding_id` (trường hợp `null`/`undefined`), hàm thoát ngay mà không fetch, bỏ qua hai thông tin định danh sẵn có là `binding.shadow_schema` và `binding.shadow_table`. Đồng thời request thiếu `page_size=200`, chỉ lấy mặc định 50 items.
3. **Lỗi Duplicate Key trong Ant Design Select**: `options={shadowRules.map(r => ({ label: r.source_field, value: r.source_field }))}` không loại bỏ các giá trị trùng lặp. Khi có nhiều mapping rule cùng chung field gốc (ví dụ các trường nested dạng `params.x`, `params.y` đều có `source_field = "params"`), Ant Design Select sinh ra lỗi cảnh báo duplicate key và gãy hành vi chọn/xóa tags.
4. **Không kết hợp `shadowColumns`**: Chưa tận dụng danh sách cột vật lý lấy từ `/api/introspection/shadow-columns/:table` để làm fallback hoặc gợi ý bổ sung cho người dùng khi danh sách rule chưa bao quát hết cột của bảng shadow.
5. **Thiếu sanitize chuỗi field khi submit**: Khi tạo rule subset, `jsonSubsetFields.join(',')` có thể chứa khoảng trắng thừa (`"field1, field2"`), khiến backend GJSON không tìm thấy trường do tính chất phân biệt ký tự chính xác.

## 3. Giải pháp Kỹ thuật Đề xuất (The Single Best Approach)

Sửa đổi file: `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx`

### A. Nâng cấp hàm `fetchShadowRules` & Lifecycle Hook
1. Cho phép `fetchShadowRules` fallback qua `shadow_schema` và `shadow_table` nếu `shadow_binding_id` rỗng hoặc không tìm thấy.
2. Thêm tham số `page_size: 200` để lấy đủ toàn bộ các field.
3. Đưa `fetchShadowRules` vào `useEffect([binding, ...])` để dữ liệu shadow rules luôn sẵn sàng trước khi người dùng nhấn nút mở modal.

```typescript
  const fetchShadowRules = useCallback(async () => {
    if (!binding) return;
    try {
      const params: Record<string, unknown> = { page_size: 200 };
      if (binding.shadow_binding_id) {
        params.binding_id = binding.shadow_binding_id;
      } else if (binding.shadow_table) {
        params.shadow_schema = binding.shadow_schema || 'public';
        params.shadow_table = binding.shadow_table;
      } else {
        return;
      }
      const { data } = await cmsApi.get('/api/v1/mapping-rules', { params });
      setShadowRules(data?.data || []);
    } catch {
      message.error('Không tải được shadow mapping rules.');
    }
  }, [binding]);
```

### B. Xây dựng danh sách options thông minh, deduplicated cho `Fields (Shadow)`
Tạo một computed list `shadowFieldOptions` dùng `useMemo`:
- Tập hợp tất cả các field khả dụng:
  + Từ `shadowRules`: lấy `source_field`, `source_path` (nếu có), và `target_column`.
  + Từ `shadowColumns`: lấy các cột vật lý của bảng shadow (loại trừ các cột hệ thống `_gpay_id`, `_raw_data`, v.v.).
- Deduplicate triệt để bằng `Set<string>`.
- Định dạng option chuẩn cho Ant Design:
  `{ label: field, value: field }`
- Thêm `optionFilterProp="label"`, `showSearch`, `allowClear`, `notFoundContent` cho `<Select mode="tags">`.

```typescript
  const shadowFieldOptions = useMemo(() => {
    const seen = new Set<string>();
    const opts: { label: string; value: string }[] = [];

    const addField = (f?: string | null) => {
      const field = (f || '').trim();
      if (!field || seen.has(field) || isSystemColumn(field)) return;
      seen.add(field);
      opts.push({ label: field, value: field });
    };

    // 1. Thêm từ shadowRules
    shadowRules.forEach((r) => {
      addField(r.source_field);
      if (r.source_path) addField(r.source_path);
      if (r.target_column) addField(r.target_column);
    });

    // 2. Bổ trợ từ shadowColumns (các cột vật lý của bảng shadow)
    shadowColumns.forEach((col) => {
      addField(col);
    });

    return opts.sort((a, b) => a.label.localeCompare(b.label));
  }, [shadowRules, shadowColumns]);
```

### C. Sanitize chuỗi field trước khi submit trong `handleCreateMapping`
Làm sạch các phần tử trong `jsonSubsetFields` (trim khoảng trắng, loại bỏ chuỗi rỗng):
```typescript
      const cleanFields = Array.from(new Set(jsonSubsetFields.map(f => f.trim()).filter(Boolean)));
      if (cleanFields.length === 0) {
        message.warning('Vui lòng nhập/chọn ít nhất 1 field hợp lệ để gộp JSONB');
        return;
      }
      const fieldsList = cleanFields.join(',');
```

### D. Áp dụng `shadowFieldOptions` vào cả 2 chế độ của Modal
- Trong **"JSON Subset"**:
  ```tsx
  <Select
    mode="tags"
    style={{ width: 'calc(100% - 140px)' }}
    placeholder="Chọn hoặc nhập tên field... (vd: user_id, metadata.status)"
    value={jsonSubsetFields}
    onChange={setJsonSubsetFields}
    tokenSeparators={[',']}
    showSearch
    optionFilterProp="label"
    options={shadowFieldOptions}
  />
  ```
- Đồng thời cập nhật `options` trong dropdown chế độ **"Field Đơn / Nested"** (`Shadow Rule (v2)`):
  Lọc bỏ duplicate `r.id` và đảm bảo an toàn null-safety.

## 4. Kế hoạch Kiểm thử & Xác minh (Verification Plan)
- **Kiểm tra biên dịch**: Chạy `npm run build` hoặc `npx tsc --noEmit` trong `cdc-cms-web` xác nhận không có lỗi kiểu.
- **Kiểm tra trực quan**:
  1. Mở modal "Tạo Master Mapping Rule thủ công", chuyển Switch sang "JSON Subset".
  2. Bấm vào dropdown "Fields (Shadow)": hiển thị danh sách các field shadow đầy đủ, không bị rỗng, không có lỗi console duplicate key.
  3. Thử tìm kiếm field bằng cách gõ text -> filter hoạt động mượt mà.
  4. Thử chọn nhiều field gợi ý và gõ thêm field mới bằng bàn phím.
  5. Nhấn "Tạo" -> Master rule được tạo thành công với định dạng chuẩn `@type_presence:{field1,field2}` hoặc `{field1,field2}`.
