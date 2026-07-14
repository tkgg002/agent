# 📚 Hard-Tech Patterns & Tripwires (Garbage Collected)

> **BẢN CHẤT**: File chứa các bẫy kỹ thuật đặc thù (Postgres, CDC, Kafka, Golang, MongoDB). Các quy trình hành vi đã được GC nén vào `tech_stack.md`. BẮT BUỘC ĐỌC TRƯỚC KHI CODE.

## 1. Golang & GORM Quirks

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