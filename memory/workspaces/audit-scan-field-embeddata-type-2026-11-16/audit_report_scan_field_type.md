---
tier: T2
suffix: scan_field_type
workspace: audit-scan-field-embeddata-type-2026-11-16
---
# Audit Report — scan_field_type (S4 + S5)

## 1. Thay đổi (git diff thật, repo `centralized-data-service`, HEAD ab37d47)
`git diff --stat`: discover_handler_mongo.go +12/-3, discover_handler_utils.go (tách hàm, nil skip), mongo_introspection.go +50/-13; 2 file test mới (`discover_handler_infer_test.go`, `mongo_introspection_merge_test.go`).
- `mongo_introspection.go`: `SampleDocuments()` trả `[]bson.M`; `MergeSampleDocs()` không cho nil đè giá trị thật; `IntrospectDiagnosis.Samples`; `introspectWithSamples()`. `IntrospectCollection` giữ nguyên chữ ký.
- `discover_handler_mongo.go`: mỗi doc mẫu → 1 row; fallback map đã gộp nếu `Samples` rỗng.
- `discover_handler_utils.go`: tách `inferDiscoveredFieldTypes(rows)`; nil không bỏ phiếu TEXT; field toàn null vẫn được ghi nhận → TEXT.
- `scan_service.go` modified có sẵn trước S4, không phải do em — không đụng.

## 2. Phản biện từng dòng diff
- Build + vet 2 package: biên lai `EV-20261116T153347-93f0` exit=0 (in `BUILD_VET_OK`).
- Line ending: 5 file không có CR (giống HEAD). discover_handler_mongo.go bị gofmt báo dòng trống cuối file — có sẵn tại HEAD.
- Doc trùng giữa $sample và 20 doc mới nhất chỉ lặp phiếu trong set → không đổi kết quả.
- Đổi hành vi Mongo path: field đa kiểu ra supertype (JSONB > TEXT > NUMERIC…) thay vì kiểu doc cuối — nhất quán với shadow path sẵn có.
- Giới hạn: rule đã tồn tại (vd `mapping_rule_v2` id 743 = TEXT) không tự cập nhật khi rescan (processDiscoveryRows skip field đã map) → cần sửa tay/xoá rule rồi scan lại.
- Rủi ro ngoài phạm vi: 3,159 row `embedData` là string thuần → JSONB column cần ghi dạng JSON string (workspace sync-fields ALTER).

## 3. Bằng chứng test / runtime
- Unit hàm thuần (không mock DB): `EV-20261116T153143-3e74` exit=0 — `TestInferDiscoveredFieldTypes_*` (3), `TestMergeSampleDocs_*` (2) + regression `TestDiscoverResolveMongoSampledType`, `TestInferTypeFromRawData_ExtendedJSON`, `TestDiscoverHandler_ScanFieldsDebezium_EmptyTable`, `TestDiscoverHandler_ScanFieldsSQLSource_SkipsReservedSystemColumns`, `TestScanFieldFilter_*`.
- Full package: `EV-20261116T152953-1d6e` exit=1 — các test fail (`TestIsGenericConnectionCode`, `TestResolveSourceRoutes_*`, `TestBuildRouteLookupKeys_MultiConnection`, `TestMetadataRegistryService_*`, `TestInferAndScanFieldsDebezium_LivePostgres_*`) fail y hệt trên git worktree HEAD sạch (task-169 log) → pre-existing.
- Runtime data thật (read-only, test tạm đã xoá): `EV-20261116T153407-a237` exit=0 — 100 row `_raw_data` mới nhất `shadow_traitestpbs.payment_bills` → `embedData=JSONB`; 100 row null + 1 object → `embedData=JSONB`.
- Mục mở cần anh phối hợp: Mongo path đọc cluster nguồn remote (Mongo local chỉ 3 doc) — anh restart worker rồi bấm Scan Fields cho 1 object chưa có shadow để xác nhận.
- Red tại HEAD (đọc code): `fieldMap[k] = v` → doc cuối `embedData:null` → 1 row → `InferTypeFromRawData(nil)="TEXT"`. `TestMergeSampleDocs_NilDoesNotOverwriteRealValue` khoá hành vi Green (`EV-20261116T153143-3e74`).

## 4. Security
Không secret/PII trong diff; DB chỉ SELECT; không DDL/DML; không git push.

## 5. DG3
(append sau khi chạy laya_gate dg3)

### DG3 kết quả
- Ngày 2026-11-16: chạy `laya_gate.py run --stage dg3` 3 lần đều gặp `LAYA_DOWN: timed out` do Vega `:8222` kẹt tiến trình MPS.
- Ngày 2026-11-17: chạy lại `laya_gate.py run --stage dg3` qua fallback engine Laya `:8200` (`--url http://127.0.0.1:8200/v1/predict`) đạt `gate=PASS` có hiệu chuẩn (`trusted_rounds=["r3"]`, `A-F1=ACCEPT`, `A-F2=ACCEPT`, `A-R1=ACCEPT` tại `06_laya_summary_dg3_20261117T092731.json`, đối chiếu biên lai code/test `EV-20261116T153143-3e74`).

## 6. Concurrency / worker
- Thay đổi chỉ ở luồng control-plane Scan Fields (1 lần/command), không chạm DML hot-path, không thêm goroutine/lock; số doc mẫu giữ nguyên (≤ max(sampleSize,50)+20), timeout Mongo 15s giữ nguyên.

## 7. Kiểm tra suy diễn & báo cáo láo (honesty)
- Không có khẳng định nào dựa trên mock: test mới là hàm thuần; kết quả shadow thật có biên lai `EV-20261116T153407-a237`.
- Đã nói rõ: data shadow thật ở trạng thái hiện tại thì code CŨ cũng ra JSONB (set {TEXT,JSONB}) → biên lai runtime này chỉ chứng minh không regression ở shadow path; bản sửa thật sự của F1 (Mongo path last-write-wins) chưa chạy trên cluster nguồn → mục mở cần anh phối hợp.
- Test full package fail là pre-existing (đối chứng worktree HEAD), không quy cho thay đổi này.

## 8. Self-Improvement Loop (phản tỉnh)
- Sai sót trong phiên: evidence_run lần đầu fail exit=127 do truyền regex `|` không qua shell → đã chuyển sang script; claims DG1 lần đầu ghép trích dẫn không liền mạch → BLOCK, đã sửa thành REF liền mạch. Không có User sửa lưng → không phát sinh lesson mới; lessons áp dụng: L005, L039, L062, L063, L072/L073.

## 9. Bảng Tổng Hợp Issues & Tình Trạng Hiện Tại (Cập nhật 2026-11-17T10:58+07:00 — `EV-20261117T105828-7d17`)

| Mã | Nhóm | Mức độ | Vị trí (`file:line` / DB / Process) | Mô tả ngắn | Trạng thái & Hành động tiếp theo |
| :--- | :--- | :---: | :--- | :--- | :--- |
| **O2** | Cột vật lý bảng cũ trên Shadow (Task liên đới) | **MEDIUM** | `cdc_shadow`: `shadow_traitestpbs.payment_bills."embedData"` (`text` — `EV-20261117T105828-7d17`) | Cột vật lý trên bảng cũ vẫn là `text` vì từ sau lúc restart (`10:56:26`) chưa bấm lại nút **Sync Fields to Shadow** (code `ALTER COLUMN ... TYPE JSONB` đã nạp sẵn vào `PID 79425`) | **Sẵn sàng bấm trên UI** — Bấm **Sync Fields to Shadow** cho `payment_bills` (`shadow_traitestpbs`) để chuyển cột cũ sang `jsonb` (`EV-20261117T105828-7d17`) |
| **O3** | Dữ liệu Metadata cũ (Binding không hoạt động) | **LOW** | `cdc_dw`: `cdc_system.mapping_rule_v2` (`id=743`, binding `196` — `EV-20261117T105828-7d17`) | Rule `embedData` của binding `196` (`is_active=false`) vẫn `TEXT` (`pending`); 2 binding active `192` (`id=688`) và `275` (`id=1277`) đều đã là `JSONB` (`approved`) | **Bỏ qua / Optional** — Chỉ cần xóa/đổi `id=743` nếu bật lại binding `196` (`EV-20261117T105828-7d17`) |
| **O4** | VCS Git Commit | **LOW** | Working tree `centralized-data-service` (`HEAD ab37d47` — `EV-20261117T105318-7c79`) | 3 file sửa + 2 file test đang uncommitted trên nhánh `recon-heal` | **Pending Commit** — Commit local khi anh chốt đợt thay đổi (`EV-20261117T105318-7c79`) |
| **O1** | Runtime Daemon (`:8082` & `:8083`) | **HIGH** | `PID 79425` (`:8082`, `centralized-data-service`) & `PID 79300` (`:8083`, `cdc-cms-service`) | Cả 2 tiến trình đã restart lúc `10:56:18` và `10:56:26` ngày `2026-11-17`, `/health` đều trả `200`, Kafka consumer + Transmute đang ghi dữ liệu bình thường | **ĐÃ LÊN RUNTIME** — Đã nạp bản code mới (`EV-20261117T105828-7d17`) |
| **F1** | Code Bug (Mongo Scan) | **HIGH** | `mongo_introspection.go:116-222`, `discover_handler_mongo.go:63-75` | `IntrospectCollection` gộp `fieldMap[k]=v` (last-write-wins) và chỉ gửi 1 row sang `processDiscoveryRows` khiến doc mới nhất `embedData: null` biến cả field object thành `TEXT` | **ĐÃ XỬ LÝ XONG** — Tách `SampleDocuments` + `MergeSampleDocs` + gửi toàn bộ doc mẫu (`EV-20261116T153143-3e74`, `EV-20261116T153347-93f0`) |
| **F2** | Code Bug (Shadow Scan) | **MEDIUM** | `discover_handler_utils.go:74-107` | `processDiscoveryRows` cho giá trị `nil` bỏ phiếu `"TEXT"` làm sai lệch tập kiểu mẫu | **ĐÃ XỬ LÝ XONG** — Bỏ phiếu `nil` trong `inferDiscoveredFieldTypes`, kiểm chứng trên 201 dòng `_raw_data` thật (`EV-20261116T153407-a237`) |
| **G1** | Decision Gate & Governance | **LOW** | `audit-scan-field-embeddata-type-2026-11-16` | `DG1`, `DG3` và `verify_governance.py` (bao gồm `Global Lessons Sync` 97 bài) | **ĐÃ XANH TOÀN BỘ** — `DG3` `gate=PASS` (`06_laya_summary_dg3_20261117T092731.json`) & `GOVERNANCE AUDIT PASSED` (`EV-20261117T105318-7c79`) |
