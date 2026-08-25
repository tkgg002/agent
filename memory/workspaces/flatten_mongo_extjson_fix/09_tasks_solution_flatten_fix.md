# Solution Profile (Updated): Fix Flatten Flatten 1 Level Only & Unwrap ExtJSON

## 1. Xác nhận Yêu cầu & Bối cảnh
- `flatten.go` chỉ thực hiện vòng lặp explode **ĐÚNG 1 CẤP MẢNG ĐẦU TIÊN** (theo `explode_path`, ví dụ `payments[*]`).
- Các thuộc tính bên trong từng phần tử (như `channelID`, `paymentId`, `amount`, `state`, `fee`) được bóc tách bình thường như document phẳng (`copy_1_to_1`).
- Riêng đối với trường `"_id": { "$oid": "6a7039c870401a4326649f7b" }`:
  - ĐÂY KHÔNG PHẢI LÀ MẢNG để loop tiếp.
  - Đây là **Mongo Extended-JSON Object Wrapper** của 1 giá trị ID duy nhất.
  - Không được lặp đệ quy vào `_id`, mà chỉ cần unwrap `{"$oid": "..."}` thành String ID `"6a7039c870401a4326649f7b"`.

## 2. Giải pháp Kỹ thuật

### A. Tầng `flatten.go`:
Giữ nguyên logic loop 1 cấp mảng duy nhất `elements := rc.ExtractArray(row.Raw, s.ExplodePath)`. Mọi phần tử `elem` trong `elements` sẽ được đưa sang `rc.ExtractColumns(elem, rc.Rules)` như 1 document phẳng đơn lẻ.

### B. Tầng `extractColumns` (`transmuter.go`):
Khi `extractColumns` chạy trên `elem` của 1 phần tử mảng:
1. Trường `path` trong rule có thể là `payments._id`, `payments[*]._id`, `payments._id.$oid` hoặc `_id`.
2. Khi query `gjson.Get(elemStr, path)` không thấy (vì `elemStr` là JSON phần tử con, không còn bọc bởi `payments.`):
   - Chuẩn hóa tách lấy key thực sự trong phần tử con (`_id`).
   - Query `gjson.Get(elemStr, "_id")` trả về object `{"$oid": "6a7039c870401a4326649f7b"}`.
3. Chạy qua `unwrapMongoExtJSON(val)`:
   - Tự động nhận diện `map[string]any{"$oid": "6a7039c870401a4326649f7b"}` và giải nén trực tiếp thành `"6a7039c870401a4326649f7b"`.
   - Không đệ quy loop mảng hay tạo mảng rỗng.

### C. Tầng Discovery / Scan Schema (`scan_service.go`):
Trong `flattenJSONWithTypes`, khi gặp `map[string]interface{}` có dạng Mongo ExtJSON (`$oid`, `$date`,...):
- Không đệ quy lặp sâu vào `$oid`.
- Coi `_id` là `TEXT` (hoặc `TIMESTAMPTZ` cho `$date`), dừng loop tại đây.
