# 08. Tasks Registry — `audit-same-updatedat-cdc-unversioned-diff-2026-11-03`

## Danh sách Hạng mục Công việc (Autonomous Full-Stack Tasks)

### TASK-01: [x] Điều tra mã nguồn `core-trans-proxy-service` & `core-trans-proxy-history-service` (S2)
- **Mô tả & Context `file:line`:**
  - Truy vết hàm `createTransHis`, `updateTransHisStatus` trong `/Users/trainguyen/Documents/work/core-trans-proxy-service/publish/cashin.publish.ts:34-44`, `/Users/trainguyen/Documents/work/core-trans-proxy-history-service/start.ts:50-55`, và `/Users/trainguyen/Documents/work/core-trans-proxy-history-service/models/transHis.model.ts:28-32, 92-114`.
  - Giải thích bằng bằng chứng code vì sao log ghi nhận `updateTransHisStatus` (`PENDING`, `error: 1172`, `n:1, nModified:1`) xuất hiện trước `createTransHis` (`insertOne`, `PROCESSING`) `23 ms`, nhưng kết quả trên Mongo v1 lại là `PENDING` còn Postgres giữ `PROCESSING` với cùng `updatedAt` (`05:17:04Z`).
- **Logs/Test:** Đối chiếu trực tiếp logic code tại `HEAD`.
- **Definition of Done (DoD):** Đã xác định chính xác 100% cơ chế `handlePromise` không `await`, `transId` có `unique: true`, và overhead của `createIpAddress(ctx)` + `$save` làm `createTransHis` ghi log trễ hơn `updateTransHisStatus` `23 ms` dù `insertOne` thực thi trước trên MongoDB.

### TASK-02: [x] Điều tra cơ chế CDC Timestamp & OCC Gate trên `centralized-data-service` (S2/S3/S5)
- **Mô tả & Context `file:line`:**
  - Truy vết cách `centralized-data-service` trích xuất `_source_ts` từ sự kiện Debezium MongoDB (`internal/handler/shadow/utils.go:72-105`, `internal/handler/shadow/kafka_consumer.go:621`), cách dedup trong batch (`internal/handler/shadow/batch_buffer_utils.go:90-153`), và cách xây dựng mệnh đề `ON CONFLICT ... DO UPDATE WHERE ...` ở cả tầng Shadow (`internal/service/shadow/schema_adapter.go:745-795`) lẫn tầng Master (`internal/service/master/transmuter.go:1695-1701`).
  - Kiểm chứng khi 2 lệnh ghi vào MongoDB diễn ra trong cùng 1 giây (cùng `payload.source.ts_ms` từ oplog BSON Timestamp hoàn toàn tính theo giây hoặc cùng `updatedAt`), điều kiện OCC `_source_ts < EXCLUDED._source_ts` và dedup trong batch xử lý như thế nào.
- **Logs/Test:** Kiểm tra unit test ordering hiện có (`test/internal/service/schema_adapter_ordering_test.go:108-139`) và chạy kiểm định qua `/laya_router` (`DG0`, `DG1`, `DG2`, `DG3`).
- **Definition of Done (DoD):** Đã xác minh 100% giả thuyết của User là đúng, lập giải pháp kèm Code Demo trong `09_tasks_solution_same_updatedat_cdc_unversioned_diff.md` và `audit_report_same_updatedat_cdc_unversioned_diff.md`, pass đủ `DG0/DG1/DG2/DG3`.

### TASK-BUG-01: [x] Self-Healing từ S5 Adversarial Audit & `/laya_router` Mode B (`S5 -> S3 -> S4 -> S5`)
- **Mô tả & Context `file:line`:**
  - **(1) Khử lặp logic `unwrapAvroUnion` tại `internal/handler/shadow/utils.go:72-95`:** `/laya_router` Mode B (`06_laya_summary_dg3review_20261103T135231.json`) phát hiện vòng lặp `if mm, ok := raw.(map[string]interface{}); ok && len(mm) == 1` trong `parseAnyInt64` bị lặp với hàm `unwrapAvroUnion(raw)` đã có sẵn tại `internal/handler/shadow/avro_helper.go:67-82`. Tái sử dụng trực tiếp `raw = unwrapAvroUnion(raw)` trong `parseAnyInt64`.
  - **(2) Đồng bộ `source.ord` sub-second tie-breaker sang peer `internal/sinkworker/utils.go:88-102`:** Truy vết đầu-cuối (`L029`) phát hiện `sinkworker` cũng có hàm `extractSourceTsMs(envelope map[string]any) int64` đọc `src["ts_ms"]` nhưng chưa cộng `min(ord, 999)` khi `ts > 0 && ts%1000 == 0`. Cập nhật `internal/sinkworker/utils.go:88-102` và bổ sung test tại `test/internal/sinkworker/sinkworker_test.go:135-170`.
- **Logs/Test:** Chạy `go test` qua `evidence_run.py` cho `internal/handler/shadow` và `test/internal/sinkworker`, chạy lại `laya_gate.py run --stage dg3` và `laya_gate.py review` đạt `PASS`.
- **Definition of Done (DoD):** Không còn lặp code trong `utils.go`, đồng bộ 100% logic giữa `shadow` và `sinkworker`, mọi test + `DG3` Mode A & Mode B đều `PASS`.
