# Context: Sửa lỗi transmuter cast bigint khi đồng bộ từ raw_data sang master
2026-06-30

## Hiện tượng
- Khi chạy action Transform thủ công hoặc đồng bộ qua job đối với bảng master, hệ thống ném lỗi:
  `nats_command ERROR: invalid input syntax for type bigint: "306.67" (SQLSTATE 22P02)`
- Lỗi này làm gián đoạn tiến trình Upsert vào bảng master, dẫn đến việc bỏ sót hoặc không cập nhật được dòng dữ liệu.

## Nguyên nhân
1. **Quá trình trích xuất dữ liệu (Extraction)**:
   - Dữ liệu thô (`_raw_data`) chứa giá trị float biểu diễn dưới dạng chuỗi (ví dụ `"306.67"`).
   - Hàm `gjsonValueToGo` hoặc `unwrapMongoExtJSON` đọc ra giá trị này dưới dạng `string` ("306.67").
2. **Quá trình kiểm tra kiểu dữ liệu (Validation)**:
   - Hàm `ValidateValue` của `TypeResolver` sử dụng `strconv.ParseFloat` để kiểm tra kiểu `BIGINT` nên chuỗi `"306.67"` được coi là hợp lệ (vì parse thành float thành công). Do đó validation cho qua.
3. **Quá trình ép kiểu (Coercion)**:
   - Hàm `coerceForColumn` nhận giá trị `"306.67"` và dataType `BIGINT`. Tuy nhiên, hàm này không có logic xử lý ép kiểu cho các kiểu số nguyên (`bigint`, `integer`, `smallint`, v.v.). Nó trả về nguyên bản chuỗi `"306.67"`.
4. **Quá trình ghi dữ liệu (Insertion)**:
   - Chuỗi `"306.67"` được đưa vào placeholder câu lệnh Raw SQL bulk upsert của Postgres, gây lỗi do Postgres không tự động cast chuỗi float sang cột bigint.

## Giải pháp
1. **Ép kiểu thông minh (Smart Coercion)**:
   - Cập nhật `coerceForColumn` để xử lý các kiểu dữ liệu cột đích là số nguyên (`bigint`, `integer`, `smallint`, `int`, `int8`, `int4`, `int2`).
   - Nếu dữ liệu nguồn là float hoặc chuỗi float (như `"306.67"`), tự động chuyển đổi nó sang float64 rồi cast về `int64` (lấy phần nguyên, ví dụ `306`) để Postgres chèn thành công.
2. **Làm giàu thông tin lỗi (Enhanced Error Logging)**:
   - Khi hàm `bulkUpsertMaster` ném lỗi query, thực hiện bóc tách các giá trị nằm trong dấu nháy kép từ error message (ví dụ `"306.67"`).
   - Dò tìm trong batch records xem cột nào đang mang giá trị này.
   - Bổ sung thông tin cột bị lỗi vào error message trả về để hiển thị rõ ràng trên log, giúp người dùng dễ dàng gỡ lỗi (DoD: thêm tên cột lỗi vào log).
