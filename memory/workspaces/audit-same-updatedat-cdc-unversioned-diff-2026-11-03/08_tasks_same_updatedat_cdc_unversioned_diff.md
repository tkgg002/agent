# 08. Tasks Registry — `audit-same-updatedat-cdc-unversioned-diff-2026-11-03`

## Danh sách Hạng mục Công việc (Autonomous Full-Stack Tasks)

### TASK-01: Điều tra mã nguồn `core-trans-proxy-history-service` & `wallet-transhis-service` (S2)
- **Mô tả & Context `file:line`:**
  - Truy vết hàm `createTransHis`, `updateTransHisStatus`, cơ chế dual-read / shadow-compare (`unversioned_diff`), và cách lưu/cập nhật `updatedAt`, `status`, `_id` trong `/Users/trainguyen/Documents/work/core-trans-proxy-history-service` và `/Users/trainguyen/Documents/work/wallet-transhis-service`.
  - Giải thích bằng bằng chứng code vì sao log ghi nhận `updateTransHisStatus` (`PENDING`, `error: 1172`, `n:1, nModified:1`) xuất hiện trước `createTransHis` (`insertOne`, `PROCESSING`) `23 ms`, nhưng kết quả trên Mongo v1 lại là `PENDING` còn Postgres giữ `PROCESSING` với cùng `updatedAt` (`05:17:04Z`).
- **Logs/Test:** Đối chiếu trực tiếp logic code tại `HEAD`.
- **Definition of Done (DoD):** Xác định chính xác 100% cơ chế sinh log, cơ chế ghi Mongo v1 và cơ chế so khớp `unversioned_diff` kèm `file:line`.

### TASK-02: Điều tra cơ chế CDC Timestamp & OCC Gate trên `centralized-data-service` (S2)
- **Mô tả & Context `file:line`:**
  - Truy vết cách `centralized-data-service` trích xuất `_source_ts` từ sự kiện Debezium MongoDB (`internal/kafka/kafka_consumer.go`, `internal/cdc/event_handler.go`) và cách xây dựng mệnh đề `ON CONFLICT ... DO UPDATE WHERE ...` ở cả tầng Shadow (`internal/service/shadow/schema_adapter.go`) lẫn tầng Master (`internal/service/master/transmuter.go`, `internal/service/master/sql_generator.go`).
  - Kiểm chứng khi 2 lệnh ghi vào MongoDB diễn ra trong cùng 1 giây (cùng `payload.source.ts_ms` từ oplog BSON Timestamp hoàn toàn tính theo giây hoặc cùng `updatedAt`), điều kiện OCC `_source_ts < EXCLUDED._source_ts` và dedup trong batch xử lý như thế nào.
- **Logs/Test:** Kiểm tra unit test ordering hiện có (`test/internal/service/schema_adapter_ordering_test.go`) và logic OCC.
- **Definition of Done (DoD):** Trả lời chính xác 100% câu hỏi của User: kịch bản *"2 lần ghi cùng mốc `updatedAt` / cùng giây làm Postgres giữ nhầm bản cũ mãi mãi nếu không có lệnh ghi tiếp theo"* có xảy ra không, cơ chế vật lý nào gây ra tại từng tầng (Debezium `source.ts_ms` vs Shadow OCC vs Master Transmute OCC), và thiết kế giải pháp triệt để kèm Code Demo (`S3` + `DG0/DG1/DG3`).
