# Plan: FixDiscoverMongoStringType

Kế hoạch chi tiết khắc phục sự cố DataType = string:

## 1. Cập nhật hàm `discoverResolveMongoSampledType` trong `discover_handler_mongo.go`
- Viết lại hàm `discoverResolveMongoSampledType` để phân giải các kiểu dữ liệu Postgres (như `TEXT`, `JSONB`, `NUMERIC`, `BIGINT`, `INTEGER`, `BOOLEAN`, `TIMESTAMP`, `TIMESTAMPTZ`).
- Nếu tập hợp kiểu dữ liệu rỗng (len == 0), mặc định trả về `"TEXT"`.
- Nếu tập hợp kiểu dữ liệu có 1 phần tử, trả về phần tử đó.
- Nếu có sự pha trộn kiểu dữ liệu (drift):
  - Ưu tiên `"JSONB"` nếu có bất kỳ trường nào là JSONB.
  - Ưu tiên `"TEXT"` nếu có sự pha trộn giữa kiểu chuỗi và các kiểu khác.
  - Nếu chỉ pha trộn các kiểu số, trả về `"NUMERIC"`.
  - Nếu pha trộn kiểu Boolean/Timestamp với các kiểu khác, trả về `"TEXT"`.
  - Không bao giờ trả về `"string"`, `"float64"`, hoặc `"int64"`.

## 2. Bổ sung Unit Test trong `discover_handler_test.go`
- Viết unit test `TestDiscoverResolveMongoSampledType` để kiểm chứng tất cả các trường hợp phân giải kiểu (bao gồm cả trường hợp trống, single type, mixed types, numeric mixtures, và các kiểu không tương thích).

## 3. Chạy kiểm thử tự động
- Chạy `go test -v ./internal/handler/source/...` để đảm bảo code biên dịch và chạy chính xác, không gây lỗi logic cho hệ thống.
