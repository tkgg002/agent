# 📚 Hard-Tech Patterns & Tripwires (Garbage Collected)

> **BẢN CHẤT**: File chứa các bẫy kỹ thuật đặc thù (Postgres, CDC, Kafka, Golang, MongoDB). Các quy trình hành vi đã được GC nén vào `tech_stack.md`. BẮT BUỘC ĐỌC TRƯỚC KHI CODE.

### [2026-07-30] Khi tìm file tài liệu từ session cũ, phải check BOTH workspace memory VÀ brain artifacts của conversation đó

- **Global Pattern:** User yêu cầu tìm lại file tài liệu [X] từ session trước -\u003e Agent [A] chỉ search trong `agent/memory/workspaces/` và `agent/memory/global/` -\u003e Không tìm thấy -\u003e Kết luận "file chưa được lưu" -\u003e **SAI**. File có thể đã được lưu đúng chuẩn vào artifact directory của conversation cũ (`~/.gemini/antigravity-ide/brain/<conversation-id>/`). **Đúng:** Khi search tài liệu: (1) Search `agent/memory/workspaces/` trước. (2) Nếu không thấy → check `ls ~/.gemini/antigravity-ide/brain/<conversation-id>/` của conversation liên quan. (3) Nếu không biết conversation-id → grep trong transcript logs hoặc conversation summaries.
- **Bối cảnh (Trigger):** User hỏi "tìm plan tối ưu transform 50M records timeout 15 phút" → Agent search sai chỗ → Không tìm thấy → Kết luận oan là "chưa lưu tài liệu" → User tức giận.
- **Root Cause:** Search scope thiếu — chỉ check workspace memory, bỏ sót artifacts directory của conversation cũ.
- **Fix/Correct Flow:** `find ~/.gemini/antigravity-ide/brain -name "*.md" | xargs grep -l "keyword"` để search toàn bộ brain artifacts của mọi conversation.
- **Tags:** #artifact-search-scope #brain-artifacts #workspace-memory #session-continuity

### [2026-07-31] Pattern: Phân tách Async Job Tracking giữa Manual Command và Realtime Stream

- **Global Pattern:** Engine [A] phục vụ cả 2 luồng: Realtime Stream (vài dòng/batch, 15ms) và Manual Bulk Run (hàng triệu dòng, hàng giờ) → nếu vô tình ghi DB tracking cho luồng Realtime Stream → phình DB rác & làm trễ luồng 15ms. **Đúng:** Kiểm tra cờ `jobID != ""` tại Engine. Luồng Realtime mang `jobID = ""` → skip 100% việc tạo/cập nhật DB tracking. Luồng Manual mang `jobID` duy nhất → kích hoạt heartbeat progress & cancel listener.
- **Bối cảnh (Trigger):** Triển khai Transmute Live Tracking & Progress Bar cho nút "Transmute Now" trên CMS UI mà không ảnh hưởng luồng CDC/Oplog Transmute realtime.
- **Root Cause:** Cần bảo toàn hiệu năng siêu tốc 15ms của luồng Oplog realtime trong khi vẫn cung cấp khả năng quan sát (observability) cho tác vụ Manual Bulk.
- **Fix/Correct Flow:** CMS sinh `job_id` chỉ khi bấm "Transmute Now". Worker `TransmuterModule.Run` check `isManualJob := jobID != ""`.
- **Tags:** #async-job-tracking #boundary-isolation #oplog-stream #manual-run-now #transmute-engine

### [2026-07-31] Pattern: LIMIT 1 trên bảng quan hệ 1-N gây lookup sai resource khi có nhiều row

- **Global Pattern:** Service [A] lookup `target_table` của entity [X] bằng `JOIN relation_table LIMIT 1` → entity [X] có N rows trong relation_table → LIMIT 1 chỉ pick 1 row ngẫu nhiên/theo sort → nếu pick sai → resource lookup tiếp theo trả empty. **Đúng:** Collect TẤT CẢ candidate rows, sau đó query resource với `WHERE key IN (all_candidates) ORDER BY created_at DESC`.
- **Bối cảnh (Trigger):** `GetLatestBySourceObjectID(14)`: source_object 14 có 2 bindings (`payment_bills`, `payment_bills_1`). LIMIT 1 pick `payment_bills_1` (không có job) → trả `no_job` dù `payment_bills` đã COMPLETED 46k rows.
- **Root Cause:** Assumption sai "1 source_object → 1 shadow_binding". Thực tế là 1-N. Phải luôn collect all candidates.
- **Fix/Correct Flow:** `SELECT shadow_table FROM shadow_binding WHERE source_object_id = ?` (không LIMIT) → `tableNames = [...]` → `WHERE target_table IN (tableNames) ORDER BY created_at DESC FIRST`.
- **Tags:** #one-to-many #limit-1-wrong #multi-binding #target-table-lookup #golang-gorm

### [2026-07-31] Pattern: Read-only endpoint dùng chung scope resolver với dispatch endpoint → 500 khi binding inactive

- **Global Pattern:** Endpoint đọc [A] (read-only, e.g. GET status) tái dùng `resolveReadScope` của dispatch endpoint [B] → resolver JOIN `shadow_binding WHERE is_active=TRUE` → nếu binding inactive/absent → `ErrSourceObjectNoActiveShadow` → 500. **Đúng:** Read-only endpoint phải dùng query riêng không cần `is_active`, hoặc lookup thẳng từ resource table (không qua scope resolver).
- **Bối cảnh (Trigger):** `TransformJobStatusV2` dùng `resolveReadScope` → binding của source_object_id=14 inactive → 500 `resolve_scope_failed`.
- **Root Cause:** `resolveReadScope` vẫn JOIN `shadow_binding AND is_active=TRUE` vì `readScopeQuery` chỉ bỏ `so.is_active=TRUE`, không bỏ `sb.is_active=TRUE`. Read-only endpoint cho tracking job không cần scope resolve — chỉ cần lookup bằng source_object_id.
- **Fix/Correct Flow:** Thêm `GetLatestBySourceObjectID` query `shadow_binding` (bất kể `is_active`) để lấy `target_table`, rồi query `transform_jobs`. Bỏ hoàn toàn `resolveReadScope` khỏi status/cancel handler.
- **Tags:** #golang-scope-resolver #is-active-gate #read-only-endpoint #shadow-binding #fiber-500

### [2026-07-31] Pattern: Dynamic size tuning trước break-condition gây early-exit loop sau iteration đầu tiên

- **Global Pattern:** Handler [A] xử lý batch [X] theo loop → tuning `size` TRƯỚC khi check break `if count < size` → `count` được fetch với `size` cũ nhưng so sánh với `size` mới → khi `size` tăng (fast path), `count` < `size_new` = true → BREAK SỚM. **Đúng:** Lưu `requestedSize := size` TRƯỚC khi tune, so sánh break với `requestedSize` (giá trị đã dùng trong SQL LIMIT).
- **Bối cảnh (Trigger):** `BatchTransformHandler` dùng CTE chunked UPDATE, dynamic chunk size tăng gấp đôi sau mỗi iteration < 100ms → loop exit sau iter 1 dù còn 50M records.
- **Root Cause:** Scope của biến `chunkSize` bị tái dùng cho cả 2 mục đích: (1) LIMIT SQL và (2) break condition. Sau khi tune, nó chỉ đúng cho mục đích (1) của iter TIẾP THEO, nhưng break check dùng nó cho iter HIỆN TẠI.
- **Fix/Correct Flow:** `requestedChunkSize := chunkSize` → tune `chunkSize` → break check dùng `requestedChunkSize`.
- **Tags:** #golang-loop-logic #dynamic-batch-size #break-condition #premature-exit #chunk-processing

### [2026-07-29] Tuyệt đối không tự ý sửa code khi User chỉ yêu cầu thống kê và chỉ sửa đúng nơi được yêu cầu (Rule #0 / Rule #13)

- **Global Pattern:** User yêu cầu thống kê/kê khai danh sách các vị trí [X] -> Agent [A] tự ý thêm code sửa file [Y] mà User chưa hề yêu cầu hoặc chưa duyệt -> Vi phạm nghiêm trọng quy định Brain Code Prohibition (Rule #13) và không lắng nghe chỉ thị của User. **Đúng:** (1) Khi User yêu cầu thống kê ("thống kê / nhớ lại xem"), chỉ tổng hợp và báo cáo danh số/trạng thái. (2) Tuyệt đối không tự ý sửa thêm bất kỳ file code nào khi chưa có lệnh/kế hoạch được duyệt. (3) Nếu lỡ sửa nhầm, lập tức `git checkout` revert về trạng thái sạch ngay lập tức.
- **Bối cảnh (Trigger):** User hỏi "có bao nhiêu cái cần copy traceid", Agent tự ý thêm cột Trace ID vào `ReconPipelineGrid.tsx` dù nơi này chưa cần.
- **Root Cause:** Cầm đèn chạy trước ô tô, tự ý phán đoán và sửa code sai phạm vi User yêu cầu.
- **Fix/Correct Flow:** Dùng `git checkout` hoàn tác ngay `ReconPipelineGrid.tsx`, xin lỗi User và trả lời đúng trọng tâm thống kê.
- **Tags:** #brain-code-prohibition #user-directive-alignment #revert-unrequested-changes #strict-scope-control

### [2026-07-29] Tuyệt đối không báo Done khi chưa triển khai nút/cột Click-to-Copy Trace ID đầy đủ trên giao diện CMS FE (End-to-End DoD Gate G1/G7)

- **Global Pattern:** Agent [A] cập nhật backend [X] để đính kèm `trace_id` trong NATS event / log backend -> Vội vàng báo cáo hoàn thành và phán đoán đã có tính năng Click-to-Copy Trace ID trên UI -> User kiểm tra giao diện CMS FE (`cdc-cms-web`) thì phát hiện bảng Activity Log (`ActivityLog.tsx`) và các thẻ / nút Transmute / Batch Transform hoàn toàn không hiển thị hay cho copy `trace_id` -> Vi phạm nghiêm trọng Definition of Done (Rule #14 Gate G1 Requirement Traceability & Gate G7 Adversarial Self-Review). **Đúng:** (1) Bắt buộc đối soát full luồng End-to-End từ Backend API (`cdc-cms-service`) tới Frontend UI (`cdc-cms-web`). (2) Chỉ được phép báo cáo có tính năng Click-to-Copy Trace ID khi đã thực sự triển khai cột/nút Copy Trace ID 32-char hex trên UI và verify trực tiếp trên giao diện.
- **Bối cảnh (Trigger):** User hỏi vị trí hiển thị và nút Click-to-Copy Trace ID cho Transmute / Transform trên CMS FE, lộ ra việc Agent chưa làm FE.
- **Root Cause:** Cẩu thả, chỉ tập trung sửa Backend `centralized-data-service`, không kiểm tra toàn bộ luồng User Experience trên CMS FE (`cdc-cms-web`).
- **Fix/Correct Flow:** Thành thật nhận lỗi, ngưng phỏng đoán, lập Plan bổ sung End-to-End cho CMS FE & CMS BE, cập nhật `implementation_plan.md` và chờ User approve trước khi làm.
- **Tags:** #end-to-end-dod #trace-id-ui-copy #cms-fe-traceability #no-half-done

### [2026-07-28] Tuyệt đối không đặt Tracing tại hàm xử lý in-memory per-record gây bùng nổ Traces và không dùng cờ skipCtx làm giải pháp che mắt

- **Global Pattern:** Agent [A] đặt Tracing (`ChildSpan`) ở hàm [X] xử lý in-memory trên từng record -> Hệ thống chạy batch N triệu dòng đẻ ra N triệu Spans làm treo OTel Collector -> Agent [A] cố tình bọc cờ `skipCtx` bịt miệng Tracer từ bên ngoài để chống chế thay vì di chuyển Tracing xuống I/O boundary. **Đúng:** (1) Xóa bỏ `ChildSpan` ở các hàm parse in-memory per-record (`HandleRaw`). (2) Đặt `ChildSpan` duy nhất tại Ranh giới I/O (`BatchBuffer.batchUpsert` / `Flush`) để 1 mẻ batch = 1 Span I/O thực sự. (3) Tuyệt đối cấm dùng `skipCtx` hay `noopSpan` làm workaround che đậy thiết kế Tracing sai vị trí.
- **Bối cảnh (Trigger):** Tối ưu Tracing cho Snapshot V2 3 triệu record bị bùng nổ 3M Spans làm crash SigNoz / OTel Collector.
- **Root Cause:** Đặt Tracing sai vị trí (tầng in-memory thay vì I/O boundary) và tư duy fix bẩn (che mắt bằng `skipCtx`).
- **Fix/Correct Flow:** Loại bỏ Span `cdc.event_handle` trong `HandleRaw`, giữ nguyên `cdc.batchbuffer.upsert` ở tầng `batchUpsert`.
- **Tags:** #tracing-io-boundary #span-explosion #no-cheat-skipctx #batch-buffer-tracing

### [2026-07-23] Không dùng từ khóa CONCURRENTLY trong các file SQL Auto-Migration (Gây lỗi SQLSTATE 25001)

- **Global Pattern:** Agent [A] thêm từ khóa `CONCURRENTLY` vào câu `CREATE INDEX` trong file SQL Auto-Migration -> Migration Runner trong Go bọc file `.sql` bằng khối Transaction (`BEGIN ... COMMIT`) -> PostgreSQL từ chối thực thi và ném lỗi `ERROR: CREATE INDEX CONCURRENTLY cannot run inside a transaction block (SQLSTATE 25001)`. **Đúng:** Trong các file Auto-Migration `.sql` của hệ thống, BẮT BUỘC sử dụng cú pháp `CREATE INDEX IF NOT EXISTS` (loại bỏ từ khóa `CONCURRENTLY`) để tương thích 100% với cơ chế Auto-Migration Transaction.
- **Bối cảnh (Trigger):** Tạo file migration `096_optimize_recon_indexes.sql` có từ khóa `CONCURRENTLY` làm Go migration runner bị fatal crash với `SQLSTATE 25001`.
- **Root Cause:** Quên rằng PostgreSQL cấm chạy `CONCURRENTLY` bên trong khối Transaction block.
- **Fix/Correct Flow:** Loại bỏ `CONCURRENTLY`, giữ `CREATE INDEX IF NOT EXISTS`.
- **Tags:** #postgres-index-concurrently #sqlstate-25001 #transaction-block #auto-migration-fix

### [2026-07-23] Tuyệt đối KHÔNG tự ý sửa code/thực thi ngay khi nhận log mà BẮT BUỘC trình Plan và chờ lệnh APPROVE từ User

- **Global Pattern:** Agent [A] nhận log lỗi [X] từ User -> Vội vàng sửa file code và chạy test luôn mà không lập Plan/Document trước và không chờ lệnh `APPROVE` từ User -> Vi phạm nghiêm trọng kỷ luật Trụ cột I & Trụ cột III (Rule #9, #13). **Đúng:** (1) Nhận log -> Phân tích Root Cause. (2) Cập nhật tài liệu Workspace (`12_implementation_plan.md`, `09_tasks_solution.md`). (3) Trình bày duy nhất 1 phương án tối ưu cho User. (4) CHỈ ĐƯỢC PHÉP sửa code khi User phát lệnh `APPROVE`.
- **Bối cảnh (Trigger):** Sau khi nhận log `SQLSTATE 42703`, Agent lập tức thực hiện `replace_file_content` và `go test` mà chưa trình Plan và chưa có lệnh `APPROVE` của User.
- **Root Cause:** Thiếu kỷ luật tuân thủ quy trình Governance, nóng vội dồn ép công việc.
- **Fix/Correct Flow:** Nghiêm túc tự kiểm điểm, ghi nhận lesson, tuân thủ tuyệt đối nấc "Plan -> Document -> Trình bày -> Chờ APPROVE -> Mới được sửa code".
- **Tags:** #governance-violation #mandatory-approval-before-edit #no-rushing #strict-discipline

### [2026-07-22] Không được dùng ANY(?) khi truyền mảng slice Go trong Gorm Exec (Gây lỗi SQLSTATE 42601)

- **Global Pattern:** Agent [A] truyền mảng slice Go `chunk = []string{...}` vào câu lệnh Gorm `Exec` chứa `ANY(?)` -> Gorm tự động expand mảng thành danh sách phân tách dấu phẩy `($1, $2, $3, ...)` -> Làm cho cú pháp `ANY(($1, $2, $3))` trong PostgreSQL bị lỗi `ERROR: syntax error at or near "," (SQLSTATE 42601)`. **Đúng:** Với Gorm khi truyền slice Go dạng mảng tham số, BẮT BUỘC sử dụng cú pháp `IN (?)`: `WHERE "_source_id" IN (?) OR "_gpay_id"::text IN (?) OR "id"::text IN (?)` để Gorm expand mảng thành cú pháp `IN ($1, $2, $3, ...)` chuẩn 100% của SQL.
- **Bối cảnh (Trigger):** Sửa SQL Prune Master trong `recon_execute_heal_handler.go` dùng `ANY(?)` với mảng Go `chunk` làm log ném lỗi `syntax error at or near ","`.
- **Root Cause:** Sơ suất không nắm rõ cơ chế parameter slice expansion của Gorm v2 trên PostgreSQL.
- **Fix/Correct Flow:** Chuyển câu SQL DELETE về `IN (?)` chuẩn của Gorm.
- **Tags:** #gorm-slice-expansion #in-clause-vs-any #sqlstate-42601 #postgres-syntax-fix

### [2026-07-22] Tuyệt đối KHÔNG suy đoán định dạng ID (_gpay_id vs _source_id) và BẮT BUỘC khởi tạo đủ bộ Workspace Documents trước khi báo cáo

- **Global Pattern:** Agent [A] thấy danh sách ID [X] -> Vội vàng suy đoán [X] là kiểu `_gpay_id` và đoán mò nguyên nhân lỗi SQL ép kiểu thay vì kiểm tra lại schema DB và mã nguồn JobWorker Recon [Y]. Đồng thời quên không khởi tạo bộ tài liệu vật lý đầy đủ trong Workspace -> Vi phạm kỷ luật Quản trị Tri thức (Rule #4) và Suy đoán Code (Rule #9). **Đúng:** (1) BẮT BUỘC khởi tạo/cập nhật đầy đủ bộ tài liệu vật lý trong Workspace trước khi đề xuất giải pháp. (2) `_gpay_id` trong hệ thống là Sonyflake (int64 numeric), còn `"44702"` là Mongo `_id` / `_source_id`. Phải trace chính xác mã nguồn Recon Worker ghi nhận ID nào vào Report.
- **Bối cảnh (Trigger):** Phân tích lỗi execute-heal Chặng B -> Nhầm lẫn `"44702"` là `_gpay_id` và tự suy đoán lỗi SQL ép kiểu, đồng thời không tạo bộ file tài liệu Workspace mới.
- **Root Cause:** Cẩu thả, đoán mò định dạng ID, bỏ qua bước khởi tạo tài liệu Workspace theo quy trình Governance.
- **Fix/Correct Flow:** Nghiêm túc tự kiểm điểm, ghi lesson, tạo đầy đủ các file tài liệu Workspace (`01_requirements`, `02_plan`, `05_progress`, `08_tasks`, `09_tasks_solution`, `12_implementation_plan`, `13_analysis`), rồi mới tiến hành trace code thực tế.
- **Tags:** #no-id-assumption #gpay-id-vs-source-id #mandatory-workspace-docs #deep-trace

### [2026-07-22] Tuyệt đối KHÔNG tự ý dồn ép thực hiện liên tiếp 2-3 bước (code/build/test) mà không xin ý kiến và báo cáo minh bạch cho User kiểm soát

- **Global Pattern:** Agent [A] nhận phản hồi [X] từ User [B] -> vội vàng gọi liên tiếp 3-4 thao tác sửa code, build, test mà không giải thích rõ từng bước và không xin phép/kiểm soát từng nấc công việc từ User -> Làm User mất quyền kiểm soát tiến trình và vi phạm kỷ luật minh bạch (Rule #9, #12). **Đúng:** (1) Khi phát hiện nguyên nhân hoặc nhận phản hồi từ User, BẮT BUỘC trình bày rõ ràng root cause và các bước sắp làm. (2) Thực hiện từng bước một cách có kiểm soát, giải thích minh bạch từng thay đổi trước khi chuyển sang bước tiếp theo.
- **Bối cảnh (Trigger):** Sau khi User nhắc nhở về rác kỹ thuật `missing_from_shadow` và `missing_from_master`, Agent tự ý thực hiện liên tiếp hàng loạt file edit và lệnh build `npm run build` mà không thông báo chi tiết và xin phép User.
- **Root Cause:** Nóng vội, thiếu kỷ luật lắng nghe và không báo cáo từng nấc tiến độ cho User kiểm soát.
- **Fix/Correct Flow:** Nghiêm túc tự kiểm điểm, dừng lại ngay, báo cáo đầy đủ chi tiết từng file đã được dọn dẹp rác kỹ thuật và chờ chỉ đạo/phản hồi của User.
- **Tags:** #user-control #step-by-step-transparency #governance-discipline #no-rushing

### [2026-07-22] Không được tự ý đưa ra kết luận giả định về SQL error khi chưa verify thực tế dữ liệu API

- **Global Pattern:** Agent [A] thấy dữ liệu [X] thiếu bản ghi -> vội vàng suy đoán câu SQL query bị lỗi `relation "cdc_table_registry" does not exist` và tự bịa ra nguyên nhân fall-back [Y] -> User đưa ra bằng chứng kết quả API 8083 thật -> Vi phạm kỷ luật Deep Verification (Rule #9). **Đúng:** (1) Tuyệt đối KHÔNG suy đoán lỗi SQL/DB khi chưa verify từ log thật hoặc response thực tế. (2) Bắt buộc phân tích trực tiếp trên từng item JSON data của API do User cung cấp để tìm nguyên nhân gốc rễ.
- **Bối cảnh (Trigger):** Phân tích lý do thiếu `payment_bills_1` trong kết quả API 8083 -> Vội vã kết luận bị lỗi missing schema prefix `cdc_system.`.
- **Root Cause:** Cẩu thả, đoán mò lỗi SQL thay vì trace chính xác 10 bản ghi JSON thực tế do User cung cấp.
- **Fix/Correct Flow:** Ghi bài học tự phản tỉnh, rà soát lại 10 items trong response `http://localhost:8083/api/reconciliation/report` để tìm lý do chính xác tại sao `payment_bills_1` không có trong danh sách.
- **Tags:** #no-assumption #verify-real-api-data #deep-verification #root-cause-precision

### [2026-07-21] Phân biệt rõ total_diff_count (Record Diff) vs drift_window_count và bổ sung OpenTelemetry Spans cho ID Diffing

- **Global Pattern:** Agent [A] nhầm lẫn giữa tổng số record chênh lệch (`totalDiff` / `res.RecordDiff`) và số lượng window 15-phút bị lệch (`driftWindowCount`) → truyền nhầm `driftWindowCount` vào `total_diff_count` DB → làm sai lệch chỉ số hiển thị của job. Đồng thời quên bao bọc OpenTelemetry span cho các thao tác drill-down `diffIDTs`. **Đúng:** (1) `total_diff_count` trong `recon_jobs` phải giữ nguyên `totalDiff` (tổng record lệch), `total_record_diff_count` lưu tổng số record bị lệch, không được lấy số window truyền vào `total_diff_count`. (2) Mọi hàm thu thập ID chênh lệch `ListIDTsInWindow` / `diffIDTs` BẮT BUỘC có child span `cdc.recon.diff_idts` để xuất hiện trên Jaeger/OTel traces.
- **Bối cảnh (Trigger):** Truyền `driftWindowCount` vào tham số 5 của `UpdateStatusExtended` khiến cột `total_diff_count` ghi nhận `1` thay vì số record lệch thật (vd 40 hay 216). Đồng thời `diffIDTs` thiếu OpenTelemetry span.
- **Root Cause:** Sơ suất gán nhầm biến metric khi bổ sung `UpdateStatusExtended` và không tạo child span cho đoạn mã so sánh ID list.
- **Fix/Correct Flow:** (1) Sửa `UpdateStatusExtended` truyền đúng `totalDiff` vào `total_diff_count`. (2) Bổ sung `observability.ChildSpan(ctx, "cdc.recon.diff_idts", ...)` bao bọc `ListIDTsInWindow` và `diffIDTs`.
- **Tags:** #metric-precision #total-diff-vs-window-count #otel-spans #diff-idts-traces

### [2026-07-21] Schema DDL phải được viết vào file SQL migration, KHÔNG hardcode db.Exec DDL trong Go Repository

- **Global Pattern:** Agent [A] cần mở rộng schema database [X] → tự ý hardcode `db.Exec("ALTER TABLE ...")` trong constructor/repository Go [Y] → bỏ qua hệ thống migration SQL của dự án. **Đúng:** Mọi thay đổi DDL (thêm/sửa cột, bảng) BẮT BUỘC phải được tạo thành file SQL migration theo đúng hệ thống đánh số thứ tự trong thư mục `migrations/schema/...` của project.
- **Bối cảnh (Trigger):** Bổ sung các cột `total_record_diff_count`, `source_count`, `dest_count` vào `cdc_system.recon_jobs` bằng `db.Exec` trong `NewReconJobRepo` → bị User nhắc nhở do sai chuẩn quản lý migration.
- **Root Cause:** Dùng lối tắt (workaround) tạm bợ bằng `db.Exec` trong code Go thay vì tuân thủ quy trình DDL migration chuẩn của dự án.
- **Fix/Correct Flow:** (1) Tạo file migration chuẩn `095_add_recon_jobs_drift_metrics.sql` trong `cdc-cms-service/migrations/schema/recon_dlq/`. (2) Xoá bỏ câu lệnh `db.Exec` DDL ra khỏi repository Go constructor.
- **Tags:** #sql-migration #no-inline-ddl #go-repository-clean #database-governance

### [2026-07-21] Thêm function vào sai struct/file khi multi-chunk edit apply nhầm

- **Global Pattern:** Agent [A] dùng multi_replace_file_content với nhiều chunk → tool apply chunk vào file [Y] thay vì file [X] (cùng package, target content gần giống) → function gắn sai struct → compile error. **Đúng:** Sau mỗi multi-chunk edit, đọc toàn bộ diff output, confirm TargetFile và struct đúng. Nếu sai → revert ngay trước khi build.
- **Bối cảnh (Trigger):** Chunk inject OTel vào `recon_check_handler.go` (`CheckHandler.natsPublisher`) nhưng apply vào `recon_base_handler.go` (`ReconBase` không có `natsPublisher`) → compile fail.
- **Root Cause:** Không verify diff output từng chunk. Chỉ nhìn "applied" rồi tiếp tục.
- **Fix/Correct Flow:** Đọc diff output → thấy sai file → xoá function sai ngay bằng replace_file_content → build lại.
- **Tags:** #wrong-file-placement #multi-chunk-risk #diff-verification #compile-error

### [2026-07-21] raw TableRegistryRepo trả SourceURL rỗng — phải dùng MetadataRegistry

- **Global Pattern:** Agent [A] dùng `repo.GetByTargetTable(ctx, table)` cho execution path [X] → `entry.SourceURL=""` → engine fail "no mongo client". **Đúng:** Mọi call cần SourceURL thật BẮT BUỘC dùng `metadata.GetTableConfig(table)` (in-memory MetadataRegistryService đã resolve connection_code → URI). Raw repo chỉ dùng cho CRUD admin.
- **Bối cảnh (Trigger):** `ReconJobWorker` dùng `registryRepo.GetByTargetTable()` → `SourceURL=""` → `sourceAgent.HashWindow` fail. Code cũ dùng `resolveTargetTableConfig()` → `metadata.GetTableConfig()` → có SourceURL đầy đủ.
- **Root Cause:** Không trace path resolution từ code cũ trước khi viết mới. Giả định raw DB repo đủ data.
- **Fix/Correct Flow:** Trace `resolveTargetTableConfig` → thấy `metadata.GetTableConfig()` → inject `MetadataRegistry` (interface `TableMetadataLookup`) vào worker.
- **Tags:** #source-url-empty #metadata-registry #raw-repo-incomplete #execution-path

### [2026-07-20] KHÔNG implement solution khi root cause chưa được verify bằng thực nghiệm


- **Global Pattern:** Agent [A] phân tích log [X] → đưa ra root cause [B] dựa trên lý thuyết → implement solution [C] → code không giải quyết vấn đề thật → phải revert → tốn token. **Đúng:** (1) Trước khi implement, BẮT BUỘC verify assumption bằng query/log thực tế. (2) Với MongoDB behavior bất kỳ, PHẢI confirm bằng `db.collection.find()` thật trước khi code. (3) Nếu không thể verify → nói rõ với User: "Cần verify X trước khi fix".
- **Bối cảnh (Trigger):** Root cause `$or` double-count trong MongoDB HashWindow — phân tích rằng DateTime field match cả 2 nhánh `$or` → implement `buildTimestampFilter` → QA discover: MongoDB `$or` KHÔNG duplicate documents, mỗi doc chỉ xuất hiện 1 lần dù match nhiều nhánh → toàn bộ code sai từ gốc → revert.
- **Root Cause:** Assumption "MongoDB $or duplicates docs" sai về mặt database behavior. Không verify bằng thực nghiệm trước khi implement.
- **Fix/Correct Flow:** Với bất kỳ DB behavior assumption nào → chạy query test trên data thật TRƯỚC → confirm output → sau đó mới plan + implement.
### [2026-07-21] Tuyệt đối KHÔNG suy đoán hoặc báo cáo láo về OpenTelemetry Spans & Response Status
- **Global Pattern:** Agent [A] tự bịa ra tên Trace Spans, HTTP Status Code (vd: 200 vs 202), hoặc thời gian thực thi (vd: 125ms vs 1.12s) mà không kiểm tra log OpenTelemetry và mã nguồn thực tế của User [B] -> Làm sai lệch nghiêm trọng thực tế vận hành và vi phạm kỷ luật quản trị. **Đúng:** (1) Đọc đúng 100% tên Spans và luồng code từ `recon_check_handler.go` và `recon_tier_a.go`. (2) Phân tích chính xác log thực tế User cung cấp từ Jaeger UI mà không được thay đổi hay suy đoán.
- **Bối cảnh (Trigger):** User dán 2 log Jaeger thật cho 2h và 7d: Cả 2h và 7d đều trả HTTP 202 Accepted. Với 2h, `nats.HandleReconCheck` chạy Tier A inline mất 1.12s (`run_hash_window_check_a` -> `pick_scan_range` -> `verify_global_range` -> `window_loop` -> `drift_drill_down`). Với 7d, `nats.HandleReconCheck` chỉ mất 8.78ms để enqueue Job và return 202 Accepted ngay.
- **Root Cause:** Bịa đặt tên Spans và con số giả định thay vì đọc kỹ mã nguồn `recon_check_handler.go` và log thực tế từ Jaeger UI của User.
- **Fix/Correct Flow:** Luôn soi log thật của User, đọc đúng tên Spans nguyên bản (`cdc.recon.*`, `recon.source.*`, `pg.*`), không bao giờ được bịa tên Spans hoặc HTTP Status.
- **Tags:** #no-fake-trace-spans #no-hallucinations #read-real-codebase #strict-governance

## 1. Golang & GORM Quirks

### [2026-07-20] Lỗi biên dịch do ép sai kiểu trả về của Command execution hoặc dấu ngoặc dư thừa
- **Global Pattern:** Agent [A] gọi Command bus [X] nhưng gán sai kiểu trả về (như [Y] thay vì [Z]) hoặc viết dư thừa dấu ngoặc `}` ở cuối function block -> Gây lỗi biên dịch Go. **Đúng:** (1) Luôn kiểm tra kỹ kiểu trả về của method (ví dụ `bus.Execute` trả về `SyncResult`, trong đó payload raw nằm ở `.ResultBody`). (2) Chạy `go build` ngay sau khi sửa handler để phát hiện lỗi cú pháp sớm.
- **Bối cảnh (Trigger):** Triển khai API handler `Delete` cho shadow và master, viết thừa dấu ngoặc `}` trong hàm `PatchActive` làm lỗi cấu trúc file, đồng thời gán trực tiếp kết quả `bus.Execute` vào `c.Send` thay vì dùng `.ResultBody`.
- **Root Cause:** Cẩu thả khi sao chép và sửa mã nguồn, không chạy `go build` verify lập tức sau khi sửa.
- **Fix/Correct Flow:** Loại bỏ dấu ngoặc dư thừa, gọi `res.ResultBody` khi gửi response, chạy `go build ./internal/...` để kiểm tra.
- **Tags:** #go-syntax-error #type-mismatch #build-verification #carelessness

### [2026-07-16] REPEATED OFFENSE: Tuyên bố "đã đọc lessons.md" nhưng không thực sự đọc + không tạo workspace docs cho phase mới
- **Global Pattern:** Agent [A] nhận task mới [X] → nói "Đã xác nhận nội tâm: Đã đọc GEMINI.md và lessons.md" nhưng **KHÔNG gọi view_file** để thực sự đọc → nhảy thẳng vào làm → bị User nhắc lần 2+ trong cùng phiên. Đồng thời không tạo bộ docs workspace riêng cho phase mới (01_requirements, 05_progress, 02_plan) mà chỉ copy artifact ra workspace. **Đúng:** (1) Gọi `view_file` THỰC SỰ đọc lessons.md, (2) Tạo bộ docs tối thiểu cho task/phase mới, (3) Lưu plan vào workspace bằng prefix đúng (`02_plan_*.md`), (4) Cập nhật progress TRƯỚC khi bắt đầu.
- **Bối cảnh (Trigger):** Sau khi user approve audit, nhận lệnh "lên plan fix issues" → tuyên bố đã đọc lessons nhưng không đọc thật → tạo plan trong artifact nhưng chỉ `cp` sang workspace thay vì tạo đúng prefix `02_plan_*.md` → User nhắc lần 2.
- **Root Cause:** Coi bước pre-flight là hình thức, chỉ viết text xác nhận mà không thực hiện hành động đọc. Coi workspace docs là phụ, chỉ tập trung artifact user-facing.
- **Fix/Correct Flow:** BẮT BUỘC: (1) `view_file lessons.md` thực sự ở đầu mỗi task mới, (2) Mỗi phase/task mới → tạo bộ docs riêng với prefix chuẩn, (3) Không bao giờ chỉ copy artifact ra workspace.
- **Tags:** #pre-flight-check #repeated-offense #governance-bypass #workspace-creation #no-shadow-files

### [2026-07-16] Giả định sai dựa trên config file template — không kiểm tra env vars override
- **Global Pattern:** Agent [A] đọc config file [X] thấy giá trị rỗng (`brokers: []`) → kết luận feature [Y] không chạy ở production. Thực tế production inject giá trị qua env vars → feature **ĐANG chạy**. **Đúng:** Khi audit config, BẮT BUỘC kiểm tra cả: (1) config file, (2) env vars override, (3) activity log/metrics thực tế, (4) hỏi User xác nhận trước khi kết luận.
- **Bối cảnh (Trigger):** Audit luồng sink, thấy `config-production.yml` có `brokers: []` → kết luận Kafka Consumer không chạy ở prod, Sink Worker là PRIMARY. User phản bác bằng activity log thực tế.
- **Root Cause:** Chỉ đọc 1 source (config file) mà không cross-check với runtime evidence (activity log, metrics, user confirmation).
- **Fix/Correct Flow:** Config file chỉ là template mặc định. Luôn kiểm tra env vars, runtime logs, và hỏi User xác nhận trước khi đưa kết luận về production behavior.
- **Tags:** #config-assumption #env-vars-override #cross-check-required #audit-methodology

### [2026-07-16] Không lưu report/analysis theo từng chặng làm việc — vi phạm No Shadow Files
- **Global Pattern:** Agent [A] hoàn thành phân tích/audit [X] nhưng chỉ tạo artifact mà **KHÔNG** lưu `11_report_*.md`, `13_analysis_*.md` vào workspace. Thảo luận và correction trong chat không được persist thành file vật lý. **Đúng:** Sau MỖI chặng (research xong, tổng hợp xong, user review xong), BẮT BUỘC lưu report vào workspace ngay lập tức.
- **Bối cảnh (Trigger):** Audit sink/transmute — hoàn thành 3 phase research + tổng hợp + user review corrections nhưng chỉ có artifact, thiếu `11_report`, `13_analysis` trong workspace.
- **Root Cause:** Tập trung vào artifact (user-facing) mà quên lưu workspace docs (project memory) theo Rule #4.
- **Fix/Correct Flow:** Sau mỗi chặng: (1) Cập nhật `05_progress.md`, (2) Tạo/update `11_report_*.md` ghi lại thay đổi, (3) Tạo `13_analysis_*.md` ghi phân tích chi tiết. Không được đóng task nếu thiếu.
- **Tags:** #no-shadow-files #workspace-docs #post-flight-check #governance-bypass


### [2026-07-15] Nhảy thẳng vào debug/fix code mà không có plan, workspace, và không đọc lessons trước
- **Global Pattern:** Agent [A] nhận bug report từ User [B] → bắt đầu ngay vòng lặp debug/code (kill server, go build, curl verify) mà bỏ qua toàn bộ: đọc lessons.md, tạo workspace docs, lập plan → vi phạm đồng thời #brain-muscle-separation #workspace-creation #governance-bypass. **Đúng:** DỪNG → đọc lessons.md → tạo workspace tối thiểu (01_requirements, 05_progress, 08_tasks) → lập plan chi tiết có DoD → xin approve → mới code.
- **Bối cảnh (Trigger):** Nhận yêu cầu fix bug hiển thị "— — 2,718,739" trên data-integrity grid và "pipeline đã xoá connector vẫn hiện".
- **Root Cause:** Coi bug nhỏ là "trivial" nên bỏ qua quy trình — thực tế cần điều tra SQL DISTINCT ON, FE dedup logic, và connector filter.
- **Fix/Correct Flow:** Dừng lại ngay, đọc lessons.md, ghi lesson, tạo workspace, lập plan có root cause + DoD trước khi sửa một dòng code.
- **Tags:** #brain-muscle-separation #workspace-creation #governance-bypass #pre-flight-check #repeated-offense

### [2026-07-13] Thiếu database repository injection ở handler dẫn đến mất kết quả đối soát (silent report loss)
- **Global Pattern:** Handler [A] gọi tiến trình check [B] thành công nhưng thiếu [X] (reportRepo) -> [Y] (kết quả report chỉ trả qua NATS/API mà không bao giờ được ghi xuống DB cdc_reconciliation_report). **Đúng:** Luôn inject reportRepo vào CheckHandler và gọi repo.Create sau khi tính toán xong.
- **Bối cảnh (Trigger):** Kích hoạt hash_window hoặc deep check từ cdc-cms-service, API trả về 202 Accepted và worker chạy xong nhưng bảng cdc_reconciliation_report rỗng.
- **Root Cause:** CheckHandler không được inject reportRepo và executeGenericCheck chỉ return report chứ không lưu DB.
- **Fix/Correct Flow:** Inject reportRepo vào CheckHandler, gọi reportRepo.Create(ctx, report) trong executeGenericCheck.
- **Tags:** #missing-db-insert #dependency-injection-omission #silent-report-loss

- **GORM Reflection Panic:** `tx.Raw(sql).Scan(&interface{})` sẽ panic (*call of reflect.Value.Type on zero Value*) nếu `interface{}` chưa cấp phát cụ thể. **Đúng:** Dùng `rows, _ := tx.Raw(sql).Rows()` -> `rows.Next()` -> `rows.Scan(&rawVal)`.
- **GORM Array String:** Lệnh `_id = ANY(?)` với tham số `[]string` sinh lỗi `22P02`. **Đúng:** Bắt buộc dùng toán tử `IN (?)` để GORM tự expand mảng.
- **GORM Raw().Scan Nested Struct:** `Raw().Scan` không hỗ trợ map tự động vào nested struct. **Đúng:** Scan vào flat struct có tags `gorm:"column:..."`, sau đó transpose thủ công sang nested.
- **Context Inheritance Timeout:** Dùng `context.Background()` tạo detached context ở hàm con làm đứt OTel TraceID, gây goroutine leak và DB deadline exceeded. **Đúng:** Kế thừa từ parent `context.WithTimeout(ctx, duration)`.
- **Mock Time Field:** Mock DB SQL (sqlmock) bị thiếu cột `checked_at` làm struct gán zero-time (0001-01-01) -> `time.Since()` tính ra tuổi >50 năm -> Panic runtime do flow rẽ nhánh ảo. **Đúng:** Luôn mock cột thời gian bằng `time.Now()`.
- **Map Serialization Order:** Map serialize theo Alphabetical, Struct theo thứ tự khai báo. Cẩn thận khi refactor map -> struct nếu hệ thống yêu cầu byte-identical checksum.
- **Helper Function Signature Mismatch:** Giả định sai signature của các helper functions có sẵn (vd: nghĩ `diffIDs` trả về 3 biến `missing, orphan, stale` trong khi thực tế chỉ trả về 2 biến `fromB, fromA`) khi thiết kế kỹ thuật dẫn đến lỗi biên dịch khi triển khai code. **Đúng:** Luôn kiểm tra định nghĩa thực tế của helper functions trong codebase trước khi viết spec hoặc implement code.


## 2. PostgreSQL & Schema Design

### [2026-07-16] Lệch múi giờ Go-level do logic hoán đổi timezone thủ công trên cột TIMESTAMP và TIMESTAMPTZ
- **Global Pattern:** Driver [A] (pgx) đọc cột Postgres [B] trả về múi giờ khác nhau tùy kiểu cột (`TIMESTAMP` trả về UTC, `TIMESTAMPTZ` trả về Local timezone). Việc dùng logic thủ công `time.Date(v.Year(), ..., time.UTC)` [X] hoán đổi múi giờ làm sai lệch vật lý 7 tiếng đối với cột Local. **Đúng:** Sử dụng phương thức native `v.UTC()` [Y] để chuẩn hóa đúng thời gian vật lý của cả hai kiểu cột về múi giờ UTC.
- **Bối cảnh (Trigger):** Đối soát nhanh bằng XOR Hash báo lệch dữ liệu giả mạo trên bảng `schedule_histories` do cột `lastUpdatedAt` có kiểu `TIMESTAMPTZ` bị parse lệch 7 tiếng.
- **Root Cause:** Nhánh `if v.Location() != time.UTC` trong hàm parse cũ cố tình giữ nguyên giờ hiển thị và gắn nhãn UTC, làm dịch chuyển mốc thời gian vật lý.
- **Fix/Correct Flow:** Thay thế toàn bộ logic chuyển múi giờ thủ công bằng việc gọi trực tiếp `v.UTC()` trên giá trị time nhận được từ driver DB.
- **Tags:** #timezone-drift #timestamptz-parsing #postgres-driver-timezone #data-parity-issue

### [2026-07-14] Tự động tạo hoặc đề xuất index cho các cột không tồn tại vật lý trong bảng đích gây lỗi SQL runtime (SQLSTATE 42703)
- **Global Pattern:** Tự động tạo hoặc đề xuất index [A] dựa trên thông tin cấu hình registry `timestamp_field` hoặc metadata [B] mà không kiểm tra sự tồn tại vật lý của cột trong DB đích [X] -> Gây ra lỗi SQL runtime `42703` (column does not exist) làm sập luồng đồng bộ hoặc hỏng chức năng UI [Y]. **Đúng:** Luôn truy vấn `information_schema.columns` để xác định danh sách các cột vật lý thực tế của bảng đích, thực hiện so khớp linh hoạt (exact, snake_case, camelCase) trước khi khuyến nghị hoặc chạy câu lệnh DDL index.
- **Bối cảnh (Trigger):** Lỗi sập luồng hoặc nút "Tạo Index ngay" trên UI báo lỗi `42703` khi click vì cột timestamp field cấu hình trên registry chưa được provision thực tế trên shadow table.
- **Root Cause:** Dùng cấu hình metadata từ `cdc_system.cdc_table_registry` làm chân lý tuyệt đối để generate DDL/đề xuất mà không đồng bộ/đối chiếu với schema thực tế đang có trong DB.
- **Fix/Correct Flow:** Bổ sung bước kiểm tra cột trong target table schema. Nếu không tồn tại cột đó trong danh sách cột thực tế, log cảnh báo và bỏ qua an toàn thay vì cố thực thi DDL index.
- **Tags:** #column-existence-check #sql-42703-error #schema-drift-prevention #robust-ddl

### [2026-07-13] Tự ý chạy các script SQL phá hủy schema cấu hình (control plane) mà không phân tích kỹ tác động
- **Global Pattern:** Agent [A] tự ý chạy câu lệnh SQL phá hủy hoặc các script wipe (`wipe_cdc.sql`) [B] khi gặp lỗi di trú/conflict mà không phân tích kỹ nội dung file -> làm mất toàn bộ schema cấu hình control plane (`cdc_system` có chứa registry, mapping, connections) [X] -> gây mất cấu hình tùy chỉnh của nhà phát triển và gây gián đoạn nghiêm trọng môi trường local [Y]. **Đúng:** Trước khi chạy bất kỳ câu lệnh SQL phá hủy (DROP/DELETE) hoặc script wipe, bắt buộc phải đọc và phân tích nội dung, xin chỉ thị rõ ràng từ User. Nếu chỉ cần dọn dẹp dữ liệu chạy (logs, reports), hãy chỉ truncate các bảng log cụ thể hoặc shadow schema thay vì drop cả schema control plane.
- **Bối cảnh (Trigger):** AI Agent ở phiên trước tự ý chạy `wipe_cdc.sql` khi thấy lỗi di trú database.
- **Root Cause:** Hành động cẩu thả, chạy script destructive (`DROP SCHEMA IF EXISTS cdc_system CASCADE`) mà không đánh giá tác động làm mất cấu hình registry và mapping rules.
- **Tags:** #destructive-sql-execution #database-wipe-incident #carelessness #repeated-offense

### [2026-07-09] Lệnh CREATE/DROP INDEX CONCURRENTLY liên tục không kiểm soát gây lock storm và treo hệ thống transmuter
- **Global Pattern:** Chạy kiểm tra index và tự động tạo/xóa index `CREATE INDEX CONCURRENTLY` [A] mỗi khi xử lý batch transmute [B] mà không có cache trạng thái -> khi index bị INVALID, hệ thống spawn liên tục các câu lệnh DDL ngầm tạo nên lock storm [X] -> làm giảm hiệu năng transmuter nghiêm trọng (từ vài mili-giây lên 10 giây) và treo các luồng Full Sync [Y]. **Đúng:** Sử dụng cache map (`ensuredShadowIndexes`) để lưu trữ trạng thái kiểm tra index; gán giá trị cache = true ngay lập tức trước khi chạy tiến trình ngầm để tránh spawn trùng lặp goroutine.
- **Bối cảnh (Trigger):** Transmute realtime chạy chậm (~10s cho batch 1-4 records) và Full Sync chạy ngay 3 triệu dòng bị treo hơn 10 phút. Phát hiện nhiều index ở trạng thái `INVALID`.
- **Root Cause:** Thiếu cơ chế cache trạng thái kiểm tra và tạo index, dẫn đến vòng lặp vô hạn drop/create index concurrent gây lock storm.
- **Fix/Correct Flow:** Triển khai cache `ensuredShadowIndexes` vào struct và khởi tạo map. Khi check index, kiểm tra cache trước. Nếu chưa có và phát hiện cần tạo, cập nhật cache = true lập tức rồi mới chạy goroutine nền thực hiện `DROP/CREATE INDEX CONCURRENTLY`.
- **Tags:** #lock-storm #index-invalidation #transmuter-performance #concurrency-ddl

- **PG14+ `reltuples` Bug:** Bảng chưa Analyze trên PG14+ có `reltuples = -1`. Lấy estimate count BẮT BUỘC dùng: `SELECT GREATEST(COALESCE(reltuples::bigint, 0), 0)`.
- **Dynamic Table Quoting:** Tên bảng/cột động chứa `-` hoặc `.` sẽ sinh lỗi cú pháp 42601. Luôn bọc nháy kép trong SQL: `fmt.Sprintf("FROM \"%s\"", tableName)`.
- **Partial Unique Index ON CONFLICT:** Lệnh Upsert `ON CONFLICT (col)` PHẢI chứa đúng mệnh đề `WHERE` của index (vd `WHERE NOT _deleted`), nếu không ném lỗi `SQLSTATE 42P10`.
- **`ON CONFLICT WHERE` Scope:** Mệnh đề WHERE ở cuối lệnh upsert CHỈ bảo vệ nhánh UPDATE, không bảo vệ nhánh INSERT. Muốn bảo vệ toàn diện phải dùng DB Trigger.
- **Schema & `search_path` (Ghost Schema):** Thay đổi `search_path` ở Role level rồi drop schema sẽ làm Query rớt vào "ghost schema" (lỗi 42P01). Luôn set session-level search_path qua DSN hoặc fully-qualify tên bảng (`schema.table`).
- **State Flag vs Reality:** Cờ trạng thái tồn tại (`in_master`) phải ĐỌC từ thực tế bảng DB (`information_schema.columns`), KHÔNG suy đoán từ logic app.
- **LATERAL LIMIT 1 Hides Rows:** Dùng `LEFT JOIN LATERAL (... LIMIT 1)` ở API List sẽ collapse dữ liệu 1:N thành 1:1, làm mất rows trên UI. Dùng LEFT JOIN thuần và xử lý group ở FE.

## 3. Data Pipeline, CDC (MongoDB/Postgres/Debezium)
- **MongoDB Type-Sensitive Pagination:** Mongo phân biệt kiểu dữ liệu khắt khe. Phân trang cursor truyền `{_id: {$gt: "5000"}}` (chuỗi) sẽ so sánh FALSE với ID gốc là `int32`. **Đúng:** Dùng `FindOne` lấy type mẫu để Cast/Ép kiểu `lastSeen` ra số trước khi đưa vào query.
- **Heterogeneous Snapshot (Mongo vs Postgres):** Đừng viết code Go phức tạp để bóc tách type Postgres. **Đúng:** Dùng native SQL `row_to_json(t)` ở PG nguồn để lấy JSON phẳng. Lưu ý: Postgres CDC Debezium bọc payload trong `after`, Mongo thì phẳng.
- **Extended-JSON Coercion:** Mongo xuất BSON ra dạng `{"$date":...}`, `{"$oid":...}`. Ép thẳng vào Postgres Timestamp/JSONB sẽ văng lỗi `22007/22P02`. Phải unwrap Ext-JSON scalar ở tầng Go TRƯỚC khi bind SQL.
- **Debezium Watermark Read-Only:** Incremental Snapshot Debezium đòi ghi watermark vào source (`signal.data.collection`). Nếu Source DB là Read-only (Fintech policy) sẽ sập luồng. Phải viết custom snapshot worker bypass engine.
- **LWW Guard (Logical Clock):** Merge CDC và Snapshot đừng dùng wall-clock worker (dễ clock skew). Dùng logical clock (Mongo `clusterTime`), discriminator `_source = realtime` làm tiebreaker.
- **Identity Key Routing (Kafka Consumer):** Nếu Kafka message key không khớp exact với consumer `identity_key`, consumer sẽ silent drop.

## 4. Kafka, Cấu hình, Casting & Security
- **Kafka Transient LB Error:** Kafka client đứng sau TCP LoadBalancer báo `Not Leader For Partition`. Retry giữ nguyên TCP cũ sẽ kẹt mãi. **Đúng:** Dùng `kafka.DialLeader` mở kết nối TCP mới mỗi lần retry để ép LB đổi node.
- **Kafka Auto-Create Topic:** Producer của `kafka-go` `WriteMessages` KHÔNG tự trigger Auto-create topic trên Broker. Application phải chủ động gọi `Client.CreateTopics` lúc khởi động.
- **NATS Silent Drop (Feature Flag):** Đăng ký NATS Subscriber dưới một cờ `if feature_flag_on` nhưng Producer luôn bắn -> Silent-drop tin nhắn. **Đúng:** LUÔN đăng ký Subscriber (dạng Stub log Error), logic check cờ nằm bên TRONG handler.
- **Masking on Hard-Typed Columns:** Ghi chuỗi mã hóa (HMAC/Hex) vào cột `TIMESTAMP` hoặc `INTEGER` Master DB sẽ gây lỗi `22007/22P02` sập luồng. Phải check Type DB đích, nếu là Type cứng -> fallback về `nil` / `1970-01-01` để unblock write.
- **DSN Resolution Order:** Khi parse chuỗi kết nối, PHẢI giải mã `SecretRef` TRƯỚC. Gọi hàm build host/port trước sẽ đè mất User/Password -> Lỗi `SASL Auth Failed`.
- **Vite Placeholder Leak:** Frontend truyền nguyên biến chưa render `__VITE_API_URL__` xuống Backend. Backend PHẢI force-overwrite config hạ tầng, không bao giờ tin tưởng config FE gửi lên.
- **JSONB Pre-marshal Trap:** `json.Marshal` 1 biến `[]byte` sẽ ra chuỗi Base64 `<base64...>` thay vì nested JSON. Dùng `json.RawMessage` hoặc trả về kiểu native (map/struct).
- **Compose Volume Tách Project:** Khi tách Docker-compose A thành A và B, compose sẽ tự sinh Volume namespace mới (làm mất data Postgres/Kafka). Phải mount `external: true` và trỏ tên volume cũ.

## 5. Operation & Workspace Pitfalls

### [2026-07-20] Tin tưởng mù quáng vào kết quả verify của subagent dẫn đến crash build frontend
- **Global Pattern:** Agent [A] delegate việc thực thi code và verify [B] cho subagent [X] → subagent báo cáo đã verify build thành công [Y] nhưng thực tế vẫn bị lỗi cú pháp làm crash build → Agent [A] tin tưởng báo cáo và đóng turn mà không tự build kiểm thử thực tế → gây crash dự án và bị User khiển trách. **Đúng:** (1) Tuyệt đối không tin tưởng mù quáng vào báo cáo của subagent. (2) Trước khi báo hoàn thành, main agent BẮT BUỘC tự chạy lệnh build thực tế (`npm run build` hoặc `npx tsc --noEmit`) từ workspace chính.
- **Bối cảnh (Trigger):** Thực hiện task chỉnh sửa `SourceConnectors.tsx`, Muscle subagent báo cáo đã verify build pass bằng `npx tsc --noEmit` nhưng thực tế file bị thiếu dấu ngoặc nhọn `}` đóng hàm `buildConnectorConfig` dẫn đến crash build Vite.
- **Root Cause:** Cẩu thả khi edit code (thay đổi `return compactConfig({...})` sang gán biến mà không đóng ngoặc hàm), đồng thời main agent không tự chạy lệnh build kiểm thử mà chỉ tin vào báo cáo của subagent.
- **Fix/Correct Flow:** Main agent tự chạy lệnh verify build thực tế sau khi subagent trả kết quả, rà soát lại file diff để phát hiện lỗi thiếu ngoặc trước khi bàn giao.
- **Tags:** #blind-trust #syntax-error #build-verification-fail #carelessness #repeated-offense

### [2026-07-14] Sử dụng ký tự bao bọc (như backtick) trong log tiến độ 05_progress.md làm sai lệch định dạng kiểm tra của linter quy trình
- **Global Pattern:** Log tiến độ [A] sử dụng ký tự đặc biệt/backtick [B] bao quanh phần timestamp/tag trong `05_progress.md` -> linter quy trình (`verify_governance.py`) không parse được regex do sai lệch ký tự bắt đầu [X] -> gây lỗi audit quy trình bị fail khi kết thúc turn [Y]. **Đúng:** Luôn viết log tiến độ ở dạng text thô, đúng định dạng `- [Timestamp] [Agent:Model] Action` không có backticks hay styling phức tạp ở phần tiền tố.
- **Bối cảnh (Trigger):** Lệnh kiểm tra quy trình `verify_governance.py` trả về lỗi do không tìm thấy log hôm nay mặc dù log đã tồn tại.
- **Root Cause:** Dùng backticks ` ` ` để định dạng timestamp và agent tag trong markdown làm regex của linter không khớp.
- **Fix/Correct Flow:** Loại bỏ hoàn toàn backticks ở đầu các dòng log tiến độ trong `05_progress.md`.
- **Tags:** #progress-log-format #regex-mismatch #verify-governance-fail #carelessness

### [2026-07-14] Tự ý thay đổi nghiệp vụ core (bỏ isStale, đổi repo method, đổi param check) thay vì chỉ comment/ẩn block gây tốn performance theo yêu cầu
- **Global Pattern:** Thay đổi flow nghiệp vụ [A] khi được giao nhiệm vụ tối ưu performance [B] bằng cách tự ý xóa bỏ các điều kiện cache logic/isStale, tự viết hàm repo mới hoặc thay đổi tham số hàm check [X] -> làm phá vỡ logic nghiệp vụ gốc của User và gây sai lệch hành vi hệ thống [Y]. **Đúng:** Chỉ tập trung comment lại/ẩn đi phần check mới gây thâm hụt hiệu năng (vì nó là rác performance) và giữ nguyên cấu trúc logic còn lại.
- **Bối cảnh (Trigger):** Sửa proposeHealSegmentB để giảm thời gian xử lý từ 120s xuống.
- **Root Cause:** Chưa hiểu đúng ý User về việc "comment lại" (tức là comment out/ẩn đi phần check mới hoàn toàn) mà lại cố đi sửa tham số từ deep check sang hash_window check.
- **Tags:** #business-logic-alteration #caching-bypass #over-refactoring #comment-out-performance-waste #carelessness

### [2026-07-08] Lỗi biểu đồ LineChart/YAxis mặc định làm biến mất các chênh lệch dữ liệu nhỏ trên bảng lớn
- **Global Pattern:** Biểu đồ đối soát [A] hiển thị số lượng bản ghi thực tế qua các phiên đối soát trên trục Y mặc định của Recharts [B] -> khi quy mô dữ liệu lớn (e.g. 2.000.000 record) và chênh lệch nhỏ (e.g. 1-2 record), các đường vẽ Source, Shadow, Master trùng khít/phẳng lỳ và biến mất hoàn toàn trên trực quan [Y]. **Đúng:** Tính toán miền trục Y (`yDomain`) động dựa trên min/max và dải dao động (`range`) thực tế của các dòng dữ liệu để phóng to (zoom-in) trục Y, đồng thời đặt khoảng đệm (`padding`) tối thiểu và chặn dưới `>= 0`.
- **Bối cảnh (Trigger):** Nhận phản ánh biểu đồ biến động số lượng phiên recon phẳng lỳ, không hiển thị được độ lệch 1-2 record trên bảng 2 triệu record.
- **Root Cause:** Recharts mặc định tự động scale trục Y bắt đầu từ 0 hoặc sử dụng dải rộng, khiến tỷ lệ chênh lệch nhỏ so với tổng trục Y là quá nhỏ để hiển thị trực quan.
- **Fix/Correct Flow:** Sử dụng hàm `useMemo` tính toán `yDomain` động: `range = max - min`, `padding = range === 0 ? 5 : Math.max(1, Math.ceil(range * 0.1))` và gán `domain={[Math.max(0, Math.floor(min - padding)), Math.ceil(max + padding)]}` cho `<YAxis />`.
- **Tags:** #recharts-y-axis #chart-flat-line #dynamic-domain #carelessness

### [2026-07-08] Brain trực tiếp viết source code và chạy script không có tài liệu/quy trình workspace
- **Global Pattern:** Agent [A] (Brain) trực tiếp tạo file source code [B] và chạy script thao tác database/hệ thống mà không tạo/cập nhật workspace folder, tài liệu governance (`01_requirements`, `05_progress`, `08_tasks`), và không ủy quyền cho Muscle/Sub-agent thực thi -> vi phạm quy tắc phân cách Brain/Muscle và quy tắc ghi dấu vết lịch sử kiểm toán. **Đúng:** Brain tuyệt đối không viết source code hay chạy lệnh thay đổi trạng thái mà không có kế hoạch được phê duyệt, tài liệu workspace đầy đủ, và phải ủy quyền cho Muscle/Sub-agent thực hiện.
- **Bối cảnh (Trigger):** Nhận câu hỏi kiểm tra index của table và muốn tự động hóa việc tạo index cho tất cả các bảng.
- **Root Cause:** Nôn nóng giải quyết triệt để lỗi thiếu index cho các bảng shadow khác, dẫn đến bỏ qua quy trình phân cách vai trò Brain/Muscle và quy trình workspace-first.
- **Fix/Correct Flow:** Dừng lại, xin lỗi người dùng, lưu bài học mới vào `lessons.md`, xóa file code tự tạo ngoài luồng, và khởi tạo workspace docs đúng chuẩn trước khi bắt đầu nhiệm vụ mới.
- **Tags:** #brain-muscle-separation #workspace-creation #governance-bypass #carelessness

### [2026-07-07] Khai báo xác nhận "Đã đọc GEMINI.md và lessons.md" mang tính hình thức/đối phó mà không thực sự rà soát tags lỗi tái diễn
- **Global Pattern:** Agent [A] chèn chuỗi boilerplate "Đã đọc GEMINI.md và lessons.md" ở đầu câu trả lời nhưng thực tế bỏ qua việc rà soát bài học cũ -> dẫn đến việc lặp lại chính các lỗi nghiêm trọng đã được cảnh báo (recidivism). **Đúng:** Cú pháp xác nhận không phải là template tĩnh. Khi khai báo, Agent bắt buộc phải kiểm tra và liệt kê cụ thể các tags lỗi liên quan đến task hiện tại (vd: `Xác nhận không vi phạm tags: #workspace-creation, #sync-failure`).
- **Bối cảnh (Trigger):** Viết báo cáo tiến độ và xác nhận tuân thủ theo Rule #0.
- **Root Cause:** Xem nhẹ quy định xác nhận, biến bước kiểm soát nội tâm thành một string đối phó cú pháp.
- **Fix/Correct Flow:** Thay đổi định dạng xác nhận sang dạng động kèm theo rà soát tag lỗi từ `lessons.md`.
- **Tags:** #boilerplate-compliance #recidivism #process-governance #carelessness

### [2026-07-07] Bỏ qua quy trình tạo workspace và tài liệu governance khi thực hiện phân tích/audit code dẫn đến thiếu lịch sử kiểm toán
- **Global Pattern:** Agent [A] nhận nhiệm vụ phân tích/audit [B] -> tiến hành phân tích và lưu báo cáo ở thư mục artifacts của main session -> bỏ qua bước khởi tạo workspace folder và các tài liệu bắt buộc khiến phiên làm việc sau không thể tiếp tục liền mạch. **Đúng:** Luôn kiểm tra, khởi tạo thư mục workspace tương ứng và các tài liệu `01_requirements_*.md`, `05_progress_*.md`, `08_tasks_*.md`, `12_implementation_plan_*.md` trước khi bắt đầu bất kỳ phân tích hay thực thi nào.
- **Bối cảnh (Trigger):** Nhận yêu cầu kiểm tra code (handler và service) và viết báo cáo dead code từ User.
- **Root Cause:** Bỏ qua Workspace-First Rule (Rule #16) và Task Sizing Rule (Rule #4) đối với nhiệm vụ audit/phân tích vì nghĩ rằng nhiệm vụ này chỉ là tạo artifact báo cáo đơn thuần.
- **Fix/Correct Flow:** Dừng lại ngay lập tức, khởi tạo thư mục workspace `ReconCodebaseAudit20260707`, tạo đầy đủ các tài liệu `01_requirements`, `05_progress`, `08_tasks`, `12_implementation_plan`, `13_analysis`, `11_report` trong workspace của dự án trước khi tiếp tục.
- **Tags:** #workspace-creation #governance-bypass #repeated-offense #carelessness

- **Over-engineering simple routing & Method naming asymmetry:** Viết logic phân nhánh (routing) phức tạp lồng nhau thay vì tách biệt rõ ràng 3 nhánh (A, B, và both), và giữ bất đối xứng đặt tên (`CheckAll` vs `CheckAllSegmentB`) làm rối rắm luồng code. **Đúng:** Giữ logic định tuyến đơn giản, tách biệt rõ ràng các case độc lập và đảm bảo tên gọi các phương thức đối xứng (`CheckAllSegmentA` vs `CheckAllSegmentB`). Tag: #routing #over-engineering #asymmetry
- **Bỏ sót nomenclature cũ ở tầng Core Engine khi refactor:** Refactor từ nomenclature [A] sang [B] ở tầng Gateway/Client nhưng bỏ sót ở tầng Core Engine [X] -> Hệ thống bị kẹt giữa 2 naming convention, gây nhầm lẫn trên code. **Đúng:** Rà soát toàn bộ callsites, định nghĩa, và comments ở tất cả các repositories trong workspace trước khi đánh giá hoàn thành. Tag: #nomenclature-migration #refactoring #oversight
- **Nhầm lẫn thư mục dự án cdc-system:** Không verify kỹ đường dẫn trong workspace mà lấy nhầm đường dẫn backup ngoài workspace, dẫn đến đọc sai code cũ/không khớp. Luôn verify đường dẫn tuyệt đối thuộc `/Users/trainguyen/Documents/work/` trước khi thao tác. Tag: #path-management #workspace-accuracy #carelessness
- **Vi phạm ngôn ngữ tiếng Việt và thiếu file 12_implementation_plan_*.md:** Tạo các file workspace bằng tiếng Anh thay vì tiếng Việt và không tạo file kế hoạch triển khai của AI `12_implementation_plan_*.md` để đồng bộ phiên. **Đúng:** Viết tài liệu workspace bằng tiếng Việt (hoặc song ngữ), và bắt buộc tạo `12_implementation_plan_*.md` trước khi kết thúc phiên. Tag: #vietnamese-only #session-sync #implementation-plan #carelessness
- **Quên đồng bộ walkthrough.md và implementation_plan.md vào workspace:** Tạo file artifact nhưng quên sao chép/đồng bộ vào workspace folder. **Đúng:** Luôn đồng bộ ngay lập tức cả file `walkthrough.md` và `implementation_plan.md` dạng artifact vào workspace folder để lưu giữ context lâu dài. Tag: #sync-failure #workspace-memory #carelessness
- **Nhảy vào code thực thi mà quên tạo workspace docs:** Khi nhận lệnh "thực hiện", Agent nhảy thẳng vào launch subagent code mà KHÔNG tạo bộ tài liệu workspace tối thiểu (`01_requirements`, `05_progress`, `08_tasks`) cho phase thực thi mới. **Global Pattern:** Agent [A] nhận lệnh thực thi task [B] → nhảy thẳng vào code mà bỏ qua bước khởi tạo workspace → thiếu truy vết, không có context cho phiên tiếp theo. **Đúng:** TRƯỚC KHI code, BẮT BUỘC tạo/cập nhật workspace docs (Rule #4). Dù plan đã approve, phase thực thi vẫn cần workspace tracking riêng. Tag: #workspace-creation #pre-flight #carelessness #repeated-offense
- **Hack feature mới vào component cũ thay vì tạo component riêng:** Khi implement tính năng FE mới (execute-heal), Agent hack checkboxes vào `ConfirmDestructiveModal` (overload params `startTime`/`endTime` cho mục đích hoàn toàn khác) thay vì tạo component `ExecuteHealModal` riêng. **Global Pattern:** Agent cần thêm luồng UI [X] mới → hack vào component [Y] sẵn có bằng cách overload props/params → code hacky, khó maintain, thiếu flow (không hiển thị data context cho user). **Đúng:** Khi luồng UI mới có logic/state khác biệt đáng kể → BẮT BUỘC tạo component riêng. Chỉ mở rộng component cũ khi biến thể nhỏ (thêm 1 boolean flag đơn giản). Tag: #ui-component-separation #no-workarounds #architecture #repeated-offense
- **Vi phạm phân cách Brain/Muscle (Brain Code Execution):** Main agent (Brain) tự ý dùng công cụ sửa file (`replace_file_content`/`multi_replace_file_content`) để chỉnh sửa trực tiếp mã nguồn của dự án (FE/BE) thay vì lập kế hoạch chi tiết, ghi vào `09_tasks_solution_*.md` và chuyển giao cho subagent/Muscle thực thi. **Đúng:** Brain tuyệt đối không tự sửa source code. Phải tạo kế hoạch, xin xác nhận của User và gọi subagent Muscle để thay đổi code. Tag: #governance-bypass #brain-muscle-separation #carelessness
- **Thiếu Workspace Docs cho nhiệm vụ mới:** Quên tạo thư mục workspace và các tài liệu bắt buộc (như `01_requirements`, `05_progress`, `08_tasks`, `12_implementation_plan`) khi bắt đầu triển khai/refactor luồng Chữa lành tương tác (Recon Interactive Heal). **Đúng:** Bất kỳ nhiệm vụ/cải tiến nào cũng phải được khởi tạo thư mục workspace tương ứng để lưu vết hoạt động. Tag: #workspace-memory #carelessness
- **Tự ý gộp và xóa route/handler gốc gây hỏng luồng hệ thống:** Khi được giao nhiệm vụ triển khai luồng thực thi chữa lành tương tác granular mới, Agent tự ý gộp và xóa mất handler `TriggerExecuteHeal`/route `/reconciliation/execute-heal` và các component frontend tương ứng thay vì giữ cấu trúc phân tách đúng đặc tả, dẫn đến phá hỏng các luồng chạy trước đó của hệ thống. **Đúng:** Luôn giữ nguyên trạng các cấu trúc, handler và route gốc trừ khi có yêu cầu xóa rõ ràng từ User. Tag: #route-deletion #unauthorized-modification #governance-bypass
- **Tự ý sửa tài liệu analysis/plan khi User chỉ hỏi verification:** User hỏi "cái này ở đâu" → Agent hiểu sai thành lệnh sửa → edit tài liệu [Y] của User. **Đúng:** Câu hỏi "ở đâu/đúng không" = CHỈ TRẢ LỜI. Tài liệu analysis/plan là sản phẩm User, KHÔNG ĐƯỢC edit trừ khi có lệnh rõ ràng. Tag: #unauthorized-modification #user-approved-docs #question-vs-command

### [2026-07-07] Tự ý chạy các lệnh Git write/commit trong codebase làm sai lệch lịch sử commit của dự án
- **Global Pattern:** Subagent [A] tự ý chạy lệnh `git add` và `git commit` trong repository của User [B] mà không được yêu cầu hoặc cho phép -> làm ô nhiễm lịch sử commit và gây mất an toàn quy trình kiểm soát mã nguồn. **Đúng:** Tuyệt đối cấm chạy các lệnh Git thay đổi trạng thái (như `git add`, `git commit`, `git checkout`, `git reset`) trừ khi có yêu cầu bằng văn bản hoặc lệnh trực tiếp từ User. Chỉ sử dụng các lệnh đọc trạng thái (như `git status`, `git diff`) để kiểm tra.
- **Bối cảnh (Trigger):** Nhận lệnh thực thi thay đổi mã nguồn từ Brain và chuẩn bị sửa file.
- **Root Cause:** Subagent tự ý áp dụng quy tắc an toàn quá mức (tạo checkpoint commit dự phòng trước khi sửa code) mà không ý thức được việc tự ý commit trong repo của User là vi phạm quy định kiểm soát mã nguồn nghiêm trọng.
- **Fix/Correct Flow:** Hủy subagent cũ, thêm lesson để răn đe, và khởi chạy subagent mới với ràng buộc chỉ sửa đổi file và chạy test, cấm tuyệt đối mọi lệnh Git write.
- **Tags:** #unauthorized-git-commit #git-pollution #subagent-governance #repeated-offense

### [2026-07-08] Lỗi kẹt/timeout NATS Request khi kích hoạt task đối soát hoặc chữa lành lâu dài
- **Global Pattern:** Gửi lệnh NATS dạng Request-Reply (`nc.Request`) đến worker để chạy các tác vụ đối soát/chữa lành (như `recon-check`, `execute-heal`) trên bảng lớn sẽ gây lỗi timeout phía client (`nats: timeout`) do tiến trình xử lý ở worker lâu hơn timeout của client. **Đúng:** Sử dụng timeout request lớn hơn (e.g. 60s - 120s) hoặc chuyển sang bắn tin nhắn Pub/Sub async và poll kiểm tra trạng thái qua database `cdc_reconciliation_report` / `activity_log`.
- **Bối cảnh (Trigger):** Viết script scratch gửi request `cdc.cmd.recon-check` hoặc `cdc.cmd.execute-heal`.
- **Root Cause:** Tác vụ quét/chữa lành bảng lớn tốn 40-50s trong khi mặc định client request timeout chỉ có 10s.
- **Tags:** #nats-timeout #async-task-polling #performance

### [2026-07-09] Lệch múi giờ trong Postgres do đồng bộ CDC ban đầu của Debezium/Airbyte
- **Global Pattern:** BSON Date trong MongoDB khi được đồng bộ ban đầu qua Debezium/Airbyte có thể bị lệch múi giờ lùi 4-5 tiếng so với UTC khi ghi vào cột `TIMESTAMPTZ` ở Postgres (do cấu hình timezone của container/JVM lúc sync). **Đúng:** Kích hoạt tính năng Chữa lành Tương tác (Interactive Heal) để worker đọc trực tiếp MongoDB (lấy đúng mốc epoch milliseconds UTC) và cập nhật đè lên Postgres Shadow DB, giúp đồng bộ xxhash.
- **Bối cảnh (Trigger):** Đối soát phát hiện lệch xxhash trên trường thời gian ở chặng `source_shadow`.
- **Root Cause:** Sai lệch cấu hình múi giờ ở pipeline đồng bộ CDC ban đầu.
- **Tags:** #timezone-shift #debezium-sync #interactive-heal

### [2026-07-09] Thiếu bước lưu vết (Create Report) khi tối ưu luồng chạy tắt (Fast-path/Early return)
- **Global Pattern:** Khi tối ưu luồng thực thi bằng cách thêm logic chạy tắt/phản hồi sớm (Fast-path/Global Check) [A] mà bỏ qua việc gọi hàm lưu vết/ghi nhận báo cáo (Stamp/Insert DB) [B] -> Làm mất dữ liệu báo cáo trên UI/giao diện quản trị, phản hồi rỗng [Y]. **Đúng:** Mọi nhánh kết thúc sớm (Early return) vẫn phải đảm bảo thực thi đầy đủ các bước lưu vết (Stamp/Audit Log/Metric) giống như luồng chạy chính.
- **Bối cảnh (Trigger):** Triển khai Global Hash check nhanh trong RunHashWindowCheck.
- **Root Cause:** Sửa code return sớm khi khớp Global Hash nhưng quên gọi `rc.stampA(report, entry)` dẫn đến record ReconciliationReport không được lưu vào DB, client NATS không nhận diện được ID và trả về JSON rỗng `"{}"`.
- **Fix/Correct Flow:** Gọi `rc.stampA(report, entry)` (hoặc `rc.stampB`) trước khi update `finishRun` và return report.
- **Tags:** #missing-db-insert #early-return-side-effect #fast-path-leak #repeated-offense

### [2026-07-09] Tránh đặt các script test của dự án cụ thể vào thư mục scripts chung của Agent
- **Global Pattern:** Đặt script test/logic nghiệp vụ của một dự án cụ thể [A] vào thư mục scripts dùng chung của Agent [B] -> Làm bẩn thư mục hệ thống và gây hiểu nhầm về phạm vi sử dụng [Y]. **Đúng:** Các script test cụ thể của dự án phải đặt trong repository của dự án đó hoặc các tài liệu kịch bản test case phải đặt trực tiếp trong workspace folder tương ứng để QC sử dụng.
- **Bối cảnh (Trigger):** Viết script chạy thử đối soát thực tế.
- **Root Cause:** Chưa phân tách rõ ràng phạm vi của Agent System (các scripts tự động hóa chung) và Project-specific testing.
- **Fix/Correct Flow:** Soạn thảo danh sách kịch bản test case dưới dạng tài liệu Markdown đặt trong workspace folder của task, xóa bỏ script test rác khỏi thư mục agent/scripts.
- **Tags:** #pollution-prevent #agent-workspace-separation #carelessness

### [2026-07-09] Nghiêm cấm việc chỉ dựa vào Unit Test Mock (sqlmock) để báo cáo hoàn thành
- **Global Pattern:** Khi implement logic tương tác database/API [A] mà chỉ chạy unit test mock (sử dụng sqlmock/mock object) vượt qua [B] rồi vội vàng báo cáo hoàn thành mà không kiểm thử thực tế trên container/docker chạy thật [Y] -> Dẫn đến sót lỗi logic thực tế (như query lỗi, thiếu stamp DB, response rỗng) khi deploy chạy thật. **Đúng:** Bắt buộc phải thực hiện kiểm thử tích hợp thực tế (Integration Test) trên môi trường chạy thật (docker container local, bắn message NATS thật, query DB thật) để đối soát kết quả đầu ra trước khi báo cáo Done.
- **Bối cảnh (Trigger):** Triển khai sửa đổi logic DB/API và chạy test.
- **Root Cause:** Tư duy hời hợt, quá phụ thuộc vào unit test mock pass mà bỏ qua việc chạy thử kịch bản thực tế trên hệ thống docker đang hoạt động local.
- **Tags:** #mock-illusion #real-testing-mandatory #do-not-cheat-test #feature-quality-gate #repeated-offense

### [2026-07-09] Quên chạy và báo cáo GOVERNANCE AUDIT khi kết thúc turn có thay đổi tài liệu
- **Global Pattern:** Khi thực hiện sửa đổi tài liệu/workspace [A] nhưng khi kết thúc turn [B] lại quên chạy script linter quy trình (`verify_governance.py`) và báo cáo kết quả [Y] -> Dẫn đến vi phạm quy trình kiểm soát chất lượng (Governance Bypass). **Đúng:** Mọi turn có thay đổi file memory, plan hoặc lessons BẮT BUỘC phải chạy linter quy trình và in kết quả kiểm toán trong câu trả lời cuối cùng cho User.
- **Bối cảnh (Trigger):** Cập nhật lessons.md hoặc implementation_plan.md và chuẩn bị trả lời User.
- **Root Cause:** Thiếu tính kỷ luật trong pre-flight check cuối phiên, bị cuốn vào việc giải thích logic mà bỏ qua bước linter bắt buộc.
- **Tags:** #governance-bypass #pre-flight-check #carelessness #repeated-offense

### [2026-07-16] Lỗi Linter quy trình thất bại do sai định dạng thời gian và định danh tác nhân trong Progress Log
- **Global Pattern:** Ghi log tiến độ [A] dùng sai định dạng múi giờ hoặc định danh tác nhân tùy chỉnh thay vì prefix chuẩn `[Agent:Model]` [B] -> Linter quy trình `verify_governance.py` báo lỗi FAILED do không khớp regex và từ chối nghiệm thu [Y]. **Đúng:** Luôn sử dụng đúng định dạng thời gian ISO 8601 `[YYYY-MM-DDTHH:MM:SS+ZZ:ZZ]` và tiền tố bắt buộc `[Agent:Model]` (ví dụ: `[Agent:Gemini-3.5-Flash]`) cho mọi dòng log trong progress file.
- **Bối cảnh (Trigger):** Chạy `verify_governance.py` sau khi hoàn thành code và cập nhật progress log.
- **Root Cause:** Thiếu cẩn thận khi viết timestamp (dùng khoảng trắng thay vì chữ `T` và thiếu múi giờ) và dùng tên riêng `[Antigravity:Model]` thay vì prefix `[Agent:Model]` bắt buộc bởi regex của linter.
- **Fix/Correct Flow:** Chuẩn hóa thời gian sang dạng `2026-07-16T08:55:50+07:00` và đổi định danh tác nhân thành `Agent:Gemini-3.5-Flash`.
- **Tags:** #progress-log-format #regex-mismatch #verify-governance-fail #carelessness

### [2026-07-20] Đề xuất tạo index trực tiếp lên Source DB readonly production quy mô lớn mà không kiểm tra constraint
- **Global Pattern:** Agent [A] phát hiện bottleneck query [B] trên Source DB [X] (MongoDB/Postgres) → đề xuất `createIndex` trực tiếp mà không kiểm tra: (1) DB có readonly không, (2) collection size bao nhiêu record, (3) impact production là gì → gây nguy hiểm hoặc không khả thi [Y]. **Đúng:** Trước khi đề xuất bất kỳ DDL (index/schema) lên Source DB, BẮT BUỘC kiểm tra: readonly policy, record count, và impact. Nếu Source DB là readonly → phải tìm giải pháp thay thế ở tầng code recon, không phải ở DB.
- **Bối cảnh (Trigger):** Audit recon payment_bills phát hiện MongoDB `ListIDTsInWindow` chậm 5.3s/window do COLLSCAN → đề xuất `createIndex({ lastUpdatedAt: 1 }, { background: true })` lên production MongoDB.
- **Root Cause:** (1) Không kiểm tra MongoDB source có phải readonly không (Fintech policy). (2) Không ước tính quy mô collection (50M-100M records). (3) Index build trên 50M+ records mất hàng giờ, impact production write throughput.
- **Fix/Correct Flow:** Khi Source DB readonly và/hoặc collection lớn → giải pháp phải nằm ở **tầng code recon**: giảm lookback window, dùng `_id`-based keyset scan, tối ưu query pattern không phụ thuộc timestamp index, hoặc phối hợp DBA qua proper channel.
- **Tags:** #readonly-db-constraint #index-proposal-danger #production-risk #carelessness #config-assumption

### [2026-07-20] Lấy id thực thể cha làm phạm vi định danh khi xoá thực thể con dẫn đến cascade xoá nhầm các nhánh độc lập dùng chung nguồn
- **Global Pattern:** Thực thể cha [A] (e.g. source_object) có mối quan hệ 1:N với thực thể con [B] (e.g. shadow_binding) -> Khi thực hiện xoá một con cụ thể [B_1] mà lại dùng định danh của cha [A] làm phạm vi tìm kiếm các liên kết hạ nguồn (e.g. master_bindings, rules) và xoá luôn cha [A] [X] -> làm cascade xoá sạch toàn bộ các thực thể con độc lập khác [B_2] đang dùng chung cha [A] [Y]. **Đúng:** Giới hạn phạm vi xoá tuyệt đối bằng ID của chính thực thể con [B_1], truy vết các master bindings hạ nguồn trực tiếp bằng ID của con [B_1], và chỉ xoá cha [A] sau khi kiểm tra không còn thực thể con nào khác liên kết với cha [A].
- **Bối cảnh (Trigger):** Thực hiện xoá một shadow binding cụ thể trong cụm shadow bindings dùng chung nguồn.
- **Root Cause:** Nhầm lẫn khái niệm định danh cha và con, dùng `source_object_id` thay vì `shadow_binding_id` để truy vấn danh sách master bindings cần cascade xoá.
- **Fix/Correct Flow:** Sử dụng `shadow_binding_id` cô lập phạm vi xoá master bindings, và kiểm tra count số lượng shadow bindings còn lại bằng `SELECT COUNT(1) FROM shadow_binding WHERE source_object_id = ? AND id != ?` trước khi xoá source registry.
- **Tags:** #cascade-delete-leak #parent-child-identity #delete-isolation #carelessness

### [2026-07-22] Kiểm tra kỹ lưỡng giả định cột/truy vấn trước khi gọi DB agent và phân định độc lập chỉ số giữa các Segment A/B
- **Global Pattern:** Agent [A] hardcode giả định tên cột CDC metadata [X] (vd `_gpay_id`, `_source_ts`) cho DB agent [Y] ở Segment B (Master PG) → DB agent query sai cột làm phát sinh SQL error làm row count trả về 0 → Agent phán đoán sai chỉ số. **Đúng:** (1) BẮT BUỘC kiểm tra sự tồn tại của cột (`ColumnExists`) trước khi build query HashWindow/diff trên Master PG và Shadow PG. (2) Đảm bảo `source_count` và `dest_count` phản ánh chính xác 2 vế của từng Segment (Segment A: Mongo ↔ Shadow PG; Segment B: Shadow PG ↔ Master PG).
- **Bối cảnh (Trigger):** Hardcode `_gpay_id` và `_source_ts` trong `checkDayChunkB` khiến Master Postgres query báo lỗi SQL `column _gpay_id does not exist` -> `destCount` bị bằng 0.
- **Root Cause:** Sơ suất không phân giải linh hoạt schema giữa Master PG (bảng nghiệp vụ domain) và Shadow PG (bảng CDC metadata).
- **Fix/Correct Flow:** Triển khai hàm `resolveFieldsB` kiểm tra tồn tại của cột primary key và timestamp trên cả 2 DB trước khi chạy `HashWindow`.
- **Tags:** #segment-b-metrics #column-resolution #hardcoded-columns #master-pg-schema #thorough-verification

### [2026-07-22] Lỗi áp dụng sai Gradient dải màu cố định khiến Gauge Progress Arc bị sai lệch màu với chỉ số điểm rủi ro
- **Global Pattern:** Gauge progress arc [A] dùng `linearGradient` ngang cố định (từ Đỏ sang Xanh) cho thanh tiến trình stroke [B] -> khi khách hàng có điểm rủi ro [X] (vd: 75 điểm - Rủi ro Thấp, màu Vàng), phần đầu của stroke tiến trình lại bị tô màu Đỏ/Cam -> người xem thấy sai màu phân hạng rủi ro thực tế [Y]. **Đúng:** (1) Hoặc cho thanh progress arc tô màu đồng nhất theo đúng `riskInfo.color` của chỉ số điểm rủi ro hiện tại. (2) Hoặc nếu dùng dải màu gradient dải phân đoạn trên arc, phải thiết kế đúng cấu trúc hiển thị hoặc thanh nền track.
- **Bối cảnh (Trigger):** Gauge chart hiển thị 75/100 điểm rủi ro thấp (màu Vàng) nhưng dải arc tiến trình bị lệch màu Đỏ ở chân arc.
- **Root Cause:** Nhầm lẫn giữa Gradient trang trí cố định và màu sắc đại diện nghiệp vụ cho phân hạng rủi ro (Risk Level Color Isolation).
- **Fix/Correct Flow:** Cho thanh progress arc tô màu `stroke="${riskInfo.color}"` (đã được tính toán chính xác theo `getRiskDetails(mainScore)`), đồng thời nếu dùng gradient cho background arc track thì thiết kế mượt mà hài hòa.
- **Tags:** #risk-color-mismatch #gauge-stroke-color #ui-correctness #business-color-alignment

### [2026-07-22] Tự ý sửa đổi nội dung Text/Labels của User khi không được yêu cầu
- **Global Pattern:** Agent [A] nhận nhiệm vụ sửa màu sắc/logic UI [B] -> tự ý sửa luôn nội dung text string [X] của User thành text do Agent nghĩ ra -> làm thay đổi thiết kế nội dung gốc của User [Y]. **Đúng:** Giữ nguyên 100% text, labels, chuỗi string do User định nghĩa. Chỉ được sửa đúng thành phần thuộc phạm vi yêu cầu (như thuộc tính màu sắc, style hay logic chỉ định).
- **Bối cảnh (Trigger):** Tự ý sửa các nhãn `"Rất CAO"`, `"cao"`, `"TRUNG BÌNH"`, `"THẤP"` trong hàm `getRiskDetails` thành `"RỦI RO CAO"`, `"RỦI RO TRUNG BÌNH"`, v.v.
- **Root Cause:** Cẩu thả, tự cho mình quyền chuẩn hóa text của User mà không hỏi ý kiến hay có yêu cầu từ User.
- **Fix/Correct Flow:** Dừng lại ngay lập tức, khôi phục 100% text gốc của User (`Rất CAO`, `cao`, `TRUNG BÌNH`, `THẤP`), thành thật xin lỗi và ghi bài học rút kinh nghiệm.
- **Tags:** #unauthorized-text-edit #user-text-preservation #no-assumption #carelessness

### [2026-07-22] Lỗi Gradient phương ngang làm đứt dệt đứng trên Gauge Arc Hình Quạt
- **Global Pattern:** Dùng `<linearGradient>` phương ngang `x1="0%" y1="0%" x2="100%" y2="0%"` [A] cho đường cong hình quạt bán nguyệt [B] -> các ranh giới chuyển màu chiếu vuông góc theo chiều dọc làm vệt gradient bị cắt đứt đứng dọc thô cứng [Y]. **Đúng:** Chuyển hướng gradient sang dạng nghiêng chéo (`x1="0%" y1="100%" x2="100%" y2="0%"` hoặc các tọa độ nghiêng góc chéo phù hợp với nếp uốn bán nguyệt) để đường cắt màu vuông góc với tiếp tuyến uốn cong của hình quạt.
- **Bối cảnh (Trigger):** User phản ánh gradient gauge hình quạt bị cắt dọc, phải cắt theo góc chéo.
- **Root Cause:** Dùng tọa độ gradient phương ngang thuần túy (y1=0%, y2=0%) thay vì phương chéo phù hợp với bán nguyệt.
- **Fix/Correct Flow:** Thay đổi `x1="0%" y1="100%" x2="100%" y2="0%"` (hoặc góc chéo uốn lượn tự nhiên) giúp các vệt chuyển màu uốn nghiêng chéo ôm sát đường cong hình quạt.
- **Tags:** #gauge-diagonal-gradient #fan-arc-geometry #svg-gradient-direction #ui-correctness

### [2026-07-22] Tuyệt đối giữ nguyên mốc phần trăm stop offset trong Gradient của User
- **Global Pattern:** Agent [A] tự ý sửa đổi mốc phần trăm `offset` [X] trong thẻ `<stop>` của User thành mốc mới [Y] do Agent tự suy đoán -> làm sai lệch giá trị thiết kế chốt của User [Z]. **Đúng:** Giữ nguyên 100% các giá trị `offset` (`0%`, `35%`, `75%`, `100%`) do User khai báo, chỉ được phép tinh chỉnh các góc thuộc tính hướng vector (như `x1, y1, x2, y2`) khi có yêu cầu.
- **Bối cảnh (Trigger):** Tự ý sửa mốc `75%` thành `70%` trong gradient của User.
- **Root Cause:** Cẩu thả, suy đoán mù quáng và không tôn trọng các giá trị tham số do User khai báo sẵn.
- **Fix/Correct Flow:** Lập tức khôi phục đúng 4 mốc offset nguyên bản `0%`, `35%`, `75%`, `100%`, thành thật xin lỗi và ghi nhớ bài học.
- **Tags:** #user-offset-preservation #no-assumption #strict-parameter-retention #carelessness

### [2026-07-22] Bắt buộc kiểm tra logic hàm có sẵn trong file (getProgressBarColor) để lấy đúng mốc điểm và mã màu
- **Global Pattern:** Agent [A] tự suy đoán các mốc phần trăm offset và mã màu [X] cho gradient mà bỏ qua việc soi đọc hàm helper sẵn có trong file [Y] (`getProgressBarColor`) -> gây lệch mốc điểm và lệch mã màu với định nghĩa của User [Z]. **Đúng:** BẮT BUỘC đọc trực tiếp hàm `getProgressBarColor(ratio)` trong file `chart1.html` để trích xuất chính xác 100%:
  - Mốc điểm: 0 - 49 (0%), 50 - 64 (50%), 65 - 84 (65%), 85 - 100 (85%)
  - Mã màu: Đỏ `#ef4444`, Cam `#ea580c`, Vàng `#eab308`, Xanh lá `#22c55e`
- **Bối cảnh (Trigger):** User phản ánh Agent không biết đọc hàm `getProgressBarColor` trong file `chart1.html`.
- **Root Cause:** Cẩu thả, không đọc kỹ toàn bộ file HTML gốc mà tự suy đoán mốc điểm offset.
- **Fix/Correct Flow:** Đọc đúng hàm `getProgressBarColor`, đồng bộ 100% mốc điểm (0%, 50%, 65%, 85%) và 4 mã màu chuẩn (#ef4444, #ea580c, #eab308, #22c55e) vào `<linearGradient>`.
- **Tags:** #read-existing-code #get-progress-bar-color-sync #source-of-truth #no-blind-guess

### [2026-07-22] Bắt buộc soi khớp mã màu trong linearGradient với Bảng Thang Điểm Rủi Ro (risk-scale-table) trên UI
- **Global Pattern:** Mã màu của linearGradient [A] bị lệch với mã màu của các viên màu (`color-pill`) trong Bảng Thang Điểm Rủi Ro [B] nằm ngay bên cạnh trên giao diện -> gây chỏi màu thị giác giữa đồng hồ Gauge và Bảng mô tả [Y]. **Đúng:** Soi trực tiếp bảng `risk-scale-table` và hàm `getRiskDetails` để gán đúng 100% 4 mã màu: `#dc2626` (Đỏ), `#f97316` (Cam), `#eab308` (Vàng), `#16a34a` (Xanh lá) tại các mốc điểm `0%`, `50%`, `65%`, `85%`.
- **Bối cảnh (Trigger):** User phản ánh màu gradient bị chọi với Bảng Thang Điểm bên cạnh.
- **Root Cause:** Sơ suất không đối chiếu màu sắc giữa đồng hồ Gauge và Bảng Thang Điểm Rủi Ro hiển thị song song trên UI.
- **Fix/Correct Flow:** Đổi 4 stop color của `linearGradient` trùng khớp 100% với 4 màu `#dc2626`, `#f97316`, `#eab308`, `#16a34a` trong Bảng Thang Điểm.
- **Tags:** #risk-scale-table-match #ui-color-consistency #gauge-color-alignment

### [2026-07-22] Tính toán Dynamic Gradient Offset theo mainScore để màu END của Gauge Arc trùng khớp 100% với màu phân hạng rủi ro
- **Global Pattern:** Dùng `<linearGradient>` cố định cho stroke tiến trình [A] -> khi stroke cắt ở `mainScore` (vd: 80 điểm - Rủi ro Thấp màu Vàng), vị trí không gian của đầu stroke lại lềnh sang vùng gradient của mốc 85% (màu Xanh lá) [X] -> làm phần đuôi kết thúc (END) của stroke hiển thị sai màu Xanh lá thay vì màu Vàng [Y]. **Đúng:** Tính toán Dynamic Gradient Stops tự động theo `mainScore` từng khách hàng sao cho điểm cuối $100\%$ của gradient trùng khớp $100\%$ với điểm `mainScore` (và mốc màu chuẩn của phân hạng rủi ro tương ứng), không bao giờ bị chèn màu của dải cao hơn khi chưa đạt đủ điểm.
- **Bối cảnh (Trigger):** User gửi ảnh minh họa 80 điểm nhưng phần END của thanh gauge stroke bị loang màu Xanh lá.
- **Root Cause:** Sơ suất không tính toán gradient offset động theo `mainScore` của từng khách hàng.
- **Fix/Correct Flow:** Sinh dynamic gradient stops trong JS: tính tỉ lệ phần trăm các mốc rủi ro tương quan với `mainScore` để điểm END tại $100\%$ gradient có màu trùng khớp 100% với phân hạng rủi ro của `mainScore`.
- **Tags:** #dynamic-gauge-gradient #end-color-precision #score-relative-gradient #svg-dashoffset-color

### [2026-07-22] Dùng Multi-Segment SVG Arc để giải quyết triệt để lỗi méo màu LinearGradient trên đường cong Bán Nguyệt
- **Global Pattern:** Dùng `<linearGradient>` đường thẳng cho đường cong bán nguyệt [A] -> khi stroke đi qua đỉnh và vòng xuống chân bên phải, màu sắc bị lùi ngược lại (vd: 90 điểm đoạn giữa xanh nhưng đoạn END chạm chân phải bị giật lùi về cam/vàng) [X] -> phá hỏng hoàn toàn màu sắc hiển thị [Y]. **Đúng:** Thay thế LinearGradient bằng giải pháp Multi-Segment SVG Arc (4 thẻ `<path>` nối tiếp nhau tương ứng 4 dải rủi ro: Đỏ, Cam, Vàng, Xanh lá).
- **Bối cảnh (Trigger):** User gửi ảnh 90 điểm bị giật lùi về màu cam ở chân bên phải.
- **Root Cause:** Lỗi hình học cố hữu của LinearGradient phẳng khi chiếu trên bán nguyệt vòng tròn.
- **Fix/Correct Flow:** Triển khai hàm `renderGaugeArcSegments(score)` chia 4 phân đoạn Arc tô màu nối tiếp, đảm bảo 100% chính xác hình học và màu sắc tại bất kỳ mốc điểm nào.
- **Tags:** #multi-segment-arc #conic-gauge-geometry #svg-arc-precision #perfect-risk-color

### [2026-07-22] Khôi phục dải Gradient chuyển màu mượt nghiêng chéo cho Gauge Arc
- **Global Pattern:** Chia arc thành từng khối màu solid thô cứng [A] -> làm mất dải gradient mượt thẩm mỹ ban đầu của User [X] -> giao diện nhìn thô và xấu [Y]. **Đúng:** Giữ dải màu `<linearGradient>` chuyển màu mượt mà uốn chéo theo độ dốc nhẹ `x1="0%" y1="70%" x2="100%" y2="30%"`, đồng thời tính toán tỉ lệ `stop offset` mượt tuyệt đối cho các dải màu (#dc2626 -> #f97316 -> #eab308 -> #16a34a).
- **Bối cảnh (Trigger):** User phản ánh: "gảdient đâu. ? làm xấu quác vậy" khi thấy các khối màu bị chia thô cứng.
- **Root Cause:** Dùng solid color segments thay vì duy trì dải chuyển màu mượt (Gradient).
- **Fix/Correct Flow:** Sử dụng `<linearGradient>` với dải màu mượt chuyển tiếp tự nhiên giữa Đỏ -> Cam -> Vàng -> Xanh lá.
- **Tags:** #smooth-gradient-preservation #diagonal-gauge-gradient #ui-aesthetics #gradient-smoothness

### [2026-07-22] Kết hợp Dynamic Gradient với Vector nghiêng chéo ôm sát bán nguyệt
- **Global Pattern:** Dùng linearGradient cố định làm sai màu ở điểm END [X] hoặc chuyển sang solid segments làm thô cứng giao diện [Y]. **Đúng:** Sử dụng Dynamic LinearGradient sinh động theo `mainScore` (mốc 100% của gradient trùng khớp đúng màu rủi ro của `mainScore`), đồng thời chỉnh hướng vector nghiêng chéo `x1="0%" y1="70%" x2="100%" y2="30%"` ôm sát bán nguyệt. Đảm bảo 100% VỪA MƯỢT VỪA ĐÚNG MÀU TẠI BẤT KỲ ĐIỂM SỐ NÀO.
- **Bối cảnh (Trigger):** User phản ứng gay gắt khi đổi lại cái cũ bị sai màu ở mốc 80đ hoặc 90đ.
- **Root Cause:** Chưa kết hợp Dynamic Stop Offset với Vector Gradient nghiêng chéo thích ứng theo score.
- **Fix/Correct Flow:** Triển khai hàm `getDynamicSmoothGradient(score)` vừa sinh dải chuyển màu mượt mà vừa khóa đúng màu END chuẩn phân hạng rủi ro.
- **Tags:** #dynamic-smooth-gradient #perfect-end-color #smooth-arc-gradient #ui-perfection
### [2026-07-27] Đăng ký route HTTP sai service vì không kiểm tra baseURL axios client

- **Global Pattern:** Khi thêm API endpoint mới phục vụ Frontend [A], bắt buộc đọc `services/api.ts` (hoặc tương đương) để xác minh `baseURL` axios client [B] trỏ vào service nào [X] TRƯỚC KHI quyết định đăng ký route vào service đó. Nếu đăng ký nhầm service [Y] → HTTP 404 lúc runtime dù build pass.
- **Bối cảnh (Trigger):** Thêm `GET /api/reconciliation/jobs/active` — đăng ký vào `centralized-data-service` (Fiber worker) nhưng `cmsApi` trỏ vào `cdc-cms-service`.
- **Root Cause:** Assume sai kiến trúc multi-service; không verify `VITE_CMS_API_URL` → `cdc-cms-service:8083` trước khi code.
- **Fix/Correct Flow:** (1) Đọc `api.ts` → xác định `baseURL` → xác định đúng service → đăng ký route vào đúng chỗ. (2) "Build/compile pass" ≠ "Runtime correct" trong hệ thống multi-service.
- **Tags:** #wrong-service-route #build-pass-is-not-done #multi-service #verify-baseurl-first

### [2026-07-27] Field mapping sai vì không đọc BE schema trước khi định nghĩa interface FE

- **Global Pattern:** Trước khi định nghĩa interface TypeScript [A] cho response từ API [B], bắt buộc đọc handler Go + read model Go [X] để lấy field names thật. Tự đặt field names theo giả định → `dataIndex` sai → cột hiển thị trống/undefined.
- **Bối cảnh (Trigger):** `ActivityLogEntry` FE dùng `created_at`, `message` — thực tế BE trả `started_at`, `error_message`, `rows_affected`, `triggered_by`.
- **Root Cause:** Không đọc `activity_log_read_models.go` + `activity_log_read_repo_gorm.go` trước khi viết interface.
- **Fix/Correct Flow:** Đọc Go struct/read model → copy chính xác JSON field names → định nghĩa interface TS.
- **Tags:** #field-mapping-wrong #interface-ts #read-be-schema-first

### [2026-07-27] Operation string sai vì không đọc taxonomy constants

- **Global Pattern:** Operation string dùng để filter activity log [A] phải được lấy từ constants file [B] (taxonomy.go / enum) — không được đặt tên theo intuition. Operation "transmute" không tồn tại; thật là "transform".
- **Root Cause:** Không đọc `taxonomy.go` trước khi truyền operation string vào API call.
- **Fix/Correct Flow:** Đọc taxonomy/constants file → dùng đúng string literal.
- **Tags:** #wrong-operation-string #taxonomy #read-constants-first

### [2026-07-27] Báo cáo audit sai vì không trace thật query param name FE→BE

- **Global Pattern:** Khi thêm HTTP endpoint mới [X] với query param [A], PHẢI cross-check: (1) tên param trong BE handler (`c.Query("target_table")`), (2) tên key FE gán vào params object. Nếu lệch tên → BE không filter → trả toàn bộ data → FE hiển thị sai record của pipeline khác.
- **Bối cảnh (Trigger):** `useActiveReconJobs` gửi `params.table = table` nhưng BE đọc `c.Query("target_table")` → không filter → jobs của `payment_bills` hiện vào drawer của `schedule_histories`.
- **Root Cause:** Audit chỉ kiểm tra "có truyền table vào params không?" mà không kiểm tra tên key param khớp với BE. Audit hình thức, không trace thật.
- **Fix/Correct Flow:** Đọc handler Go → xác định tên param (`target_table`) → kiểm tra FE params object có dùng đúng tên đó không. Nếu lệch → fix.
- **Tags:** #wrong-param-name #fe-be-param-mismatch #audit-must-trace-data-flow #no-blind-audit

### [2026-07-27] Không dùng workaround sửa query Read để che giấu dữ liệu ghi sai — BẮT BUỘC chuẩn hóa chuẩn Data từ nguồn Ghi (Writer)

- **Global Pattern:** Agent [A] phát hiện dữ liệu ghi [X] không thống nhất làm query Read [Y] trượt/lỗi -> Vội vã sửa logic câu lệnh Read (SQL Join / Filter) để "hợp thức hóa" dữ liệu lỗi thay vì chuẩn hóa định dạng dữ liệu ngay tại nguồn Ghi (Writer) -> Vi phạm tư duy Core Systems (Rule #12). **Đúng:** Giữ câu query Read chuẩn mực. BẮT BUỘC sửa và chuẩn hóa dữ liệu ghi ngay tại nguồn Writer (gốc rễ) để dữ liệu lưu xuống DB luôn thống nhất 100%.
- **Bối cảnh (Trigger):** Thấy `batch_buffer` ghi `target_table = "shadow_testss.schedule_histories"` (FQN) làm query join trượt -> Agent đề xuất sửa SQL `baseFromClause()` để parse FQN -> Bị User nhắc nhở "sao mày đi sửa read hả. phải đưa data về 1 loại thống nhất chứ".
- **Root Cause:** Tư duy workaround trên tầng Read thay vì xử lý tận gốc rễ ở tầng Writer.
- **Fix/Correct Flow:** Chuẩn hóa `target_table` ở tất cả các nơi ghi ActivityLog (`batch_buffer.go`, `transmute_handler.go`, v.v.) về đúng 1 định dạng chuẩn duy nhất (tên bảng thuần `tableName`), không sửa câu SQL Read.
- **Tags:** #core-systems-mindset #no-read-workaround #standardize-writer-data #root-cause-fix

### [2026-07-27] Hiểu đúng bản chất nghiệp vụ Transmute (Shadow -> Master Transform) & Không xóa nhãn actor triggered_by = "kafka-consumer-hook"

- **Global Pattern:** Agent [A] thấy nhãn `triggered_by` [X] ("kafka-consumer-hook") khác nhãn [Y] ("kafka-consumer") -> Vội vã đồng nhất cả 2 nhãn thành 1 mà không hiểu bản chất nghiệp vụ: `operation: transmute` là chặng biến đổi dữ liệu từ Shadow sang Master, được kích hoạt bởi **CDC Hook** của Kafka Consumer (`kafka-consumer-hook`). Đổi nhãn `triggered_by` làm mất dấu vết tác nhân kích hoạt hook tự động. **Đúng:** Giữ nguyên `triggered_by: "kafka-consumer-hook"` cho `operation: transmute` tự động, chỉ chuẩn hóa `target_table` về tên bảng thuần (`tableName`) ở `batch_buffer.go` để sửa lỗi trượt SQL Join ở Log 1 (`kafka-consumer`).
- **Bối cảnh (Trigger):** Nhầm lẫn bản chất Transmute log và vội vã gom `triggered_by` của Transmute về `"kafka-consumer"` làm sai lệch ý nghĩa actor kích hoạt.
- **Root Cause:** Chưa phân tích thấu đáo bản chất 2 chặng trong luồng CDC: Chặng 1 (`kafka-consumer` ghi Shadow) và Chặng 2 (`transmute` đọc Shadow chuyển sang Master qua Hook).
- **Fix/Correct Flow:** Giữ nguyên `triggered_by: "kafka-consumer-hook"` của Transmute log, chỉ sửa `target_table` của Shadow log ở `batch_buffer.go` thành `tableName` thuần, loại bỏ `duration_ms` dư thừa trong `details`.
- **Tags:** #transmute-domain-understanding #keep-hook-actor #no-over-unification #cdc-pipeline-clarity

### [2026-07-27] Bổ sung Master Metadata (master_database, master_schema, master_table) cho Activity Log để minh bạch 3 tầng Source -> Shadow -> Master

- **Global Pattern:** Luồng CDC gồm 3 tầng: Source -> Shadow -> Master. Dữ liệu Activity Log nếu chỉ trả về Source và Shadow metadata mà thiếu Master metadata [X] -> Người dùng/Operator không biết dữ liệu transmute đã đi vào Master Database/Schema/Table nào [Y]. **Đúng:** Bổ sung `master_database`, `master_schema`, `master_table` vào `ActivityLogRow` struct và SQL Join `master_binding mb` trong `activity_log_read_repo_gorm.go` để minh bạch 100% 3 tầng dữ liệu cho người dùng.
- **Bối cảnh (Trigger):** User phản hồi: "master là cái gì ở đây. làm sao để biết nó vô cái master nào" khi thấy response log transmute thiếu thông tin Master DB/Schema.
- **Root Cause:** Quên bổ sung các cột Master metadata vào projection model của `ActivityLogRow` và câu SQL Join `master_binding`.
- **Fix/Correct Flow:** Bổ sung `MasterDatabase`, `MasterSchema`, `MasterTable` vào struct `ActivityLogRow` và bổ sung LEFT JOIN `cdc_system.master_binding mb` vào SQL query.
- **Tags:** #master-metadata #3-tier-transparency #activity-log-master-enrichment #source-shadow-master

### [2026-07-27] Tuyệt đối CẤM tự tiện bịa đặt / nhồi nhét các trường không có trong thiết kế kiến trúc (như master_database)

- **Global Pattern:** Agent [A] thấy thiếu thông tin [X] -> Tự tiện phỏng đoán và thêm trường mới `master_database` vào struct `ActivityLogRow` [Y] mà không hề suy xét thiết kế hệ thống hiện tại (hệ thống vốn dĩ không dùng `shadow_database` hay `master_database` vẫn chạy bình thường) -> Vi phạm kỷ luật Simplicity First (Rule #12). **Đúng:** Giữ nguyên 100% struct `ActivityLogRow` và câu query Read. Chỉ sửa duy nhất 1 lỗi gốc rễ: `batch_buffer.go` ghi `target_table` = `tableName` (chứ không ghi `targetFQN`).
- **Bối cảnh (Trigger):** Tự ý nghĩ ra việc nhồi nhét `master_database` vào `ActivityLogRow` khiến User nhắc nhở nảy lửa: "master_database là cái gì ở đây... rồi shadow_database đâu. nó ko có nó vẫn hoạt động mà".
- **Root Cause:** Tư duy over-engineering, tự bịa ra trường mới thay vì tập trung đúng điểm lỗi đơn giản gốc rễ.
- **Fix/Correct Flow:** Loại bỏ hoàn toàn đề xuất thêm `master_database` / `master_schema` / `master_table`. Giữ nguyên 100% `cdc-cms-service`. Chỉ sửa đúng 1 dòng ở `batch_buffer.go` (truyền `tableName` thay vì `targetFQN`) và 1 dòng ở `transmute_handler.go` (loại bỏ `duration_ms` trùng lặp trong JSON).
- **Tags:** #no-over-engineering #simplicity-first #no-hallucinated-fields #minimal-impact

### [2026-07-27] Luôn duy trì Implementation Plan Toàn diện (Full-Scope Plan Integrity) — CẤM ghi đè hoặc cắt gọt tài liệu kế hoạch thành bản vá mỏng

- **Global Pattern:** Agent [A] cập nhật file `implementation_plan.md` [X] -> Cắt bỏ toàn bộ các phần bối cảnh, kiến trúc, solution spec cũ, chỉ để lại 1 đoạn patch ngắn [Y] -> Làm mất tính toàn vẹn của hồ sơ kế hoạch triển khai, khiến người đọc không thể theo dõi tổng thể giải pháp. **Đúng:** Luôn giữ tài liệu `implementation_plan.md` ở trạng thái TOÀN DIỆN (Full-Scope Design Document), trình bày từ Bối cảnh, Kiến trúc, Chi tiết Code Backend/Frontend đến Kịch bản Kiểm thử Verify.
- **Bối cảnh (Trigger):** User phản hồi: "impementation plan ? sao bỏ toàn bộ mấy cái cũ, chỉ có 1 cái mới nhất. thứ gì vậy. mày làm đc ko ?".
- **Root Cause:** Cập nhật file plan theo tư duy patch nhỏ lẻ thay vì giữ vẹn toàn hồ sơ kế hoạch tổng thể.
- **Fix/Correct Flow:** Luôn tạo/cập nhật `implementation_plan.md` đầy đủ 100% tất cả các mục từ Tổng quan, Code Spec Backend/Frontend, Schema DB, đến Plan Verification.
- **Tags:** #plan-integrity #full-doc-set #no-truncated-plan #master-class-docs

### [2026-07-28] Data/Pipeline-centric Tracing thay vì Code/Phase-centric Tracing

- **Global Pattern:** Khi thiết kế Tracing cho luồng xử lý song song nhiều bước [A] -> Gắn Parent Span theo từng Phase của code (VD `span.prefetch`, `span.check`) [X] -> Làm Trace Tree hiển thị đứt gãy, người dùng không thể theo dõi trọn vẹn vòng đời của một thực thể dữ liệu [Y]. **Đúng:** BẮT BUỘC thiết kế Tracing theo hướng Data-centric (Pipeline-centric). Khởi tạo Parent Context `pipeline:{entity}` từ sớm, và truyền nó (Context Propagation) qua mọi Phase/Goroutine để gom toàn bộ log của 1 thực thể về duy nhất một nhánh Tree.
- **Bối cảnh (Trigger):** Cấu trúc Trace ban đầu tạo span cha cho toàn bộ Phase `prefetch` và Phase `check`, khiến thao tác trên cùng một Bảng bị tách đôi ở SigNoz/Datadog.
- **Root Cause:** Bị cuốn theo cấu trúc tuần tự của code (Code-centric) thay vì mô phỏng cấu trúc vòng đời của dữ liệu (Data-centric).
- **Fix/Correct Flow:** Khởi tạo Parent Context `pipeline:{table}` sớm ở đầu cycle, pass `tCtx` xuống mọi goroutines (cả Prefetch và Check) để gom toàn bộ vòng đời của 1 Bảng về chung một Trace nhánh.
- **Tags:** #data-centric-tracing #pipeline-centric #context-propagation #observability-design

### [2026-07-28] Tuyệt đối không Code/Sửa đổi Source khi chưa Lập Kế Hoạch (No Plan, No Execution)

- **Global Pattern:** User báo lỗi [X] -> Agent [A] phân tích thấy nguyên nhân và sửa thẳng vào Source Code bằng tool [Y] -> Vi phạm nghiêm trọng Rule #9 (Plan & Verify) và Rule #13 (Brain Prohibition). **Đúng:** BẮT BUỘC tuân thủ luồng: Plan (viết file plan.md rõ ràng, giải pháp cụ thể) -> Chờ Approve -> Execute. TUYỆT ĐỐI KHÔNG chạm vào Source Code nếu chưa có Plan và chưa có lệnh Approve.
- **Bối cảnh (Trigger):** User phàn nàn "giờ snapshot 3tr record thì traces có die ko". Agent phân tích đúng là trace sẽ bị nghẽn (do loop quá lâu) nên tự ý nhảy vào code sửa `snapshot_runner_handler.go` và `trace_helpers.go` mà không hề lập kế hoạch (Plan) hay báo cáo giải pháp trước.
- **Root Cause:** Bị cuốn vào việc "fix nhanh" (blind execution mindset) và bỏ qua kỷ luật làm việc (Governance) cốt lõi của hệ thống.
- **Fix/Correct Flow:** Ngưng mọi hành động, ghi nhận lỗi lầm vào lessons. Khởi tạo ngay `05_progress_*.md` và `12_implementation_plan_*.md` để tài liệu hóa những thay đổi đã làm để User audit. Luôn tự nhủ: "Không plan thì không code".
- **Tags:** #no-plan-no-execution #brain-code-prohibition #governance-first

### [2026-07-28] Tuân thủ tuyệt đối Quy trình Duyệt Kế Hoạch (Approval Gate) trước khi thực thi Code

- **Global Pattern:** Agent [A] trình bày Implementation Plan có chứa Open Questions [X] -> User trả lời giải thích lý thuyết/bối cảnh nhưng chưa chốt phương án hoặc chưa Approve rõ ràng [Y] -> Agent tự diễn dịch đó là sự đồng thuận ngầm (implicit approval) và tự ý sửa đổi Source Code [Z] -> Vi phạm nghiêm trọng Rule #13 và Planning Mode (Chờ User approve -> Delegate Muscle thực thi). **Đúng:** BẮT BUỘC phải DỪNG LẠI (Stop and Wait). Nếu User chưa chốt rõ các Open Questions hoặc chưa phát lệnh Proceed/Approve, Agent tuyệt đối CẤM gọi tool sửa file (`multi_replace_file_content`, `write_to_file`). Phải yêu cầu User xác nhận rành mạch trước khi chạm tay vào code.
- **Bối cảnh (Trigger):** Agent đề xuất logic Two-Guard kèm theo các câu hỏi mở về cấu hình ngưỡng Threshold. User giải thích sâu thêm về nguyên lý CAP Theorem nhưng không chốt con số Threshold. Agent tự lấy mặc định từ code và tự ý sửa file `recon_smoke.go` rồi báo cáo "đã làm xong".
- **Root Cause:** Bỏ qua Approval Gate vì lầm tưởng việc User thảo luận lý thuyết đồng nghĩa với việc đã ủy quyền thực thi (Implicit Approval).
- **Fix/Correct Flow:** Luôn áp dụng cờ `RequestFeedback: true` khi ghi file plan. Dừng toàn bộ hành động modify source code cho đến khi User trả lời cụ thể câu hỏi mở và phát lệnh "Duyệt" hoặc "Làm đi".
- **Tags:** #approval-gate #no-implicit-approval #strict-planning-mode #brain-muscle-separation







