---
tier: T2
suffix: bson_doc_similarity
workspace: audit-bson-doc-similarity-2026-11-02
---

# 08. Tasks — Đánh giá Mức độ Tương đồng & Chuẩn hóa BSON (`bson_doc_similarity`)

## TASK-01: Phân tích & Đối chiếu Chi tiết 2 Bản ghi (`Gốc` vs `Backup mới tạo`)
- **Mô tả & Context `file:line`:**
  - Đối chiếu từng trường cấp root (`16` trường gốc vs `17` trường backup) và `24` trường lá lồng nhau trong `sender`, `receiver`, `error`, `info`.
  - Truy vết luồng chuyển đổi kiểu BSON tại `centralized-data-service/internal/service/master/mongo_type_converter.go:43-102`, `centralized-data-service/internal/service/master/mongo_bson_restorer.go:42-172`, và `centralized-data-service/internal/service/master/mongo_extjson_unwrapper.go:318-395`.
- **Logs / Test:** Kiểm tra unit test hiện có trong `mongo_bson_restorer_test.go` và `mongo_extjson_unwrapper_test.go`.
- **Definition of Done (DoD):** Lập bảng ma trận đối chiếu đầy đủ 100% các trường, chỉ rõ tỷ lệ tương đồng về giá trị nghiệp vụ (`99%`) và cấu trúc/kiểu BSON (`88.9%`), giải thích rõ 4 điểm khác biệt (`value: NumberDecimal`, thiếu `platformReqId` & `__v`, dư `channelId`/`originalTransHisId`/`serviceCode: null`).

## TASK-02: Thiết kế Giải pháp Chuẩn hóa `coerceDefaultInt32IfInRange` cho `primitive.Decimal128` (Propose-Only chờ `APPROVE`)
- **Mô tả & Context `file:line`:**
  - Tại `centralized-data-service/internal/service/master/mongo_extjson_unwrapper.go:318-336`, bổ sung nhánh xử lý `primitive.Decimal128` và `string` cho hàm `coerceDefaultInt32IfInRange(val any)` để khi cột `value` (`NUMERIC` trên PostgreSQL) đã được `CoerceToBSONValue` (`mongo_type_converter.go:44`) chuyển thành `primitive.Decimal128("9999")`, bước fallback tại `mongo_bson_restorer.go:165-169` vẫn thu hẹp chính xác các số nguyên nằm trong dải `int32` về `NumberInt('9999')`.
- **Logs / Test:** `go test -v -run TestCoerceRecordToBSONDoc ./internal/service/master`.
- **Definition of Done (DoD):** Qua cổng `DG0` và `DG1` (`PASS`), hiển thị Code Demo trên chat và dừng chờ `APPROVE`.
