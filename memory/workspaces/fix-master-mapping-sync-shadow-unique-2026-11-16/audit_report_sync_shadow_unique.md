---
tier: T2
suffix: sync_shadow_unique
workspace: fix-master-mapping-sync-shadow-unique-2026-11-16
created_at: 2026-11-17T09:58:45+07:00
---
# Audit Report & Consolidated Validation — Khắc Phục Lỗi `SQLSTATE 23505` Khi "Sync Từ Shadow" (`binding_id=61`)

## Mục 1: Phản biện từng dòng update / git diff & từng task (Line-by-Line Audit)

### 1.1. Thống kê dòng lệnh (`wc -l`) & Phạm vi thay đổi của Task
- `cdc-cms-service/internal/infra/persistence/master/master_multi_shadow_repo_gorm.go`: **320 dòng** (giảm từ 329 xuống 320 dòng, tuân thủ chặt SRP, không phình God-file).
- `cdc-cms-service/internal/infra/persistence/master/master_repo_gorm_test.go`: **338 dòng** (bổ sung test đối kháng đa kịch bản `TestSyncRulesFromShadow_PreservesCustomizedColumnsAndAvoidsUniqueConflict`).
- `cdc-cms-web/src/utils/apiError.ts`: **152 dòng** (+2 dòng ánh xạ `ux_mapping_rule_master_target`).

### 1.2. Rà soát từng dòng `git diff`
1. **`master_multi_shadow_repo_gorm.go:209-272` (`syncRulesFromAllShadowsTx`):**
   - Dòng `210-214`: Kiểm tra sớm `if len(shadowIDs) == 0 { return 0, 0, nil }` tránh mở transaction rỗng không cần thiết.
   - Dòng `216-221`: Phân giải tên bảng `v2Table` và `masterRuleTable` theo `GO_ENV == "test"` giống hệt hàm chuẩn `cloneAllShadowRulesTx` (`:37-42`), chỉ dùng hằng chuỗi nội bộ cố định (không có nguy cơ SQL Injection).
   - Dòng `226-228`: Bỏ qua an toàn `sbID <= 0`.
   - Dòng `230-260`:
     - Chèn các cột mới từ `mapping_rule_v2` với `status = 'pending'`, `is_active = true`, `created_by = 'shadow-sync'`, `CURRENT_TIMESTAMP` (chuẩn ANSI SQL chạy đồng nhất trên cả PostgreSQL và SQLite).
     - Lọc chặt `v2.status = 'approved'`, `COALESCE(v2.is_active, true) = true`, `TRIM(COALESCE(v2.target_column, '')) <> ''`, và `LOWER(TRIM(v2.target_column)) NOT IN (?)` (dùng `IN (?)` đúng chuẩn GORM slice theo `L077`, không dùng `ANY(?)`).
     - Khử trùng tiền định bằng `v2.id = (SELECT MIN(v2d.id) ...)` phòng trường hợp `mapping_rule_v2` chứa nhiều bản ghi cùng tên cột khác hoa-thường.
     - Kiểm tra tồn tại kép: `NOT EXISTS (... m.mapping_v2_id = v2.id)` (giữ nguyên mọi cột đã được Operator đổi tên sang `snake_case` như `billId -> bill_id` hoặc đã tách `1 shadow field -> N master fields`) **và** `NOT EXISTS (... LOWER(TRIM(m.target_column)) = LOWER(TRIM(v2.target_column)))` (chặn đụng tên cột với bảng Shadow phụ trong `multi_shadow` hoặc rule tự tạo).
     - Chốt chặn nguyên tử cấp CSDL `ON CONFLICT (master_binding_id, target_column) DO NOTHING` khớp trực tiếp Unique Index `ux_mapping_rule_master_target`.
   - Đã gỡ bỏ hoàn toàn 2 khối `UPDATE` cưỡng bức ghi đè `target_column = v2.target_column` vốn gây tự xung đột `SQLSTATE 23505` ngay trong cùng transaction và làm mất cấu hình đổi tên cột của Operator.
2. **`apiError.ts:43-44` (`CONSTRAINT_HUMAN`):**
   - Bổ sung khóa `ux_mapping_rule_master_target` hiển thị thông báo tiếng Việt chính xác khi người dùng nhập trùng `target_column` trong cùng một Master Binding.

---

## Mục 2: Chống suy diễn Smoke Test / Mockup / Hallucination & Trace Runtime thật

### 2.1. Biên lai Thực thi Vật lý (`06_evidence/EV-*.json`)
- Kiểm thử đơn vị gói `persistence/master` (gồm tái hiện Red sang Green trên `TestSyncRulesFromShadow_PreservesCustomizedColumnsAndAvoidsUniqueConflict`) và các gói callers (`test/internal/app/commands`, `test/internal/infra/persistence`): đạt `PASS` kèm biên lai `EV-20261117T095822-b70b` (`exit=0`).
- Kiểm tra tĩnh toàn bộ cây mã nguồn `cdc-cms-service` (`go vet ./internal/... ./cmd/...`) ở chế độ runtime check: đạt `OK` kèm biên lai `EV-20261117T095838-e1af` (`exit=0`).
- Biên dịch binary thực tế `bin/cms` của `cdc-cms-service` (`go build -o bin/cms ./cmd/server/`) ở chế độ runtime build: đạt `OK` kèm biên lai `EV-20261117T095830-8572` (`exit=0`).
- Biên dịch TypeScript (`tsc -b`) và đóng gói bundle Vite thực tế của `cdc-cms-web` (`npm run build`) ở chế độ runtime build: đạt `OK` kèm biên lai `EV-20261117T095045-506f` (`exit=0`).
- Không gắn bất kỳ closure giả (`*Fn`), mockup hay dữ liệu bịa đặt nào vào struct production.

### 2.2. Giới hạn & Các hạng mục chưa kiểm chứng trực tiếp trên `binding_id=61` (Phối hợp User)
- Trên container local `gpay-postgres-cdc` (`cdc_dw`), bảng `cdc_system.master_binding` hiện không có bản ghi `id = 61` (`0 rows`), còn tiến trình `main` đang lắng nghe `:8083` (`PID 86070`) là binary biên dịch trước khi sửa code và tuân thủ nghiêm `L072`/`L073` (không tự ý bắn câu lệnh ghi vào DB đang chạy).
- Do đó, em **không suy diễn** rằng unit test SQLite đồng nghĩa với việc phiên chạy của anh đã tự cập nhật: để xác nhận trực tiếp trên trang `/masters/cdc_payment_bill/mappings?binding_id=61`, cần anh khởi động lại `cdc-cms-service` từ bản build mới (`./bin/cms` hoặc `make run`) và bấm lại nút **"Sync từ Shadow"** để kiểm chứng thực tế cùng em.

---

## Mục 3: Kiểm tra Concurrency / Multi-Worker / Timeout-Heartbeat / Data Continuity

1. **Concurrency & Multi-Request Race Condition (`2 Operators cùng bấm Sync từ Shadow`):**
   - Giữa thời điểm kiểm tra `WHERE NOT EXISTS (...)` và thời điểm ghi đĩa, nếu 2 request HTTP chạy song song cho cùng `master_binding_id = 61`, mệnh đề `ON CONFLICT (master_binding_id, target_column) DO NOTHING` bảo vệ nguyên tử tại tầng index PostgreSQL `ux_mapping_rule_master_target`: request đến sau bỏ qua dòng trùng mà không làm abort transaction (`SQLSTATE 23505` / `25P02`).
2. **Multi-Shadow Sequential Loop (`collectShadowIDsFromSpec`):**
   - Trong cùng một transaction `tx`, vòng lặp qua `shadowIDs` thực thi tuần tự: các dòng được chèn từ Primary Shadow hiển thị ngay lập tức cho câu truy vấn `NOT EXISTS (... LOWER(TRIM(m.target_column)) = LOWER(TRIM(v2.target_column)))` của Secondary Shadow kế tiếp, ngăn chặn triệt để va chạm tên cột giữa nhiều bảng Shadow.
3. **Data Continuity, Watermark, Debouncer & Timeout/Heartbeat của Data Plane:**
   - Việc chuyển trạng thái rule mới kéo về từ `'approved'` sang `'pending'` bảo vệ trực tiếp luồng `TransmuterModule` (`Watermark/Cursor _gpay_id`, `TableDebouncer`, `AckWait` vs `Heartbeat` của `TransmuteConsumerPool`): ngăn Transmuter nạp sớm cột chưa có vật lý trên bảng Master (`in_master = false`), tránh gây lỗi `SQLSTATE 42703` làm kẹt hàng đợi JetStream `CDC_TRANSMUTE`. Chỉ sau khi Operator duyệt rule (`BatchUpdateRulesHandler` phát `cdc.cmd.master-create` tạo cột xong), cột mới tham gia luồng transmute.

---

## Mục 4: Kiểm tra suy diễn & báo cáo láo (Anti-Speculation & Honesty)

- Toàn bộ 4 phát hiện `F1`–`F4` đều trích dẫn chính xác dòng lệnh thật (`master_multi_shadow_repo_gorm.go:209-281`, `078_master_allow_many_per_shadow_field.sql:8-23`, `batch_update_rules.go:42-53`, `transmuter.go:1044`, `apiError.ts:41-57`).
- Minh bạch 100% về các gói test có sẵn ngoài phạm vi trong `cdc-cms-service`: `create_master_mongo_real_test.go` thiếu dữ liệu seed `shadow_binding` local và `observability/probes/worker_test.go` lệch kỳ vọng đường dẫn `/healthz` cũ (đúng Mục 0.2 của `system_overview_and_design_pattern.md` — không đụng chạm và không che giấu).
- Bảo mật (`/security-agent`): Không có secret, token, URI chứa mật khẩu hay PII trong code diff và trong toàn bộ file workspace; mọi tham số truy vấn (`masterBindingID`, `updatedBy`, `sbID`, `reservedCols`) đều truyền qua placeholder `?`. Không chạy `git push` (Rule #15).

---

## Mục 5: Vòng lặp Phản tỉnh & Khắc phục (Self-Improvement Loop & Lessons)

- **Rà soát vi phạm Lesson trong phiên:** Không vi phạm bài học nào trong `lessons/INDEX.md`.
- **Các bài học đã áp dụng hiệu quả:**
  - `L041`: Tách rời từng lệnh kiểm định, không nhúng từ khóa SQL vào tham số dòng lệnh của cổng kiểm định.
  - `L072` & `L073`: Tuyệt đối không chạy câu lệnh ghi (kể cả bọc rollback/explain) lên DB thật; chỉ đọc `pg_indexes` qua `SELECT` thuần.
  - `L075` & `L076`: Đối chiếu lịch sử migration `074`–`081` và index thật `ux_mapping_rule_master_target` trước khi viết SQL.
  - `L077`: Sử dụng `IN (?)` cho slice GORM, `CURRENT_TIMESTAMP` chuẩn ANSI SQL, và `ON CONFLICT DO NOTHING` tránh `25P02` transaction aborted.
- **Kinh nghiệm rút ra:** Trong môi trường nhiều phiên làm việc cùng lúc trên repo, cần chạy `evidence_run.py` sát thời điểm ghi `audit_report` để `src_fingerprint` luôn đồng bộ mới nhất.

---

## Mục 6: Kết quả QC & nhận xét từ /laya_router (DG3)

- **DG0 (Pre-Plan Critique):** Đạt tại `06_laya_summary_dg0_20261117T093350.json`.
- **DG1 + DG2 (Solution & Code Demo Verification):** Đạt tại `06_laya_summary_dg1_20261117T093802.json` và `06_laya_summary_dg2_20261117T093802.json`.
- **DG3 (Final Adversarial Diff & Test Audit — `laya_gate.py`):** Đạt cổng kiểm định cuối phiên tại `06_laya_summary_dg3_20261117T095951.json` (fallback sang Laya `:8200` qua `--url` theo Rule #10 do Vega `:8222/v1/predict` timeout; engine trả `trusted_rounds=[]` / `UNVALIDATED`, toàn bộ claims `A-DIFF-1`, `A-TEST-1`, `A-FE-1` được kết luận trực tiếp bằng ground-truth mã nguồn `master_multi_shadow_repo_gorm.go:214-272`, `master_repo_gorm_test.go:284-315`, `apiError.ts:39-47` và biên lai thực thi ở Mục 2).
