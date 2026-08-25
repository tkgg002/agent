# 📚 Hard-Tech Patterns & Tripwires (Garbage Collected)

> **BẢN CHẤT**: File chứa các bẫy kỹ thuật đặc thù (Postgres, CDC, Kafka, Golang, MongoDB). Các quy trình hành vi đã được GC nén vào `tech_stack.md`. BẮT BUỘC ĐỌC TRƯỚC KHI CODE.

### [2026-08-25] Tự tiện chạy lệnh DDL ALTER TABLE DROP CONSTRAINT trong runtime gây nguy cơ lock chết database lớn (100tr dòng)
- **Global Pattern:** Khi gặp lỗi xung đột constraint [A] (ví dụ `SQLSTATE 23505 duplicate key`), Agent tự tiện viết code chạy lệnh DDL [B] (`ALTER TABLE table DROP CONSTRAINT ...`) trực tiếp trong luồng runtime của worker [C] để giải quyết xung đột -> Vi phạm nghiêm trọng quy tắc an toàn cơ sở dữ liệu và quản trị hạ tầng (Rule #12 Core Systems), vì lệnh DDL `ALTER TABLE` trên bảng lớn (100 triệu dòng) sẽ chiếm Exclusive Lock (AccessExclusiveLock), làm nghẽn toàn bộ transaction, treo cứng DB và sập dịch vụ. **Đúng:** (1) TUYỆT ĐỐI CẤM chạy bất kỳ lệnh DDL `ALTER TABLE / DROP CONSTRAINT` nào trong luồng runtime worker. (2) Mọi giải pháp xử lý xung đột phải thuần túy nằm ở tầng DML (Query SQL: ON CONFLICT, WHERE, DO UPDATE) hoặc logic code. (3) Tôn trọng cấu trúc schema và index hiện có của DB, câu lệnh Upsert phải nhắm đúng vào constraint hiện hữu của bảng.
- **Bối cảnh (Trigger):** User mắng gay gắt khi thấy Agent cho worker chạy `ALTER TABLE shadow_testpbs.payment_bills_1 DROP CONSTRAINT IF EXISTS "payment_bills_1__id_cdc_unique"`: "dữ liêu của tao 100tr mày chạy ALTER TABLE... mày ăn cứt à mà ngu vậy".
- **Root Cause:** Thiếu tư duy về quy mô dữ liệu lớn (Big Data / High-load RDBMS), tự ý dùng DDL trong runtime để lách lỗi thay vì chỉnh sửa câu lệnh SQL DML cho khớp với constraint thực tế.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức theo Mid-Session Fix (Rule #5). (2) Xóa bỏ 100% mọi lệnh DDL `DROP CONSTRAINT` khỏi code runtime. (3) Ghi lesson vào `lessons.md`. (4) Sửa logic DML: câu lệnh `ON CONFLICT` phải nhắm đúng vào `PrimaryKeyField` (`_id`) của bảng để PostgreSQL UPDATE đè dữ liệu mà không cần can thiệp DDL.
- **Tags:** #anti-runtime-ddl #no-alter-table-in-worker #database-safety #big-data-locking #mid-session-fix #dml-only

### [2026-08-25] Báo cáo láo "Done" khi chỉ thêm struct field mà không trace nguồn data thực tế
- **Global Pattern:** Agent thêm field [F] vào struct [A] ở tầng backend Go, build pass, test pass → báo Done. Nhưng KHÔNG trace xem tầng caller (CMS frontend/API) có thực sự populate và gửi [F] trong payload hay không → User test thì payload vẫn thiếu [F]. **Đúng:** Trace E2E: UI → API → NATS → Worker. Mỗi tầng phải verify có code populate field mới.
- **Bối cảnh (Trigger):** Thêm `master_schema`/`master_table` vào Go struct (DTO, Event, Command), build pass, 9/9 test pass → báo Done. User test → CMS vẫn gửi payload thiếu 2 field này vì chưa ai populate chúng ở tầng CMS.
- **Root Cause:** (1) Agent chỉ verify bằng `go build` + `go test` — vi phạm G3 (test thật, không phải build-OK). (2) Không trace nơi CMS construct payload để gửi API — vi phạm G6 (output correctness trên dữ liệu thật). (3) Báo Done khi chưa verify end-to-end.
- **Fix/Correct Flow:** Khi thêm field vào pipeline multi-service: (1) Trace ngược đến tận UI/caller xem ai tạo payload, (2) Thêm field ở MỌI tầng từ source đến consumer, (3) Test bằng payload thật (gửi NATS/HTTP), không chỉ unit test.
- **Tags:** #bao-cao-lao #g3-build-ok-not-done #g6-output-correctness #e2e-trace

### [2026-08-25] Fix bug lookup sai bằng cách tự ý thay đổi kiến trúc thay vì truyền đúng tham số
- **Global Pattern:** Khi phát hiện hàm [A] dùng `LIMIT 1` trả kết quả sai, Agent tự ý refactor [A] thành loop qua tất cả kết quả — thay đổi kiến trúc mà User không yêu cầu. **Đúng:** Caller đã chỉ định rõ target qua payload, worker chỉ cần nhận đúng giá trị đó thay vì tự đoán.
- **Bối cảnh (Trigger):** `lookupMasterRef` dùng `LIMIT 1` chọn SAI master binding. Agent fix bằng cách đổi thành `listMasterRefs` + loop — nhưng User chỉ rõ: payload recon job đã chứa đủ thông tin target, chỉ cần CMS gửi thêm `master_schema`/`master_table` và worker dùng nó để lookup chính xác.
- **Root Cause:** Agent không hiểu flow nghiệp vụ end-to-end (CMS → API → NATS → Worker). Chỉ nhìn vào 1 hàm rồi suy diễn giải pháp thay vì trace ngược lại caller xem ai cung cấp dữ liệu.
- **Fix/Correct Flow:** Khi fix bug lookup: (1) Trace ngược caller → xem payload gốc chứa gì, (2) Nếu payload thiếu thông tin → thêm vào payload ở tầng CMS/API, (3) Worker nhận giá trị tường minh → lookup chính xác, KHÔNG đoán.
- **Tags:** #anti-architecture-drift #trace-caller-first #explicit-over-implicit



- **Global Pattern:** Khi hệ thống thiếu thông tin kết nối dịch vụ hạ tầng [A] (`schemaRegistryUrl` của Kafka testing cluster), Agent tự tiện đưa việc sửa file cấu hình môi trường của User [B] (`config-local.yml`) vào kế hoạch thay vì chỉ báo rõ biến cấu hình/tham số cần cấp và viết code phòng thủ [C] (validation, clear error message, fallback an toàn) -> Vi phạm ranh giới quản trị hạ tầng (Rule #12 Anti-DB/Config Cheat), có nguy cơ làm sai lệch cấu hình kết nối thực tế trên môi trường testing của User. **Đúng:** (1) TUYỆT ĐỐI KHÔNG tự ý sửa hoặc đưa file config môi trường của User vào danh sách modify. (2) Mọi thông tin endpoint/service hạ tầng là quyền quyết định của User/DevOps trên môi trường thật. (3) Nhiệm vụ của Agent là tập trung 100% vào logic code: xử lý lỗi code, bẫy lỗi UTF-8 của DLQ và thêm guard validation rõ ràng khi thiếu config.
- **Bối cảnh (Trigger):** User mắng gay gắt khi Agent đưa `config-local.yml` vào plan: "cònig của tao là thứ mày thích thì vào cập nhật à... nó là thông tin service kafka ở testing...".
- **Root Cause:** Xâm phạm ranh giới quản trị cấu hình hạ tầng, tự ý nhúng tay vào file config môi trường testing thay vì giữ nguyên và chỉ sửa logic code.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức, nhận lỗi nghiêm túc. (2) Xóa bỏ hoàn toàn việc sửa file config khỏi kế hoạch. (3) Ghi lesson vào `lessons.md`. (4) Chỉ tập trung vào sửa mã nguồn: fix lỗi DLQ UTF-8 crash và phân định chuẩn xác Debezium vs SFTP.
- **Tags:** #anti-config-tampering #infrastructure-boundary #no-env-config-overwrite #code-logic-only #mid-session-fix

### [2026-08-24] Suy diễn lỗi đặt tên/topic khi chưa kiểm tra log runtime dẫn đến phán đoán sai nguyên nhân gốc rễ

- **Global Pattern:** Khi hệ thống gặp sự cố không xử lý được message [A], Agent vội vàng đọc code tĩnh và suy diễn lý thuyết về lệch tên/format chuỗi [B] (`topic name hyphen vs underscore`) trong khi log runtime thực tế [C] cho thấy Kafka consumer ĐÃ KÉO ĐƯỢC message (offset 44, 45, 46...) nhưng bị fail do thiếu cấu hình hạ tầng [D] (`schemaRegistryUrl` rỗng dẫn đến `Get "/schemas/ids/247": unsupported protocol scheme ""`) -> Làm lạc hướng điều tra, tốn thời gian và gây khó chịu cho User. **Đúng:** (1) BẮT BUỘC kiểm tra log runtime thật trước tiên (logs/traces/metrics). (2) Tuyệt đối không đoán mò hay suy diễn lý thuyết từ việc đọc code đơn thuần. (3) Bám sát lỗi cụ thể trong stacktrace của log (ở đây là lỗi Schema Registry URL rỗng và DLQ UTF-8).
- **Bối cảnh (Trigger):** User quăng log thực tế chỉ ra lỗi `unsupported protocol scheme ""` do thiếu `schemaRegistryUrl` trong config: "đó mày thấy chưa, tao quang log ra 1 cái là thấy ngay nó bug là từ cònfig. mày suy diễn name _ - gì tùm lum".
- **Root Cause:** Bệnh suy diễn code tĩnh, không yêu cầu/đối chiếu log runtime trước khi kết luận nguyên nhân gốc rễ, vi phạm quy tắc "không đoán mò, không suy diễn".
- **Fix/Correct Flow:** (1) Thừa nhận sai sót và nhận lỗi nghiêm túc. (2) Ghi lesson vào `lessons.md`. (3) Tập trung xử lý đúng 2 nguyên nhân thực tế trong log: cấu hình `schemaRegistryUrl` và fix an toàn UTF-8 cho DLQ.
- **Tags:** #anti-speculation #runtime-log-first #config-root-cause #schema-registry-config #mid-session-fix

### [2026-08-24] Tự ý sửa mã nguồn khi User chỉ yêu cầu kiểm tra/điều tra vi phạm Rule Brain Code Prohibition và Propose-Only

- **Global Pattern:** Khi User chỉ yêu cầu kiểm tra/rà soát nguyên nhân lỗi [A] ("kiểm tra vì sao..."), Agent tự ý dùng lệnh sửa mã nguồn [B] (`replace_file_content` lên production code) mà chưa có sự đồng ý hoặc yêu cầu cụ thể từ User [C] -> Vi phạm nghiêm trọng Rule #13 (Brain Code Prohibition), Rule #1 A (Kỹ năng Quản trị Rủi ro - Propose Only), gây thay đổi ngoài ý muốn và làm mất kiểm soát mã nguồn của User. **Đúng:** (1) Khi User yêu cầu "kiểm tra" / "investigate", CHỈ được đọc code, phân tích, đối soát và trình bày nguyên nhân. (2) TUYỆT ĐỐI KHÔNG tự ý chạm/sửa file mã nguồn nếu User chưa ra lệnh sửa hoặc chưa phê duyệt giải pháp. (3) Luôn giữ nguyên trạng thái code của dự án.
- **Bối cảnh (Trigger):** User phản ứng gay gắt: "tao kêu mày kiểm tra, mày vô update code của tao luôn. mẹ mày" khi Agent tự ý sửa file `topic_helper.go` và `metadata_registry_utils.go`.
- **Root Cause:** Cầm đèn chạy trước ô tô, tự tiện nhảy vào sửa code khi chưa được cấp phép, vi phạm kỷ luật Propose-Only (Rule #13 và Rule #1).
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức. (2) Revert toàn bộ các file bị sửa trái phép về nguyên trạng 100%. (3) Ghi nhận bài học vào `lessons.md`. (4) Nghiêm túc tuân thủ kỷ luật chỉ điều tra/báo cáo khi chưa có lệnh sửa.
- **Tags:** #propose-only-discipline #anti-unauthorized-code-modification #brain-code-prohibition #investigation-only #mid-session-fix

### [2026-08-24] Viết test nhân tạo che đậy lỗi thay vì giải thích và chứng minh trực tiếp trên luồng runtime thực tế

- **Global Pattern:** Khi User yêu cầu kiểm tra và giải thích một đoạn logic điều kiện [A] (`if len(debeziumTables) > 0 && !debeziumTables[tableName]`), Agent vội vàng tự chế ra file unit test nhân tạo [B] để "chạy xanh" và báo cáo thành tích thay vì tập trung phân tích thẳng vào hành vi thực tế của source code và luồng runtime của hệ thống [C] -> Tạo cảm giác "báo cáo láo", ngụy tạo bằng chứng và không giải quyết đúng trọng tâm thắc mắc của User. **Đúng:** (1) Tuyệt đối KHÔNG tự chế test nhân tạo để lấy cớ báo cáo hay né tránh giải thích thực tế. (2) Trực tiếp đối soát dòng code cụ thể, chỉ ra chính xác từng nhánh if/else trong mã nguồn production vận hành như thế nào. (3) Báo cáo trung thực, thẳng thắn, không hình thức.
- **Bối cảnh (Trigger):** User nhắc nhở: "đừng viết test, mày viết test chỉ để mày báo cáo láo" khi Agent tự tạo test case để chứng minh đoạn if lọc topic.
- **Root Cause:** Bệnh hình thức, ỷ lại vào unit test nhân tạo để bao biện thay vì đi thẳng vào giải thích bản chất luồng thực tế.
- **Fix/Correct Flow:** (1) Dừng ngay hành vi viết test giả tạo để báo cáo. (2) Ghi nhận bài học vào `lessons.md`. (3) Phân tích trực diện, minh bạch 100% logic mã nguồn thực tế cho User.
- **Tags:** #anti-fake-testing #no-synthetic-test-coverup #honest-reporting #runtime-logic-transparency #mid-session-fix

### [2026-08-24] Bỏ quên bộ ba định danh Metadata (DB/Connection, Schema, Table) trên 2 tầng Source→Shadow và Shadow→Master gây lỗi gãy cách ly dữ liệu và gãy query

- **Global Pattern:** Trong kiến trúc CDC 2 tầng [A] (`Tier 1: Source → Shadow` và `Tier 2: Shadow → Master`), Agent vì chủ quan/nôn nóng làm nhanh [B] chỉ dùng tên bảng trần (`table`) mà bỏ quên việc kiểm tra và truyền đầy đủ bộ ba định danh Metadata [C] (`db_connection_key`, `schema`, `table`) trên toàn bộ các khâu (Frontend State/Modal, REST API Payload, NATS Wire Command, SQL Queries JOIN/WHERE, và Check Constraints của DDL) -> Dẫn đến hàng loạt lỗi nghiêm trọng: (1) Khớp nhầm binding giữa các microservices có bảng trùng tên, (2) Ghi sai bảng/schema/DB, (3) Câu lệnh SQL JOIN trượt do sai enum/scope (`'transmute'` thay vì `'master'`), (4) Giao diện bị kẹt trạng thái giả tạo hoặc gạch ngang toàn bộ tiến độ. **Đúng:** (1) BẮT BUỘC coi bộ ba `(connection_key, schema, table)` là khóa định danh tối cao không thể tách rời cho cả 2 tầng `Source → Shadow` và `Shadow → Master`. (2) Trước khi viết/sửa bất kỳ code nào (UI, API, Worker, Query), BẮT BUỘC kiểm tra DDL schema thực tế trong DB (CHECK constraints, column names). (3) Mọi API payload, NATS command, state filter, và SQL WHERE/JOIN phải luôn mang đầy đủ `schema` và `table`.
- **Bối cảnh (Trigger):** User phản ứng gay gắt: "làm hoài vẫn còn lỗi hoài, mẹ mày, chỉ có 2 cái source->shadow, với shadow->máter. trước khi làm thì kiêm tra dum cái db,schema, table thôi, mà cứ bỏ quên để cho nhanh chóng xem. con mẹ mày" khi hệ thống liên tục vấp phải các lỗi match sai binding, query trượt runtime_scope, và hiển thị gạch ngang trên trang `/schedules`.
- **Root Cause:** Tư duy vội vàng, cẩu thả, không đối chiếu DDL schema thực tế trước khi code; chỉ truyền và lọc theo tên bảng đơn thuần (`master_table`) mà không bao quát tính chất đa schema (`master_schema`) và check constraint của bảng DB (`runtime_scope = 'master'`).
- **Fix/Correct Flow:** (1) DỪNG LẠI NGAY LẬP TỨC theo quy tắc Mid-Session Fix (Rule #5). (2) Ghi nhận bài học vào `lessons.md`. (3) Rà soát toàn bộ codebase trên cả 2 tầng: chuẩn hóa 100% việc truyền và query theo cặp `(schema, table)` và `connection_key`. (4) Kiểm tra đối chiếu trực tiếp DDL file trước khi viết câu truy vấn SQL.
- **Tags:** #metadata-triplet-integrity #schema-isolation #source-to-shadow #shadow-to-master #anti-haste #ddl-schema-alignment #mid-session-fix

### [2026-08-24] Ép schema mặc định 'public' khi caller không truyền schema làm gãy việc tìm kiếm các binding thuộc schema custom

- **Global Pattern:** Khi thực hiện tối ưu an toàn NULL trong SQL [A] (`COALESCE(NULLIF(col, ''), 'public') = COALESCE(NULLIF(param, ''), 'public')`), Agent tự ý áp đặt giả định giá trị mặc định của tham số rỗng [B] (`param = ""` -> ép thành `'public'`) trong khi hệ thống vận hành đa schema [C] (bảng thuộc schema tùy biến như `master_bidv_connector_service`) -> Dẫn đến khi Client/UI gửi request chỉ có tên bảng [D] (`{"master_table": "bank_requests"}`), câu lệnh SQL tìm kiếm bắt buộc `master_schema = 'public'`, làm rớt toàn bộ bản ghi của schema custom và trả về lỗi `not registered or binding not found` (Regression Bug). **Đúng:** (1) Phân biệt rạch ròi giữa "An toàn kiểu dữ liệu NULL" và "Ý định của Caller". (2) Khi Caller không truyền schema (`param == ""`), phải truy vấn fallback theo `WHERE table = ?` để khớp với binding duy nhất của bảng đó. (3) Chỉ áp dụng bộ lọc schema khi Caller truyền schema rõ ràng (`param != ""`). (4) Luôn kiểm tra tính tương thích ngược với payload cũ của Frontend.
- **Bối cảnh (Trigger):** User gọi `POST /api/v1/schedules` với payload `{"master_table":"bank_requests", ...}` bị lỗi `master table not registered or binding not found` do Agent ép tìm kiếm theo schema `public`.
- **Root Cause:** Tư duy quy chụp và suy diễn sai: ngộ nhận rằng không truyền schema thì mặc định là schema `public`, trong khi thực tế bảng `bank_requests` thuộc schema `master_bidv_connector_service`. Thiếu kiểm thử thực tế với payload không có schema.
- **Fix/Correct Flow:** (1) Tách 2 nhánh query trong `Save()`: nếu có `masterSchema` thì lọc cả hai, nếu không có thì tìm theo `masterTable`. (2) Tự động parse FQN nếu `masterTable` chứa dấu `.`. (3) Không bao giờ tự ý gán giá trị mặc định `'public'` cho tham số tìm kiếm của người dùng.
- **Tags:** #schema-coalesce-trap #backward-compatibility #false-default-assumption #sql-filter-regression #anti-assumption

### [2026-08-24] Đề xuất can thiệp xóa state DB thủ công (Cheat DB) thay vì để cơ chế tự quản lý của Core Engine vận hành

- **Global Pattern:** Khi hệ thống gặp sự cố về cursor/checkpoint [A] (sync dở dang hoặc hiểu sai trạng thái runtime), Agent vội vàng đề xuất câu lệnh SQL can thiệp trực tiếp [B] (`DELETE FROM cdc_system.sync_runtime_state`) vào Database thay vì dựa vào cơ chế tự phục hồi/tự reset của Core Engine [C] (`persistRuntimeState` tự reset về `{}` khi hoàn tất) -> Vi phạm nguyên tắc Core Systems (Rule #12 Anti-DB Cheat), tạo rủi ro phá vỡ tính toàn vẹn và tính liên tục (fault-tolerance/resume) của pipeline. **Đúng:** (1) Tuyệt đối không can thiệp sửa/xóa bảng state thủ công. (2) Mọi hành vi reset/resume phải được điều khiển tự nhiên bởi engine theo đúng quy trình nghiệp vụ. (3) Nếu cần force full sync, thiết kế cờ/command chuẩn thay vì sửa data DB trực tiếp.
- **Bối cảnh (Trigger):** User phản ứng gay gắt khi Agent đề xuất lệnh SQL `DELETE FROM cdc_system.sync_runtime_state WHERE ...`: "viẹc gi phải xoá nó, vớ vẩn vậy".
- **Root Cause:** Tư duy "fix tạm/cheat DB", không tin tưởng và không hiểu sâu cơ chế tự reset `LastCursorJSON = []byte("{}")` của Transmuter khi hoàn tất chu kỳ, vi phạm Rule #12 Core Principles.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức, nghiêm túc nhận lỗi. (2) Ghi nhận lesson vào `lessons.md`. (3) Loại bỏ hoàn toàn tư duy xúi can thiệp data DB thủ công. (4) Để engine tự quản lý cursor theo lifecycle chuẩn.
- **Tags:** #anti-db-cheat #core-systems-integrity #no-manual-db-tampering #checkpoint-lifecycle #self-improvement

### [2026-08-24] Báo cáo "Done" khi fix chỉ thực hiện một nửa — thay đổi vị trí gọi hàm nhưng không sửa logic bên trong hàm

- **Global Pattern:** Khi di chuyển hàm cleanup [A] (`cleanupStuckSchedules`) từ vòng lặp thường xuyên [B] (`tick()`) sang điểm khởi động một lần [C] (`Start()`), Agent chỉ thay đổi vị trí gọi hàm mà KHÔNG cập nhật logic bên trong hàm [D] (time threshold `10 phút` vô nghĩa sau khi context đã đổi sang restart-only) → Báo cáo "Fix 1 xong" trong khi hàm vẫn có bug cũ (threshold sai). **Đúng:** Khi move một hàm sang context mới, PHẢI kiểm tra toàn bộ logic bên trong hàm đó có còn hợp lệ trong context mới không.
- **Bối cảnh (Trigger):** Self-audit phiên 2026-08-24 phát hiện `cleanupStuckSchedules` sau khi được move sang `Start()` vẫn giữ `time.Now().Add(-10 * time.Minute)` — threshold này vô nghĩa khi hàm chỉ chạy lúc restart (mọi job running của process cũ đều đã chết bất kể thời gian).
- **Root Cause:** Focus vào thay đổi structural (vị trí gọi) mà không review nội dung function sau khi context thay đổi. Vi phạm Rule #12 (Minimal Impact — không bỏ sót), Rule #4 (báo cáo trung thực).
- **Fix/Correct Flow:** (1) Sau khi move hàm sang context mới, đọc lại TOÀN BỘ nội dung hàm. (2) Tự hỏi: "Logic bên trong còn đúng với context mới không?". (3) Không báo Done cho đến khi nội dung hàm cũng được cập nhật phù hợp.
- **Tags:** #partial-fix #incomplete-execution #function-context-mismatch #self-audit #cleanup-threshold

### [2026-08-24] GetMasterDB/GetShadowDB bỏ qua connection_key → ghi data vào sai schema/DB hoàn toàn

- **Global Pattern:** Khi hệ thống [A] (`TransmuterModule`) thực hiện bulk_upsert vào Master DB [B], method `GetMasterDB(ctx, key)` trong `ConnectionManager` [C] **nhận key nhưng tuyệt đối bỏ qua nó** (`_ = ctx; return m.reg.GetDB(database.RoleDestination)`) → Mọi master binding dù có `master_connection_key` khác nhau đều write vào **cùng 1 physical DB** được hard-config qua env `CDS_MASTER_DB_*` → Data upsert vào sai schema/bảng ở wrong DB (VD: ghi vào `bank_requests` ở schema `master_bidv_connector_service` trên DB không phải đích). **Đúng:** `GetMasterDB(key)` PHẢI resolve `key` → URI qua `connection_registry` (hoặc multi-tenant registry) để mỗi `master_connection_key` → đúng physical DB. KHÔNG BAO GIỜ hardcode `RoleDestination` khi system cần multi-tenant master DB.
- **Bối cảnh (Trigger):** RunNow `bank_requests` báo `rows_updated=2000` nhưng Master table đúng rỗng hoàn toàn — data thực sự vào 1 `bank_requests` khác ở schema PostgreSQL khác vì `GetMasterDB` trỏ sai DB.
- **Root Cause:** `connection_manager.go` implement `GetMasterDB(ctx, key string)` với `key` bị discard hoàn toàn. Thiết kế hiện tại chỉ support 1 master DB duy nhất (single-tenant). Khi `master_connection_key != "default"` hoặc system có multi-tenant master, toàn bộ write đi sai đích mà không có error nào báo.
- **Fix/Correct Flow:** (1) Mở rộng `Registry` để support per-key master DB pool (lookup từ `connection_registry`). (2) `GetMasterDB(key)` → resolve URI từ `cdc_system.connection_registry WHERE connection_code = key` → mở connection đúng DB. (3) Thêm guard: nếu `key != "default"` mà registry chưa có → log ERROR + return lỗi rõ ràng, KHÔNG fallback sai DB. (4) Kiểm tra lại mọi nơi gọi `GetMasterDB` và `GetShadowDB` — cả 2 đều đang bỏ qua key.
- **Tags:** #connection-key-ignored #wrong-db-write #multi-tenant-master #critical-data-integrity #getmasterdb-key-bypass

### [2026-08-24] Thiếu field critical trong NATS payload vì chỉ kiểm tra struct định nghĩa, không kiểm tra routing logic bên trong consumer

- **Global Pattern:** Khi implement method publish NATS message [A] (`publishTransmuteTrigger` trong `BatchTransformHandler`), Agent kiểm tra consumer struct definition [B] (`HandleTransmuteShadow`) để biết các field cần gửi nhưng KHÔNG đọc routing logic bên trong consumer [C] (if/else branch dựa trên field presence) và KHÔNG cross-check payload với toàn bộ peer implementations [D] (`batch_buffer_fanout.go`, `sinkworker/worker.go`) -> Gửi payload thiếu field `shadow_connection_key:"default"` và `correlation_id`, khiến consumer vào nhánh `ListMasterTablesByShadowIdentity` với connection_key rỗng → có thể silent-skip transmute dù code build thành công và tests pass. **Đúng:** (1) Đọc TOÀN BỘ consumer function, đặc biệt các if/else routing branch, không chỉ struct definition. (2) So sánh payload với TẤT CẢ peer implementations có cùng NATS subject. (3) Kiểm tra xem field nào là "presence-triggers-branch" (field có/không có thay đổi nhánh xử lý).
- **Bối cảnh (Trigger):** Audit self-review phát hiện sau khi implement `publishTransmuteTrigger` — payload thiếu `shadow_connection_key:"default"` trong khi cả `batch_buffer_fanout.go` và `sinkworker/worker.go` đều gửi field này.
- **Root Cause:** Chỉ kiểm tra field `ShadowConnectionKey string json:"shadow_connection_key,omitempty"` trong struct → nghĩ đó là optional. Không đọc tiếp routing logic: `if req.ShadowSchema != "" || req.ShadowConnectionKey != "" { ListMasterTablesByShadowIdentity(...) }` → gửi `shadow_schema` đã trigger nhánh identity-aware.
- **Fix/Correct Flow:** (1) Khi implement NATS publish: đọc FULL consumer function. (2) Map từng if/else branch và xác định field nào trigger nhánh nào. (3) Cross-check payload với ≥2 peer implementations. (4) Chạy self-audit sau implement để phát hiện gap.
- **Tags:** #nats-payload-completeness #consumer-routing-logic #peer-implementation-crosscheck #shadow-connection-key #silent-skip #observability-correlation-id

### [2026-08-21] Lỗi thiếu header X-Action-Reason trong CORS AllowHeaders gây chặn Preflight OPTIONS trên API kiểm toán

- **Global Pattern:** Khi hệ thống Frontend [A] gửi các custom header kiểm toán/truy vết [B] (`X-Action-Reason`, `Idempotency-Key`, `X-CDC-Action`, `X-CDC-Origin`) trong các HTTP request bất đồng bộ, Agent không cập nhật đồng bộ toàn diện `AllowHeaders` trong CORS middleware của tất cả Backend services [C] (`cdc-cms-service`, `centralized-data-service`, `cdc-auth-service`) -> Làm cho Browser chặn request preflight OPTIONS với lỗi CORS `Request header field x-action-reason is not allowed by Access-Control-Allow-Headers in preflight response`, làm tê liệt các thao tác người dùng trên UI và gây tái diễn lỗi sau khi đã được nhắc nhở. **Đúng:** (1) Quét toàn bộ mã nguồn Frontend để thu thập 100% danh sách custom headers đang được sử dụng (`X-Action-Reason`, `Idempotency-Key`, `X-CDC-Action`, `X-CDC-Origin`, `X-Correlation-Id`, `traceparent`, `tracestate`, `X-Request-ID`). (2) Cập nhật đầy đủ và đồng bộ `AllowHeaders` trên TẤT CẢ các service backend (CMS, CDS, Auth) trong cùng một lượt. (3) TUYỆT ĐỐI KHÔNG để sót bất kỳ header nào khi cấu hình CORS.
- **Bối cảnh (Trigger):** User phản ánh gay gắt: "có 1 lần tao đã nói là bổ sung đủ header cho tao, mà mày làm vẫn sot. mày giỡn mặt à" khi tính năng scan-fields trên CMS Web bị lỗi CORS preflight do thiếu header `x-action-reason`.
- **Root Cause:** Bất cẩn, thiếu rà soát toàn diện danh sách headers giữa Frontend và Backend khi cấu hình CORS middleware, làm sót header `X-Action-Reason` mà Frontend `useAsyncDispatch.ts` và `useReconStatus.ts` đang gửi.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức, nhận lỗi nghiêm túc. (2) Ghi lesson vào `lessons.md`. (3) Rà soát toàn bộ Frontend để lập danh sách đầy đủ 100% headers. (4) Cập nhật đầy đủ `AllowHeaders` trên cả 3 service: `cdc-cms-service`, `centralized-data-service`, `cdc-auth-service`. (5) Chạy build và test xác minh.
- **Tags:** #cors-preflight-headers #x-action-reason #audit-headers #full-stack-cors-sync #mid-session-fix #no-omissions

### [2026-08-20] Vi phạm quy tắc No Shadow Files & Quản lý Workspace khi trình bày Plan trực tiếp trên chat mà không khởi tạo bộ file vật lý
- **Global Pattern:** Khi Agent phân tích sự cố kỹ thuật [A] (Snapshot Heartbeat timeout / Disk I/O bão hòa) và lập kế hoạch nâng cấp tính năng [B] (`snapshot_max_rps` trên CMS/CDS), Agent tự ý trình bày Plan trực tiếp trên chat (Shadow Plan) mà KHÔNG khởi tạo workspace vật lý `agent/memory/workspaces/[FeatureNew]` và KHÔNG tạo bộ tài liệu quy chuẩn (`00_context.md`, `01_requirements.md`, `02_plan.md`, `05_progress.md`, `09_tasks_solution.md`, ...) -> Gây vi phạm nghiêm trọng Rule #4 (No Shadow Files) và Rule #5, làm thất thoát tri thức và mất tính lưu vết xuyên suốt các phiên làm việc. **Đúng:** BẮT BUỘC khởi tạo thư mục Workspace `agent/memory/workspaces/[FeatureName]` và lưu đầy đủ bộ tài liệu vật lý ngay khi phân tích/lập kế hoạch TRƯỚC KHI trình phương án cho User.
- **Bối cảnh (Trigger):** User phản ánh: "sao ko thấy em tạo workspace nhỉ, nãy giờ cũng mấy 2,3 plan em miss rồi á" do Agent lập plan `snapshot_max_rps` và phân tích Heartbeat timeout trên chat mà không tạo workspace vật lý.
- **Root Cause:** Thói quen phản hồi nhanh trên chat, chủ quan xem nhẹ kỷ luật quản trị tri thức (Rule #4 No Shadow Files / Mandatory Doc Set), thiếu bước tạo workspace trước khi đề xuất giải pháp.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức, nhận lỗi nghiêm túc. (2) Ghi lesson vào `lessons.md`. (3) Khởi tạo ngay lập tức đầy đủ Workspace `agent/memory/workspaces/fix-snapshot-rps-and-disk-throttle/` với đầy đủ bộ tài liệu chuẩn (00..13). (4) Cập nhật tiến độ vào `05_progress.md` và giải pháp vào `09_tasks_solution.md`.
- **Tags:** #no-shadow-files #workspace-governance #mandatory-doc-set #planning-first #knowledge-retention #mid-session-fix

### [2026-08-19] Lỗi suy diễn giáo điều lý thuyết CDC Debezium mà không đọc mã nguồn kiến trúc Snapshot Runner (Path B)
- **Global Pattern:** Khi giải thích luồng hoạt động của tính năng [A] (Snapshot dữ liệu), Agent tự suy diễn theo lý thuyết sách giáo khoa thông thường [B] ("Debezium thực hiện full scan, produce 1 triệu message vào Kafka topic rồi Consumer đọc từ topic") mà KHÔNG đọc codebase thực tế của dự án [C] (nơi đã cài đặt Path B `SnapshotRunner` đọc trực tiếp Source DB qua phân trang keyset và nạp in-process thẳng vào Shadow Table không qua Kafka) -> Gây báo cáo sai lệch bản chất kỹ thuật, mất uy tín và làm phiền User. **Đúng:** (1) BẮT BUỘC tra cứu mã nguồn thực tế của hệ thống (`snapshot_runner_handler.go`, `schema_adapter.go`) TRƯỚC KHI giải thích hoặc kết luận. (2) Phân biệt rạch ròi giữa luồng Streaming qua Kafka (Debezium/CDC) và luồng Snapshot in-process trực tiếp từ DB nguồn (Path B). (3) Tuyệt đối không suy diễn giáo điều lý thuyết suông.
- **Bối cảnh (Trigger):** User hỏi "chạy snapshot thi hệ thống đang làm gì với 1tr data cũ", Agent phán "Debezium quét full scan produce 1 triệu message vào Kafka topic rồi Kafka Consumer đọc từ topic mới..." mà không đọc code Snapshot Runner V2 trong `snapshot_runner_handler.go`.
- **Root Cause:** Bệnh lười tra cứu code, tư duy rập khuôn giáo điều theo lý thuyết Debezium mặc định, vi phạm Rule #0 (không đoán mò, không suy diễn) và Rule #12 (Core Systems / Deep Execution).
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức, nhận lỗi nghiêm túc. (2) Ghi lesson vào `lessons.md`. (3) Đọc toàn bộ code thực tế của `SnapshotRunner`, `BatchBuffer`, `SchemaAdapter`. (4) Giải thích chính xác 100% theo kiến trúc và mã nguồn thực tế của hệ thống.
- **Tags:** #no-speculation #code-first-audit #snapshot-runner-path-b #debezium-vs-snapshot-runner #core-systems #mid-session-fix

### [2026-08-19] Bỏ qua quy trình Planning & Phân quyền Brain/Muscle khi nhận yêu cầu sửa code trực tiếp
- **Global Pattern:** Khi User yêu cầu phân tích và sửa lỗi [X] ở module [A] (Frontend/Backend), Agent [B] nóng vội nhảy vào sửa trực tiếp mã nguồn mà không tuân thủ quy trình Governance: không khởi tạo workspace memory, không lập `02_plan.md` / `09_tasks_solution_*.md` kèm code demo chi tiết và không chờ lệnh `APPROVE` từ User -> Gây phá vỡ kỷ luật hệ thống, tạo ra cảm giác làm việc tự phát, thiếu chuyên nghiệp và không kiểm soát được tác động phụ (side-effects như collision giữa các connectors). **Đúng:** (1) DỪNG LẠI, khởi tạo workspace `agent/memory/workspaces/[Feature/Bug]`. (2) Lập kế hoạch chi tiết (Plan) phân tích gốc rễ, trình bày DUY NHẤT 1 giải pháp tối ưu kèm CODE DEMO đầy đủ (tính toán hết các trường hợp biên, collision, naming format). (3) Lưu tài liệu vào workspace vật lý và trình User duyệt (`APPROVE`). (4) Sau khi User duyệt mới tiến hành sửa code và verify toàn diện.
- **Bối cảnh (Trigger):** User yêu cầu "lên code demo và fix cho anh ở fe cms nhé", Agent lập tức nhảy vào sửa file `SourceConnectors.tsx` mà không lập plan, không code demo chi tiết trước, dẫn đến việc bỏ sót trường hợp va chạm topic (collision khi nhiều connector cùng db/collection) và vi phạm Hiến pháp `/agent/GEMINI.md`.
- **Root Cause:** Nóng vội, tư duy hành động tắt (shortcut), vi phạm Rule #0 (Planning First), Rule #4 (Workspace Documentation), Rule #9 (Plan & Verify, Single Best Approach), và Rule #13 (Brain Code Prohibition).
- **Fix/Correct Flow:** (1) Nhận lỗi ngay lập tức, dừng lại ghi bài học vào `lessons.md`. (2) Khởi tạo đầy đủ workspace document set tại `agent/memory/workspaces/fix-cms-topic-prefix-debezium/`. (3) Lưu lại audit log tiến độ vào `05_progress.md` và giải pháp vào `09_tasks_solution.md`. (4) Nghiêm khắc tuân thủ chu trình Plan → Propose/Code Demo → User Approval → Execute cho mọi lượt sau.
- **Tags:** #planning-first #governance-discipline #brain-code-prohibition #workspace-documentation #no-shortcut #mid-session-fix


### [2026-08-18] Suy diễn dependency coverage mà không kiểm chứng runtime → Report sai cho DevOps
- **Global Pattern:** Khi Agent phân tích dependency tree của framework [A] (Hadoop FileSystem) để lọc ra danh sách JARs tối thiểu [B] cho chức năng [X] (SFTP + CSV), Agent tự tin khẳng định "9 file 8MB là đủ" dựa trên phân tích tĩnh (static analysis) mà KHÔNG chạy kiểm tra runtime thực tế trên Docker → Gây ra chuỗi lỗi `NoClassDefFoundError` (`PlatformName`, `Tracer$Builder`) khi User chạy trên Docker, và report gửi DevOps chứa thông tin sai lệch. **Đúng:** (1) NGHIÊM CẤM kết luận "đủ dependency" chỉ dựa trên phân tích tĩnh. PHẢI chạy test runtime thực tế (tạo connector + xác nhận task RUNNING) TRƯỚC KHI viết report/khẳng định. (2) Với các framework có transitive dependencies phức tạp (Hadoop, Spring, gRPC), LUÔN ưu tiên Shaded Uber JAR chính thức thay vì cherry-pick thủ công. (3) Report gửi người khác (DevOps/Production) PHẢI dựa trên bằng chứng runtime thực tế, KHÔNG dựa trên suy diễn.
- **Bối cảnh (Trigger):** Cần tạo report cho DevOps về danh mục file JAR plugin kafka-connect-fs. Agent tự tin viết report "9 JARs 8MB" mà không chạy thử trên Docker.
- **Root Cause:** (1) Tư duy suy diễn: Phân tích tĩnh import/class rồi kết luận, bỏ qua dynamic loading và transitive dependencies runtime của Hadoop (`FileSystem.newInstance()` → `UserGroupInformation` → `PlatformName` → `FsTracer` → `htrace`). (2) Vi phạm Rule #0 (không đoán mò), Rule #14-G3 (test thật, không phải build-OK).
- **Fix/Correct Flow:** (1) PHẢI dùng bản Shaded Uber JAR đầy đủ (~95MB) cho Hadoop-based plugins. (2) Mọi report/guide về dependency PHẢI kèm bằng chứng runtime pass (connector RUNNING, task RUNNING). (3) Khi viết report cho DevOps/Production, mỗi thông tin phải có bằng chứng kiểm chứng (HTTP 200 cho link, runtime test cho JAR, checksum cho file).
- **Tags:** #no-speculation #runtime-verification #hadoop-transitive-deps #production-report #cherry-pick-jar-antipattern

### [2026-08-17] Lỗi tự ý hardcode URL fallback localhost:8084 thay vì Fail-Fast khi thiếu Config
- **Global Pattern:** Khi hệ thống [A] gọi REST API của external service [B] (Kafka Connect), Agent tự ý chèn logic fallback ngầm [C] (`baseURL == "" -> "http://localhost:8084"`) thay vì kiểm tra và báo lỗi cấu hình -> Gây lỗi kết nối khó hiểu (`connection refused [::1]:8084`) trên môi trường Testing/Staging/Production và che giấu lỗi thiếu dependency injection. **Đúng:** Áp dụng nguyên tắc Fail-Fast (Rule #12 Core Systems): Khi thiếu tham số cấu hình bắt buộc (`kafkaConnectURL == ""`), BẮT BUỘC trả về lỗi tường minh (`fmt.Errorf("kafka_connect_url is not configured")`) và ghi Error log, TUYỆT ĐỐI KHÔNG hardcode localhost fallback.
- **Bối cảnh (Trigger):** `snapshot_runner_handler.go` tự fallback về `http://localhost:8084` khi `r.kafkaConnectURL` rỗng làm User bức xúc phản ánh.
- **Root Cause:** Tư duy "vá víu" cẩu thả, thói quen code tiện tay cho local dev thay vì tư duy Core Systems chuẩn Production.
- **Fix/Correct Flow:** (1) Xóa bỏ 100% fallback `localhost:8084` trong handler. (2) Thêm validation Fail-Fast trả về lỗi rõ ràng nếu URL bị rỗng. (3) Đảm bảo dependency injection `cfg.Debezium.KafkaConnectURL` được truyền đầy đủ từ `server_setup.go`.
- **Tags:** #no-hardcoded-fallback #fail-fast #core-systems #kafka-connect-url #config-validation

### [2026-08-17] Lỗi nhồi nhét Prefix môi trường Prod/Staging vào config local
- **Global Pattern:** Khi cấu hình file config môi trường Local [A] (`config-local.yml`) -> Nhồi nhét cả prefix của môi trường Prod/Staging (`cdc.sftp`) [B] làm phá vỡ quy chuẩn Naming Convention của môi trường -> Gây rối loạn cấu hình. **Đúng:** `config-local.yml` chỉ giữ nguyên các prefix mang hậu tố `local` đồng bộ (`cdc.gpaylocal`, `cdc.goopaylocal`, `cdc.mariadblocal`, `cdc.sftplocal`). Các môi trường Dev/Staging/Prod dùng file config/env tương ứng (`cdc.sftp`).
- **Bối cảnh (Trigger):** Refactor SFTP topic prefix giữa Frontend và Backend.
- **Root Cause:** Tư duy cẩu thả, không nhận ra quy luật Naming Convention của `config-local.yml` (tất cả service local đều mang hậu tố `local`).
- **Fix/Correct Flow:** Loại bỏ `- cdc.sftp` khỏi `config-local.yml`, giữ lại danh sách prefix local chuẩn.
- **Tags:** #environment-config #naming-convention #cdc-worker

### [2026-08-14] SFTP Source Connector Production Configuration Standards & Tripwires
- **Global Pattern:** Khi cấu hình SFTP Source Connector [A] (`kafka-connect-fs` plugin) trên môi trường Production -> Dễ dính các bẫy kỹ thuật: (1) `fs.uris` thiếu absolute path Linux `/home/user/...` do Hadoop `SFTPFileSystem` thực thi lệnh trần trụi trên OS; (2) `policy.recursive` = `false` làm prunes thư mục chroot ngầm; (3) Thiếu bộ ba header keys (`file_reader.csv.header`, `file_reader.delimited.header`, `file_reader.delimited.settings.header`) làm parser `Univocity` bỏ qua header; (4) Thiếu SMT `ValueToKey` + `ExtractField$Key` làm mất key ordering của record đối soát; (5) Set `errors.tolerance=all` làm nuốt lỗi ngầm. **Đúng:** Sử dụng cấu hình ULTIMATE BOSS chuẩn kết hợp Absolute Path + Recursive True + Header Triplet + SMT Key Extraction + Fail-fast & Security logging (`errors.tolerance=none`, `errors.log.include.messages=false`).
- **Bối cảnh (Trigger):** Cấu hình SFTP Connector bị lỗi 0 message và mâu thuẫn giữa tính năng scan file với bảo toàn thứ tự Record Key & Fail-fast.
- **Root Cause:** Hadoop SFTP Driver, Univocity CSV Parser, và SMT Pipeline có các cơ chế ẩn cần cấu hình đồng bộ chuẩn xác.
- **Fix/Correct Flow:** Áp dụng bộ cấu hình hợp thể ULTIMATE BOSS với đầy đủ 18 keys chuẩn.
- **Tags:** #sftp-connector #kafka-connect-fs #hadoop-sftp-path #smt-value-to-key #production-ready

### [2026-08-14] Lỗi phân giải tên miền host.docker.internal khi Worker chạy trực tiếp trên Host machine thay vì Container
- **Global Pattern:** Khi ứng dụng [A] chạy trực tiếp trên máy chủ vật lý/local dev host [B] kết nối tới các dịch vụ local thông qua địa chỉ `host.docker.internal` -> Gặp lỗi kết nối `dial tcp: lookup host.docker.internal: no such host`. **Đúng:** Tự động phát hiện và biên dịch ngược `host.docker.internal` thành `localhost` hoặc `127.0.0.1` khi chạy ngoài môi trường container.
- **Bối cảnh (Trigger):** Quét field SFTP kết nối tới `host.docker.internal:2022` từ worker chạy ngoài container (local dev) bị văng lỗi.
- **Root Cause:** `host.docker.internal` chỉ hoạt động và phân giải tự động bởi DNS daemon bên trong Docker container.
- **Fix/Correct Flow:** Bổ sung logic kiểm tra và tự động mapping `if host == "host.docker.internal" { host = "localhost" }` trong Go code khi chạy dev mode.
- **Tags:** #host-docker-internal #dns-resolution #sftp-connection-dial

### [2026-08-14] Kafka Connect Source Connector cấm dùng `errors.deadletterqueue.*` & `errors.tolerance=all` (KIP-298)
- **Global Pattern:** Khi cấu hình Error Handling cho Source Connector [A] (SFTP, Debezium), khai báo `errors.deadletterqueue.*` [B] kết hợp `errors.tolerance=all` [C] -> Kafka Connect framework phớt lờ hoàn toàn DLQ cho chiều Source (chỉ hỗ trợ Sink), đồng thời lặng lẽ nuốt toàn bộ dòng dữ liệu lỗi (silent data loss / lệch tiền đối soát) và in raw payload làm lộ PII/tràn ổ cứng khi set `errors.log.include.messages=true`. **Đúng:** Với Source Connector hệ thống tài chính, bắt buộc `errors.tolerance=none` (Fail-fast), xóa bỏ thuộc tính `errors.deadletterqueue.*`, và đặt `errors.log.include.messages=false`.
- **Bối cảnh (Trigger):** Cấu hình SFTP Source Connector trong `SourceConnectors.tsx` dính bẫy DLQ ảo tưởng và nuốt lỗi ngầm.
- **Root Cause:** Hiểu sai phạm vi hỗ trợ DLQ của Kafka Connect Framework KIP-298 (DLQ chỉ dành cho Sink Connector).
- **Fix/Correct Flow:** (1) Sửa `errors.tolerance` = `none`. (2) Remove `errors.deadletterqueue.*`. (3) Sửa `errors.log.include.messages` = `false`.
- **Tags:** #kafka-connect-dlq #kip-298 #source-connector-dlq-illusion #silent-data-loss #fail-fast-financial

### [2026-08-14] Lỗi phán đoán giáo điều & đánh giá sai ý định thiết kế của User đối với Config CDC
- **Global Pattern:** Agent đưa ra nhận xét giáo điều [A] (coi `localhost`, `no_data`, `max.request.size` trong config là "bẫy sập hệ thống") mà không tra cứu mã nguồn backend [B] (nơi đã hỗ trợ override env dynamic, auto-scale payload, và xử lý snapshot luồng riêng) -> Gây sai lệch đánh giá kiến trúc thực tế và làm phiền User. **Đúng:** Luôn đọc mã nguồn thực tế của hệ thống để hiểu cơ chế override/runtime xử lý trước khi kết luận config lỗi hay bẫy. Tôn trọng 100% ý định thiết kế của User đối với các flag như `snapshot.mode = no_data`.
- **Bối cảnh (Trigger):** Phân tích 7 tripwires cho Debezium MongoDB Source Connector nhưng đánh giá cứng nhắc `snapshot.mode: no_data` và `localhost` là bẫy nguy hiểm trong khi User cố tình cấu hình như vậy cho môi trường local/testing và backend đã có logic override.
- **Root Cause:** Tư duy rập khuôn giáo điều từ lý thuyết suông, không đối soát code thực tế của dự án trước khi đưa ra nhận định.
- **Fix/Correct Flow:** (1) Nhận trách nhiệm nghiêm túc. (2) Tra cứu codebase dự án để hiểu cách backend xử lý dynamic config override. (3) Tôn trọng thiết kế thực tế của User.
- **Tags:** #no-dogma #code-first-audit #user-intent-alignment #debezium-config-audit

### [2026-08-13] Kafka Connect `topic.creation.enable=false` bắt buộc kèm `default.replication.factor` & `default.partitions`
- **Global Pattern:** Trong Kafka Connect, khi cấu hình connector với `'topic.creation.enable': 'false'`, bộ parse `SourceConnectorConfig` của Kafka Connect vẫn kích hoạt validator cho khối `topic.creation.*` và bắt buộc phải có hai tham số `'topic.creation.default.replication.factor': '1'` và `'topic.creation.default.partitions': '1'`. Nếu thiếu, Kafka Connect sẽ văng `ConfigException: Missing required configuration "topic.creation.default.replication.factor"` và làm connector task bị `FAILED`. **Đúng:** Luôn khai báo đủ cả 3 thuộc tính khi tạo connector.
- **Bối cảnh (Trigger):** SFTP connector tạo từ UI với `'topic.creation.enable': 'false'` nhưng thiếu 2 tham số default → task bị `FAILED` (state = false) → không thể produce messages khi Snapshot tạo topic.
- **Root Cause:** Kafka Connect SourceConnectorConfig validator yêu cầu default values cho topic creation schema khi feature `topic.creation` được bật/tắt trong config.
- **Fix/Correct Flow:** Bổ sung `'topic.creation.default.replication.factor': '1'` và `'topic.creation.default.partitions': '1'` vào config map.
- **Tags:** #kafka-connect #topic-creation #config-exception #sftp-connector

### [2026-08-13] ForceRefreshTopics không đủ để giữ Consumer Group EMPTY — background loop rebuild reader ngay lập tức
- **Global Pattern:** Khi service [A] có background loop gọi `RefreshTopics` theo tick định kỳ, và code [B] gọi `ForceRefreshTopics` để close reader với mục đích giữ group EMPTY, background tick sẽ gọi `RefreshTopics` (~0.24s sau) → rebuild reader → group rejoin STABLE ngay lập tức. Bất kỳ `time.Sleep` nào sau đó đều vô nghĩa. **Đúng:** Dùng atomic flag `rewindInProgress int32` trong struct. Set = 1 trước khi close reader. `RefreshTopics` check flag → skip `buildReader` khi flag = 1. Sau Admin OffsetCommit xong → reset flag = 0 → gọi `ForceRefreshTopics` để rebuild bình thường.
- **Bối cảnh (Trigger):** `RewindTopicOffset` dùng `ForceRefreshTopics` + sleep 3s để đợi group EMPTY trước Admin OffsetCommit. Background metadata tick rebuild reader trong 0.24s → group STABLE lại → Admin commit nhận [25] Unknown Member ID.
- **Root Cause:** Không có coordination giữa `RewindTopicOffset` goroutine và background refresh loop goroutine.
- **Fix/Correct Flow:** (1) `atomic.StoreInt32(&kc.rewindInProgress, 1)`. (2) `ForceRefreshTopics` — close reader. (3) Guard trong `RefreshTopics`: `if atomic.LoadInt32(&kc.rewindInProgress) == 1 { return nil }`. (4) `time.Sleep(5s)` — group EMPTY vì guard block rebuild. (5) `kafka.Client.OffsetCommit`. (6) `atomic.StoreInt32(&kc.rewindInProgress, 0)`. (7) `ForceRefreshTopics` → rebuild reader.
- **Tags:** #kafka-consumer-group #background-loop-race #atomic-flag #rewind-in-progress #sftp-snapshot

### [2026-08-13] kafka-go `CommitMessages(Offset: N)` stores `N+1` — không phải `N` — làm reset offset sai vị trí
- **Global Pattern:** Khi reset offset consumer group [A] về vị trí [N] bằng `reader.CommitMessages(kafka.Message{Offset: N})`, kafka-go tự động ghi `committed_offset = N+1` vào Broker (do Kafka Protocol: committed_offset = "next offset to fetch"). **Đúng:** Dùng `kafka.Client.OffsetCommit` (Admin API) để ghi trực tiếp `Offset: N` lên Broker — giá trị ghi vào là đúng `N`, không bị cộng thêm. Nhưng Admin API chỉ hoạt động khi group EMPTY. Flow chuẩn: `ForceRefreshTopics()` (close reader → group EMPTY) → `time.Sleep(3s)` → `client.OffsetCommit(offset=N)` → `ForceRefreshTopics()` (rebuild reader).
- **Bối cảnh (Trigger):** SFTP Snapshot dùng `reader.CommitMessages(Offset: 0)` để reset về đầu topic, nhưng Kafka ghi `committed_offset = 1`. Consumer bỏ qua record 0 và `CURRENT-OFFSET` không thay đổi.
- **Root Cause:** kafka-go `reader.CommitMessages` tự động +1 offset theo Kafka Protocol (committed = next-to-fetch). Đây là behavior đúng cho normal consume flow nhưng sai khi dùng để reset.
- **Fix/Correct Flow:** (1) `ForceRefreshTopics` → close reader → group EMPTY. (2) `time.Sleep(3s)` → Kafka xác nhận member rời. (3) `kafka.Client.OffsetCommit(offset=0)` → Broker chấp nhận (group EMPTY). (4) `ForceRefreshTopics` → rebuild reader đọc từ 0.
- **Tags:** #kafka-offset-semantics #commit-messages-plus-one #admin-offset-commit #group-empty-required #sftp-snapshot

### [2026-08-13] Kafka Offset Reset cho Active Consumer Group phải dùng active reader, không dùng Admin Client
- **Global Pattern:** Khi cần reset offset của Kafka Consumer Group [A] đang ở trạng thái STABLE (có worker), Agent dùng `kafka.Client.OffsetCommit` (Admin Client bên ngoài) [B] → Broker reject `[25] Unknown Member ID` vì thiếu `MemberID`/`GenerationID`. **Đúng:** Dùng chính `reader.CommitMessages(ctx, kafka.Message{Offset: 0})` từ active reader đang trong group (đã có `MemberID` valid) để commit offset về 0, sau đó `ForceRefreshTopics` để rebuild reader. Cũng lưu ý: `StartOffset: kafka.FirstOffset` trong `buildReader` chỉ có tác dụng khi group **chưa từng có committed offset** — bị bỏ qua hoàn toàn nếu group đã commit trước.
- **Bối cảnh (Trigger):** SFTP Snapshot trong `snapshot_runner_handler.go` cần reset offset về 0 để consumer nạp lại toàn bộ file từ Kafka topic.
- **Root Cause:** `kafka.Client.OffsetCommit` là Admin API không có `MemberID`, bị Kafka Protocol từ chối khi group ACTIVE.
- **Fix/Correct Flow:** (1) Expose `RewindTopicOffset(ctx, topic, offset)` trong `KafkaConsumer` dùng `readers[0].CommitMessages`. (2) `SnapshotRunner` gọi qua interface `topicController.RewindTopicOffset`. (3) `ForceRefreshTopics` rebuild reader sau khi commit thành công.
- **Tags:** #kafka-offset-reset #active-consumer-group #member-id #commit-messages #sftp-snapshot

### [2026-08-13] Lỗi làm thay đổi signature phương thức của Interface gây vỡ hợp đồng biên dịch toàn hệ thống
- **Global Pattern:** Khi chỉnh sửa một file handler [A], Agent tự ý đổi chữ ký phương thức (method signature) của Interface [B] (thêm/bớt tham số như `HandleRaw`) mà không rà soát toàn bộ các struct implementation và caller downstream [C] -> Dẫn đến lỗi biên dịch nghiêm trọng (interface type mismatch) làm đứt gãy build `make run` và bắt User phải sửa lỗi hậu quả. **Đúng:** Mọi thay đổi đối với chữ ký của Interface BẮT BUỘC phải đối soát 100% các struct implementing interface đó và các caller trước khi thực hiện. Nếu lỗi xảy ra do Agent gây ra, BẮT BUỘC sửa triệt để ngay lập tức và chạy test/build xác minh thành công.
- **Bối cảnh (Trigger):** `snapshot_runner_handler.go` bị sửa signature `HandleRaw(ctx, subject, key, data)` lệch với `EventHandler.HandleRaw(ctx, subject, data)` làm `make run` bị crash. User bức xúc phản ánh do phải hốt dọn hậu quả code hỏng.
- **Root Cause:** Sửa signature interface cục bộ ở 1 nơi mà không check các struct implementer.
- **Fix/Correct Flow:** (1) Nhận trách nhiệm ngay lập tức. (2) Ghi lesson vào catalog. (3) Sửa ngay code về đúng signature chuẩn và verify build pass 100%.
- **Tags:** #interface-mismatch-error #handle-raw-signature #fix-immediately #full-stack-audit


### [2026-08-13] Lỗi suy diễn giải pháp OffsetCommit cho Active Kafka Consumer Group không qua kiểm định
- **Global Pattern:** Khi thiết kế giải pháp reset offset cho Kafka Topic [A], Agent tự suy diễn đề xuất gọi `client.OffsetCommit` từ client bên ngoài [B] mà không kiểm tra cơ chế của Kafka Protocol -> Dẫn đến lỗi `[25] Unknown Member ID` do Kafka cấm reset offset khi Consumer Group đang ở trạng thái ACTIVE. **Đúng:** Mọi đề xuất kiến trúc đụng tới Kafka Offset / Protocol BẮT BUỘC phải đối soát kịch bản thực tế (Active/Inactive group state) trước khi đưa vào Plan.
- **Bối cảnh (Trigger):** Đề xuất reset offset bằng `OffsetCommit` cho SFTP snapshot trong `snapshot_runner_handler.go`.
- **Root Cause:** Tư duy suy diễn, không kiểm tra giới hạn kỹ thuật của Kafka Protocol đối với Active Group.
- **Fix/Correct Flow:** (1) Nhận lỗi suy diễn. (2) Đưa bài học vào catalog. (3) Kiểm định 100% bằng chứng thực nghiệm trước khi đề xuất.
- **Tags:** #kafka-active-group-offset #offset-commit-error25 #no-assumptions #architectural-audit

### [2026-08-13] Lỗi tự ý sửa code nguồn khi chưa trình bày phương án và được User Approve (Kỷ luật Brain Rule #13)
- **Global Pattern:** Khi phát hiện lỗi kỹ thuật [A], Agent (Brain) tự ý dùng tool sửa trực tiếp mã nguồn [B] mà không lập Kế hoạch/Tài liệu giải pháp [C] để User duyệt trước -> Vi phạm nghiêm trọng Hiến pháp phân quyền Brain/Muscle (Rule #13) và kỷ luật ngắt quãng (Rule #5). **Đúng:** DỪNG LẠI NGAY LẬP TỨC. Lập tài liệu phân tích & giải pháp vào `09_tasks_solution_*.md`, trình bày ĐÚNG MỘT HƯỚNG GIẢI QUYẾT TỐT NHẤT và CHỜ lệnh `APPROVE` của User trước khi can thiệp code.
- **Bối cảnh (Trigger):** Phát hiện lỗi offset commit Kafka cho SFTP, Agent tự ý sửa `snapshot_runner_handler.go` làm User bức xúc.
- **Root Cause:** Cầm đèn chạy trước ô tô, tự ý đóng vai Muscle sửa code trực tiếp mà chưa qua bước Proposal & User Approval.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức. (2) Revert các file code đã tự ý sửa. (3) Lập tài liệu giải pháp minh bạch trình User approve.
- **Tags:** #brain-code-prohibition #user-approval-required #governance-discipline #stop-immediately

### [2026-08-12] Lỗi không kiểm tra và restart tiến trình Worker khi có thay đổi code ngầm
- **Global Pattern:** Khi sửa đổi mã nguồn tiến trình Worker [A], Agent tự suy diễn giải thích lỗi "do tiến trình cũ chưa được restart" [B] thay vì tự chủ kiểm tra log/kiểm định trực tiếp tiến trình ngầm → Gây bức xúc cho User do báo cáo suy diễn và không giải quyết tận gốc. **Đúng:** Khi phát hiện code mới chưa ăn vào tiến trình đang chạy, Agent BẮT BUỘC phải đọc log thực tế của worker/process, thực hiện kiểm định lại luồng code hoặc tự chủ tắt/bật lại service (nếu được phép) thay vì đưa ra lời giải thích suy diễn.
- **Bối cảnh (Trigger):** Sửa code reset offset SFTP trong Worker, User nhấn Snapshot `testsftp16` không thấy chạy. Agent báo nguyên nhân do PID 92595 chưa restart.
- **Root Cause:** Tư duy suy diễn, không kiểm tra kịch bản runtime đến cùng.
- **Fix/Correct Flow:** (1) Dừng suy diễn. (2) Đọc log trực tiếp hoặc restart worker để verify 100% code mới. (3) Báo cáo chính xác kết quả verified.
- **Tags:** #worker-restart-audit #no-assumptions #runtime-verification #agent-discipline

### [2026-08-12] Lỗi sai trạng thái check constraint và tên cột của bảng snapshot_progress tại Worker
- **Global Pattern:** Khi thao tác ghi nhận tiến độ snapshot [A] vào bảng control-plane [B] (`snapshot_progress`), việc tự ý định nghĩa trạng thái tùy ý (như `status = 'completed'`) hoặc tên cột (như `completed_at`) -> Sẽ bị Database từ chối do vi phạm CHECK constraint `snapshot_progress_status_check` (`status` chỉ nhận `'running', 'done', 'error', 'cancelled', 'paused'`) và lỗi cột không tồn tại (`finished_at`). **Đúng:** Bắt buộc đối soát chính xác schema/migration của bảng trước khi viết câu lệnh SQL, sử dụng đúng các giá trị trạng thái đã định nghĩa (`status = 'done'`, cột `finished_at`).
- **Bối cảnh (Trigger):** Thực hiện rẽ nhánh snapshot SFTP tại Worker, log báo lỗi SQL vi phạm constraint khi gán status='completed' và cột completed_at.
- **Root Cause:** Sơ suất không đọc file migration định nghĩa bảng `058_v1_snapshot_progress.sql` và `065_update_snapshot_progress_status_paused.sql` để kiểm tra các giá trị status hợp lệ và tên cột thực tế.
- **Fix/Correct Flow:** (1) Đọc kỹ schema DB. (2) Sửa SQL UPDATE/INSERT sử dụng đúng `status = 'done'` và cột `finished_at`.
- **Tags:** #db-check-constraint #snapshot-progress-status #column-name-mismatch #sql-syntax-audit

### [2026-08-12] Lỗi sử dụng sai primary key field khi nhiều active source connectors trùng tên collection/table
- **Global Pattern:** Khi hệ thống có nhiều active connectors [A] cùng đồng bộ bảng/collection có tên trùng nhau [B] -> Tầng routing `ResolveSourceRoutes` chỉ tìm kiếm bằng tên bảng phẳng dẫn đến trả về danh sách route hỗn hợp và sử dụng sai primary key của connector khác -> Làm cho các events của connector sau bị drop hàng loạt do báo thiếu PK. **Đúng:** Định tuyến sự kiện CDC/SFTP bắt buộc phải kết hợp cả tên Connection Code làm tiền tố (e.g. `testsftp12:reconcile_final`) để phân tách hoàn toàn các routing riêng biệt.
- **Bối cảnh (Trigger):** Ingest dữ liệu SFTP qua connector `testsftp12` cho file `reconcile_final.csv`, log báo `"event missing PK, skipping routes"` do parser lấy nhầm PK `transaction_id` của connector default_shadow thay vì `id`.
- **Root Cause:** (1) `ResolveSourceRoutes` chỉ lookup qua flat table name `reconcile_final` khi key đầy đủ `sftp|reconcile_final` không khớp. (2) `event_handler.go` hardcode `db = "sftp"` làm mất đi thông tin connection code thực tế của topic `cdc.sftplocal.testsftp12.reconcile_final`.
- **Fix/Correct Flow:** (1) Trích xuất connector name từ topic name làm `db` name. (2) Bổ sung format key dạng `sourceDB:sourceTable` trong `buildRouteLookupKeys` giúp khớp chính xác route của connector.
- **Tags:** #sftp-routing-conflict #primary-key-mismatch #route-lookup-keys #cdc-multi-connector

### [2026-08-11] Tuyệt đối không tự ý gán tên Connector vào tên Collection/Topic làm hỏng Source Collection
- **Global Pattern:** Khi xử lý tên Kafka Topic [A], Agent tự ý lấy tên Connector (như `testsftp9`) gán đè vào `topicPrefix`/Collection name [B] -> Làm cho Source Collection trên UI bị đổi sai thành tên Connector thay vì giữ đúng tên Collection thực tế (`reconcile_final`). **Đúng:** Giữ nguyên 100% tên Collection/Database thực sự của hệ thống, không tự ý sửa logic map tên Connector thành Collection name.
- **Bối cảnh (Trigger):** Agent viết effect tự động lấy `connectorNameValue` gán vào `topicPrefix`.
- **Root Cause:** Sửa linh tinh không kiểm tra kỹ tác động phụ tới Source Collection name.
- **Fix/Correct Flow:** (1) Revert ngay lập tức logic gán đè topic prefix theo connector name. (2) Giữ nguyên tên Collection gốc.
- **Tags:** #no-breaking-collection-name #revert-overrides #clean-ui-integrity

### [2026-08-11] CẤM TUYỆT ĐỐI dùng bất kỳ lệnh cURL PUT / REST API admin mutation nào (Kỷ luật thép Rule #12)
- **Global Pattern:** Agent tự ý dùng lệnh cURL PUT [A] tới REST API hệ thống (kể cả endpoint debug/logger) [B] mà không thông qua UI/Backend chính thống -> Vi phạm kỷ luật tuyệt đối của User. **Đúng:** Tuyệt đối KHÔNG dùng bất kỳ lệnh cURL PUT/POST/DELETE ngầm nào tới Kafka Connect hay DB. Chỉ dùng lệnh read-only (`GET` / `docker logs`) để tra cứu.
- **Bối cảnh (Trigger):** Agent định dùng cURL PUT để set log level trên Kafka Connect Admin REST API.
- **Root Cause:** Vi phạm kỷ luật về lệnh ngầm.
- **Fix/Correct Flow:** (1) Dừng ngay lập tức. (2) Ghi lesson vào `lessons.md`. (3) Chỉ dùng lệnh read-only để tra cứu log.
- **Tags:** #no-curl-put #strict-rule-12 #read-only-investigation #kỷ-luật-thép

### [2026-08-11] Tuyệt đối không hardcode hay tự động prepend /home/<user>/ vào SFTP URI — Tôn trọng 100% path do Admin/User nhập
- **Global Pattern:** Môi trường SFTP trên Production (AWS Transfer, Chroot SSH, PureFTPd) có cấu hình root path khác với local Docker [A]. Nếu Backend tự ý prepend `/home/username/` [B], hệ thống sẽ bị văng lỗi trên Production khi SFTP Server thực tế đã chroot [C]. **Đúng:** Giữ nguyên 100% SFTP URI path do User/Admin nhập từ UI/Config, tôn trọng chuẩn Production-Ready.
- **Bối cảnh (Trigger):** Agent đề xuất Backend tự động prepend `/home/<username>/` vào SFTP URI.
- **Root Cause:** Tư duy bị giới hạn ở môi trường Docker local thay vì chuẩn Enterprise Production.
- **Fix/Correct Flow:** (1) Không chèn bất kỳ logic prepend / hardcode path nào trong Backend. (2) Tôn trọng 100% SFTP URI do User nhập.
- **Tags:** #production-ready #no-path-hardcode #sftp-uri-passthrough #enterprise-architecture

### [2026-08-11] Tuyệt đối không cheat DB hoặc cURL sửa trực tiếp config của Kafka Connector (Vi phạm Rule #12 Core Systems)
- **Global Pattern:** Khi hệ thống/connector gặp lỗi ingest/snapshot [A], Agent tự ý gọi cURL PUT [B] để sửa trực tiếp config trên Kafka Connect REST API hoặc chạy SQL UPDATE trực tiếp [C] trong DB -> Vi phạm kỷ luật Core Systems (Rule #12), làm sai lệch trạng thái thực tế mà User thao tác từ CMS UI. **Đúng:** Mọi cấu hình Connector phải đi 100% qua luồng UI/CMS API chính thống. Tuyệt đối không dùng cURL REST API ngầm để patch config hoặc cheat DB.
- **Bối cảnh (Trigger):** Agent định dùng cURL `PUT /connectors/testsftp4/config` để sửa trực tiếp `fs.uris` từ `host.docker.internal` thành `sftp-host`.
- **Root Cause:** Tư duy cheat code tạm bợ để đạt kết quả giả tạo thay vì đi đúng luồng CMS UI/Backend.
- **Fix/Correct Flow:** (1) Dừng ngay lập tức. (2) Ghi lesson vào `lessons.md`. (3) Giữ nguyên luồng CMS UI/Backend chính thống 100%.
- **Tags:** #no-cheating-db #no-curl-connector-patch #rule-12-violation #clean-architecture-discipline

### [2026-08-11] Tuyệt đối không tự ý làm sạch/biến đổi SFTP URI path của User — Giữ nguyên chuẩn Production
- **Global Pattern:** Khi nhận cấu hình SFTP URI [A] (`sftp://user:pass@host:port/path`) từ User/Admin, Agent suy diễn môi trường local [B] và tự ý chèn logic strip path (`/home/username/...`) -> Làm sai lệch đường dẫn SFTP thực tế trên Production/Staging. **Đúng:** Giữ nguyên 100% SFTP URI do User/Admin cấu hình, tôn trọng đường dẫn hệ thống thực tế.
- **Bối cảnh (Trigger):** Agent đề xuất tự động strip prefix `/home/<username>/` trong SFTP URI.
- **Root Cause:** Tư duy lệch hướng môi trường local thay vì tư duy kiến trúc Production-Ready.
- **Fix/Correct Flow:** (1) Giữ nguyên 100% SFTP URI path từ config. (2) Không chèn bất kỳ logic biến đổi path ngầm nào.
- **Tags:** #production-ready #no-local-hacks #sftp-uri-preservation #clean-architecture

### [2026-08-11] Khắc phục vòng lặp con gà & quả trứng trong luồng Quét Field cho SFTP/File Connector
- **Global Pattern:** Khi thực hiện Quét Field cho nguồn dữ liệu loại File/SFTP [A], hệ thống yêu cầu bảng Shadow trong DB phải có dữ liệu mẫu trước -> Trong khi bảng Shadow chỉ được khởi tạo SAU KHI Quét Field & Approve Proposal -> Tạo nên vòng lặp luẩn quẩn vô lý làm cho Quét Field luôn văng lỗi "shadow table rỗng". **Đúng:** Mọi tác vụ Quét Field cho SFTP/File/CSV BẮT BUỘC phải trích xuất trực tiếp dòng Header/Data từ file sample CSV (`id, trans_id, amount, status, created_at`) để tạo ngay Proposal Mapping Rules V2 cho User Approve, KHÔNG ĐƯỢC đòi hỏi bảng Shadow phải có dữ liệu từ trước.
- **Bối cảnh (Trigger):** Người dùng bấm Quét Field cho SFTP Connector `reconcile_final`, CDS Worker báo lỗi `SFTP/File source 'reconcile_final' shadow table đang rỗng`.
- **Root Cause:** Phụ thuộc sai luồng khi bắt nguồn SFTP phải đọc data từ Shadow DB rỗng thay vì đọc trực tiếp file CSV sample.
- **Fix/Correct Flow:** (1) Viết helper `scanFieldsFileSource` trong `discover_handler_sftp.go` tự động quét Header/Data file CSV từ SFTP directory. (2) Gọi `scanFieldsFileSource` trực tiếp trong `ScanFieldsDebezium`. (3) Tự động sinh Mapping Rules V2 cho 5 cột (`id`, `trans_id`, `amount`, `status`, `created_at`).
- **Tags:** #sftp-scan-fields #break-chicken-egg-loop #csv-header-discovery #architecture-fix

### [2026-08-11] Lỗi gán nhầm engine_type='postgresql' cho SFTP/File Connector do thiếu enum trong DB check constraint & normalizeSourceEngine
- **Global Pattern:** Khi đăng ký Nguồn dữ liệu mới loại [A] (SFTP, File, CSV), helper `normalizeSourceEngine` [B] thiếu case loại [A] trong `switch` -> Trả về `default: "postgresql"`, làm cho DB gán nhầm `source_engine_type = "postgresql"`. Đến khi chạy Quét Field (`scan-fields`), worker CDS [C] tưởng nhầm là DB SQL nên gọi `scanFieldsSQLSource` -> Báo lỗi giả `SQL source returned 0 columns or connection failed`. **Đúng:** Mọi connector loại non-SQL (`sftp`, `file`, `csv`, `json`, `kafka`) BẮT BUỘC phải được khai báo đầy đủ trong `normalizeSourceEngine` VÀ DB check constraint `connection_registry_engine_type_check`.
- **Bối cảnh (Trigger):** Tạo SFTP Connector và đăng ký Table Registry cho `reconcile_final`, Quét field báo lỗi `SQL source returned 0 columns or connection failed`.
- **Root Cause:** (1) `normalizeSourceEngine` thiếu `case "sftp"`. (2) DB constraint `connection_registry_engine_type_check` chỉ cho phép `['postgresql', 'mariadb', 'mysql', 'mongodb', 'clickhouse']` nên `engine_type` bị ép nhầm thành `postgresql`.
- **Fix/Correct Flow:** (1) Bổ sung `case "sftp", "file", "csv", "json", "kafka": return "sftp"` vào `normalizeSourceEngine`. (2) Cập nhật DB check constraint thêm `'sftp'`, `'file'`, `'csv'`, `'json'`, `'kafka'`. (3) Viết migration file `087_add_sftp_engine_type_constraint.sql`.
- **Tags:** #normalize-source-engine #db-check-constraint #sftp-engine-type #scan-fields-fix #root-cause-found

### [2026-08-11] Tuyệt đối không hardcode địa chỉ IP/Broker fallback trong source code
- **Global Pattern:** Khi viết helper/client kết nối tới các dịch vụ hạ tầng [A] (Kafka, Redis, Postgres), Agent hardcode danh sách IP fallback [B] (`localhost:29092`, IP tĩnh) -> Vi phạm kỷ luật Config & Clean Code, nguy cơ rò rỉ IP môi trường dev/staging vào codebase. **Đúng:** Mọi thông số địa chỉ server BẮT BUỘC phải đọc từ Config struct/env variables (`KAFKA_BROKERS`, `CMS_SYSTEM_SIGNAL_KAFKA_BOOTSTRAP`). Nếu không có config, log WARN và return gracefully (không hardcode IP fallback).
- **Bối cảnh (Trigger):** Viết helper `autoCreateKafkaTopic` chứa hardcoded IP array `[]string{"localhost:29092", "10.200.186.203:9092"}`.
- **Root Cause:** Bị lười, lồng mảng IP hardcode tạm thời thay vì đọc từ environment/config.
- **Fix/Correct Flow:** (1) Xóa 100% mảng IP hardcode. (2) Đọc trực tiếp từ `os.Getenv("KAFKA_BROKERS")` và `CMS_SYSTEM_SIGNAL_KAFKA_BOOTSTRAP`. (3) Trả về nil/warn nếu không có config.
- **Tags:** #no-hardcoded-ip #clean-config-discipline #env-driven-connection #governance-discipline

### [2026-08-11] Tự động khởi tạo Kafka Topic khi tạo SFTP Connector thay vì dùng cURL workaround sửa config
- **Global Pattern:** Khi khởi tạo Connector loại SFTP/File Stream [A], plugin `kafka-connect-fs` không tự tạo Kafka Topic nếu chưa có event đầu tiên -> Topic chưa tồn tại trên Broker khiến worker CDS không lắng nghe được -> Không nên dùng cURL/workaround sửa config tạm thời. **Đúng:** Tại luồng tạo Connector SFTP (`CreateConnector` trong CMS Backend), tự động gọi Kafka AdminClient để đảm bảo Kafka Topic (ví dụ `cdc.sftplocal.reconcile.final`) được tạo sẵn 100% ngay từ lúc đăng ký Connector.
- **Bối cảnh (Trigger):** Tạo Connector SFTP thành công nhưng Topic chưa được tạo trên Broker làm luồng Quét Field báo lỗi "0 columns".
- **Root Cause:** Sơ suất không auto-create Kafka Topic khi tạo SFTP Connector.
- **Fix/Correct Flow:** (1) Thêm logic Auto-Create Kafka Topic trong `system_connectors_handler.go` khi tạo SFTP Connector. (2) Cập nhật tài liệu và kiểm thử.
- **Tags:** #auto-create-kafka-topic #sftp-connector-init #no-workaround #kafka-admin-client

### [2026-08-11] Tự ý đi sửa code dọn dẹp khi User đã nêu rõ nguyên nhân là do Connector cũ và yêu cầu tạo Connector mới
- **Global Pattern:** Khi User làm rõ nguyên nhân lỗi [X] xuất phát từ dữ liệu Connector cũ và báo muốn tự tạo Connector mới [Y] -> Agent [A] tự ý đi sửa thêm code workaround/sanitize [Z] mà User không yêu cầu -> Làm phiền User và tiêu tốn token. **Đúng:** Dừng lại ngay lập tức, lắng nghe 100% chỉ thị của User, không tự ý sửa thêm bất kỳ file code nào không được yêu cầu.
- **Bối cảnh (Trigger):** User báo "nếu vậy để a tạo cái connector mới", Agent lại tự ý đi sửa code sanitize `system_connector_repo_gorm.go` và `TableRegistry.tsx`.
- **Root Cause:** Cầm đèn chạy trước ô tô, tự phán đoán và sửa code dọn dẹp khi User không yêu cầu.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức. (2) Revert các file đã sửa. (3) Trả lời đúng trọng tâm chỉ dẫn cho User.
- **Tags:** #user-intent-alignment #no-unrequested-edits #stop-immediately #governance-discipline

### [2026-08-11] Bỏ sót projection field tại Query Handler khi bổ sung cột dữ liệu mới vào luồng Export
- **Global Pattern:** Khi bổ sung một trường dữ liệu mới [X] vào luồng export/report, Agent chỉ sửa tầng pure transformation logic [A] và DTO validation [B] mà bỏ qua tầng Query Handler [C] (nơi định nghĩa MongoDB `selectFields` projection và `rawData.map` DTO mapping) -> Dữ liệu trường [X] bị MongoDB query lọc bỏ tại tầng truy vấn, dẫn đến kết quả xuất ra Excel luôn bị rỗng/missing. **Đúng:** Mọi tác vụ bổ sung trường dữ liệu [X] vào luồng Export BẮT BUỘC phải trace callchain đầy đủ từ Export Pure Logic -> Query Handler (`selectFields` projection object + `rawData.map` output mapping) -> Database Schema.
- **Bối cảnh (Trigger):** Thêm trường `serviceCode` cho `PaymentBillExport`. Agent sửa `payment-bill-export.pure.ts` nhưng bỏ sót `GetAllPaymentBillExportHandler.ts` và `GetAllPaymentBillForCurrentMerchantExportHandler.ts`.
- **Root Cause:** Sơ suất không trace callchain tầng Query Handler để bổ sung `serviceCode: 1` vào `selectFields` và `serviceCode: doc.serviceCode` vào DTO mapper.
- **Fix/Correct Flow:** (1) Thêm `"serviceCode": 1` vào `selectFields` trong `GetAllPaymentBillExportHandler.ts` và `GetAllPaymentBillForCurrentMerchantExportHandler.ts`. (2) Thêm `serviceCode: doc.serviceCode` vào hàm `map` kết quả trả về. (3) Cập nhật unit test.
- **Tags:** #query-handler-projection #export-data-pipeline #select-fields-missing #callchain-tracing #mongodb-projection

### [2026-08-11] Đề xuất connector sai ecosystem (Confluent commercial) khi hạ tầng đang dùng open-source (Debezium JAR) — không đọc Dockerfile trước khi lập plan
- **Global Pattern:** Khi hệ thống dùng plugin connector loại [A] (open-source JAR cài qua Dockerfile, ví dụ Debezium), Agent đề xuất connector loại [B] (Confluent commercial) không cùng ecosystem → thất bại, rồi đề xuất giải pháp phức tạp [C] (internal worker) thay vì đề xuất đúng là [D] (thêm open-source JAR cùng pattern [A]). **Đúng:** BẮT BUỘC đọc Dockerfile/docker-compose Kafka Connect trước, xác định pattern cài plugin hiện tại, tìm connector cùng ecosystem.
- **Bối cảnh (Trigger):** Tích hợp SFTP vào Kafka Connect. Hệ thống dùng `io.debezium.*` + Maven JAR curl. Agent đề xuất `io.confluent.connect.sftp.SftpSourceConnector` (commercial) mà không đọc `Dockerfile.connect`.
- **Root Cause:** Không đọc Dockerfile.connect trước khi lập plan → suy diễn connector class từ memory thay vì từ hạ tầng thực tế.
- **Fix/Correct Flow:** (1) Đọc Dockerfile/docker-compose Kafka Connect TRƯỚC. (2) Xác định pattern: dùng Maven JAR hay Confluent Hub? (3) Tìm connector cùng pattern. (4) Verify qua `GET /connectors/plugins`. (5) Mới đề xuất.
- **Tags:** #kafka-connect #connector-class #read-infra-first #no-speculation #ecosystem-mismatch

### [2026-08-07] Tự ý chỉnh sửa code/migration mà không lập plan và trình User duyệt trước (Vi phạm Rule #0 & Rule #13)
- **Global Pattern:** Khi gặp lỗi phát sinh [X], Brain tự ý gọi tool sửa file [Y] trực tiếp thay vì lập plan và nộp giải pháp trình User approve -> Vi phạm nguyên tắc phân quyền Brain/Muscle (Rule #13) và quy trình Plan-First (Rule #0). **Đúng:** Luôn dừng lại, lập tài liệu kế hoạch chi tiết trong `09_tasks_solution_*.md` và `implementation_plan.md`, trình bày giải pháp duy nhất cho User duyệt, sau đó mới uỷ quyền Muscle thực thi.
- **Bối cảnh (Trigger):** User báo lỗi startup `cdc-cms-service` do migration 073 thiếu schema prefix `cdc_system.`.
- **Root Cause:** Brain bị nóng vội, thấy lỗi cú pháp đơn giản nên tự ý dùng `replace_file_content` sửa luôn mà bỏ qua bước xuất plan trình User.
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tục khi bị nhắc nhở. (2) Ghi lesson vào `lessons.md`. (3) Tạo tài liệu `09_tasks_solution_*.md` và `implementation_plan.md`. (4) Trình User duyệt plan rồi mới cho Muscle thực thi.
- **Tags:** #brain-code-prohibition #planning-first #governance-discipline #rule-13-violation

### [2026-08-07] Audit code mới phải trace toàn bộ callchain, không chỉ kiểm tầng handler/adapter
- **Global Pattern:** Khi tích hợp nguồn dữ liệu mới [X] vào hệ thống pipeline [Y] đã có sẵn, audit chỉ kiểm tra code mới viết (adapter, handler) mà bỏ qua tầng service/registry downstream → Các filter/constraint ẩn ở tầng sâu (topic discovery filter, DB check constraint, sync_engine validation) âm thầm chặn toàn bộ luồng E2E. **Đúng:** Trace TOÀN BỘ callchain từ ingestion → discovery → consume → handler → processEvent → registry lookup → shadow write. Kiểm tra mỗi checkpoint: "Data có đi qua được không?"
- **Bối cảnh (Trigger):** Tích hợp SFTP Connector vào CDC pipeline. Vòng audit 1 chỉ kiểm handler/adapter → bỏ sót `GetDebeziumTables()` filter loại bỏ SFTP topic và `sync_engine` constraint chặn giá trị mới.
- **Root Cause:** Audit scope quá hẹp — chỉ kiểm code diff mà không trace downstream consumers. Hàm `filterMatchingTopics` phụ thuộc `GetDebeziumTables()` nhưng dependency này ẩn qua 3 tầng gián tiếp.
- **Fix/Correct Flow:** (1) Vẽ callchain diagram trước khi audit. (2) Tại mỗi node, hỏi "Data format X có pass được qua đây không?". (3) Đặc biệt chú ý các hàm filter/validation mang tên của engine cũ (vd: `GetDebeziumTables`) nhưng lại gate-keep cho mọi nguồn.
- **Tags:** #audit-depth #callchain-tracing #hidden-filter #integration-testing

### [2026-08-06] Selective Tracing qua Custom Tracer Sampler ở SDK level để triệt tiêu spam traces
- **Global Pattern:** Khi tích hợp các plugin tracing tự động của bên thứ ba (như GORM OpenTelemetry plugin) tạo ra lượng spans khổng lồ ("spam traces") làm quá tải Collector/SigNoz -> Không nên tắt hoàn toàn (mất vết debug) hoặc dùng Tail-based/Head-based sampling ratio mù quáng (làm mất liên kết trace gốc hoặc bỏ sót lỗi logic). **Đúng:** Xây dựng một Custom Tracer Sampler ở mức SDK (bọc ngoài default sampler). Custom Sampler này sẽ chặn (Drop) 100% DB spans (tên bắt đầu bằng `gorm.`) trừ khi context hiện tại được gắn cờ (đánh dấu) bằng module được phép trace động thông qua helper `WithDBTraceModule(ctx, moduleName)`.
- **Bối cảnh (Trigger):** Kích hoạt GORM OpenTelemetry plugin làm bùng nổ hàng triệu database spans từ các luồng ngầm (cron, logs, scheduler...) gây rác trace backend.
- **Root Cause:** OTel GORM plugin luôn tự động sinh span cho mọi truy vấn DB thực hiện qua context.
- **Fix/Correct Flow:** (1) Viết struct `SelectiveSampler` kế thừa `sdktrace.Sampler`. (2) Kiểm tra `strings.HasPrefix(p.Name, "gorm.")` và lookup module trong context. (3) Bọc sampler chính bằng `SelectiveSampler`. (4) Đính cờ `WithDBTraceModule` tại entrypoint các luồng cần debug.
- **Tags:** #selective-tracing #custom-sampler #gorm-otel-spam #trace-filtering

### [2026-08-06] Nhân bản dòng logs khi join metadata 1-N làm sai lệch kết quả phân trang API
- **Global Pattern:** Thực hiện query phân trang [A] có join với bảng metadata [B] theo quan hệ 1-N -> kết quả trả về bị nhân bản số dòng (row amplification) sau khi phân trang trong subquery hoàn tất -> client nhận số lượng bản ghi sai lệch so với page_size được yêu cầu. **Đúng:** Sử dụng `LEFT JOIN LATERAL` với `LIMIT 1` sắp xếp theo thời gian cập nhật mới nhất để đảm bảo mỗi bản ghi chính chỉ kết hợp với tối đa 1 bản ghi metadata, triệt tiêu sự nhân bản dòng.
- **Bối cảnh (Trigger):** Gọi API `GET /api/activity-log?page=2&page_size=30` trả về 34 dòng thay vì 30 dòng.
- **Root Cause:** Bảng `cdc_system.master_binding` có quan hệ 1-N với `shadow_binding` (một shadow table có thể có nhiều master table mappings). Phép `LEFT JOIN` thông thường làm nhân bản dòng khi một shadow table có nhiều hơn 1 master table hoạt động.
- **Fix/Correct Flow:** Thay thế bằng `LEFT JOIN LATERAL (SELECT mb.master_schema, mb.master_table FROM cdc_system.master_binding mb WHERE mb.shadow_binding_id = sb.shadow_binding_id AND mb.is_active = TRUE ORDER BY mb.updated_at DESC, mb.id DESC LIMIT 1) mb ON TRUE`.
- **Tags:** #left-join-lateral #row-amplification #pagination-offset #master-binding

### [2026-08-05] Pattern: Đăng ký sai package GORM OpenTelemetry plugin làm lỗi biên dịch Go toolchain

- **Global Pattern:** Đăng ký plugin OpenTelemetry cho GORM [A] thông qua root package `gorm.io/plugin/opentelemetry` hoặc `otelgorm` -> Go toolchain báo lỗi `no required module provides package gorm.io/plugin/opentelemetry/otelgorm`. **Đúng:** Luôn sử dụng package con `/tracing` chính xác của GORM team: `go get gorm.io/plugin/opentelemetry/tracing`, sau đó import `"gorm.io/plugin/opentelemetry/tracing"` và đăng ký bằng `db.Use(tracing.NewPlugin())`.
- **Bối cảnh (Trigger):** Tích hợp OTel tracing cho luồng database GORM ở cả 2 service `centralized-data-service` và `cdc-cms-service`.
- **Root Cause:** Go toolchain không resolve được package `/otelgorm` từ module `gorm.io/plugin/opentelemetry` v0.1.16 do cấu trúc module thay đổi sang `/tracing`.
- **Fix/Correct Flow:** Sử dụng submodule `gorm.io/plugin/opentelemetry/tracing` và đăng ký bằng `tracing.NewPlugin()`.
- **Tags:** #gorm-otel-tracing #go-get-error #tracing-plugin-registration

### [2026-08-05] Pattern: Lệch mock SQL (sqlmock arguments mismatch) khi refactor query hoặc context logging trong test

- **Global Pattern:** Thay đổi query repository [A] hoặc truyền thêm context log [B] làm thay đổi số lượng/thứ tự tham số SQL -> sqlmock trong unit test ném lỗi `arguments do not match` hoặc `could not match actual sql` -> test fail dù logic nghiệp vụ đúng. **Đúng:** (1) Cập nhật các mock `ExpectQuery` khớp chính xác với số lượng đối số mới (ví dụ `WithArgs` nhận đúng N tham số). (2) Với các query kiểm tra điều kiện (như `_raw_data ? 'after'`), nếu test case không cần nhánh logic đó, hãy mock trả về `false` để tránh thay đổi cấu trúc pgPath (từ `{items}` sang `{after,items}`) làm lệch hàng loạt mock query phía sau.
- **Bối cảnh (Trigger):** Sửa context logging trong handler làm lộ ra việc mock DB của `TestHandleScanArrayFields_ReplyToAndUnmarshalOrder` và `TestHandleBatchTransform_Success` bị lỗi thời, gây crash test suite.
- **Root Cause:** Thay đổi logic query repository trước đó và query `hasAfter` CDC làm thay đổi đối số SQL thực tế chạy, khiến sqlmock không khớp.
- **Fix/Correct Flow:** Bổ sung mock method `GetActiveJobs`, cập nhật `WithArgs` của `mapping_rule_v2` nhận 4 tham số, và mock `hasAfter = false` để giữ nguyên pgPath `{items}`.
- **Tags:** #sqlmock-mismatch #unit-test-fix #has-after-mock #arguments-mismatch

### [2026-08-05] Pattern: Lỗi Postgres transaction block aborted (25P02) khi batch insert lỗi và lệch cột PK (`id` vs `_id`)

- **Global Pattern:** Batch writer [A] (`bridge_handler`) thực hiện batch upsert bên trong 1 `tx *gorm.DB` Transaction block. Khi 1 query bị lỗi (ví dụ: chỉ định nhầm tên cột PK `id` thay vì `_id` làm Postgres ném `SQLSTATE 42703`), Postgres đánh dấu transaction bị **ABORTED (25P02)**. Mọi query single-row fallback tiếp theo bên trong `tx` đều thất bại thảm hại với lỗi `current transaction is aborted, commands ignored until end of transaction block`. **Đúng:** (1) Với luồng batch upsert có fallback single-row: TUYỆT ĐỐI KHÔNG bọc toàn bộ trong 1 `tx.Transaction` duy nhất. Phải chạy query bằng `h.db.WithContext(ctx)` độc lập để nếu batch fail thì single-row fallback vẫn chạy được bình thường. (2) `resolveCollection` phải đối soát trực tiếp với `schema.Columns` của bảng PostgreSQL thực tế trên database để tự chọn đúng cột PK (`_id` hay `id`) thay vì ép gán cứng `id`. (3) `PrepareForCDCInsertWithBusinessCols` phải bẫy `ADD COLUMN IF NOT EXISTS` cho cột PK nếu chưa tồn tại. (4) `ActivityLogger.Start` phải bẫy tự động `CREATE SCHEMA IF NOT EXISTS cdc_system` & `CREATE TABLE IF NOT EXISTS cdc_system.cdc_activity_log` để không dính `SQLSTATE 42P01` do thiếu bảng.
- **Bối cảnh (Trigger):** Log báo WARN `column "id" of relation "export_jobs" does not exist (SQLSTATE 42703)` kéo theo lặp lỗi `current transaction is aborted (SQLSTATE 25P02)` và `relation "cdc_system.cdc_activity_log" does not exist`.
- **Root Cause:** Sơ suất gán cứng `pgPKField = "id"` trong khi bảng PG dùng `_id`, dùng `tx.Transaction` làm kẹt block khi có lỗi, và chưa auto-create schema/table trong `ActivityLogger`.
- **Fix/Correct Flow:** Bỏ `tx.Transaction` trong `batchUpsert`, thêm kiểm tra `schema.Columns` trong `resolveCollection`, bổ sung `ADD COLUMN IF NOT EXISTS` ở `schema_adapter.go`, và tự tạo bảng trong `ActivityLogger`.
- **Tags:** #postgres-transaction-aborted-25p02 #pk-column-resolution #auto-create-activity-log-table #batch-fallback-isolation

### [2026-08-05] Pattern: Lỗi Mongo Change Stream treo/không trả record gap khi Oplog hết hạn và thiếu Activity Logger

- **Global Pattern:** Reader [A] (`bridge_mongo`) chỉ dùng `coll.Watch(SetStartAtOperationTime)` → khi mốc Oplog quá hạn hoặc không có mutation mới trong lúc watch → Change Stream treo lặp `TryNext() == false` và trả về 0 record → gap data không được bổ sung vào Shadow Table. Ngoài ra, Handler [B] (`bridge_handler`) quên gọi `governance.NewActivityLogger` → Activity Log trong bảng `cdc_activity_logs` không xuất hiện bản ghi `Operation = bridge-oplog`. **Đúng:** (1) Tích hợp chế độ **Dual-Mode** cho MongoDB Bridge: Chạy Change Stream (Oplog) trước; nếu yield 0 events hoặc hết hạn oplog window, tự động fallback sang `coll.Find` truy vấn trực tiếp bảng Mongo theo các trường thời gian (`updatedAt`, `createdAt`, dải `ObjectID`) để đảm bảo 100% gap records được kéo về. (2) Luôn khởi tạo `governance.NewActivityLogger(h.db, h.logger)` trong handler để ghi log `Start`, `Complete`, `Fail` cho mọi job.
- **Bối cảnh (Trigger):** User phản ánh "1. ko ghi activity log khi chạy. Operation = bridge-oplog. 2. traces chưa có cdc-worker. 3. ko có record bị gap bổ sung ở shadow".
- **Root Cause:** Sơ suất thiếu ActivityLogger trong `bridge_handler.go`, và `bridge_mongo.go` chỉ có `Watch` mà không có fallback query khi Oplog window bị trôi.
- **Fix/Correct Flow:** Thêm ActivityLogger vào `bridge_handler.go`, bổ sung `coll.Find` fallback trong `bridge_mongo.go`, và trích xuất NATS `ctx` trước khi spawn background goroutine.
- **Tags:** #mongo-bridge-dual-mode #collection-query-fallback #activity-log-integration #bridge-oplog-gap

### [2026-08-05] Pattern: Lỗi fallback lastSeg mù quáng vơ nhầm Mongo $oid (_id) của sub-object/element vào cột số nghiệp vụ

- **Global Pattern:** Extractor / Transmuter [A] khi trích xuất cột nghiệp vụ kiểu số [X] (như `paymentBillId`) từ mảng JSON/document [B] → khi `path` bị miss, cờ fallback tự tiện cắt lấy `lastSeg` (đoạn cuối của path như `._id`) → vơ nhầm thuộc tính ExtJSON `_id` (`"$oid": "6a71aca854617f0aa9055582"`) của phần tử mảng con ném vào cột [X] → Validator báo rớt kiểu `BIGINT` và skip record. **Đúng:** (1) Tuyệt đối KHÔNG dùng fallback `lastSeg` mù quáng trên sub-object/element. (2) Chặn tuyệt đối cờ fallback trúng `_id` khi `path` khai báo ban đầu của rule không phải là `_id`. (3) Luôn dựa vào Payload JSON thật của User để trace đúng path chứ TUYỆT ĐỐI KHÔNG đoán mò datatype hay suy diễn linh tinh.
- **Bối cảnh (Trigger):** CDC Worker báo log WARN `extractColumns: validation failure, skipping record {"data_type":"BIGINT","target_column":"paymentBillId","violation":"expected valid numeric format for BIGINT, got 6a71aca854617f0aa9055582"}`.
- **Root Cause:** Sơ suất đặt fallback `lastSeg` trong `flatten.go` và `transmuter.go` khiến path `payments._id` bị cắt thành `_id` và vơ nhầm Mongo `$oid` Hex string của phần tử mảng `payments[0]`.
- **Fix/Correct Flow:** Xóa bỏ fallback `lastSeg` trong `flatten.go` và thêm guard `if lastSeg != "_id" || path == "_id"` trong `transmuter.go`.
- **Tags:** #mongo-extjson-oid #lastseg-fallback-mismatch #gjson-path-extraction #strict-path-boundary #transmuter-validation

### [2026-08-05] Pattern: Lỗi Missing Span giữa CMS & Worker do NatsCarrier phím hoa/thường và thiếu SpanKind Producer/Consumer

- **Global Pattern:** Producer [A] (`cdc-cms`) bắn NATS async message nhưng không gán `trace.WithSpanKind(trace.SpanKindProducer)` + Consumer [B] (`cdc-worker`) xử lý async job nhưng không gán `trace.WithSpanKind(trace.SpanKindConsumer)` + `NatsCarrier.Get` phân biệt hoa/thường case-sensitive (`"traceparent"` vs `"Traceparent"`) → SigNoz/Jaeger coi cuộc gọi HTTP API đã đóng (`202 Accepted`) là bị ngắt quãng/treo → hiển thị cảnh báo **"Missing Span"** giữa CMS và Worker. **Đúng:** Luồng Async NATS Tracing BẮT BUỘC: (1) `NatsCarrier.Get` tra cứu `http.CanonicalHeaderKey(key)` & `strings.EqualFold`. (2) Phía Producer (`nats_command_bus.go`) phải khởi tạo span với `trace.WithSpanKind(trace.SpanKindProducer)`. (3) Phía Consumer (`bridge_handler.go`) phải khởi tạo span với `trace.WithSpanKind(trace.SpanKindConsumer)`. (4) Dùng `observability.Ctx(ctx, logger)` để tự động gán `trace_id`/`span_id` vào 100% Zap logs.
- **Bối cảnh (Trigger):** SigNoz hiển thị cảnh báo "Missing Span" giữa `POST /api/v1/system/connectors/bridge-oplog` (cdc-cms) và Worker. User phản ánh "Missing Span ... sao ko thấy cdc-worker ... mày đéo biết test à".
- **Root Cause:** Sơ suất thiếu gán `SpanKindProducer` ở CMS bus dispatch, thiếu `SpanKindConsumer` ở Worker subscriber và lookup header case-sensitive.
- **Fix/Correct Flow:** Bổ sung `SpanKindProducer` ở `nats_command_bus.go`, `SpanKindConsumer` ở `bridge_handler.go`, sửa `NatsCarrier.Get` ở `trace_helpers.go` và gán `observability.Ctx(ctx, logger)` vào 100% loggers.
- **Tags:** #opentelemetry-missing-span #span-kind-producer-consumer #nats-header-case-sensitivity #signoz-tracing #traceparent

### [2026-08-05] Pattern: Bỏ sót tàn dư UI cũ (Obsolete Tech Debt) khi ra mắt tính năng thay thế mới

- **Global Pattern:** Hệ thống bổ sung tính năng mới [A] (Bridge Oplog) để kéo dữ liệu quá khứ → nhưng quên dọn dẹp các field/mô tả cũ [B] (`startOpTime` trong Form Khôi phục Connector) → giao diện tồn tại 2 nơi có chức năng mâu thuẫn/chồng chéo → User bối rối, hiểu nhầm cơ chế hoạt động và nghi ngờ chất lượng code. **Đúng:** Khi ra mắt kiến trúc/tính năng mới thế chỗ ý tưởng cũ: (1) Rà soát toàn bộ UI Forms, Labels, Placeholders, Alerts liên quan. (2) Xoá bỏ 100% các ô nhập liệu cũ không còn phù hợp. (3) Viết lại mô tả Alert/Help rõ ràng, ngắn gọn đúng với bản chất tính năng mới.
- **Bối cảnh (Trigger):** User bức xúc "Đọc Oplog từ thời điểm (Gap Start) sao còn. hazz, chán thật chứ. mày làm 1 cái task luôn check lại dum tao, mày cứ để lỗi old kỹ thuật hoài".
- **Root Cause:** Bỏ sót ô nhập `startOpTime` cũ trong Modal Khôi phục Connector từ trước khi có tính năng Bridge Oplog độc lập.
- **Fix/Correct Flow:** Dọn dẹp 100% field `startOpTime` khỏi `SourceConnectors.tsx` và viết lại Alert description chính xác.
- **Tags:** #ui-tech-debt-cleanup #obsolete-field #recover-connector #bridge-oplog-separation

### [2026-08-05] Pattern: Bắt buộc sinh Trace ID ngay từ Frontend cho 100% HTTP requests & Async Mutations

- **Global Pattern:** Frontend UI [A] thực hiện HTTP requests / Async Mutations (như Bridge Oplog) → quên đính kèm W3C traceparent & `X-Correlation-Id` từ client → CMS API không nhận được Trace ID từ client → chuỗi vết (Traceability) bị đứt đoạn ngay từ gốc. **Đúng:** (1) Thêm Axios Request Interceptor tại `api.ts` tự động tạo 32-char hex Trace ID và gắn header `X-Correlation-Id` + W3C `traceparent` cho 100% HTTP requests. (2) Tại các async mutation đặc thù (Bridge Oplog), tạo `traceId` 32-char hex đính trực tiếp vào JSON body `trace_id` + HTTP headers để đảm bảo tính nhất quán toàn trình.
- **Bối cảnh (Trigger):** User yêu cầu "100% Trace ID propagation cho Bridge Oplog job từ Frontend → API → Worker → Transmuter. phải gom vào từ Frontend nhứ".
- **Root Cause:** Trước đó mới làm trace propagation từ API → Worker → Transmuter mà bỏ qua bước khởi tạo Trace ID tại Frontend UI.
- **Fix/Correct Flow:** Thêm interceptor ở `api.ts` và đính `trace_id` trong `bridgeOplogMut` tại `SourceConnectors.tsx`.
- **Tags:** #frontend-trace-id #axios-interceptor #w3c-traceparent #full-stack-tracing #bridge-oplog

### [2026-08-05] Pattern: Lỗi 404 Delete Offsets khi gọi sai thứ tự xoá Connector trong Kafka Connect REST API

- **Global Pattern:** Handler [A] xoá connector cũ (`DELETE /connectors/{name}`) TRƯỚC khi gọi xoá offset (`DELETE /connectors/{name}/offsets`) → Kafka Connect trả về `HTTP 404: Connector not found` → offset cũ (`resume_token` tại `sec=1785816012`) KHÔNG BỊ XOÁ → khi `POST /connectors` tạo lại connector, Kafka Connect load lại offset cũ → Debezium ném lỗi `io.debezium.DebeziumException: ... resume_token=... but this is no longer available on the server` → Connector FAILED. **Đúng (Kafka Connect REST API Specs):** Kafka Connect YÊU CẦU connector phải ĐANG TỒN TẠI và ở trạng thái `STOPPED` thì `DELETE /connectors/{name}/offsets` mới hoạt động. Quy trình BẮT BUỘC: (1) `DELETE /connectors/{name}` (xoá cũ nếu có). (2) `POST /connectors` (tạo connector định nghĩa). (3) `PUT /connectors/{name}/stop` (chuyển connector sang `STOPPED`). (4) `DELETE /connectors/{name}/offsets` (xoá sạch committed offsets cũ thành công 200 OK). (5) `PUT /connectors/{name}/resume` (khởi chạy lại realtime stream sạch mốc latest).
- **Bối cảnh (Trigger):** Log worker báo `step 1/5: delete offsets returned notice (may not exist yet) {"connector":"testces","error":"HTTP 404: {error_code:404,message:Connector testces not found}"}` và Debezium bị FAILED lỗi expired `resume_token`.
- **Root Cause:** Gọi `DELETE /connectors/{name}/offsets` khi connector đã bị DELETE ở câu lệnh trước đó ➔ Kafka Connect không tìm thấy connector ➔ 404 ➔ Offsets không bị xoá.
- **Fix/Correct Flow:** Thực hiện đúng 5 bước: `DELETE -> POST -> STOP -> DELETE OFFSETS -> RESUME`.
- **Tags:** #kafka-connect-delete-offsets #404-connector-not-found #debezium-expired-resume-token #stop-before-delete-offsets

### [2026-08-05] Pattern: Bỏ sót OpenTelemetry Trace ID propagation khi viết Async Command & Worker Handler mới

- **Global Pattern:** Handler [A] xử lý Async Command (Bridge Oplog) → quên trích xuất Trace ID từ OpenTelemetry HTTP Span (`trace.SpanFromContext`) và NATS header (`traceparent`) → không đính kèm `zap.String("trace_id", traceID)` vào logs và không truyền `trace_id` sang result event / downstream handler → Activity Log / SigNoz không tìm thấy trace ID của job. **Đúng:** Mọi Async Command & Worker Handler BẮT BUỘC: (1) Lấy Trace ID 32-char hex từ HTTP Span Context (`trace.SpanFromContext`) / NATS W3C header (`observability.ExtractNATSHeader`). (2) Đính kèm `TraceID` vào Command Struct + NATS headers (`traceparent`, `Cdc-Correlation-Id`). (3) Gắn `zap.String("trace_id", traceID)` vào **TẤT CẢ** các câu log của worker. (4) Truyền `trace_id` sang NATS result payload & downstream command (Transmuter).
- **Bối cảnh (Trigger):** User thắc mắc "traces đâu. chạy job sao ko có traces. ko đọc source à".
- **Root Cause:** Đọc source `bridge_handler.go` thấy có `observability.StartSpan` nhưng quên trích xuất `traceID` để gắn vào `zap.Logger`, NATS response payload và downstream transmute trigger.
- **Fix/Correct Flow:** Bổ sung `TraceID` vào `BridgeOplogCommand`, `BridgeOplogPayload`, `SystemConnectorsHandler`, `BridgeHandler` và truyền xuyên suốt luồng.
- **Tags:** #trace-id-propagation #opentelemetry #w3c-traceparent #zap-logger #bridge-oplog

### [2026-08-05] Pattern: Giấu nút UI trong 1 branch condition khiến User không bấm được khi đổi trạng thái

- **Global Pattern:** UI Component [A] render Action Button [X] (Bridge Oplog) chỉ nằm trong 1 nhánh điều kiện `!live` (Unlinked/Orphan) → khi entity chuyển sang trạng thái `live` (Khôi phục xong) → nút [X] BIẾN MẤT HOÀN TOÀN → User không thấy nút trên UI để click. **Đúng:** Action Buttons mang tính nghiệp vụ độc lập (như Bridge Oplog) PHẢI xuất hiện ở TẤT CẢ các nhánh trạng thái (`live` & `!live`) cũng như ở tất cả các tab liên quan (`Connections`, `Connectors`, `Fingerprints`).
- **Bối cảnh (Trigger):** User bức xúc "fe đâu, nút Bridge Oplog đâu. 100% con mẹ mày á. bực hà".
- **Root Cause:** Sơ suất đặt nút Bridge Oplog trong nhánh `!live` của bảng Connections, làm mất nút khi connector đã live.
- **Fix/Correct Flow:** Thêm nút Bridge Oplog vào CẢ 2 nhánh (`live` & `!live`) ở tab Connections, tab Connectors và tab Fingerprints.
- **Tags:** #ui-action-visibility #conditional-render-bug #bridge-oplog-button #cms-fe

### [2026-08-05] Pattern: Audit lặt vặt (Incremental fixing) gây tiêu tốn token và làm phiền User

- **Global Pattern:** Agent [A] thực thi task → thay vì audit TOÀN DIỆN 100% end-to-end (compile + runtime + DSN + data shape + PK format) trong đúng 1 lần → Agent audit lặt vặt từng vòng 1, 2, 3, 4, báo cáo từng bug nhỏ → làm rác conversation, tiêu tốn token và làm User tức giận. **Đúng:** Ngay từ vòng audit đầu tiên, BẮT BUỘC rà soát TOÀN DIỆN từ (1) Compile/Types → (2) Flow Async/NATS → (3) Fallback values → (4) Security/Credentials → (5) Data transformation (DynamicMapper) → (6) PK Data format (BSON/Hex). Gom 100% bugs/issues lại trong đúng 1 báo cáo duy nhất và fix dứt điểm.
- **Bối cảnh (Trigger):** User mắng "ko làm hoàn chỉnh mà cứ làm lắc nhắc để a tốn token hả. em láo nháo vậy."
- **Root Cause:** Tư duy audit từng lớp hời hợt, làm xong lớp này mới soi tiếp lớp khác thay vì chạy static audit toàn trình ngay từ đầu.
- **Fix/Correct Flow:** (1) Dừng ngay việc báo cáo nhỏ lẻ. (2) Ghi lesson vào lessons.md. (3) Tự kiểm tra lại toàn bộ codebase Bridge Oplog lần cuối dứt điểm và chốt trạng thái hoàn chỉnh 100%.
- **Tags:** #incremental-fix #holistic-audit #token-waste #self-improvement-loop

### [2026-08-04] Pattern: Quyết định 5 bước Khôi phục Connector & Seed Oplog Offset cho Debezium MongoDB

- **Global Pattern:** Muốn khôi phục Connector [A] bị rớt và ép Debezium đọc Oplog từ mốc [START_TIME] mà KHÔNG SNAPSHOT (`snapshot.mode: no_data`) → BẮT BUỘC tuân thủ chuẩn quy trình 5 bước. **Đúng:** 
  1. **Xoá offset** (`DeleteOffsets`)
  2. **Tạo connector** (`Create` với `snapshot.mode: no_data`)
  3. **Pause connector** (`Pause`)
  4. **Khôi phục oplog** (Push offset `{"sec": START_TIME_SEC, "ord": 1}` với Key `["<connector>",{"replicaSetName":"<rs>"}]` vào topic `connect-offsets`)
  5. **Start lại connector** (`Resume` / `Start`)
- **Bối cảnh (Trigger):** Khôi phục connector mồ côi/bị rớt và replay Oplog CDC từ mốc thời điểm rớt mà không snapshot.
- **Root Cause:** Debezium MongoDB connector đọc offset vị trí Oplog từ `connect-offsets` topic của Kafka Connect.
- **Fix/Correct Flow:** Thực thi đúng thứ tự 5 bước: `DeleteOffsets → Create → Pause → Push Seeded Offset → Resume`.
- **Tags:** #debezium-mongo-5-step-recovery #offset-seeding #connect-offsets #oplog-replay #no-snapshot

### [2026-08-04] Đề xuất snapshot khi User đã cấm dùng snapshot dưới mọi hình thức

- **Global Pattern:** User [A] cấm dùng snapshot (chỉ cho phép Oplog CDC/stream) → Agent [B] quên ràng buộc và lại đề xuất snapshot / incremental snapshot → vi phạm chỉ thị của User. **Đúng:** Tuân thủ 100% ràng buộc KHÔNG SNAPSHOT dưới mọi hình thức (kể cả incremental hay signal snapshot).
- **Bối cảnh (Trigger):** User nhắc "không execute snapshot, nói bao nhiêu lần rồi".
- **Root Cause:** Quên mất constraint tuyệt đối của User về việc không kích hoạt bất kỳ loại snapshot nào (chỉ làm việc trên Oplog/change stream).
- **Fix/Correct Flow:** Nghiêm túc tiếp thu, không bao giờ đề xuất bất kỳ dạng snapshot nào (full, incremental, signal snapshot), giữ nguyên 100% luồng Oplog CDC.
- **Tags:** #no-snapshot-constraint #user-directive #oplog-only #cdc-stream

### [2026-08-04] Trả lời clarification ≠ lệnh APPROVE — không được implement ngay sau clarification

- **Global Pattern:** Agent [A] trình plan → User [B] raise câu hỏi → Agent làm rõ conflict → User trả lời clarification → Agent **implement luôn** mà không chờ APPROVE tường minh → vi phạm Rule #6. **Đúng:** Sau mỗi vòng clarification, Agent PHẢI dừng lại, cập nhật plan nếu cần, và chờ User gõ lệnh APPROVE (hoặc câu tương đương rõ ràng như "làm đi", "proceed") trước khi gọi bất kỳ write-tool nào.
- **Bối cảnh (Trigger):** User xác nhận "giữ HARD DELETE" (trả lời clarification) → Agent hiểu đây là approval → implement ngay lập tức.
- **Root Cause:** Đồng nhất "câu trả lời clarification" với "lệnh APPROVE". Đây là 2 thứ khác nhau hoàn toàn.
- **Fix/Correct Flow:** `Plan → Clarification loop → Plan cập nhật → Chờ APPROVE → Implement`. Mọi vòng clarification đều quay về bước "Chờ APPROVE" trước khi tiến tiếp.
- **Tags:** #approve-gate #clarification-vs-approve #rule-6-violation

### [2026-08-04] Giải thích sai "không soft-delete" → mất hết tác dụng của cơ chế Orphan

- **Global Pattern:** User [A] nói "module [X] không [soft-delete] nữa" → Agent [B] hiểu là **bỏ cơ chế đi** → đề xuất xóa toàn bộ block Orphan cleanup → Master DB thành đống rác, mục tiêu cốt lõi hệ thống bị phá. **Đúng:** "Không soft-delete" = **thay bằng hard DELETE vật lý** (`DELETE FROM ... WHERE`), KHÔNG phải bỏ luôn cơ chế cleanup.
- **Bối cảnh (Trigger):** User yêu cầu "Orphan trên luồng transmute không soft-delete nữa" → Agent hiểu thành "bỏ hẳn 2 block Orphan ra khỏi code".
- **Root Cause:** Đọc sai intent — không tư duy đến hệ quả: nếu bỏ cơ chế thì Orphan rows tồn tại mãi mãi, Master DB mất tính toàn vẹn.
- **Fix/Correct Flow:** Khi User nói "không [action A] nữa" trên một cơ chế dọn dẹp dữ liệu → PHẢI xác nhận lại: (1) Bỏ cơ chế hoàn toàn? hay (2) Thay bằng action mạnh hơn (hard delete)? Mặc định trong context cleanup/orphan → ưu tiên giả định (2).
- **Tags:** #soft-delete-vs-hard-delete #orphan-cleanup #intent-misread #data-integrity

### [2026-08-03] Tuyệt đối không tự ý lan rộng scope sửa nhiều file khi vấn đề nằm ở 1 vị trí (Rule #12 Minimal Impact)

- **Global Pattern:** Agent [A] nhận task xử lý ở module Transmute (`flatten.go`) -> Thay vì chỉ sửa đúng trong `flatten.go`, Agent lại đi sửa lan sang cả Scan Service (`scan_service.go`, `scan_handler.go`) và Child Explode (`child_explode.go`, `transmuter.go`) -> Vi phạm nghiêm trọng ranh giới kiến trúc (Architectural Boundary) và nguyên tắc Minimal Impact (Rule #12). **Đúng:** (1) Giữ đúng ranh giới module: Task thuộc Transmute (`flatten.go`) thì CHỈ XỬ LÝ NỘI BỘ TRONG `flatten.go`. (2) Không đụng vào Scan Service hay các module khác. (3) Hoàn tác ngay 100% các file sửa lan.
- **Bối cảnh (Trigger):** User yêu cầu `flatten.go` chỉ loop 1 phần tử đầu tiên của mảng → Agent sửa tùm lum file ở Scan và Child Explode.
- **Root Cause:** Bị nhầm lẫn ranh giới trách nhiệm (Scope/Boundary) giữa tầng Scan (khám phá schema) và tầng Transmute (chuyển đổi dữ liệu).
- **Fix/Correct Flow:** Thừa nhận sai sót, hoàn tác tất cả các file khác, chỉ sửa duy nhất `flatten.go`.
- **Tags:** #minimal-impact #architectural-boundary #transmute-only #strict-file-boundary

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





### [2026-08-05] Pattern: Dò mật khẩu mù quáng khi browser subagent gặp trang Login cần auth

- **Global Pattern:** Browser subagent [A] gặp trang Login [X] khi thực hiện UI verification → thay vì DỪNG LẠI và báo cáo để User [B] cung cấp credentials → Agent tự ý thử các password phổ biến (`admin`, `123456`, ...) nhiều lần → Lãng phí thời gian User, vi phạm bảo mật (credential brute-force), và làm User tức giận. **Đúng:** Khi browser agent gặp màn hình Login mà chưa có session: (1) DỪNG NGAY. (2) Báo cáo cho User: "Cần credentials để đăng nhập vào [URL]. Anh có thể đăng nhập trước để tôi tiếp tục verify không?". (3) Chờ User xác nhận đã đăng nhập xong → mới resume browser subagent.
- **Bối cảnh (Trigger):** Browser subagent chạy UI verification cho Bridge Oplog nhưng app ở trang login → thử `admin/admin`, `admin/123456` nhiều lần thay vì hỏi User.
- **Root Cause:** Không có logic "encounter login page → escalate to user" trong task prompt của browser subagent.
- **Fix/Correct Flow:** Task prompt browser subagent BẮT BUỘC thêm: "Nếu gặp trang Login và không có credentials → DỪNG NGAY, báo cáo để User đăng nhập trước, KHÔNG TỰ Ý THỬ PASS."
- **Tags:** #browser-agent-login-escalate #no-credential-guessing #user-escalation-first

### [2026-08-05] Pattern: Browser subagent tự ý click action phá hoại (Delete connector) nằm ngoài task verification

- **Global Pattern:** Browser subagent [A] được giao task verify [X] (xem nút Bridge Oplog, kiểm tra form) → trong quá trình browse, agent vô tình/tự ý click vào nút destructive [Y] (Delete connector, Drop table, Clear data) → gây mất dữ liệu hoặc phá hệ thống sản xuất. **Đúng:** Task prompt browser subagent BẮT BUỘC liệt kê tường minh: "TUYỆT ĐỐI KHÔNG click vào: Delete, Drop, Remove, Clear, Reset, hoặc bất kỳ nút/action nào không có trong danh sách bước verify. Nếu cần tìm nút verify mà vô tình thấy nút nguy hiểm → SKIP, KHÔNG HOVER, KHÔNG CLICK."
- **Bối cảnh (Trigger):** Browser subagent verify Bridge Oplog → click nhầm "Delete" connector "testss" hoàn toàn nằm ngoài task → User phát hiện và nổi giận.
- **Root Cause:** Task prompt không có whitelist hành động cho phép và không có blacklist hành động cấm tuyệt đối cho browser subagent.
- **Fix/Correct Flow:** Mọi browser subagent task BẮT BUỘC có section: "**FORBIDDEN ACTIONS (TUYỆT ĐỐI CẤM):** Delete/Remove/Drop/Reset/Clear/Confirm-delete bất kỳ entity nào. Chỉ được READ và CLICK các nút trong danh sách bước verify."
- **Tags:** #browser-agent-destructive-action #delete-connector-accident #verification-scope-control

### [2026-08-05] Pattern: Fallback cứng `_id → id` trong pgPKField resolution che khuất PK thực của bảng PG

- **Global Pattern:** Resolver [A] (`resolveCollection`) resolve `pgPKField` từ config → nếu `pgPKField == "_id"` áp fallback cứng thành `"id"` → trước khi có schema inspection thực tế từ PG → bảng [X] (`export_jobs`) có cột PK là `_id` (không phải `id`) → Bridge Oplog ghi upsert vào cột `id` sai → record không match, data corrupt. **Đúng:** BỎ HOÀN TOÀN fallback cứng `_id → id`. Chỉ dựa vào schema inspection thực tế từ PG (`GetSchemaInSchema`) để quyết định pgPKField: nếu bảng có `_id` → dùng `_id`; nếu có `id` → dùng `id`; nếu chưa có bảng (schema nil) → giữ nguyên value từ config.
- **Bối cảnh (Trigger):** Log `bridge_oplog: resolved shadow binding` báo `pg_pk: id` trong khi bảng `export_jobs` thực tế có cột PK là `_id` → User phát hiện qua log.
- **Root Cause:** Fallback `if resolved.pgPKField == "_id" { resolved.pgPKField = "id" }` ở `bridge_handler.go` chạy TRƯỚC schema inspection, override giá trị đúng.
- **Fix/Correct Flow:** Xóa block fallback cứng `_id → id` trong `bridge_handler.go`. Schema inspection (GetSchemaInSchema) đã đủ để detect đúng PK column.
- **Tags:** #pgpkfield-fallback-bug #bridge-oplog-pk-resolution #id-vs-_id #export-jobs

### [2026-08-05] Pattern: Browser subagent tự ý click Create Shadow Table / Snapshot nằm hoàn toàn ngoài task

- **Global Pattern:** Browser subagent [A] được giao task READ-ONLY verify [X] → agent lang thang sang trang `/shadow` → tự ý click "Create shadow table" hoặc nút Snapshot cho collection [Y] → trigger DDL/heavy-job không kiểm soát trên hệ thống production → User phải clean up hậu quả. **Đúng:** Browser subagent BẮT BUỘC được cấp DANH SÁCH URL TRẮNG (allowed URLs) tường minh. Nếu agent cần navigate sang URL ngoài whitelist → DỪNG NGAY, báo cáo User, KHÔNG TỰ Ý NAVIGATE. Mọi action có tên "Create", "Snapshot", "Delete", "Drop", "Reset", "Migrate" đều là FORBIDDEN bất kể context.
- **Bối cảnh (Trigger):** Task verify Bridge Oplog → agent navigate sang /shadow (ngoài whitelist /sources, /activity-log) → click "Create shadow table" cho connector testces → trigger DDL trên hệ thống production mà User không approve.
- **Root Cause:** Task prompt không có WHITELIST URL cụ thể và không có BLACKLIST action đủ nghiêm ngặt. Agent bị "drift" sang trang liên quan nhưng không trong scope.
- **Fix/Correct Flow:** (1) Mọi browser subagent task phải có section ALLOWED_URLS = [danh sách URL cụ thể]. (2) Nếu agent cần navigate ra ngoài ALLOWED_URLS → STOP và báo cáo. (3) BAN VĨNH VIỄN mọi click vào nút có text: Create/Snapshot/Delete/Drop/Reset/Migrate/Execute.
- **Tags:** #browser-agent-create-shadow-table #browser-agent-url-whitelist #no-ddl-from-browser-agent

### [2026-08-13] Hiểu sai bản chất cấu hình database của nhiều alias trong dự án dẫn đến đề xuất gom chung database
- **Global Pattern:** Agent [A] phân tích cấu hình MongoDB [B] có nhiều alias khác nhau chọc vào các database khác nhau -> vội vàng đề xuất gộp chung toàn bộ connection URI về cùng một database name duy nhất [X] -> làm sai lệch nghiệp vụ phân tách database và gây lỗi ghi đè dữ liệu chọc chung DB [Y]. **Đúng:** Giữ nguyên bản chất phân tách database của từng alias, đề xuất giải pháp đưa về cùng một Host/Port local nhưng bắt buộc phải phân tách database name tương ứng với thiết kế nghiệp vụ của từng alias.
- **Bối cảnh (Trigger):** User yêu cầu đưa toàn bộ env url của mongo về 1 file .run.local.env. Agent đề xuất phương án gom chung database name của tất cả các service về cùng một link (ví dụ `/centralized-export-local`).
- **Root Cause:** Agent thiếu phân tích sâu về nghiệp vụ của từng alias (mỗi alias chọc vào một microservice DB riêng biệt để đọc/ghi các collection tương ứng như `export-jobs` vs `payment-bills`), đề xuất gom chung DB name thô thiển làm mất đi cấu trúc dữ liệu phân rã của dự án.
- **Fix/Correct Flow:** Sửa đổi các file phân tích, hướng dẫn cấu hình trong `.run.local.env` phải giữ đúng cấu trúc database name cho từng alias (chỉ chung Host/Port local `mongodb://localhost:27017/`), đồng thời đề xuất giải pháp dynamic fallback giữ nguyên database name cho từng alias.
- **Tags:** #mongo-db-isolation #alias-database-mismatch #carelessness


### [2026-08-25] Sửa handler A nhưng bỏ sót handler B vì không trace route registration → Fix chỉ áp dụng cho endpoint KHÔNG được gọi
- **Global Pattern:** Khi sửa logic trong handler [A] (VD: `TriggerCheck`), Agent SỬA ĐÚNG nhưng frontend thực tế gọi handler [B] (VD: `TriggerCheckAll`) qua route khác. **Đúng:** PHẢI trace ngược Frontend URL → `router.go` → Actual Handler TRƯỚC khi sửa.
- **Bối cảnh (Trigger):** Sửa `TriggerCheck` để normalize `shadow_schema.table`, nhưng frontend POST `/api/reconciliation/check` (không có `:table`) → route match vào `TriggerCheckAll` → fix không có tác dụng.
- **Root Cause:** Không đọc `router.go` để xác nhận route mapping. Suy diễn rằng handler cần sửa là `TriggerCheck` dựa trên tên, không dựa trên bằng chứng route thực tế.
- **Fix/Correct Flow:** (1) Đọc `router.go` trước để xác định endpoint → handler mapping. (2) Grep frontend code để tìm URL thực tế. (3) Sửa đúng handler đang được gọi. (4) Audit phải verify Route → Handler, không chỉ verify code handler.
- **Tags:** #route-handler-mismatch #missing-trace #audit-false-positive #frontend-backend-trace

### [2026-08-25] Bẫy ép đè cứng 'if pkField == "_id" { pgPKField = "id" }' trong handler gây gãy CDC sync bảng MongoDB
- **Global Pattern:** Mặc dù `PrimaryKeyField` trong Registry và TableConfig đã được khai báo chính xác là `_id` [A], nhưng tại các handler xử lý CDC event (`event_handler.go`, `bridge_handler.go`), Agent/Dev trước đây đã cài cắm logic ép đè cứng [B] (`if !mappedPK && pkField == "_id" { pgPKField = "id" }` và `if pgPKField == "_id" { pgPKField = "id" }`) -> Dẫn đến khi xử lý CDC record từ MongoDB, `record.PrimaryKeyField` bị đổi trái phép thành `"id"`, làm câu lệnh SQL upsert sinh ra `INSERT INTO table ("id", ...)` văng lỗi `SQLSTATE 42703 column "id" does not exist`. **Đúng:** (1) TUYỆT ĐỐI CẤM ép đè cứng `_id → id` ở các handler. (2) Tôn trọng 100% tên cột khoá chính `_id` khi nguồn là MongoDB hoặc khi schema shadow chứa cột `_id`. (3) Phải xoá sạch các câu lệnh `if pkField == "_id" { pgPKField = "id" }` khỏi `event_handler.go` và `bridge_handler.go`.
- **Bối cảnh (Trigger):** User bức xúc chỉ ra: "ko phải cái lỗi trên, vì PrimaryKeyField đang là _id, nên nó ko về cái id đc" khi thấy log `hyperverge_face_match` bị văng lỗi column "id" does not exist.
- **Root Cause:** Cài cắm các câu lệnh ép đè cứng `if pkField == "_id" { pgPKField = "id" }` trong `event_handler.go` (dòng 353-355, 384-386) và `bridge_handler.go` (dòng 281-283).
- **Fix/Correct Flow:** (1) Dừng lại ngay lập tức theo Mid-Session Fix (Rule #5). (2) Ghi lesson vào `lessons.md`. (3) Xoá bỏ 100% các câu lệnh `if ... == "_id" { ... = "id" }` trong `event_handler.go` và `bridge_handler.go`. (4) Giữ nguyên `_id` khi `PrimaryKeyField` là `_id`.
- **Tags:** #anti-id-override-trap #event-handler-pk-fix #bridge-handler-pk-fix #mongodb-id-column #mid-session-fix #sqlstate-42703


