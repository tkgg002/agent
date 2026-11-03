# 08. Tasks — `mongo_createindex_dupkey`

## Danh sách Công việc (`S1 -> S6`)

- [x] **TASK-01 (S1 Intake & S2 Investigate — Exhaustive Log & Codebase Verification):**
  - **Mô tả & Context `file:line`:**
    - Kiểm tra nhật ký runtime `/tmp/cds-worker.log:45654, 49977, 50020, 51208, 51924` để xác minh vì sao **mọi cột** (`_id`, `transId`, `idempotencyKey`) trên `devtest.transaction_history_2` đều lỗi khi tạo Unique Index.
    - Đối chiếu với `centralized-data-service/internal/service/master/transmuter.go:936-955`, `centralized-data-service/internal/service/master/transmuter_identity.go:85-98, 130-155`, `centralized-data-service/internal/service/master/transmuter_mongo.go:205-248`, `centralized-data-service/internal/service/governance/index_driver_mongo.go:208-225, 260-311`, và `centralized-data-service/internal/handler/base/base_handler.go:209-222`.
  - **Logs/Test:** Bằng chứng runtime từ `/tmp/cds-worker.log` và `ps -o pid,lstart -p 5148`.
  - **Definition of Done:** Chỉ rõ chính xác 100% nguyên nhân gốc rễ cho cả 3 nhóm cột mà không dùng bất kỳ mẹo lách ("cheat code") nào.

- [x] **TASK-02 (S2.5 Pre-Plan Critique DG0 & S3 Solution + DG1/DG2):**
  - **Mô tả & Context `file:line`:**
    - Thiết kế phương án tối ưu duy nhất trong `09_tasks_solution_mongo_createindex_dupkey.md` tập trung xử lý dứt điểm lỗi tạo Unique Index trên MongoDB (`L001`, `L004`):
      1. (`F1` — Chuẩn hóa `mongoIndexDriver.CreateIndex` & `ListIndexes` tại `centralized-data-service/internal/service/governance/index_driver_mongo.go:208-226, 267-364` không dùng cheat code): Chặn gửi `unique: true` cho index đơn cột `_id` (`field '_id' already has a built-in unique index (_id_) in MongoDB`), tự động bật `idxOpts.SetSparse(true)` cho Unique Index không phải Partial Index (`spec.IsUnique && !spec.IsPartial`), lấy mẫu document có chứa trường (`$exists: true`) qua `resolveMongoSampleDocForColumns` để ánh xạ `snake_case -> camelCase` chính xác, và hiển thị `UNIQUE SPARSE INDEX` trong `ListIndexes`.
      2. (`F2` — Bảo vệ khóa chính `_id` trên MongoDB Master tại `centralized-data-service/internal/service/master/transmuter_identity.go:85-105, 130-165`): Đảm bảo `resolveConflictTarget` và `normalizeMasterPKRule` luôn nhận diện và điền `source_field = "id"`, `data_type = "TEXT"` cho rule `target_column = "_id"` khi `EngineType == "mongodb"` để ngăn nhân bản document gây lỗi `DuplicateKey` trên các cột nghiệp vụ.
      3. (`F3` — Giữ trọn thông báo lỗi `E11000 ... dup key` tại `centralized-data-service/internal/handler/base/base_handler.go:209-222`): Nâng giới hạn cắt chuỗi trong `SanitizeAdminError` từ `240` lên `512` ký tự.
    - Chạy cổng `DG1 + DG2` (`laya_gate.py run --stage dg1`) đạt `PASS` và trình bày **Code Demo** trực tiếp trên khung chat chờ User `APPROVE`.
  - **Logs/Test:** Kết quả `laya_gate.py run --stage dg1` đạt `PASS`.
  - **Definition of Done:** Hồ sơ `09_tasks_solution_mongo_createindex_dupkey.md` hoàn chỉnh, qua `DG1/DG2`, nhận `APPROVE` từ User.

- [x] **TASK-03 (S4 Execute — Implement Approved MongoDB Index Governance & Error Sanitization Fixes):**
  - **Mô tả & Context `file:line`:**
    - Thực thi sửa đổi trên `centralized-data-service/internal/service/governance/index_driver_mongo.go:208-226, 267-364`, `centralized-data-service/internal/service/master/transmuter_identity.go:85-105, 130-165`, và `centralized-data-service/internal/handler/base/base_handler.go:209-222`.
    - Bổ sung unit & integration test trong `centralized-data-service/internal/service/governance/index_driver_test.go`, `centralized-data-service/internal/handler/base/base_handler_test.go`, và `centralized-data-service/internal/service/master/transmuter_mongo_test.go`.
    - Chạy kiểm thử và build `/tmp/cds-worker-bin` qua `evidence_run.py` để lấy biên lai vật lý `EV-...`.
  - **Logs/Test:** Biên lai `EV-20261102T160923-aae5`, `EV-20261102T160931-f389`, `EV-20261102T160935-1363`, `EV-20261102T160941-ee74`, `EV-20261102T160955-65d3`, `EV-20261102T161007-3204`.
  - **Definition of Done:** Toàn bộ unit/integration test và build binary đạt `PASS` (`exit=0`).

- [x] **TASK-04 (S5 Mandatory Final Adversarial Audit, Self-Healing Loop `AF-01..AF-04`, DG3 & S6 Close):**
  - **Mô tả & Context `file:line`:**
    - Phản biện gắt gao toàn bộ quá trình, phát hiện và tự khắc phục khép kín (`S5 -> S3 -> S4 -> S5`) 4 điểm `AF-01`..`AF-04`:
      1. **`AF-01` (Single-Responsibility & Line-Budget Guard `L015`):** Tách toàn bộ hàm phân giải đặc tả cột, lấy mẫu `sampleDoc` (`$exists: true`), `parseMongoPartialFilter` và `formatByteSize` khỏi `index_driver_mongo.go` sang file đơn trách nhiệm `centralized-data-service/internal/service/governance/index_driver_mongo_spec.go`, giảm kích thước `index_driver_mongo.go` xuống dưới mức `HEAD` (`226 <= 328` dòng).
      2. **`AF-02` (Tương thích ngược `IndexOptionsConflict` & Mã lỗi HTTP 400 `natsIndexErrorStatus`):** Chuẩn hóa thông báo lỗi đơn cột `_id` bắt đầu bằng `"cannot create index on '_id': ..."` để `natsIndexErrorStatus` (`introspection_index_handler.go:12-20`) trả về đúng HTTP `400 Bad Request`; xử lý tương thích ngược khi index cùng tên đã tồn tại trên MongoDB (`IndexOptionsConflict` code 85 / `IndexKeySpecsConflict` code 86).
      3. **`AF-03` (Triệt tiêu `{ idempotencyKey: null }` tường minh trong `mongo_bson_restorer.go:143-172`):** Trong MongoDB, `sparse: true` chỉ bỏ qua document vắng mặt trường (`missing`), nhưng vẫn đánh index document có `{ idempotencyKey: null }` tường minh. Sửa `ApplySourceFieldTypesToBSONDoc` để lược bỏ `optionalNullableFields` (`idempotencyKey`, `requestId`) cả sau bước `applyBSONTypeAtPath` khi giá trị là `nil` hoặc chuỗi rỗng `""`, đảm bảo document ghi xuống MongoDB hoàn toàn vắng mặt trường đó.
      4. **`AF-04` (Zero-Mockup Live MongoDB Verification `L006`, `L025`):** Bổ sung bài test tích hợp chạy trực tiếp trên MongoDB server thật (`TestMongoIndexDriver_LiveMongo_SparseUniqueAndNullOmission` trong `index_driver_test.go`) trên collection tạm độc lập (`t.Cleanup` tự xóa, không chạm DB của User).
    - Xuất báo cáo hợp nhất `audit_report_mongo_createindex_dupkey.md`, ghi `DONE` vào `05_progress_mongo_createindex_dupkey.md`, và chạy `verify_governance.py` đạt `PASSED` 🟢.
  - **Logs/Test:** Biên lai `EV-...` thật từ `evidence_run.py`, `06_laya_summary_dg3_*.json`, `06_laya_summary_dg3review_*.json`, `audit_report_mongo_createindex_dupkey.md`.
  - **Definition of Done:** `verify_governance.py --workspace /Users/trainguyen/Documents/work/agent/memory/workspaces/audit-mongo-createindex-dupkey-2026-11-02 --suffix mongo_createindex_dupkey` đạt `PASSED` 🟢.
