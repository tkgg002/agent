# 11 Report: Báo cáo Tổng kết Chuẩn hóa Canonical MongoDB Extended JSON

## 1. Tổng quan thay đổi
Giải quyết triệt để lỗi khi quét/flatten các trường lồng của MongoDB chứa toán tử Canonical Extended JSON (ví dụ `info.fee: {"$numberInt": "100"}`). Trước đây, bộ scan đào sâu vào key `$numberInt` làm sinh ra trường `info.fee.$numberInt`, khiến CMS hiển thị thành `$numberInt` và Transmuter có nguy cơ bind sai kiểu hoặc NULL. 

Sau khi chuẩn hóa:
- Bộ scan nhận diện toàn bộ các wrapper Canonical Mongo ExtJSON, dừng đệ quy tại cấp thuộc tính cha (`info.fee`), gán đúng kiểu dữ liệu PostgreSQL (`INTEGER`, `BIGINT`, `NUMERIC`, `TEXT`, `TIMESTAMPTZ`).
- Tầng Transmuter unwrap ExtJSON đồng bộ cả `$numberInt` và `$numberDouble`.

## 2. Chi tiết các tệp tin đã thay đổi

| STT | Đường dẫn tệp tin | Số dòng thay đổi | Tóm tắt thay đổi |
|:---:|:---|:---:|:---|
| 1 | `internal/service/source/scan_service.go` | ~45 dòng | Bổ sung hàm `isCanonicalMongoExtJSON` và guard chặn đệ quy trong `flattenJSONWithTypes`. |
| 2 | `internal/handler/scan/scan_handler.go` | ~45 dòng | Bổ sung `isCanonicalMongoExtJSON` và guard chặn đệ quy trong `flattenJSONWithTypes` đồng bộ với `scan_service.go`. |
| 3 | `internal/service/source/source_router.go` | ~20 dòng | Mở rộng `InferTypeFromRawData` nhận diện thêm `$numberDouble`, `$timestamp`, `$binary`, v.v. |
| 4 | `internal/service/master/transmute/flatten.go` | ~10 dòng | Bổ sung bóc tách `$numberInt` và `$numberDouble` trong `unwrapExtJSON`. |
| 5 | `internal/service/source/registration_test.go` | ~2 dòng | Bổ sung tham số `topicPrefix` ("cdc.goopay") cho `topicNameFor` sửa lỗi build test cũ. |
| 6 | `internal/handler/scan/scan_array_path_test.go` | ~35 dòng | Viết unit test `TestFlattenJSONWithTypes_CanonicalMongoExtJSON` kiểm thử `info.fee` và `items[*].sub_fee`. |
| 7 | `internal/service/source/scan_service_test.go` | ~30 dòng | Viết unit test `TestScanService_FlattenJSONWithTypes_CanonicalMongoExtJSON`. |
| 8 | `internal/service/source/source_router_test.go` | ~25 dòng | Mở rộng test cases cho 25 biến thể ExtJSON trong `TestInferTypeFromRawData_ExtendedJSON`. |
| 9 | `internal/service/master/transmute/strategy_test.go` | ~25 dòng | Viết unit test `TestUnwrapExtJSON`. |

## 3. Rà soát An toàn & Bảo mật (Security Auto-Check - Rule #3)
- **Injection / Sanitization:** Không thực hiện can thiệp raw SQL nối chuỗi không an toàn. Tất cả kiểu dữ liệu Postgres được ánh xạ qua hằng số whitelist rõ ràng (`INTEGER`, `BIGINT`, `NUMERIC`, `TEXT`, `TIMESTAMPTZ`).
- **Data Integrity:** Không làm biến dạng `_raw_data` gốc. Giữ nguyên tính toàn vẹn Ground Truth theo Rule #12.
- **Secrets / PII:** Không có secret hoặc PII bị rò rỉ trong log hoặc mã nguồn.
- **Kết luận:** **PASS 100% Security & Quality Gates (G1–G8)**.

---

## 4. Cập nhật Phase 2: Refactor DRY & Unwrapping Coercion

### 4.1. Tóm tắt thay đổi Phase 2
- **Tái cấu trúc DRY:** Export `IsCanonicalMongoExtJSON` từ `scan_service.go`, xóa bỏ hoàn toàn 55 dòng code trùng lặp trong `scan_handler.go`, tinh gọn 50 dòng trong `source_router.go`. Logic ExtJSON gom về Single Source of Truth duy nhất.
- **Coercion & Unwrapping hoàn chỉnh:** Nâng cấp `unwrapExtJSON` trong `transmute/flatten.go` với `mongoNumberToGo` ép kiểu string số sang `int64`, parse float64 qua `strconv.ParseFloat`, bóc tách `$date` epoch millis và `$timestamp` thành `time.Time` UTC, ngăn ngừa triệt để lỗi ép kiểu pgx `22P02`.

### 4.2. Danh sách file thay đổi trong Phase 2
| Tệp tin | Thay đổi |
|:---|:---|
| `internal/service/source/scan_service.go` | Export `IsCanonicalMongoExtJSON` làm Single Source of Truth |
| `internal/handler/scan/scan_handler.go` | Xóa 55 dòng trùng lặp, dùng `source.IsCanonicalMongoExtJSON` |
| `internal/service/source/source_router.go` | Tinh gọn 50 dòng, gọi trực tiếp `IsCanonicalMongoExtJSON` |
| `internal/service/master/transmute/flatten.go` | Bổ sung `mongoNumberToGo`, coercion int64/float64, bóc tách `$date` và `$timestamp` |
| `internal/service/master/transmute/strategy_test.go` | Cập nhật 11 test cases khẳng định coercion kiểu dữ liệu chuẩn |

### 4.3. Kết quả nghiệm thu
- `go test` trên tất cả các package liên quan: **PASS 100%**.
- `go build` toàn bộ targets (`./internal/...`, `admin-api`, `sinkworker`, `worker`): **PASS Exit Code 0**.
