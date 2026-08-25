# Requirements: Fix Lỗi Flatten Mongo Extended-JSON (_id.$oid / $date) Bị Rỗng

## 1. Bối cảnh
Khi xử lý mảng phần tử (Array Explode / Flatten / Transmute) từ dữ liệu nguồn MongoDB (qua Debezium/CDC), các phần tử trong mảng có chứa các đối tượng MongoDB Extended-JSON đặc thù như:
```json
"_id": { "$oid": "6a7039c870401a4326649f7b" }
```
hoặc `$date`, `$numberLong`.

## 2. Hiện trạng & Lỗi
- Tầng **Discovery / Scan (`flattenJSONWithTypes`)**: Coi `"_id"` là 1 sub-map kinh doanh thông thường và tiếp tục loop đệ quy vào các key con có tiền tố `$oid` -> Tạo ra path dạng `payments[*]._id.$oid`.
- Tầng **Mapper / Extract (`MapColumnsFromElement` & `extractColumns`)**:
  1. `MapColumnsFromElement` trong `child_explode.go` trực tiếp lấy `element["_id"]` thu được `map[string]interface{}{"$oid": "..."}` nhưng KHÔNG unwrap Mongo ExtJSON, dẫn đến `convertType` không thể ép về `TEXT` (ra chuỗi dạng `map[$oid:...]` hoặc lỗi rỗng).
  2. `extractColumns` trong `transmuter.go` khi query path `payments._id.$oid` trên 1 element con bị trượt gjson query (và fallback `lastSeg="$oid"` cũng trượt), khiến giá trị trả về bị `null`/rỗng.

## 3. Định nghĩa hoàn thành (DoD)
1. Thêm cơ chế nhận biết Mongo Extended-JSON (`$oid`, `$date`, `$numberLong`, v.v.) trong tầng Scan Service để coi các wrapper này là **Scalar Value** (TEXT/DATETIME), không tiếp tục loop đệ quy sâu vào key có `$`.
2. Tích hợp `unwrapMongoExtJSON` vào `MapColumnsFromElement` trong `child_explode.go` để giải nén tự động `{"$oid": "..."}` -> `"6a7039c870401a4326649f7b"`.
3. Chuẩn hóa path resolver khi `gjson.Get` query các trường trong element mảng (loại bỏ prefix mảng cha nếu query trực tiếp trên mảng con).
4. Unit tests pass 100% cho cả cases Mongo ExtJSON `_id`, `$date`, `$numberLong` trong mảng flatten.
