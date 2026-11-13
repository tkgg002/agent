---
tier: T2
suffix: payment_v2
workspace: audit-payment-v2-2026-11-10
---
# 08 Tasks — Audit Tài Liệu `Payment-v2` & Đánh Giá Mức Độ Đáp Ứng Hệ Thống

- [x] **Task 1 (S2 Investigate — Đọc tài liệu Confluence `Payment-v2`):**
  - **Context:** Trang Confluence `page_id=512557312` (`Payment-v2`) và 3 trang con (`513179664`, `512590004`, `512458941`) + 5 Inline Comments.
  - **Action:** Truy xuất toàn bộ nội dung `body.storage` và các trang con/bình luận qua Confluence REST API (tuân thủ Rule #20).
  - **DoD:** Nắm đầy đủ 100% cấu trúc bảng nguồn, bảng đích, quy tắc đồng bộ/chuyển đổi dữ liệu, đối soát và yêu cầu vận hành của `Payment-v2`.
- [x] **Task 2 (S2 Investigate & S2.5 DG0 — Đối chiếu Thực trạng Hệ thống `data-hub`):**
  - **Context:** `centralized-data-service` (`internal/service/master/transmuter.go:1726-1754`, `internal/service/master/transmuter_utils.go:361-375,574-600`, `internal/service/master/mongo_bson_restorer.go:19-140`, `internal/service/master/mongo_extjson_unwrapper.go:185-192,351-356`, `internal/service/master/mongo_stale_field_pruner.go:20-23`, `internal/service/recon/recon_multi_shadow_b.go:154-179`).
  - **Action:** Đối chiếu từng yêu cầu CDC trong `Payment-v2` với codebase `data-hub` (`file:line`); chỉ rõ các chỗ đang bị hardcode tên cột của `transaction_history` và tên cột metadata `source_field_types`/`source_field_values` cần xóa bỏ hoàn toàn để nhận diện qua cờ `JSON Subset (Type/Presence | Value)`; chạy `laya_gate.py run --stage dg0` (`PASS`).
  - **DoD:** Phân loại rõ các hạng mục `READY` vs 2 hạng mục cần xử lý 100% tổng quát (`F1`: Postgres Array Literal `TEXT[]`; `F2`: Nhận diện cột qua cờ `JSON Subset` của Mapping Rule và xóa bỏ 100% hardcode tên cột trong `recon_multi_shadow_b.go`, `mongo_bson_restorer.go`, `mongo_extjson_unwrapper.go`, `mongo_stale_field_pruner.go`).
- [x] **Task 3 (S3 Solution & DG1/DG2 — Lập Báo cáo Nhận định & Đề xuất Giải pháp 100% Tổng quát):**
  - **Context:** `09_tasks_solution_payment_v2.md`.
  - **Action:** Thiết kế giải pháp 100% generic (Zero Table/Column Hardcoding) cho `F1` và `F2` kèm Code Demo, chạy `laya_gate.py run --stage dg1` (`PASS`) và trình User duyệt.
  - **DoD:** `09_tasks_solution_payment_v2.md` đầy đủ, qua cổng `DG1/DG2`, trình bày rõ ràng trên chat.
- [x] **Task 4 (S4 Execute — Thực thi `F1` & `F2` trên `centralized-data-service`):**
  - **Context:** `internal/service/master/transmuter_utils.go`, `internal/service/master/transmuter.go`, `internal/service/master/mongo_type_converter.go`, `internal/service/master/mongo_bson_restorer.go`, `internal/service/master/mongo_extjson_unwrapper.go`, `internal/service/master/mongo_stale_field_pruner.go`, `internal/service/recon/recon_multi_shadow_b.go`, `internal/service/recon/recon_tier_b.go`.
  - **Action:** Tạo restore-point git local; triển khai `F1` (Postgres Array Literal `TEXT[]`, `BIGINT[]`, `INTEGER[]`, `UUID[]` trong `coerceForColumn` & `sqlBindValueTransmute`) và `F2` (nhận diện cột `Type/Presence` & `Value Subset` qua cờ `IsTypePresenceRule` / `IsJSONValueSubsetRule` của Mapping Rule, xóa bỏ 100% hardcode tên cột trong `recon_multi_shadow_b.go`, `mongo_bson_restorer.go`, `mongo_extjson_unwrapper.go`, `mongo_stale_field_pruner.go`); chạy test qua `evidence_run.py` lấy biên lai `EV-...`.
  - **DoD:** Toàn bộ test suite `internal/service/master` và `internal/service/recon` PASS 100% kèm biên lai `EV-...`, không còn bất kỳ chuỗi tên cột hardcode nào.
- [x] **Task 5 (S5 Audit & S6 Close — Kiểm toán Phản biện 6 Bước, `DG3` & Xuất `audit_report_payment_v2.md`):**
  - **Context:** `audit_report_payment_v2.md`, `05_progress_payment_v2.md`.
  - **Action:** Thực hiện 6 bước S5 Audit (Line-by-Line `git diff`, đo `wc -l`, Anti-Smoke-Test Speculation, `EV-...` receipts, `laya_gate.py run --stage dg3` sau lần sửa code cuối), xuất `audit_report_payment_v2.md`, append `DONE` vào `05_progress_payment_v2.md` và chạy `verify_governance.py`.
  - **DoD:** `DG3` PASS, `verify_governance.py --workspace audit-payment-v2-2026-11-10 --suffix payment_v2` PASSED 🟢.
- [x] **Task 6 (S2/S3 Audit & Propose-Only — Kiểm tra Kiểu mảng PostgreSQL `TEXT[]`, `BIGINT[]`, `INTEGER[]`, `SMALLINT[]`, `UUID[]` trên CMS & Data Plane):**
  - **Context:** `cdc-cms-web/src/constants/postgresDataTypes.ts:1-23`, `cdc-cms-web/src/constants/clickhouseDataTypes.ts:123-165`, `cdc-cms-service/pkgs/utils/dbutil/validate.go:7-44`, `cdc-cms-service/pkgs/utils/pg_validator.go:9-37`, `cdc-cms-service/internal/infra/persistence/shadow/shadow_schema_reader_gorm.go:33-50`, `centralized-data-service/internal/service/shadow/type_resolver.go:41-59`, `centralized-data-service/internal/service/master/master_ddl_generator.go:174-224`, `centralized-data-service/internal/service/shadow/schema_adapter.go:1099-1118`.
  - **Action:** Giải thích rõ bản chất kiểu mảng PostgreSQL (`data_type = 'ARRAY'`, `udt_name = '_text' | '_int8' | '_int4' | '_int2' | '_uuid'`, cú pháp DDL `TEXT[]`); đối chiếu toàn diện mức độ hỗ trợ trên Data Plane vs Control Plane (`cdc-cms-service`) vs Frontend (`cdc-cms-web`); lập phương án bổ sung kèm Code Demo trong `09_tasks_solution_payment_v2.md` và qua cổng `DG0` + `DG1/DG2`.
  - **DoD:** Báo cáo `file:line` rõ ràng, qua `DG0` & `DG1`, giữ nghiêm chế độ Propose-Only chờ User `APPROVE`.
- [x] **Task 7 (S2 Investigate & Gap Analysis — Truy vết lý do thiếu kiểu mảng `_` / `TEXT[]` trên UI `/shadow/:id/mappings` và Master Mapping):**
  - **Context:** `cdc-cms-web/src/pages/MappingFieldsPage.tsx:410-467`, `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:770-854`, `cdc-cms-web/src/constants/postgresDataTypes.ts:1-23`, `cdc-cms-service/internal/infra/persistence/shadow/shadow_schema_reader_gorm.go:33-50`, `cdc-cms-service/pkgs/utils/dbutil/validate.go:31-44`.
  - **Action:** Xác định chính xác tại sao cột Data Type Current hiển thị `_TEXT ⚠️` nhưng ô Data Type Target và Master Data Type lại không có lựa chọn `_TEXT` hay `TEXT[]`; phân tích rõ khoảng trống (gaps) ở cả 3 tầng UI, CMS Backend Validator và Schema Introspection Normalizer.
  - **DoD:** Chỉ rõ root cause kèm `file:line` cụ thể, giải thích cặn kẽ vì sao hệ thống đang thiếu chỗ này và đề xuất giải pháp đồng bộ chuẩn DDL `TEXT[]` hoặc `_TEXT`.
- [x] **Task 8 (S2 Investigate — Xác minh cơ chế chuyển đổi dữ liệu mảng MongoDB sang cú pháp PostgreSQL Array Literal `{...}` cho kiểu `_text` vs `JSONB`):**
  - **Context:** `centralized-data-service/internal/service/master/transmuter_utils.go:361-366`, `centralized-data-service/internal/service/master/transmuter_subset_array_utils.go:21-120`, `centralized-data-service/internal/service/master/transmuter_subset_array_utils_test.go:20-53`.
  - **Action:** Điều tra và xác nhận cơ chế `isPostgresArrayColumnType` + `formatPostgresArrayLiteral` trong lõi Data Plane; đối chiếu 2 trường hợp cụ thể: mảng chuỗi nguyên thủy (`refundedOperationIds`) tự động chuyển đổi sang `{...}` khi map vào `_text`, và mảng object con phức tạp (`item: [...]`) được chuẩn hóa lưu trữ vào cột `JSONB` để bảo toàn khả năng truy vấn cấu trúc JSON của PostgreSQL.
  - **DoD:** Khẳng định chính xác hiện trạng chuyển đổi sang `{...}` đã có trong runtime CDC Data Plane kèm `file:line`, phân định ranh giới kiến trúc `_text` vs `JSONB`.
- [ ] **Task 9 (S4 Execute — Triển khai hỗ trợ đầy đủ kiểu mảng PostgreSQL trên CMS Web, CMS Backend & Data Plane):**
  - **Context:** `cdc-cms-web/src/constants/postgresDataTypes.ts`, `cdc-cms-web/src/pages/MappingFieldsPage.tsx`, `cdc-cms-web/src/constants/clickhouseDataTypes.ts`, `cdc-cms-service/pkgs/utils/dbutil/validate.go`, `cdc-cms-service/pkgs/utils/pg_validator.go`, `cdc-cms-service/internal/infra/persistence/shadow/shadow_schema_reader_gorm.go`, `centralized-data-service/internal/service/shadow/type_resolver.go`, `centralized-data-service/internal/service/shadow/schema_adapter.go`.
  - **Action:**
    1. `cdc-cms-web`: Bổ sung các kiểu mảng (`TEXT[]`, `BIGINT[]`, `INTEGER[]`, `SMALLINT[]`, `UUID[]`, `VARCHAR[]`, `BOOLEAN[]`, `NUMERIC[]`, `TIMESTAMPTZ[]`, `DATE[]`, `JSONB[]` cùng alias catalog `_TEXT`, `_INT8`, `_INT4`, `_INT2`, `_UUID`, `_VARCHAR`, v.v.) vào `POSTGRES_DATA_TYPES`; chuẩn hóa đối soát `_TEXT` <-> `TEXT[]` trong `MappingFieldsPage.tsx` để xóa bỏ false drift warning ⚠️; hỗ trợ hiển thị array types trên `MasterMappingFieldsPage.tsx`.
    2. `cdc-cms-service`: Mở rộng `IsValidDataType` trong `dbutil/validate.go` và `pg_validator.go` hỗ trợ hậu tố `[]` và tiền tố `_`; cập nhật `shadow_schema_reader_gorm.go` ánh xạ chuẩn `ARRAY` + `_udt` sang ANSI `TYPE[]`.
    3. `centralized-data-service`: Mở rộng `reTypeWhitelist` trong `type_resolver.go` và `ListColumnsWithType` trong `schema_adapter.go` nhận diện chính xác kiểu mảng và udt_name.
    4. Chạy toàn bộ test suites và build của 3 services qua `evidence_run.py` lấy biên lai `EV-...`.
  - **DoD:** Toàn bộ unit tests pass, frontend build thành công, không còn lỗi 400 validation trên CMS, không còn false drift warning ⚠️ khi target/current là kiểu mảng.
