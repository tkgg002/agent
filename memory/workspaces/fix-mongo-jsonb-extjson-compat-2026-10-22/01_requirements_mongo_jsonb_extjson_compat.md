---
tier: T2
workspace: fix-mongo-jsonb-extjson-compat-2026-10-22
suffix: mongo_jsonb_extjson_compat
created_at: 2026-10-22T09:53:00+07:00
---

# 01 Requirements: Giải quyết xung đột định dạng MongoDB Extended JSON trong cột JSONB đối với các Service cũ (`mongo_jsonb_extjson_compat`)

## 1. Bối cảnh & Vấn đề (Problem Statement)
- Trong hệ thống `data-hub` (`centralized-data-service`), ở phiên bản chuẩn hóa `CanonicalMongoExtJSON` (commit `1473d0f` ngày 30/09/2026, hồ sơ `AnalysisRawDataDebeziumVsSnapshot20260929`), toàn bộ payload MongoDB đi qua tầng Shadow (`kafka_consumer.go`, `sinkworker/utils.go`, `snapshot_runner_handler.go`, `recon_heal_fetch.go`) đã được chuyển về chuẩn **MongoDB Canonical Extended JSON (v2)** (`bson.MarshalExtJSON(doc, true, false)`):
  - `_id` / `ObjectId`: `{"$oid": "67230efb7ed87a55a4ce8393"}`
  - `Date`: `{"$date": "2024-10-31T05:00:43.935Z"}` (Relaxed) hoặc `{"$date": {"$numberLong": "1730350843935"}}` (Canonical)
  - `Number` (`int32`, `int64`, `double`, `decimal128`): `{"$numberInt": "123123123"}`, `{"$numberLong": "123123123"}`, `{"$numberDouble": "123.45"}`, `{"$numberDecimal": "123.45"}`
  - Các kiểu BSON mở rộng khác: `$timestamp`, `$binary`, `$regularExpression`, `$symbol`...
- Tại tầng Master (`transmuter.go:1915-1917` và `mapping_utils.go:68-72`), khi map sang cột đích kiểu `JSONB` / `JSON`, hệ thống hiện giữ nguyên xi cấu trúc Canonical Extended JSON từ Shadow (`_raw_data`) mà không bóc (unwrap) hoặc chuyển đổi theo nhu cầu của từng consumer/service.
- **Vấn đề thực tế (Issue):**
  - Một số **service cũ** đọc dữ liệu `JSONB` đã viết sẵn các hàm (functions / struct unmarshal / SQL JSONB path queries) quen với định dạng phẳng không có lớp bọc `$numberInt` / `$numberLong` / `$numberDouble` (hoặc viết theo dạng `{"number": 123123123}` hay `"number": "123123123"` trực tiếp ở key, thay vì lồng object `{"number": {"$numberLong": "123123123"}}`).
  - Nếu đổi cứng toàn cục ở Shadow (bỏ Canonical ExtJSON) thì sẽ làm gãy cơ chế bảo toàn kiểu BSON của Shadow/Recon/Scan đã xây dựng ở `AnalysisRawDataDebeziumVsSnapshot20260929` và `canonical-mongo-extjson-scan`; ngược lại, nếu giữ cứng Canonical ExtJSON đẩy thẳng vào mọi cột `JSONB` của Master mà không có cơ chế bóc tách/chuyển đổi linh hoạt theo rule (`transform_fn` / `unwrap` đệ quy trên cột `JSONB`), các service cũ sẽ đọc lỗi dữ liệu.

## 2. Yêu cầu Nghiệp vụ & Kỹ thuật (Requirements)
- **R1 (Phân tích toàn diện luồng JSONB MongoDB từ Shadow -> Master -> Service cũ):** Xác định chính xác toàn bộ điểm chạm (`file:line`) nơi MongoDB Extended JSON được tạo ra ở Shadow (`kafka_consumer.go`, `sinkworker/utils.go`, `snapshot_runner_handler.go`, `recon_heal_fetch.go`, `schema_adapter_coerce.go`) và nơi nó được chuyển sang cột `JSONB` của Master (`transmuter.go`, `transmuter_utils.go`, `transform_registry.go`, `mapping_utils.go`, `transmute/flatten.go`).
- **R2 (Bảo toàn BSON Metadata ở Shadow & Không phá vỡ tương thích ngược — Lesson L079):** Giữ nguyên chuẩn Canonical Extended JSON tại `_raw_data` của tầng Shadow để không làm hỏng đối soát (Recon), Scan schema (`IsCanonicalMongoExtJSON`) và các pipeline mới.
- **R3 (Giải pháp chuẩn hóa/bóc tách đệ quy Extended JSON cho cột `JSONB` phục vụ Service cũ):** Thiết kế phương án kỹ thuật tối ưu nhất (kèm Code Demo đầy đủ trong `09_tasks_solution_mongo_jsonb_extjson_compat.md`) cho phép:
  1. Bóc tách đệ quy (Deep/Recursive Unwrap) toàn bộ các wrapper BSON Extended JSON (`$oid`, `$date`, `$numberInt`, `$numberLong`, `$numberDouble`, `$numberDecimal`, `$timestamp`, `$binary`...) bên trong cấu trúc `JSONB` (cả nested object và array) khi map sang Master hoặc qua `transform_fn` trong `transform_registry.go` + SQL helper trong `mapping_utils.go`.
  2. Hỗ trợ chuẩn xác cả 2 chế độ biểu diễn trường số trong `JSONB` mà các service cũ đang dùng:
     - Chế độ **Native JSON Scalar (`unwrap_extjson` / `extjson_to_plain_json`)**: bóc `$numberInt`/`$numberLong`/`$numberDouble` về JSON number (`{"number": 123123123}`), `$oid` về hex string (`"67230efb7ed87a55a4ce8393"`), `$date` về chuỗi ISO-8601 (`"2024-10-31T05:00:43.935Z"`).
     - Chế độ **Stringified Number (`unwrap_extjson_num_str` / `extjson_to_string_number`)**: bóc `$numberInt`/`$numberLong`/`$numberDouble`/`$numberDecimal` về JSON string (`{"number": "123123123"}`), `$oid` về hex string, `$date` về ISO-8601 string — phục vụ các service cũ đã viết func đọc `number:"123123123"`.
- **R4 (Full-Stack Cấu hình `transform_fn` trên từng Field khi `Master Data Type` là `JSONB` / `JSON` tại trang `/masters/:id/mappings`):**
  1. **Database & CMS Backend (`cdc-cms-service`):** Khôi phục cột `transform_fn VARCHAR(100)` trên bảng `cdc_system.mapping_rule_master` (do migration `075` đã drop mất và chỉ đang đọc `v2.transform_fn`), cập nhật `masterRuleSelectJoin` dùng `COALESCE(m.transform_fn, v2.transform_fn) AS transform_fn`, và hỗ trợ cập nhật `transform_fn` qua API `PUT /api/v1/master-mapping-rules/:id/data-type` (hoặc `/transform-fn`) và `POST /api/v1/master-mapping-rules`.
  2. **Worker Transmuter (`centralized-data-service`):** Cập nhật `loadRules` (`transmuter.go:998`) và `Generate` (`master_ddl_generator.go:116`) sang `COALESCE(m.transform_fn, v2.transform_fn) AS transform_fn` để nhận `transform_fn` cấu hình riêng trên từng field của Master.
  3. **CMS Frontend (`cdc-cms-web` — `MasterMappingFieldsPage.tsx`):** Tại cột **`Master Data Type`** trên trang `/masters/:id/mappings?binding_id=...`, khi `baseType === 'JSONB' || baseType === 'JSON'`, hiển thị `<Select>` cho phép chọn trực tiếp `transform_fn` của field đó (Mặc định giữ nguyên Shadow, hoặc bóc tách theo `extjson_relaxed`, `extjson_relaxed_num_str`, `extjson_unwrap_plain`, `extjson_unwrap_num_str`).

## 3. Tiêu chuẩn Nghiệm thu (Definition of Done - Gates G1–G8)
- Hoàn thành S1 Intake, S2 Investigate với đầy đủ dẫn chứng `file:line`.
- Hoàn thành S3 Solution & Code Demo trong `09_tasks_solution_mongo_jsonb_extjson_compat.md` và vượt qua cổng kiểm định `/laya_router` (`DG1 + DG2` đạt `PASS`).
- Trình anh xem xét và chờ lệnh `APPROVE` trước khi thực thi S4 (Rule #1, Rule #13).
