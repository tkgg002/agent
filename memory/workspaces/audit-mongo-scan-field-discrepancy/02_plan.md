# 02_plan.md - Kế hoạch Kiểm toán & Phản tỉnh (Audit Plan)

## 1. Mục tiêu
Thực hiện audit chuyên sâu, đối soát toàn bộ chuỗi xử lý từ khi kích hoạt lệnh `Scan Fields` đến khi trả kết quả về bảng Review Mapping trên giao diện CDC Web, chỉ ra chính xác từng nguyên nhân gây ra sự sai lệch về trường dữ liệu và kiểu dữ liệu.

## 2. Các bước thực hiện
- [x] **Bước 1: Tra cứu & Đối chiếu Mã nguồn Thực tế (Code-First Tracing)**:
  - Kiểm tra `discover_handler.go` (`ScanFieldsDebezium`), `discover_handler_mongo.go` (`scanFieldsMongoSource`), `discover_handler_utils.go` (`processDiscoveryRows`).
  - Kiểm tra `source_router.go` (`InferTypeFromRawData`), `discovery_utils.go` (`bsonToPGType`, `inferMongoCols`).
  - Kiểm tra `mongo_introspection.go` (`IntrospectCollectionDiagnose`, `IntrospectCollection`).
  - Kiểm tra `scan_service.go` (`ScanRawData`).
  - Kiểm tra Frontend `MappingFieldsPage.tsx`.
- [x] **Bước 2: Phân tích Gốc rễ & Phản biện (Root Cause & Critical Thinking Audit)**:
  - Phân tích hiện tượng 1: Tại sao BSON Extended JSON (`$date`, `$oid`) bị ép kiểu thành `JSONB` trong `InferTypeFromRawData` và `processDiscoveryRows`.
  - Phân tích hiện tượng 2: Tại sao kết quả scan bị lệch schema (có `requestData`, `responseData` nhưng thiếu `bankTransactionId`, `logs`) do cơ chế lấy mẫu dữ liệu (Natural Order Find Limit 10 / Shadow Sample 100).
  - Phân tích hiện tượng 3: Array fields (`logs`) trong kiến trúc Flatten vs Explode.
- [x] **Bước 3: Tổng hợp Báo cáo Kiểm toán (Audit Report)**:
  - Tạo tài liệu `audit_report_mongo_scan_field.md` và `13_analysis_mongo_scan_field.md` trong workspace.
- [x] **Bước 4: Đề xuất Phương án Tối ưu (The Single Best Approach)**:
  - Thiết kế giải pháp nâng cấp Type Inferrer nhận diện BSON Wrapper (`$oid`, `$date`, `$numberDecimal`, `$numberLong`).
  - Nâng cấp Sampling Engine đa tầng (Sample latest + Sample distinct `requestType` hoặc tăng sample pool).
