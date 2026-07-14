# Requirements - Sửa lỗi mapping MongoDB Ext-JSON Date/Timestamp vào Postgres timestamp/timestamptz

## 1. Vấn đề hiện tại
- Khi thực hiện đồng bộ hoặc chuyển dữ liệu (shadow sync / transmute) từ MongoDB sang PostgreSQL, một số trường kiểu thời gian (ví dụ: `identityDob` có giá trị `{"$numberLong":"-126403200000"}` hoặc các trường kiểu `$date`) bị ném lỗi:
  `failed to encode args[24]: unable to encode map[string]interface {}{"$numberLong":"-126403200000"} into binary format for timestamptz (OID 1184): cannot find encode plan`
- Lý do: Tầng ép kiểu `CoerceValue` của `SchemaAdapter` chưa xử lý việc ép kiểu cho các cột đích kiểu `timestamp`, `timestamptz`, `date` từ PostgreSQL. Nó chỉ pass qua giá trị gốc (ở dạng map `{"$numberLong": ...}` hoặc `{"$date": ...}`) khiến driver PGX không mã hóa được sang định dạng nhị phân của Postgres.

## 2. Giải pháp yêu cầu
- Bổ sung logic ép kiểu thời gian trong `CoerceValue` cho các kiểu cột đích: `"timestamp with time zone", "timestamp without time zone", "timestamptz", "timestamp", "date"`.
- Chuyển đổi an toàn từ các dạng biểu diễn thời gian:
  1. `time.Time` và `*time.Time`
  2. Chuỗi RFC3339, RFC3339Nano, các định dạng ngày giờ cơ bản khác hoặc chuỗi số.
  3. Kiểu số đại diện cho epoch milliseconds hoặc epoch seconds (dùng ngưỡng giá trị để tự động phân biệt milliseconds và seconds).
  4. Bản đồ `map[string]interface{}` chứa khóa `$date` hoặc `$numberLong` (theo định dạng Extended JSON của MongoDB).
- Thêm kiểm thử đơn vị chi tiết để bảo vệ logic chuyển đổi này.
- Biên dịch lại dự án và chạy bộ kiểm thử để đảm bảo mọi thứ pass 100%.
