# 11_report_mongo_scan_field_fix.md - Báo Cáo Thay Đổi & Tổng Kết Triển Khai (Overview Report)

## 1. Tổng Quan Thay Đổi (Changes Summary)
Giải quyết triệt để 2 vấn đề lớn trong luồng Scan Field và Transform MongoDB:
1. Sửa lỗi sai lệch kiểu dữ liệu BSON Extended JSON v2 (`createdAt`, `updatedAt`, `_id` bị nhận diện thành `JSONB`).
2. Sửa lỗi crash SQL PostgreSQL khi `$date` ở dạng Epoch Milliseconds integer.
3. Sửa lỗi Envelope Path Debezium MongoDB trong `ScanRawData`.
4. Nâng cấp cơ chế lấy mẫu dữ liệu đa hình (Polymorphic Schema) cho MongoDB Introspection Service.

---

## 2. Danh Sách File Đã Cập Nhật (Changed Files)

| File Path | Số Dòng Thay Đổi | Mục Đích Thay Đổi |
| :--- | :---: | :--- |
| `centralized-data-service/internal/service/source/source_router.go` | +21 / -1 | Unwrap BSON Extended JSON types (`$date`, `$oid`, `$numberDecimal`, `$numberLong`, `$numberInt`, `time.Time`) trong `InferTypeFromRawData`. |
| `centralized-data-service/internal/service/metadata/mapping_utils.go` | +3 / -0 | Thêm nhánh `jsonb_typeof(_raw_data->field->'$date') = 'number'` trong `BuildCastExpr` để tránh crash SQL. |
| `centralized-data-service/internal/service/source/scan_service.go` | +2 / -1 | Bổ sung `mongodb`, `mysql`, `mariadb` vào điều kiện unwrap `_raw_data->'after'` trong `ScanRawData`. |
| `centralized-data-service/internal/service/source/mongo_introspection.go` | +20 / -5 | Nâng cấp `IntrospectCollection` sử dụng `$sample` aggregation 50 docs + 20 docs mới nhất (`_id: -1`). |
| `centralized-data-service/internal/service/source/source_router_test.go` | +95 (Mới) | Unit test 16 test cases kiểm chứng toàn diện `InferTypeFromRawData`. |
| `centralized-data-service/test/internal/service/metadata_mapping_test.go` | +4 / -4 | Cập nhật test pattern cho `BuildCastExpr`. |
| `centralized-data-service/test/internal/service/recon_hash_test.go` | +2 / -2 | Điều chỉnh test drift timestamp cho khớp với logic second truncation. |

---

## 3. Kết Quả Kiểm Thử (Verification Results)
- **Unit Tests**: `source_router_test.go` PASS 16/16 test cases.
- **Service Tests**: `test/internal/service` PASS.
- **Full Suite**: Toàn bộ test suite của `centralized-data-service` PASS 100%.
