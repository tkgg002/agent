# Context: FixDiscoverMongoStringType

## Sự cố DataType = string
Trong quá trình quét và tự động phát hiện schema từ MongoDB/Debezium:
- Hàm `discoverResolveMongoSampledType` trong file `internal/handler/source/discover_handler_mongo.go` chịu trách nhiệm giải quyết kiểu dữ liệu mẫu (sampled datatype) được trích xuất từ dữ liệu mẫu của MongoDB.
- Trước khi thực hiện tái cấu trúc phân chia ranh giới worker CDC, kiểu dữ liệu trả về từ hàm `InferTypeFromRawData` là kiểu thô (Go/Mongo types như `string`, `float64`, `int64`). Do đó, hàm `discoverResolveMongoSampledType` được thiết kế để ánh xạ các kiểu thô đó.
- Tuy nhiên, sau khi `InferTypeFromRawData` được refactor để trả về trực tiếp các kiểu dữ liệu Postgres hợp lệ (như `"TEXT"`, `"BIGINT"`, `"NUMERIC"`, `"JSONB"`, `"BOOLEAN"`, `"TIMESTAMP"`), hàm `discoverResolveMongoSampledType` vẫn chưa được cập nhật và vẫn tiếp tục tìm kiếm các kiểu dữ liệu thô như `"float64"`, `"int64"`, và mặc định trả về `"string"`.
- Vì thế, nếu có type drift hoặc kiểu dữ liệu không xác định, hoặc tập hợp kiểu lớn hơn 1, nó luôn trả về `"string"`.
- Kiểu dữ liệu `"string"` này được lưu trực tiếp vào trường `data_type` của bảng `cdc_system.mapping_rule_v2`.
- Khi luồng provisioning shadow table hoặc đồng bộ DDL chạy lệnh `alter-column`, `base.IsSafeType` kiểm tra kiểu dữ liệu của cột. Do `"string"` không phải là kiểu dữ liệu Postgres hợp lệ, hệ thống báo lỗi:
  `{"level":"error","msg":"command failed","command":"alter-column","error":"invalid data_type"}`

## Giải pháp khắc phục
1. Thay đổi logic trong hàm `discoverResolveMongoSampledType` của `discover_handler_mongo.go` để giải quyết các kiểu Postgres viết hoa hợp lệ thay vì các kiểu Go/Mongo cũ.
2. Thiết kế cơ chế phân giải kiểu dữ liệu (type resolution) tối ưu, lựa chọn kiểu an toàn nhất (supertypes) như `TEXT`, `NUMERIC`, `JSONB` khi phát hiện nhiều kiểu dữ liệu khác nhau trên cùng một trường mẫu.
3. Đảm bảo hàm này không bao giờ trả về `"string"`, `"float64"` hay `"int64"` (các kiểu không hợp lệ trong PostgreSQL).
4. Bổ sung unit tests kiểm chứng hành vi phân giải kiểu dữ liệu của `discoverResolveMongoSampledType` trong `discover_handler_test.go`.
