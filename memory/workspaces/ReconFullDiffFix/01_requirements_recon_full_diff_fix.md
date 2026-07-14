# Yêu cầu sửa lỗi đối soát Full Search (Full Diff) không có kết quả

Dự án yêu cầu sửa lỗi chức năng đối soát "Full Search" (Full Diff / Tier 2) không trả về kết quả hoặc trả về kết quả trống/không đúng thực tế.

## 1. Yêu cầu chi tiết
- **Phân tích và khắc phục:** Tìm hiểu nguyên nhân gốc rễ khiến truy vấn dữ liệu từ Shadow DB (Postgres) và Source DB (Postgres/MongoDB) dựa trên khoảng thời gian truyền vào từ UI không khớp nhau hoặc bị lỗi/trả về rỗng.
- **Xử lý kiểu dữ liệu trường Timestamp:**
  - Cột `_source_ts` trong Shadow DB được lưu dưới dạng `BIGINT` (epoch milliseconds).
  - Khi so khớp khoảng thời gian trong truy vấn SQL của `TimeBoundedDiffMissingFromShadow`, hiện tại code đang truyền vào giá trị `time.Time` (`startTime`, `endTime`) trực tiếp, dẫn đến PostgreSQL không so khớp được hoặc gây lỗi truy vấn do không tự động cast từ `TIMESTAMP` sang `BIGINT`.
  - Cần tự động phát hiện kiểu dữ liệu của cột timestamp được resolve (`dstTS` và `srcTS` đối với Postgres source/dest) để cast tham số thời gian phù hợp (hoặc `time.Time` hoặc `int64` epoch milliseconds / epoch seconds).
- **Chất lượng đầu ra (DoD):**
  - Đảm bảo sửa lỗi triệt để cả 3 nơi:
    1. Truy vấn Shadow DB trong `TimeBoundedDiffMissingFromShadow`.
    2. Truy vấn Source DB (nếu là Postgres) trong `listIDsInWindowPostgres`.
    3. Truy vấn Source DB (nếu là Postgres) trong `streamIDsPostgresInTimeRange`.
  - Phải build và test thành công, không gây ảnh hưởng đến các luồng đối soát khác.
