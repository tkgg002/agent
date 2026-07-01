# 📚 Hard-Tech Patterns & Tripwires (Garbage Collected)

> **BẢN CHẤT**: File chứa các bẫy kỹ thuật đặc thù (Postgres, CDC, Kafka, Golang, MongoDB). Các quy trình hành vi đã được GC nén vào `tech_stack.md`. BẮT BUỘC ĐỌC TRƯỚC KHI CODE.

## 1. Golang & GORM Quirks
- **GORM Reflection Panic:** `tx.Raw(sql).Scan(&interface{})` sẽ panic (*call of reflect.Value.Type on zero Value*) nếu `interface{}` chưa cấp phát cụ thể. **Đúng:** Dùng `rows, _ := tx.Raw(sql).Rows()` -> `rows.Next()` -> `rows.Scan(&rawVal)`.
- **GORM Array String:** Lệnh `_id = ANY(?)` với tham số `[]string` sinh lỗi `22P02`. **Đúng:** Bắt buộc dùng toán tử `IN (?)` để GORM tự expand mảng.
- **GORM Raw().Scan Nested Struct:** `Raw().Scan` không hỗ trợ map tự động vào nested struct. **Đúng:** Scan vào flat struct có tags `gorm:"column:..."`, sau đó transpose thủ công sang nested.
- **Context Inheritance Timeout:** Dùng `context.Background()` tạo detached context ở hàm con làm đứt OTel TraceID, gây goroutine leak và DB deadline exceeded. **Đúng:** Kế thừa từ parent `context.WithTimeout(ctx, duration)`.
- **Mock Time Field:** Mock DB SQL (sqlmock) bị thiếu cột `checked_at` làm struct gán zero-time (0001-01-01) -> `time.Since()` tính ra tuổi >50 năm -> Panic runtime do flow rẽ nhánh ảo. **Đúng:** Luôn mock cột thời gian bằng `time.Now()`.
- **Map Serialization Order:** Map serialize theo Alphabetical, Struct theo thứ tự khai báo. Cẩn thận khi refactor map -> struct nếu hệ thống yêu cầu byte-identical checksum.

## 2. PostgreSQL & Schema Design
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
- **Nhầm lẫn thư mục dự án cdc-system:** Không verify kỹ đường dẫn trong workspace mà lấy nhầm đường dẫn backup ngoài workspace, dẫn đến đọc sai code cũ/không khớp. Luôn verify đường dẫn tuyệt đối thuộc `/Users/trainguyen/Documents/work/` trước khi thao tác. Tag: #path-management #workspace-accuracy #carelessness
- **Vi phạm ngôn ngữ tiếng Việt và thiếu file 12_implementation_plan_*.md:** Tạo các file workspace bằng tiếng Anh thay vì tiếng Việt và không tạo file kế hoạch triển khai của AI `12_implementation_plan_*.md` để đồng bộ phiên. **Đúng:** Viết tài liệu workspace bằng tiếng Việt (hoặc song ngữ), và bắt buộc tạo `12_implementation_plan_*.md` trước khi kết thúc phiên. Tag: #vietnamese-only #session-sync #implementation-plan #carelessness
- **Quên đồng bộ walkthrough.md và implementation_plan.md vào workspace:** Tạo file artifact nhưng quên sao chép/đồng bộ vào workspace folder. **Đúng:** Luôn đồng bộ ngay lập tức cả file `walkthrough.md` và `implementation_plan.md` dạng artifact vào workspace folder để lưu giữ context lâu dài. Tag: #sync-failure #workspace-memory #carelessness