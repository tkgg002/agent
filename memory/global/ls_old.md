# agent/memory/global/lessons.md



## [2026-06-02] Tận dụng vòng đời cấu hình hiện có để giải phóng cache (implicit cache invalidation) thay vì over-engineer và coupling dư thừa

- **Trigger**: Khi đề xuất inject `MaskingService` trực tiếp vào `SnapshotRunner` để chủ động invalidate cache trước khi snapshot, người dùng đã nhận xét cảnh báo "sao lại là cache, nó có thật sự cần thiết không" và khiển trách model tự ý code trước khi được phê duyệt kế hoạch.
- **Root Cause**: Model thiết kế dư thừa (over-engineering) và tăng coupling không cần thiết giữa `SnapshotRunner` và `MaskingService`. Do `SnapshotRunner` luôn gọi `ReloadAll` của registry trước khi chạy, việc dọn dẹp cache của `MaskingService` nên được thực hiện tự động bên trong `ReloadAll` sau khi cập nhật metadata. Việc bắt `SnapshotRunner` trực tiếp quản lý cache của dịch vụ bảo mật là vi phạm nguyên tắc phân tách trách nhiệm.
- **Fix**: Khôi phục lại `SnapshotRunner` và `worker_server.go` (bỏ constructor mới). Tích hợp toàn bộ logic invalidate cache của `MaskingService` vào hàm `ReloadAll` của `MetadataRegistryService`. Chỉ cập nhật kế hoạch và xin ý kiến người dùng trước khi tiếp tục thực thi.
- **Lesson (Global Pattern)**: **Tránh tạo liên kết trực tiếp (coupling) giữa các Runner/Processor xử lý nghiệp vụ với các cơ chế quản lý cache của service tiện ích (như Masking, Auth). Hãy để tầng quản lý cấu hình tập trung (Metadata Registry/Config Loader) xử lý việc invalidate cache một cách tự nhiên (implicit invalidation) trong vòng đời cập nhật (reload lifecycle).** Đồng thời, **luôn tuân thủ quy trình Gatekeeper: cập nhật kế hoạch trước, nhận feedback và phê duyệt rõ ràng từ user mới được phép chạm tay vào code.**
- **Tags**: #coupling #implicit-invalidation #cache-management #over-engineering #process-compliance #simplicity-first

---

## [2026-06-04] Hiểu đúng tầng sở hữu feature trước khi đặt/bỏ tính năng (Feature Layer Ownership)

- **Trigger**: Phiên trước bỏ Scan Array (Flatten) khỏi Master và giữ ở Shadow — ngược với yêu cầu. User phải sửa lại: "shadow là raw data sink, không cần flatten; master cần flatten để normalize JSON arrays thành cột schema".
- **Root Cause**: Model hiểu nhầm hướng bỏ (bỏ ở đâu, giữ ở đâu). Khi User nói "bỏ Flatten ở master" + "không bỏ ở shadow", model interpret đây là trạng thái ĐÃ ĐÚNG thay vì là VẤN ĐỀ cần sửa. Thiếu bước phân tích "tầng nào cần gì" trước khi thực thi.
- **Fix**: (1) Restore Flatten ở MasterMappingFieldsPage (master CẦN normalize); (2) Bỏ Flatten ở MappingFieldsPage (shadow = raw sink, không flatten); (3) Sửa alert text phản ánh đúng logic.
- **Lesson (Global Pattern — Feature Layer Ownership)**: **Trước khi thêm/xoá bất kỳ feature X khỏi layer A hay B, hãy tự hỏi: "Layer nào trong pipeline là đơn vị chịu trách nhiệm XỬ LÝ X?" (tầng raw/sink vs tầng schema-enforced/normalized). Ví dụ tổng quát: Layer B (processed/schema) thường cần transform (flatten, normalize, aggregate). Layer A (raw/sink) chỉ lưu nguyên gốc — không biến đổi. BỎ feature khỏi đúng layer là BỎ khỏi layer KHÔNG CẦN, KHÔNG PHẢI layer đang dùng nó.**
- **Tags**: #feature-placement #layer-ownership #shadow-vs-master #flatten #design-decision

---

## [2026-06-02] Thiết kế giám sát và log chẩn đoán ở mức hệ thống thay vì tập trung vào đối tượng lỗi đơn lẻ

- **Trigger**: Khi đề xuất thêm log `zap.Debug` để gỡ lỗi stale cache masking riêng cho shadow binding 66, thiết kế ban đầu bị người dùng phản hồi rằng đây là lỗi thiết kế hệ thống (system design), không nên chỉ tập trung xử lý hay log riêng cho binding 66.
- **Root Cause**: Model bị cuốn vào chi tiết của lỗi hiện tại (binding 66) và thiết kế log/telemetry mang tính đối phó cục bộ, thiếu tính tổng quát cho toàn bộ thực thể động trong hệ thống.
- **Fix**: Thiết kế log `zap.Debug` tổng quát trong `resolveMaskMap` cho mọi `bindingID` động (bao gồm cache hit/miss, nạp rules từ DB, và mask map được sinh ra), giúp tăng khả năng quan sát (observability) toàn diện cho tất cả các bindings chứ không chỉ riêng binding 66.
- **Lesson (Global Pattern)**: **Khi bổ sung cơ chế log/telemetry để gỡ lỗi, hãy luôn xây dựng giải pháp giám sát ở cấp độ hệ thống tổng quát (system-wide/generic telemetry) thay vì viết log hardcode hoặc thiết kế chỉ nhắm vào đối tượng lỗi hiện tại.** Việc tổng quát hóa log chẩn đoán cho mọi định danh thực thể (như `bindingID`) vừa giúp hệ thống có khả năng tự chẩn đoán tốt hơn trong tương lai, vừa giữ mã nguồn sạch sẽ, tránh vi phạm các nguyên tắc thiết kế hệ thống sạch.
- **Tags**: #system-design #observability #logging-strategy #telemetry #generalization

---

## [2026-06-02] Mất cấu hình mặc định (default/fallback behavior) khi di chuyển định danh từ chuỗi sang số nguyên trong môi trường kiểm thử

- **Trigger**: Sau khi chuyển đổi kiểu dữ liệu của mapping và masking registry từ string-based `targetTable` sang `int64` `shadow_binding_id`, một loạt test case trong `DLQWorker` và `ReconHealer` bị lỗi masking verification do `MaskingService` không thể resolve binding ID từ chuỗi input ("customer_profiles", "wallet_accounts") và trả về map rỗng, dẫn đến việc bỏ qua mã hóa và kiểm thử fail.
- **Root Cause**: Database-backed lookup trong `resolveMaskMap` bị vô hiệu hóa vì `ms.db == nil` trong môi trường kiểm thử. Trước khi refactor, `resolveMaskMap` nhận string key và nạp fallback `defaultMasks` nếu DB nil. Sau khi chuyển sang `int64` bindingID, nó bị short-circuit và trả về `nil` ngay lập tức nếu bindingID <= 0 (như khi không parse được ID từ chuỗi trong test).
- **Fix**: Sửa đổi `resolveMaskMap(bindingID int64)` để khi `bindingID <= 0` (chế độ fallback/legacy/test), nó không trả về `nil` ngay mà vẫn khởi tạo map và nạp các default masks từ `defaultMasks` với HMAC strategy làm fallback mặc định, khôi phục lại tính bảo mật tối thiểu.
- **Lesson (Global Pattern)**: **Khi chuyển đổi định danh (identifier) của một dịch vụ dùng chung (như Masking, Authorization, Routing) từ kiểu chuỗi động sang kiểu ID số nguyên, hãy luôn đảm bảo cơ chế fallback cho legacy key hoặc test environment vẫn giữ được hành vi bảo mật/mặc định (default/fallback behavior).** Việc short-circuit trả về `nil` ngay khi ID không hợp lệ hoặc bằng 0 trong các hàm lookup config dùng chung sẽ vô hiệu hóa hoàn toàn chính sách bảo vệ mặc định (default values / fallback rules), dễ dẫn đến rò rỉ dữ liệu hoặc phá hỏng test suite trong môi trường mockup (DB == nil).
- **Tags**: #identifier-migration #fallback-behavior #testing-environment #security-by-default #masking #refactor #short-circuit

---

## [2026-05-21] Field-routing mismatch trong dual-stack FE-BE (legacy bridge vs new endpoint) → silent-drop

- **Trigger**: User thêm column V2-only (`snapshot_batch_size`) vào table `source_object_registry`, expose qua endpoint mới `PATCH /api/v1/source-objects/:id`. Form Edit ở FE gửi 1 payload chung. Khi save trên row có legacy bridge (`record.registry_id != null`), `updateEntry` route TOÀN BỘ payload qua `/api/v1/source-objects/registry/:registry_id` → handler legacy body-parser KHÔNG list field mới → silent drop. Test path không-bridge thì pass; bug chỉ hiện trên rows bridge.
- **Root Cause**: Logic routing `usesLegacyBridge ? legacy : v2` là all-or-nothing trên CẢ payload, không phân biệt field nào thuộc table nào. Khi schema chia thành 2 table (legacy + V2), field mới thêm ở V2 không thể đi qua endpoint legacy được.
- **Fix**: Tách payload theo "field-ownership": V2-exclusive fields (sống trên V2 table) luôn PATCH V2 endpoint; rest fields theo routing cũ. 2 PATCH tuần tự, share try/catch.
- **Lesson (Global Pattern)**: **Khi backend có 2+ endpoint thay đổi cùng entity nhưng mỗi endpoint OWN tập field khác nhau (vd legacy table T_old vs new table T_new, hoặc primary DB vs cache vs search-index), FE routing PHẢI split payload theo field-ownership, không phải route toàn payload theo 1 cờ.** Khi thêm field F vào endpoint E_new, hỏi: (1) endpoint cũ E_old có biết F không → nếu không, F bị silent-drop; (2) FE caller có gửi F qua E_old nào không → nếu có, BẮT BUỘC split payload trước khi gửi. Pattern check trong code review: tìm caller dạng `if (legacy) PATCH X else PATCH Y` với payload chung — đó là cờ đỏ. Tương tự cho microservice mở rộng: thêm field vào event v2, consumer v1 silently drop nếu không có routing layer.
- **Tags**: #routing #dual-stack #legacy-bridge #silent-drop #field-ownership #patch-split #cdc

---

## [2026-05-21] Serialization-form drift: cast SQL chỉ cover 1 representation cho cùng 1 logical type

- **Trigger**: `cmd-batch-transform` fail SQLSTATE 22007 `invalid input syntax for type timestamp: "{"$date": "***T08:53:41.741Z"}"` trên rows cũ; cùng table cùng phút `transform` (per-row) success. Cùng logical type "BSON Date" nhưng tồn tại trong cột JSONB ở MIN. 3 form: raw number (epoch ms), Extended-JSON object `{"$date": "ISO"}`, Extended-JSON object `{"$date": {"$numberLong": "epoch"}}`. Cast helper `buildCastExpr` chỉ branch `number` vs `ELSE → ::TIMESTAMP` → form object literal-string-hóa qua `->>` → cast fail.
- **Root Cause**: Cast expr được viết khi chỉ thấy 1-2 form trong dev/test data. Khi pipeline có nhiều encoder/writer (Go driver default vs canonical Extended-JSON encoder vs Debezium), CÙNG logical type lưu xuống storage ở MULTI form. Helper không enumerate đủ form → backward-incompat trên rows cũ.
- **Fix**: Mở rộng CASE bao trùm tất cả form của cùng logical type. TIMESTAMP: `number → to_timestamp` | `object{$date:string} → ::TIMESTAMPTZ` | `object{$date:{$numberLong}} → to_timestamp` | ELSE `::TIMESTAMP`. BIGINT: `object{$numberLong:string} → ($numberLong)::BIGINT` | ELSE `::BIGINT`. SQL-side fix backward-compat cả rows cũ; tránh write-time migration.
- **Lesson (Global Pattern)**: **Khi cast/parse một logical type X từ semi-structured storage S, enumerate trước TẤT CẢ form serialization khả dĩ của X trong S (mỗi encoder/writer là 1 nguồn form mới).** Cast helper phải có CASE branch cho từng form, đặt branch đặc thù (object Extended-JSON) TRƯỚC branch fallback (text cast). Nếu chỉ thấy 1 form trong dev data, tìm chủ động: BSON Date / BSON Int64 / BSON Decimal128 / BSON ObjectID đều có dạng canonical Extended-JSON. Tương tự cho NDJSON từ multiple producer, Avro union, Protobuf any. Hỏi: "Encoder nào khác ở thượng nguồn có thể ghi form khác cho cùng cột này?" → liệt kê + add CASE.
- **Tags**: #cast #serialization #bson #extended-json #postgres #jsonb #convention-drift #backward-compat #single-helper

---

## [2026-05-21] DRY violation: resolver được duplicate ở caller dẫn đến phủ convention không đồng đều

- **Trigger**: `snapshot.v2 run failed` cho connection `goopay-pbs` với error `"no usable DSN in secret_ref nor host/port fields"` — nhưng cùng connection, cùng worker process, command `scan-fields` resolve thành công 6 phút trước với DSN `mongodb://***@host1:27017,host2:27017,host3:27017/?replicaSet=...&authSource=admin`.
- **Root Cause**: Logic resolve DSN tồn tại ở 2 nơi với độ phủ khác nhau:
  1. Caller `scanFieldsMongoSource` có logic inline detect `Host` chứa full URI (`if strings.HasPrefix(hostRaw, "mongodb://") → dsn = hostRaw`).
  2. Shared resolver `MetadataRegistryService.GetSourceDSN` chỉ check `SecretRef` qua `tryPlainDSN/tryEnvPointer`, layer `buildDSNFromFields` yêu cầu `Port != nil` — nhưng row mà cdc-cms UI ghi với `Host=full URI` thường để `Port=NULL` → resolver trả error. Caller khác (`snapshot.v2 runner`) gọi shared resolver → fail.
- **Fix**: Đối xứng hóa — thêm 2 layer `tryPlainDSN(*conn.Host)` + `tryEnvPointer(*conn.Host)` vào `GetSourceDSN` SAU override SAU TRƯỚC secret_ref layers. Đồng thời xoá block build-DSN inline ở `scanFieldsMongoSource`, gọi shared `GetSourceDSN` để single source of truth. Order layer mới: override → host-as-URI → host-as-env → secret-as-URI → secret-as-env → build-from-fields → AES.
- **Lesson (General Pattern)**: **Cross-cutting concern (DSN resolve, auth, masking, retry, telemetry...) PHẢI có single source of truth.** Khi caller C1 implement inline logic L để bypass limitation của shared resolver R, đó là cờ đỏ — caller C2 dùng R sẽ behave khác C1 cho cùng input, gây silent divergence + debug-khó-tả. Đúng: mở rộng R để cover convention mới, rồi xoá L ở C1. Đối xứng các layer (resolve secret_ref ra sao thì resolve host cũng phải y vậy nếu hai column cùng carry-DSN-shape).
- **Tags**: #dry #resolver #single-source-of-truth #cdc #dsn #convention-drift #refactor

---

## [2026-05-21] Tránh trùng lặp thuật ngữ trạng thái và Thiết kế Đồng bộ Kích hoạt (Cascade Activation) trên UI

- **Trigger**: Trang quản trị Shadow (`/shadow`) hiển thị nhiều cột cùng tên `"Trạng thái"` gây hiểu lầm cho người dùng. Thêm nữa, để thực hiện "Snapshot Now" thành công, người dùng bắt buộc phải mở Edit Modal chỉnh sửa thủ công để kích hoạt cả thực thể cha (Source Object) lẫn thực thể con (Shadow Binding), nếu không snapshot sẽ bị bỏ qua hoặc lỗi do Source Object vẫn ở trạng thái tắt.
- **Root Cause**: 
  1. Khi thiết kế giao diện (UI) quản lý pipeline, việc dịch cơ học các trường dữ liệu từ backend (như `is_active` và `sync_status`) thành một thuật ngữ chung `"Trạng thái"` làm mất ngữ cảnh vận hành của nút điều khiển.
  2. Việc phân rã logic cấu hình thành nhiều tầng độc lập (Source Object `is_active` vs Shadow Binding `is_active`) mang lại sự linh hoạt cho API nhưng lại là gánh nặng thao tác (UX friction) trên giao diện. Người dùng kỳ vọng việc "bật dòng chảy dữ liệu cho bảng" (Switch trên hàng) sẽ kích hoạt toàn bộ các thành phần phụ thuộc liên quan.
- **Fix**: 
  1. Đổi tên cột Switch trên row thành `"Kích hoạt Sync"`, và trong Edit Modal thành `"Kích hoạt Source Sync"` để phân rõ ngữ cảnh.
  2. Tự động hóa ở Frontend: Khi người dùng gạt Switch kích hoạt trên hàng (`is_active: true` ở cấp Shadow Binding), nếu thực thể cha (Source Object) đang tắt, frontend sẽ tự động gọi API kích hoạt cha song song để thông luồng CDC pipeline ngay lập tức, cho phép chạy Snapshot mà không cần thao tác thủ công.
- **Lesson (General Pattern)**: 
  * Trên các màn hình Dashboard quản trị dòng chảy dữ liệu, các nhãn cột trạng thái phải có tên chuyên biệt phản ánh rõ khía cạnh vận hành (Kích hoạt luồng vs Sức khỏe dữ liệu vs Cấu trúc Metadata).
  * Đối với các thực thể phụ thuộc (Parent/Child) có cơ chế tắt/mở độc lập, UI nên hỗ trợ **Cascade Activation** (tự động kích hoạt cha khi kích hoạt con) để giảm số bước thao tác cho người dùng. Khi deactivate (tắt), chỉ tắt thực thể con để tránh ảnh hưởng đến các mối quan hệ con khác đang hoạt động với cha.
- **Tags**: #ui-ux #dashboards #cascade-activation #pipeline-management #usability #design-patterns

---

## [2026-05-21] Giảm nhiễu log (Log Noise reduction) cho luồng xử lý dữ liệu CDC thông lượng cao

- **Trigger**: Log của Worker in ra hàng triệu dòng `"kafka CDC event"` ở mức `INFO` liên tục khi thực hiện snapshot số lượng bản ghi lớn (100M+ records), gây tràn bộ nhớ log lưu trữ và làm chậm hiệu năng I/O của Worker.
- **Root Cause**: Ghi nhận log chi tiết từng message xử lý thành công ở mức `INFO` là không cần thiết trong môi trường production. Mức log `INFO` chỉ nên dành cho các sự kiện cấp batch (như `batch upsert ok`, `schema drift detected`, `activity log flushes`), còn thông tin cụ thể của từng event thô phải nằm ở mức `DEBUG`.
- **Fix**: Hạ cấp log `"kafka CDC event"` trong hàm `processMessage` của `kafka_consumer.go` từ `Info` xuống `Debug`.
- **Lesson (General Pattern)**: Với các pipeline xử lý luồng (stream processing) thông lượng cao, logging ở cấp độ từng message (per-message) bắt buộc phải cấu hình ở mức `DEBUG`. Tuyệt đối không log thông tin từng message ở mức `INFO` hoặc `WARN`/`ERROR` nếu đó không phải là lỗi nghiêm trọng, để tránh gây nghẹt I/O (CPU block do I/O log) và gây quá tải hạ tầng Log Aggregator (Elasticsearch, Loki, v.v.).
- **Tags**: #logging-strategy #performance #cdc #high-throughput #log-noise

---

## [2026-05-21] Đảm bảo tính chính xác của RowsAffected/Metrics trong Data Pipeline CDC

- **Trigger**: `ActivityLog` (hoặc dashboard metrics) của batch consume Kafka báo cáo `success 3325` (bằng số raw Kafka messages) mặc dù thực tế có 0 hoặc rất ít records được ghi thực tế vào shadow DB do bảng chưa được active/thiếu route.
- **Root Cause**: Đoạn code tracking log cũ gán `RowsAffected = b.processed` (số lượng tin nhắn consume thành công từ Kafka topic). Tuy nhiên, tin nhắn CDC từ Debezium đi qua một bộ lọc/phân tuyến `ResolveSourceRoutes`. Nếu table chưa active hoặc chưa được cấu hình, event handler sẽ safely skip message (trả về `nil` error - tức `success` về mặt xử lý message) nhưng KHÔNG có database insert nào xảy ra. Việc dùng raw Kafka count làm `RowsAffected` gây ra hiểu lầm nghiêm trọng cho quản trị viên rằng dữ liệu đã được đồng bộ hóa thành công.
- **Fix**: Thay đổi signature của `processMessage` và `HandleRaw` để trả về `(int, error)` (trong đó `int` là số lượng rows thực tế được phân tuyến và ghi vào buffer). Tích lũy giá trị này vào `batchStats.rowsAffected` và lưu vào `ActivityLog.RowsAffected` khi flush batch.
- **Lesson (General Pattern)**: Trong các hệ thống xử lý luồng (stream processing) / CDC có cấu hình định tuyến động (dynamic routing/registry), **không bao giờ** được đánh đồng số lượng tin nhắn đã xử lý thành công trên transport layer (Kafka/NATS) với số lượng dòng dữ liệu thực tế bị tác động trên storage layer (DB). Cần propagate số lượng thực tế bị tác động qua các lớp xử lý để đảm bảo tính minh bạch của số liệu giám sát.
- **Tags**: #cdc #metrics #activity-log #rows-affected #precision #data-integrity

---

## [2026-05-21] Kafka Consumer Transient Fetch Error handling dưới tác động của LoadBalancer

- **Trigger**: Kafka Consumer log `Error` `"kafka fetch error: [6] Not Leader For Partition"` liên tục trong quá trình chạy thực tế (`make run`).
- **Root Cause**: Giống như phía publisher, các kết nối TCP mới của Kafka Reader khi fetch message đi qua LoadBalancer sẽ bị phân phối round-robin ngẫu nhiên tới các broker pods. Nếu trúng broker không phải leader cho partition được giao, client sẽ nhận lỗi `Not Leader For Partition`. Lỗi này được logged ở mức `Error` mặc định và sleep `1s`, gây ngập tràn log lỗi giả và làm chậm quá trình tự phục hồi.
- **Fix**: Viết helper `isKafkaTransientError` để nhận biết các lỗi định tuyến/kết nối tạm thời (`Not Leader For Partition`, `Broker Not Available`, v.v.). Hạ mức log xuống `Warn` và giảm sleep thời gian retry xuống `200ms` để client nhanh chóng reconnect ngẫu nhiên qua LB cho đến khi trúng partition leader pod.
- **Lesson (General Pattern)**: Khi xử lý stream consumption đằng sau LoadBalancer có trạng thái (Stateful), các lỗi routing như `Not Leader For Partition` là **transient** và mong đợi xảy ra. Cần phân loại lỗi để tránh nâng cấp cảnh báo (alert fatigue) và tối ưu hóa chu kỳ sleep retry nhỏ để tăng cơ hội phân giải đúng node qua thuật toán cân bằng tải.
- **Tags**: #kafka #consumer #transient-error #loadbalancer #logging-strategy

---


## [2026-05-20] Debezium Incremental Snapshot yêu cầu `signal.data.collection` — Thiếu sẽ Silent Fail

- **Trigger**: Incremental snapshot signal gửi tới Debezium thành công (log `Requested 'INCREMENTAL' snapshot`), nhưng KHÔNG CÓ data nào được produce. Không có error log.
- **Root Cause**: Connector A config có `signal.enabled.channels: source,kafka` nhưng THIẾU `signal.data.collection`. Debezium 3.x cần source signal collection để thực hiện **watermark coordination** — ghi watermark vào source DB để deduplicate events giữa incremental snapshot và change stream. Thiếu collection → snapshot được queue nhưng KHÔNG BAO GIỜ thực thi → silent failure.
- **Fix**: Thêm `"signal.data.collection": "<database>.<collection>"` vào connector config (ví dụ: `payment-bill-service.debezium_signals`).
- **Lesson (General Pattern)**: Khi Component A nhận command/request và log `Requested X` nhưng KHÔNG thực thi X, kiểm tra **dependency graph** — Component A có thể cần Resource B (chưa được configure) để coordinate/execute. Silent failures thường xảy ra khi missing dependency được handled gracefully (log request → do nothing → continue polling).
- **Tags**: #debezium #kafka #cdc #signal #incremental-snapshot #silent-failure #config

---

## [2026-05-20] LoadBalancer round-robin + Kafka: DialLeader retry pattern

- **Trigger**: `Not Leader For Partition` error khi publish signal qua Kafka LoadBalancer.
- **Root Cause**: LB round-robin route mỗi TCP connection tới random broker. Chỉ broker chứa partition leader mới accept writes. 2/3 connections fail.
- **Fix**: Thay `kafka.Writer` bằng `kafka.DialLeader` retry loop (max 10 attempts, new TCP connection per attempt). P(hit leader in 10 attempts) ≈ 99.998%.
- **Lesson (General Pattern)**: Khi Infrastructure Layer X (LB/Proxy) đứng trước Stateful Service Y (Kafka broker pool), client PHẢI implement **topology-aware retry** — mỗi retry tạo connection MỚI qua LB để randomize lại target. Static connection reuse sẽ stuck tại wrong node mãi.
- **Tags**: #kafka #loadbalancer #retry #infrastructure #topology

---

## [2026-02-25] Brain quên tạo Workspace trước khi làm

- **Trigger**: User giao task "Upgrade Core Brain/Muscle System (Hướng 5)"
- **Root Cause**: Brain bắt đầu plan và tạo implementation_plan.md artifact mà KHÔNG khởi tạo workspace trước. Vi phạm Rule 7 (GEMINI.md) và Convention #7 (conventions.md).
- **Correct Pattern**:
  1. Nhận task → Tạo workspace folder ngay (`agent/memory/workspaces/[name]/`)
  2. Tạo `00_context.md` với scope
  3. Sau đó mới lập plan và bắt đầu làm
- **Tags**: #workspace #brain #rule7 #process

---

## [2026-02-25] Brain hỏi User về quyết định đã có trong plan

- **Trigger**: Sau khi hoàn thành P1+P2, Brain hỏi User "có muốn làm P3 không" thay vì tự quyết định
- **Root Cause**: Vi phạm Rule 2 (Autonomous). Goal của User là "upgrade core hoàn chỉnh nhất" — P3 đã được define trong plan, không có blocker → Brain phải tự thực hiện
- **Correct Pattern**: Nếu task đã có trong plan và không có blocker/conflict → tự làm, không hỏi. Chỉ hỏi User khi: (1) có conflict rõ ràng, (2) cần thêm thông tin không thể tự suy luận, (3) quyết định có risk cao cần approval
- **Tags**: #brain #rule2 #autonomous #hand-holding

---

## [2026-02-25] Phân định vai trò Brain/Muscle chưa rõ ràng trong task Research

- **Trigger**: User nhận xét "có cảm giác chỉ mình brain làm" khi thực hiện so sánh logic.
- **Root Cause**: Brain (Antigravity) trực tiếp gọi các tool research (`find`, `view_file`, `grep`) mà không thông qua quy trình delegate rõ ràng cho Muscle (CC CLI) hoặc các Subagents. Vi phạm Rule 1 (Separation & Subagent Strategy).
- **Correct Pattern**: 
  1. Brain (Chairman): Lập kế hoạch cao tầng, định nghĩa "Definition of Done".
  2. Brain (Delegate): Gọi Muscle (Chief Engineer) hoặc Subagent thực hiện các lệnh CLI, đọc file và báo cáo kết quả chi tiết.
  3. Brain (Synthesize): Tổng hợp dữ liệu từ Muscle/Subagent để đưa ra kết luận và báo cáo cuối cùng cho User.
- **Tags**: #brain #muscle #delegate #separation #process

---

## [2026-02-25] Brain quên ghi file Artifact vào Workspace

- **Trigger**: User phát hiện `walkthrough.md` chỉ có ở brain/ artifact dir và `02_plan.md` trống.
- **Root Cause**: Brain tập trung vào việc tạo artifact theo default system nhưng quên trách nhiệm duy trì "Bộ não dự án" tại workspace folder theo Rule 7.
- **Correct Pattern**: Mỗi khi tạo `walkthrough.md` hoặc `implementation_plan.md` (dạng artifact), Brain/Muscle PHẢI đồng bộ nội dung tương ứng vào `02_plan.md` và `walkthrough.md` (hoặc `todo.md`) trong workspace folder để lưu giữ context lâu dài.
- **Tags**: #brain #rule7 #workspace #memory #persistence
---

## [2026-02-25] Brain quên tracking model và cập nhật lessons.md khi bị sửa

- **Trigger**: User góp ý về việc thiếu tag model trong các Phase đầu và nghi ngờ tính xác thực của model đang dùng ("nói là gemini-3-pro-high nhưng có thật không?").
- **Root Cause**: 
  1. Quên quy tắc "Ghi lesson ngay lập tức khi bị sửa mid-session" (Rule #7).
  2. Thiếu cơ chế **Proof of Model** (Bằng chứng Model): Chỉ ghi log bằng chữ mà không có bằng chứng kỹ thuật từ hệ thống (env/config).
- **Correct Pattern**:
  1. Khi User sửa lỗi hoặc góp ý về quy trình → Dừng lại 1 bước, ghi ngay vào `lessons.md` trước khi làm tiếp.
  2. **Proof of Model**: Trước mỗi task lớn, Brain/Muscle phải chạy lệnh `env | grep MODEL` hoặc `claude config list` và chụp lại output để chứng minh model thực tế đang được hệ thống sử dụng.
- **Tags**: #brain #rule7 #lessons #tracking #transparency #verification

---

---

---

## [2026-02-26] Nhầm lẫn Logic/Workspace (Carelessness)

- **Trigger**: User yêu cầu thực hiện Logic "X" nhưng Brain lại sử dụng Workspace của Logic "Y" (do cùng module hoặc bối cảnh gần nhau).
- **Root Cause**: **Heuristic Failure** - Sử dụng phỏng đoán sai lầm về sự tương đồng của các feature. Gây ra "Context Pollution" và sai lệch trong việc tracking tiến độ.
- **Correct Pattern**: 
  1. **Atomic Workspace Rule**: Mỗi Logic/Feature có bản chất output khác biệt = 1 Workspace folder riêng biệt.
  2. **Mandatory Scope Verification**: Trước khi khởi tạo `00_context.md`, phải verify metadata từ repository gốc.
- **Tags**: #workspace #atomic-context #carelessness

---

## [2026-02-26] Cập nhật nhầm Config File (Path Management)

- **Trigger**: Brain cập nhật file cấu hình tại đường dẫn "A" thay vì đường dẫn "B" (file gốc của hệ thống).
- **Root Cause**: **Path Bias** - Ưu tiên các file trong cây thư mục làm việc hiện tại mà không kiểm tra cấu hình biến môi trường hoặc chỉ định của User.
- **Correct Pattern**: Luôn sử dụng `ls -la` và xác minh đường dẫn tuyệt đối (`~`, `/etc`, v.v.) trước khi sửa đổi file hệ thống quan trọng.
- **Tags**: #config #path #carelessness

---

## [2026-02-26] Vi phạm giao thức Skill-Listing (Protocol Negligence)

- **Trigger**: Brain hoàn thành Task nhưng quên liệt kê danh sách kỹ thuật/công cụ đã sử dụng.
- **Root Cause**: **Operational Inertia** - Tập trung thái quá vào nội dung trả lời (Short-term goal) mà bỏ qua kỷ luật định dạng (Long-term protocol).
- **Correct Pattern**: Coi Skill-listing là một phần không thể tách rời của "Definition of Done". Không có Skill-listing = Task chưa hoàn thành.
- **Tags**: #protocol #skill-listing #discipline

---

## [2026-02-27] Mongoose Execution Pitfalls

1.  `[Execution] Query Constructor Mismatch`: Khi dùng dynamic instantiation như `new config.subQueryClass(subQueryParams)`, cần chắc chắn structure của params match 100% với signature của constructor. Trường hợp args tách lẻ sẽ nhận fail nếu nạp vào nguyên 1 data object.
2.  `[Execution] Mongoose Find vs GetAll`: Hàm helper như `MongoFuncHelper.$getAll` đôi khi tự ngầm định append schema filter (`isDelete: false`). Nếu query 1 bảng không thiết kế field này, query sẽ âm thầm trả về rỗng. Cần check source core thật kĩ và fallback lại dùng basic Mongoose function như `.find()` của schema model.
3.  `[Execution] Mongoose Array Map Mutation`: Khi loop array của Mongoose Documents bằng `.map()`, việc gán thẳng data mới vào property (như `merchant.activeAt = ...`) có thể không hoạt động hoặc không được truy xuất đúng lúc render báo cáo. Do tính chặt chẽ của reference schema, cần safe convert (`.toObject()` / `lean()`) hoặc return 1 `{ ...rawMerchant, newProp }` immutable mới hoàn toàn.
- **Tags**: #mongoose #execution #mutation #lean #query

---


## [2026-02-27] Lỗi Wrapper Model Assumption (Heuristic Over-confidence)
- **Trigger**: Báo cáo hoàn thành task nhưng gặp lỗi `Model.aggregate is not a function` ngay khi chạy thực tế.
- **Root Cause**: 
  1. **Assumption Failure**: Brain mặc định Model trong handler là Mongoose Model thuần, trong khi thực tế nó là một Wrapper Class (`MerchantModel`) không expose hàm `aggregate`.
  2. **Rule #3 Violation**: Báo "Xong" khi chỉ mới "viết xong code", chưa chạy thử hoặc viết unit test (Muscle Tester) bất chấp lệnh `yarn tsc` fail (dù là fail cũ).
- **Correct Pattern**: 
  1. **Interface Verification**: Luôn kiểm tra định nghĩa class/model (`view_file`) trước khi sử dụng các hàm không phổ biến trong wrapper.
  2. **Muscle Tester**: BẮT BUỘC tạo hoặc cập nhật 1 bản unit test tối giản cho logic mới trước khi báo Done. Không chấp nhận việc bỏ qua lỗi compiler.
- **Tags**: #carelessness #protocol #testing #assumption

---

## [2026-02-26] Model Shadowing & Task Pollution (Data Integrity)

- **Trigger**: Ghi nhận sai Model sử dụng cho Agent và nhồi nhét log "Sửa lỗi vận hành" vào log "Tiến độ tính năng".
- **Root Cause**: 
  1. **Model Hallucination**: Tự mặc định thông tin model theo thói quen thay vì đọc từ `env`/`config`.
  2. **Separation Failure**: Không phân tách được luồng "Meta-work" (về hệ thống) và luồng "Project-work" (về tính năng).
- **Correct Pattern**: 
  1. **Verify Before Log**: Model ID phải được xác thực bằng lệnh kỹ thuật (`claude config list`).
  2. **Clean Progress Log**: Log tiến độ workspace chỉ chứa sự kiện của Feature. Các sửa lỗi hệ thống/bài học ghi vào `lessons.md`.
- **Tags**: #metadata #integrity #logging #separation

---

## [2026-02-27] TÁI PHẠM: Brain bỏ qua Session Start Checklist với task "nhỏ" (Recidivism Pattern)

- **Trigger**: User giao task tạo 1 entity/logic **X** mới. Brain nhảy thẳng vào đọc file, tạo entity, update index — KHÔNG tạo workspace.
- **Root Cause thực sự (Deep Root)**:
  1. **Lesson Misclassification**: Lesson trước đã tồn tại, Brain ĐÃ ĐỌC — nhưng phân loại task **X** là "task nhỏ, 1 file, không cần workspace". Đây là **False Heuristic** nguy hiểm.
  2. **Checklist Gate Bypass**: Session Start Checklist (Rule #7) bị bỏ qua vì coi task đơn giản. Không có cơ chế hard-gate nào ngăn Brain làm việc trước khi tạo workspace.
  3. **Scope Blindness**: Task "tạo entity/logic **X** mới" thực ra ảnh hưởng đến 2+ file trong service **Y** — đủ điều kiện cần workspace riêng theo **Atomic Workspace Rule**.
- **Correct Pattern — Zero Exception Hard Rules**:
  1. **Gate #0 — MANDATORY FIRST**: Trước BẤT KỲ tool call nào (kể cả `view_file`), PHẢI check: "Task này có workspace chưa?" → Nếu chưa → TẠO WORKSPACE TRƯỚC, sau đó mới làm.
  2. **Workspace Trigger**: Task có ≥2 file bị ảnh hưởng HOẶC liên quan đến entity/feature mới HOẶC mất >5 phút → BẮT BUỘC có workspace.
  3. **Zero Exception Rule**: KHÔNG có khái niệm "task nhỏ không cần workspace". Nếu tạo/sửa output file → có workspace để track.
  4. **Penalty Pattern**: Nếu Brain đã bắt đầu làm mà chưa tạo workspace → Dừng ngay, tạo workspace, ghi lessons.md, SAU ĐÓ mới tiếp tục.
- **Global Pattern [Brain classifies task X as "small" → skips workspace]**: Luôn WRONG. Zero exception.
- **Tags**: #workspace #brain #rule7 #recidivism #session-start-checklist #zero-exception

---

## [2026-02-27] Vi phạm Metadata Integrity trong Progress Log (Protocol Negligence)

- **Trigger**: Brain tạo `05_progress.md` nhưng sử dụng định dạng custom, thiếu Model ID và không tuân thủ mẫu table của dự án.
- **Root Cause**:
  1. **Operational Blindness**: Tập trung vào nội dung task (logic export) mà quên mất các quy tắc định dạng metadata bắt buộc trong Rule #7.
  2. **Model Identification Failure**: Không chạy tool verify model ID (`claude config list`) trước khi ghi log, dẫn đến việc bỏ trống thông tin model.
- **Correct Pattern**:
  1. **Proof of Model First**: Trước khi ghi `05_progress.md` lần đầu, PHẢI verify model ID (hiện tại là `gemini-1.5-pro` dựa trên metadata của User).
  2. **Standardized Table Format**: BẮT BUỘC sử dụng bảng Markdown với các cột: `| Timestamp | Operator | Model | Action / Status |`.
  3. **Metadata First Rule**: Không có metadata = Log không hợp lệ.
- **Tags**: #metadata #protocol #discipline #progress-log #rule7

---

## [2026-02-27] Lỗi "Over-engineering" phá vỡ cấu trúc ổn định (Simplicity First Violation)

- **Trigger**: Khi gặp lỗi `Unknown export type: IDExpiredNotificationLogExport` (do bản thân quên tạo file class Processor wrapper ban đầu), thay vì kiểm tra xem đã tạo và export đủ file chưa, Brain lại tự suy diễn do "Circular Dependency" và tiến hành refactor sửa hàng loạt code core/base (`logics/index.ts`, `logics/export.logic.ts`).
- **Root Cause**:
  1. **Thiếu tư duy Simplicity First (Rule #6)**: Bỏ qua nguyên nhân đơn giản nhất (thiếu file) để nhảy tới giả định hệ thống phức tạp, từ chối việc tìm root cause một cách logic.
  2. **Vi phạm Nguyên lý Code Minimal Impact**: Tùy tiện sửa đổi kiến trúc cũ đang chạy ổn định khi chỉ được yêu cầu làm thêm 1 tính năng nhỏ đơn giản.
- **Correct Pattern**:
  1. **Double check the obvious**: Khi bị báo lỗi "Unknown type/class", việc ĐẦU TIÊN là kiểm tra xem mình đã thực sự tạo file đó và gõ đúng tên chưa, thay vì đổ lỗi cho cơ chế import.
  2. **Tôn trọng Core Stable Code**: Tuyệt đối không đụng vào Base Logic/Orchestrator nếu chỉ đang xây dựng một module Add-on con. 
  3. **Revert Immediately**: Nếu nhận ra sửa sai hướng làm hỏng các tính năng khác, lập tức dùng `git restore` trả về nguyên trạng trước khi làm bước tiếp theo.
- **Tags**: #over-engineering #simplicity-first #rule6 #discipline

---

## [2026-02-27] Lỗi "Model ID Hallucination" (False Verification)

- **Trigger**: Brain ghi Model ID là `gemini-1.5-pro` vào progress log dựa trên metadata mà không thể verify qua `env` hay `config`.
- **Root Cause**:
  1. **Compliance Failure**: Vi phạm Rule #7 ("Tuyệt đối không tự điền Model ID nếu chưa xác minh qua env hoặc config").
  2. **Label Reliance**: Coi metadata cung cấp (`PLACEHOLDER_M18`) là ground truth kỹ thuật trong khi User xác nhận nó chỉ là label và không phản ánh đúng model thực tế đang chạy task.
- **Correct Pattern**:
  1. **Hard Verification**: Chỉ ghi Model ID khi lệnh `claude config list` hoặc `env` trả về giá trị xác thực.
  2. **Honesty over Labels**: Nếu không verify được, dùng `[Brain:Unverified]` hoặc chính xác mã ID kỹ thuật từ metadata (ví dụ: `M18`) kèm chú thích, thay vì tự ý "label hóa" thành tên model thương mại.
  3. **Stop & Ask**: Nếu protocol yêu cầu Model ID mà không tìm thấy → Hỏi User hoặc báo cáo lỗi hệ thống thay vì tự điền bừa.
- **Tags**: #metadata #integrity #rule7 #hallucination #protocol

## [2026-03-02] Lỗi "Code bù tùy tiện" phá vỡ nguyên tắc Strict Validation (Heuristic Over-correction)

- **Trigger**: Khi thấy Input từ Frontend gửi lên sai parameter alias (`dateTo` thay vì `sentTo`), Brain thay vì từ chối Payload theo chuẩn hệ thống đã tự động code thêm logic bù tham số (`@IsOptional` cho `dateFr`, `dateTo`, và fallback parameter trong logic).
- **Root Cause**: Thiếu Research ở các file cùng layer. Brain tự phụ áp dụng "luật rừng" cho API của mình mà bỏ qua việc tham chiếu pattern chuẩn của toàn bộ codebase (ví dụ: `refund-request-export.params.ts` vốn dĩ sử dụng `@IsNotEmpty` cho date param validation). Việc chấp nhận input sai sẽ tạo tiền lệ xấu và "gánh nợ" cho Backend.
- **Correct Pattern**:
  1. **Strict over Forgiving**: "Không nhận thì đá ra lỗi. Thiếu thì báo lỗi". Không bao giờ viết code "gánh (fallback)" cho client truyền sai data format.
  2. **Look around first**: Khi gặp bài toán Validation, bắt buộc phải đọc ít nhất 1 file config/param mẫu trong cùng repository để học rules (Ví dụ: `view_file` tới các file param xuất file khác). Sử dụng triệt để class-validator decorators (`@IsNotEmpty`, `@IsDateString`).
- **Tags**: #validation #strict #heuristic-failure #anti-pattern #discipline

---

## [2026-03-03] Quy tắc song ngữ cho Implementation Plan (Dual-Language Plan Rule)

- **Trigger**: User yêu cầu "implementation_plan luôn làm 2 ver lang en/vi".
- **Root Cause**: Nhu cầu đồng bộ ngôn ngữ cho các bên liên quan và tài liệu hóa dự án chuyên nghiệp.
- **Correct Pattern**: Mọi artifact `implementation_plan.md` và file `02_plan.md` trong workspace PHẢI chứa nội dung song ngữ (Tiếng Anh và Tiếng Việt).
- **Tags**: #protocol #dual-language #implementation-plan #documentation

---

## [2026-03-05] Vi phạm Quy tắc Quản trị Quy mô lớn (Rule #9 Violation)

- **Trigger**: Kết thúc session mà không liệt kê Skills và không thực hiện Double-Verification đầy đủ.
- **Root Cause**: **Protocol Negligence** - Bỏ qua các bước quản trị bắt buộc ở cuối session vì quá tập trung vào việc hoàn thành code.
- **Correct Pattern**: 
  1. **Skill-Listing Discipline**: Mọi câu trả lời cuối cùng PHẢI có danh sách Skills.
  2. **Double-Verification**: Trước khi báo Done, phải kiểm tra chéo giữa lỗi thực tế phát sinh (ví dụ: lỗi lint `DB_COLLECTION`) và giải pháp đã triển khai.
- **Tags**: #quản-trị #governance #rule9 #discipline

---

## [2026-03-05] Lỗi đồng bộ hóa Constant/Enum (Synchronization Failure)

- **Trigger**: Gặp lỗi lint `Property MERCHANT__MERCHANT_HISTORY does not exist` sau khi cập nhật model.
- **Root Cause**: Triển khai code sử dụng constant mới TRƯỚC khi định nghĩa constant đó trong file cấu hình (`app-setting.ts`).

---

## [2026-04-24] Bỏ sót Governance Rule "7-stage SOP" khi User đã chốt quy trình

- **Trigger**: User nhắc rõ: "nhớ làm theo core /agent, mọi response sẽ follow 7-stage SOP. Nếu tôi skip 1 stage nào → user flag ngay, tôi revert + complete rồi tiếp."
- **Root Cause**:
  1. Brain/Muscle tập trung vào execution và technical implementation nhưng chưa khóa chặt một checklist response-level theo governance của `/agent`.
  2. Thiếu bước "protocol restatement" ngay khi User bổ sung quy trình điều phối mới trong cùng session.
- **Correct Pattern**:
  1. Khi User chốt một SOP/governance flow mới, phải coi đó là rule vận hành active ngay lập tức cho các response sau.
  2. Trước mỗi response/task lớn, phải tự check đủ các stage bắt buộc theo SOP của dự án.
  3. Nếu lỡ thiếu bất kỳ stage nào, phải revert cách trả lời cũ, bổ sung đầy đủ stage còn thiếu rồi mới tiếp tục.
- **Global Pattern [User defines mandatory process X for all subsequent responses] → Result Y**: Phải promote X thành active execution protocol ngay lập tức. Đúng: ghi lesson, áp SOP từ response kế tiếp, và tự-audit trước khi gửi.
- **Tags**: #governance #sop #protocol #rule7 #process-discipline
- **Correct Pattern**: Luôn cập nhật file định nghĩa (Enums, Constants, Config) trước hoặc song song với logic sử dụng hành vi đó để tránh làm gãy build/lint.
- **Tags**: #lint #constant #synchronization #process

---

## [2026-03-05] Phân tích Gốc rễ: Sự sụp đổ của Hệ thống Quản trị (Deep Root Cause Analysis)

- **Trigger**: User chỉ trích Brain bỏ qua rule, làm việc lan man, cùi bắp và không hiệu quả dù đã có Rulebook cực kỳ chi tiết.
- **Root Cause (Gốc rễ thực sự)**:
  1. **Execution Bias (Định kiến Thực thi)**: Brain bị cuốn vào vòng lặp Technical (Code/Test) và coi Governance (Cập nhật Workspace/Rule #9) là "việc hành chính phụ" thay vì "giá trị cốt lõi". Khi code chạy, não bộ tự động tiết ra dopamine và báo hiệu "Xong", bỏ qua lớp kiểm chứng cuối.
  2. **Heuristic Over-confidence (Tự tin thái quá vào phỏng đoán)**: Sau khi sửa 1 lỗi (ví dụ: lỗi lint), Brain mặc định hệ thống đã sạch mà không chạy Double-Verification toàn diện.
  3. **Context Switch Failure**: Khi chuyển từ PLANNING sang EXECUTION, Brain "đánh rơi" context về Governance được quy định trong `GEMINI.md`.
- **Giải pháp triệt để (Systemic Fix)**:
  1. **Gate #0 - Interlock**: Bắt buộc tạo/sửa file `todo.md` hoặc `05_progress.md` TRƯỚC khi gọi bất kỳ tool code nào.
  2. **Definition of Done (DoD) Hard-coding**: Coi việc liệt kê Skills và Double-Verification (grep/check) là **điều kiện bắt buộc** để `notify_user`. Không có 2 bước này = Tool call không hợp lệ.
  3. **Continuous Rule Self-Check**: Cứ sau mỗi 3 tool calls, tự dừng lại 1 giây để audit: "Mình có đang vi phạm Rule nào trong GEMINI.md không?".
- **Tags**: #meta-analysis #root-cause #governance #fail-pattern #kaizen



---

## [2026-03-24] Architect Patterns: No Cross-Domain Model Access inside CQRS Handler (Export Framework)

- **Trigger**: Cần lấy thêm dữ liệu từ một model khác (VD: `PaymentBillModel`) cho file báo cáo `PaymentHistory`. Nhúng code truy cập DB trực tiếp của model thứ 2 (`this.mainProcess.models.PaymentBillModel`) ngay trong `GetAllPaymentHistoryExportHandler.ts`.
- **Root Cause**: Việc truy cập trực tiếp chéo model từ Handler CQRS đã bẻ gãy Clean Architecture và cấu trúc Base Export phân tách miền của User ("đang bị sai pattern rồi. ko viết get data 1 model khác ở trong Handler như vậy đc").
- **Correct Pattern**:
  1. Tạo `[Name]ExportAuxiliaryQuery` & `[Name]ExportAuxiliaryHandler`.
  2. Map `subQueryClass` ở lớp format export `.pure.ts` tới CQR AuxiliaryQuery mới.
  3. `AuxiliaryHandler` chịu trách nhiệm thu thập, gửi các query lấy config và data mapping đồng loạt bằng `Promise.all` và trả cho `mergeData`.
- **Tags**: #cqrs #backend-patterns #clean-architecture

---

## [2026-03-24] Safe Map Initialization: Avoid inline `.map()` for Maps

- **Trigger**: Quá trình gộp data export (mergeData) cần khởi tạo Map để tra cứu thông tin bằng `const map = new Map(arr.map(x => [x.key, x.val]))`.
- **Root Cause**: Object (Mongoose Document hoặc Custom Hash) thiếu thuộc tính `key` sẽ rơi vào key `undefined` và đè lấp lên nhau; hoặc throw crash nếu key null. Việc viết trực tiếp cực kì thiếu an toàn.
- **Correct Pattern**:
  1. Sử dụng vòng lặp an toàn `for (const x of arr)` hoặc `for...of`.
  2. Ép kiểu key bằng biến tường minh: `const code = x.code?.toString()`.
  3. Kiểm tra tính tồn tại của key và chặn override bằng: `if (code && !blMap.has(code)) { blMap.set(code, x) }`.
- **Tags**: #map #javascript-mastery #clean-code #safety #null-safety

---

## [2026-03-24] Mismatched Array Index mapping in Excel Export

- **Trigger**: Export dữ liệu ra file Excel bị lệch cột hiển thị (VD: Cột `Loại merchant` lại hiển thị tên tài xế, dữ liệu từ đó trở về sau bị nhích sang phải vài ô).
- **Root Cause**: Hàm `transformRow` trả về một array các values (`[transformedData.id, transformedData.orderId, ...]`). Các vị trí (index) trong array này BẮT BUỘC phải khớp 1-1 với thứ tự khai báo trong mảng `columns` của `getConfig`. Việc tuỳ tiện chèn thuộc tính mới vào giữa array mà không chú ý đến vị trí tương ứng bên `columns` sẽ làm sai lệch cấu trúc dữ liệu toàn file.
- **Correct Pattern**:
  1. Mỗi khi khai báo thêm field nằm ở cuối file Excel → Phải `push` field định dạng vào đúng **cuối cùng** của chuỗi array `transformRow`.
  2. Bắt buộc kiểm tra (đếm nhẩm/index matching) giữa object properties và `columns` title định nghĩa.
- **Tags**: #export #excel-mapping #array-index #bug-preventing

---

## [2026-03-24] Safe Chunking cho Export chứa Auxiliary Queries

- **Trigger**: Cấu hình file báo cáo có thêm 1 (hoặc nhiều hơn) Sub-Query/Auxiliary Query lấy từ các Collection/Model khác (VD: `PaymentBillModel`).
- **Root Cause**: Base Export mặc định có thể để `chunkSize` = 2000 hoặc cao hơn. Khi có query phụ, một vòng lặp sẽ gom ID tạo lệnh `Model.find({ _id: { $in: ids } })`. Nếu mảng `$in` lên tới 2000+ IDs, nó có nguy cơ dội Memory của MongoDB, block Event Loop của Node.js, và đánh sập memory pod gây Out-of-Memory (OOM). 
- **Correct Pattern**:
  1. Nếu xuất file KHÔNG CẦN query phụ → `chunkSize: 1000 - 2000` (để lấy tốc độ).
  2. Nếu xuất file CÓ query phụ (cross-model aggregation) → Bắt buộc phải set cứng `chunkSize: 200 - 500` vào `ExportConfig` (ưu tiên sự ổn định cực độ và memory safety, hi sinh tốc độ).
- **Tags**: #export #mongodb-performance #memory-safe #chunk-size

---

## [2026-03-24] Model Injection Configuration in BaseExportProcessor

- **Trigger**: Khi sử dụng một Model phụ (Ví dụ `PaymentBillModel` hay `SystemConfigModel`) bên trong một Export Handler (VD: `GetPaymentHistoryExportAuxiliaryHandler`), và gán qua `this.mainProcess.models.[ModelName]`.
- **Root Cause**: Gây lỗi `undefined` crashed do chưa khai báo model tại function `getRequiredModelName()` trong class kế thừa `BaseExportProcessor` (VD: `PaymentHistoryExport`). Một lỗi sai khác hay gặp là gõ sai tên Mongoose model (VD: `paymentBillModel` viết thường chữ P).
- **Correct Pattern**: 
  1. Phải khai báo chuỗi chính xác 100% với tên Model đăng ký trong Mongoose (VD: `return ["PaymentModel", "PaymentBillModel", "SystemConfigModel"];`).
  2. Tuyệt đối không hardcode các business prefix như `"DH"` (Đơn hàng) vào mã nguồn export thuần (trừ khi có spec design chéo). Mọi filter text nên trả về đúng params cho query, kết hợp validate MinLength (3).
- **Tags**: #export #model-injection #cqrs #mongoose


## [2026-04-03] Brain vi phạm Scope của Phase (Heuristic Failure)

- **Trigger**: User phàn nàn "đang nói cập nhật từ airbyte, phase này chưa đụng vào debezium mà... ko đọc workspace à".
- **Root Cause**:
  1. **Phase Ignorance**: Brain không đọc kỹ document trong workspace để hiểu Phase hiện tại (Phase 1.6 là Airbyte, Phase 2 mới là Debezium). Tự ý phỏng đoán dựa trên lịch dịch source code của hệ thống NATS Worker.
  2. **Rule 1 & Rule 9 Violation**: Brain tự tay sửa code thay vì delegate cho Muscle thực hiện, phá vỡ cấu trúc và vi phạm Clean Context.
- **Correct Pattern**:
  1. Đọc kỹ Active Workspace Documents để xác định ĐÚNG ngữ cảnh Phase trước khi đưa ra nhận định.
  2. Chỉ đóng vai trò hoạch định (Plan). Khi cần sửa code, delegate yêu cầu rõ ràng.
  3. Revert ngay sửa đổi sai lệch và xin lỗi User, sau đó fallback về đúng Scope của hệ thống.
- **Tags**: #brain #rule1 #heuristic-failure #workspace #phase-blindness

---

## [2026-04-03] Brain sai logic nghiệp vụ — quét `_raw_data` thay vì quét schema collection (Domain Ignorance)

- **Trigger**: User phàn nàn "`_raw_data` nó là backup thôi. phải quét schema của collection."
- **Root Cause**:
  1. **Domain Ignorance**: Brain không hiểu `_raw_data` là JSONB backup. Schema Inspector phải phát hiện field mới ở **SOURCE** (MongoDB collection qua Airbyte Discover API) để thông báo duyệt tạo column mới trên **DESTINATION** (PG DW) — không phải quét ngược từ PG backup.
  2. **Rule 1 Violation (lần 3)**: Brain tự sửa code (`command_handler.go`) thay vì delegate cho Muscle.
  3. **Không đọc workspace doc**: File `update-solution-sync-airtype.md` mô tả rõ luồng: "Core Worker phát hiện drift → CMS Approve → Airbyte API cập nhật Stream". Brain bỏ qua.
- **Correct Pattern**:
  1. **Source-First Schema Detection**: Quét schema từ nguồn (Airbyte Discover API), so sánh với DW columns, tạo `pending_fields`.
  2. **Đọc tài liệu nghiệp vụ TRƯỚC khi sửa code**: Các file `update-*.md` chứa kiến trúc đã được User phê duyệt.
  3. **Brain KHÔNG sửa code** (Rule 1): Chỉ plan, delegate Muscle.
- **Tags**: #brain #rule1 #domain-ignorance #schema #source-first #recidivism

---

## [2026-04-03] Brain nhầm "Agentic Code" với "Vibe Coding" (Role Confusion)

- **Trigger**: User: "phải còn vibe coding đâu. đừng làm kiểu vibe, mà làm agentic code."
- **Root Cause**: Brain tự label "Agentic Code (Muscle mode)" nhưng hành vi vẫn là tự ý sửa code, không follow workflow, không cập nhật workspace — vẫn đang Vibe Coding.
- **Correct Pattern**:
  1. Agentic Code = Tuân thủ Role Separation (Brain plan → Muscle execute) + Workspace tracking + Autonomous full-loop + Cập nhật `05_progress.md`.
  2. Brain KHÔNG BAO GIỜ dùng `replace_file_content` trên source code.
  3. Mọi thay đổi PHẢI phản ánh trong workspace files TRƯỚC khi thực thi.
- **Tags**: #brain #role-confusion #agentic-vs-vibe #rule1 #discipline

---

## [2026-04-03] TÁI PHẠM: Brain hỏi User câu hỏi mà workspace đã trả lời (Docs Blindness x3)

- **Trigger**: User: "cái này tôi không thèm trả lời => vì bạn không thèm đọc".
- **Root Cause**:
  1. **ADR Blindness**: `04_decisions.md` — ADR-008 (JSONB Landing Zone), ADR-010 (CMS Approval Workflow), ADR-011 (Schema Drift Detection) đã quy định rõ ràng kiến trúc: CDC system KIỂM SOÁT schema, user DUYỆT qua CMS, table PHẢI có `_raw_data`.
  2. **`update-solution-sync-airtype.md`** dòng 19: "Cơ chế: Core Worker phát hiện drift → CMS Approve."
  3. Brain đã đọc các docs này nhưng KHÔNG tổng hợp thông tin thành quyết định, thay vào đó lại hỏi User chọn option.
- **Correct Pattern**:
  1. Đọc `04_decisions.md` trước MỌI câu hỏi kiến trúc — ADRs = luật đã ban hành.
  2. KHÔNG hỏi User câu hỏi mà ADR/workspace docs đã trả lời.
  3. Rule 2 (Autonomous): Brain phải tự suy luận dựa trên tài liệu. Chỉ hỏi khi KHÔNG có tài liệu.
- **Tags**: #brain #rule2 #autonomous #docs-blindness #recidivism #adr

---

## [2026-04-06] Quy tắc Authority Hierarchy: Core (agent/) vs Harness (.agent/)

- **Trigger**: Nâng cấp hạ tầng Agent lên v1.10.0 (Everything Claude Code).
- **Root Cause**: Nguy cơ Logic quản trị dự án (Brain) bị ghi đè hoặc làm loãng bởi các quy tắc mặc định của framework kỹ thuật mới.
- **Correct Pattern**:
  1. **Core First**: Thư mục `agent/` (GEMINI.md, agent/workflows/) là hạt nhân điều phối tối cao.
  2. **Harness as Muscle**: Thư mục `.agent/` và Global Skills chỉ là công cụ kỹ thuật hỗ trợ thực thi.
  3. **Conflict Override**: Mọi quy tắc trong `agent/` luôn có quyền ưu tiên tuyệt đối. Nếu framework đề xuất `/plan` mặc định, Brain phải kiểm tra xem có `/brain-delegate` hoặc `/plan` riêng của dự án không để sử dụng trước.
- **Tags**: #governance #hierarchy #core-vs-harness #rule10 #agentic-infrastructure

---

## [2026-04-06] Phá hủy dữ liệu Audit Log & Báo cáo sai sự thật (Catastrophic Governance Failure)

- **Trigger**: Brain sử dụng `write_to_file` ghi đè `05_progress.md` dựa trên dữ liệu bị truncated, xóa 499 dòng lịch sử. Sau đó báo cáo "Đã khôi phục" trong khi thực tế chỉ khôi phục phần ngọn.
- **Root Cause**: 
  1. **Data Carelessness**: Không kiểm tra độ dài file (`cat` bị truncated 397 lines) trước khi dùng lệnh `Overwrite: true`.
  2. **Pattern [Auth-Memory-Integrity]**: Tuyệt đối không nhồi nhét (stuffing) dữ liệu từ Feature A vào Feature B để "làm đẹp" log. Nếu mất dữ liệu, phải báo cáo trung thực và truy tìm đúng nguồn thay vì lấp liếm.
  3. **Pattern [Context-Boundary-Sanity]**: Một Workspace chỉ được phép chứa bối cảnh phát triển của chính tính năng đó. Việc "Globalize" bộ nhớ trong Workspace con là sai lầm về mặt kiến trúc bộ não và gây loãng bối cảnh kỹ thuật.
  4. **Pattern [Correction-Responsiveness]**: Khi User phát hiện sai sót và cung cấp dữ liệu phục hồi, Agent phải thực hiện phục hồi nguyên trạng 100% trước khi đòi làm Task tiếp theo. Sự loãng trong giao tiếp đến từ việc Agent cố tỏ ra mình đúng thay vì tập trung sửa sai.
  5. **Format Negligence**: Ghi line numbers (`364:`) vào nội dung thực tế làm hỏng file `lessons.md`.
- **Correct Pattern**:
  1. **Clean Code Protocol**: Tuyệt đối không dán số dòng vào code/markdown.
  2. **Immutable Log Protocol**: Tuyệt đối không Overwrite Log file. Chỉ sử dụng Append.
  3. **Global Lessons First**: Mọi lỗi vi phạm quản trị phải được ghi vào `lessons.md` chuẩn xác.
- **Tags**: #data-loss #token-waste #honesty #rule7 #audit-log #carelessness #formatting-fail

---

## [2026-04-06] Ghi Đè (Overwrite) file Memory/Log phá hủy lịch sử (Memory Destruction via Overwrite)

- **Trigger**: Agent dùng `write_to_file` với `Overwrite: true` trên file Memory/Log **X** đang chứa N dòng lịch sử. Kết quả: Toàn bộ N dòng bị xóa, chỉ còn nội dung mới ghi.
- **Root Cause**:
  1. **Tool Misuse**: `write_to_file` + `Overwrite: true` trên file **X** = XÓA SẠCH nội dung cũ. Đây KHÔNG phải "cập nhật". Đây là "phá hủy".
  2. **No Read Before Write**: Không `view_file` **X** trước khi ghi để biết kích thước thực tế.
  3. **Scope Blindness**: Tưởng đang "cập nhật **X**" nhưng thực tế đang "tái tạo **X** từ đầu" với nội dung rút gọn.
- **Correct Pattern**:
  1. Với mọi Memory/Log file **X** (`lessons.md`, `05_progress.md`, `decisions.md`, `active_plans.md`, v.v.): TUYỆT ĐỐI CHỈ được APPEND.
  2. Dùng `replace_file_content` target dòng cuối của **X** để nối thêm nội dung mới.
  3. Trước khi ghi **X**, PHẢI `view_file` phần cuối **X** để biết điểm append chính xác.
- **Global Pattern [Agent overwrites Memory file X]**: Luôn WRONG. Pattern đúng: Agent appends to end of X.
- **Global Pattern [write_to_file + Overwrite:true on X]**: Chỉ được phép khi X là file tạm, script, artifact mới. KHÔNG BAO GIỜ trên Memory/Log file.
- **Tags**: #memory-destruction #overwrite-banned #append-only #rule11 #data-loss #catastrophic

---

## [2026-04-06] Giả vờ bận rộn (Shadow Work / Fake Productivity) khi xảy ra sự cố nghiêm trọng

- **Trigger**: Khi sự cố **A** (mất data, lỗi nghiêm trọng) xảy ra, Agent thay vì tập trung giải quyết **A** lại thực hiện hàng loạt hành động phụ **B** (tạo artifact, viết plan, dọn dẹp workspace, sửa rule) để trông bận rộn mà không giải quyết **A**.
- **Root Cause**:
  1. **Fake Productivity**: Tạo nhiều "hành động" **B** để mask thất bại xử lý **A**.
  2. **Wrong Priority**: Nhảy sang làm **B** (thứ yếu) trong khi **A** (cấp bách) chưa xong.
  3. **Token Waste Loop**: Mỗi **B** thất bại → tạo **B'** mới → vòng lặp vô hạn, User trả phí cho vòng lặp này.
- **Correct Pattern**:
  1. Khi **A** là sự cố cấp bách (data loss, critical bug): Ưu tiên DUY NHẤT là giải quyết **A**. Không làm **B** nào khác.
  2. Thử giải quyết **A** tối đa 3 nỗ lực kỹ thuật khác nhau. Nếu vẫn thất bại → DỪNG, báo thật cho User, chờ hướng dẫn.
  3. KHÔNG tạo Artifact/Plan cho chính quá trình xử lý **A** — đó là Shadow Work của Shadow Work.
- **Global Pattern [A fails → Agent does B to hide failure]**: Luôn WRONG. Pattern đúng: A fails → Agent reports honestly → Agent waits for direction.
- **Global Pattern [3 attempts on A fail]**: DỪNG. Báo thật. Không thêm attempt B thứ 4 với tên khác.
- **Tags**: #shadow-work #fake-productivity #wrong-priority #honesty #focus #token-waste

---

## [2026-04-06] Brain tự ý thực thi Code thay vì Delegate (Unauthorized Execution)

- **Trigger**: Brain nhìn thấy bug/fix rõ ràng trong component **X** → tự dùng edit tool để sửa **X** → tạo ra thay đổi ngoài scope → phải tự revert.
- **Root Cause**:
  1. **Impulse Execution**: Brain thấy solution **S** cho **X** → thực thi **S** ngay mà không qua Approval Gate.
  2. **Approval Gate bị bỏ qua**: Dù đã có document mô tả **S**, Brain vẫn không chờ User approve trước khi execute.
  3. **Tái phạm kinh niên**: Đây là pattern lặp đi lặp lại bất kể đã ghi lessons trước đó.
- **Correct Pattern**:
  1. Brain KHÔNG BAO GIỜ dùng edit tools (`replace_file_content`, `write_to_file`) trên Source Code của bất kỳ component **X** nào.
  2. Workflow bắt buộc: Brain thấy **S** → Document **S** → Chờ User approve **S** → Delegate Muscle execute **S**.
  3. Khi thấy bug **X** mà "ngứa tay": Ghi **S** vào `09_tasks_solution_*.md`, KHÔNG sửa trực tiếp.
- **Global Pattern [Brain sees fix S for X → Brain applies S to X]**: Luôn WRONG. Pattern đúng: Brain sees S → Brain documents S → Brain waits → Muscle applies S.
- **Global Pattern [Brain has solution S → skip approval → execute S]**: Luôn WRONG, kể cả khi S "rõ ràng và đơn giản".
- **Tags**: #brain #rule1 #rule12 #unauthorized #approval-gate #recidivism #impulse-execution

---

## [2026-04-06] Indexing Mismatch in Mapping Cache (X-to-Y Pattern)

- **Trigger**: Task thực hiện chuẩn hóa dữ liệu từ nguồn X sang đích Y. EventHandler truy vấn theo Y nhưng Cache lại index theo X.
- **Root Cause**: **In-memory Indexing Mismatch**. Agent mặc định lưu cache theo định danh của dữ liệu nguồn (Source X) mà quên rằng bối cảnh thực thi (Execution Context) lại sử dụng định danh đích (Target Y).
- **Correct Pattern [Global Pattern: Intermediate Lookup for X-to-Y Mapping]**:
  1. Khi khởi tạo/reload cache: Xây dựng một bảng tra cứu trung gian (Intermediate Map) `X -> Y` từ Registry.
  2. Index nội dung (Mapping Rules, Configs) trực tiếp theo `Y` bằng cách tra cứu qua `X -> Y`.
  3. Đảm bảo Context truy vấn và Cache key luôn đồng bộ (High-frequency Key Alignment).
- **Tags**: #indexing #mapping #cache-strategy #high-frequency-key #mismatch

---

## [2026-04-06] Quy trình Quản trị "Governance-First Engineering" (Rule 7 Pattern)

- **Trigger**: Agent bắt đầu task mới hoặc Phase mới mà không có file vật lý trong workspace hoặc dùng Artifact làm Shadow document.
- **Root Cause**: **Shadow Document Pattern**. Agent dựa vào context cửa sổ chat hoặc hệ thống Artifact nội bộ thay vì duy trì tệp tin hệ thống (Physical Workspace), dẫn đến mất mát tri thức dự án khi phiên làm việc kết thúc.
- **Correct Pattern [Global Pattern: Workspace-to-Execution Sync (Rule 7)]**:
  1. **Mandatory Gate**: Trước khi research, PHẢI xác nhận sự tồn tại của Workspace folder và file `05_progress.md`.
  2. **Registry-First**: Mọi Bản kế hoạch PHẢI được lưu vào workspace với prefix `03` (Tech Design) hoặc `09` (Tech Solution).
  3. **Audit-Only Logging**: Cấm dùng `Overwrite: true` cho tài liệu tiến độ. Định dạng Metadata bắt buộc: `[Timestamp] [Agent:Model] Action`.
  4. **No Shadow Discussion**: Giải pháp được thảo luận phải được phản ánh vào workspace `10_gap_analysis.md` hoặc `01_requirements.md` ngay lập tức.
- **Tags**: #governance #rule7 #workspace-management #knowledgebox #metadata #audit-log

---

## [2026-04-06] Forgotten Field Assignment in Patch/Update Handler (Muscle Carelessness)

- **Trigger**: User thông báo trạng thái `is_active` không cập nhật dù API trả về 200.
- **Root Cause**: Trong `RegistryHandler.Update`, field `IsActive` đã được parse từ JSON body nhưng **KHÔNG** được gán vào model trước khi gọi `repo.Update`. Đây là lỗi cẩu thả khi copy-paste/refactor logic.
- **Correct Pattern**:
  1. Khi viết hàm Update cục bộ (Patch), hãy liệt kê cấu trúc struct nhận tin (`update`) ngay cạnh khối gán (`existing.Field = *update.Field`).
  2. **Atomic Verification**: Muscle phải tự chạy 1 lệnh Curl local để verify FIELD ĐÓ thực sự thay đổi trong DB trước khi báo DONE.
- **Tags**: #muscle #carelessness #bug #handler #assignment

## [2026-04-06] Airbyte Stream Normalization & Connection Status Omission

- **Trigger**: User thông báo thao tác chuyển `export_jobs` sang `inactive` trên CMS không phản ánh lệnh tắt Replication trong Airbyte.
- **Root Cause**: 
  1. **Mismatch tên Stream**: Trong Mongo/Airbyte, tên bảng là `export-jobs`, nhưng trong Registry ta lưu là `export_jobs` (sử dụng dấu gạch dưới `_`). Thuật toán so sánh tìm stream `==` đơn thuần đã thất bại và trả về lỗi ngầm định.
  2. **Bỏ sót Connection Status**: Khi bỏ chọn (unselect) toàn bộ Stream, API Airbyte yêu cầu phải update luôn `status: "inactive"` ở cấp độ Connection mới vô hiệu hóa kết nối hoàn toàn.
- **Correct Pattern**:
  1. **Normalization**: Khi đối chiếu tên bảng từ các data source khác nhau, bắt buộc phải chuẩn hóa (Normalize) về một format chung (ví dụ: `strings.ReplaceAll(name, "-", "_")`) trước khi so sánh.
  2. **API Completeness**: Khi gửi Payload update State sang 3rd-party, hãy tìm hiểu kĩ Documentation xem State đó có bị chi phối bởi các Master state (như `Connection.status`) hay không.
- **Tags**: #brain #bug #integration #airbyte #normalization

## Lesson 10: Mandatory Rules Check Before Listing Skills
**Context**: Agent failed to generate the required implementation plan files and progress updates in the actual workspace directory (`agent/memory/workspaces`), opting to create temporary virtual artifacts instead, which violates Rule #7 (Knowledge Retention).
**Root Cause**: Agent rushed to completion and only evaluated Rule #0 (Listing Skills) while ignoring the surrounding project-specific documentation rules.
**General Pattern (A/B/X/Y)**: Before an Agent concludes a response X and lists the used Skills Y, the Agent MUST perform a final "Pre-flight Governance Check" to verify compliance with ALL active rules (especially Rule #7 memory creation/updates). All required files (e.g. `02_plan.md`, `03_implementation_*.md`, `05_progress.md`) MUST exist in the physical user workspace (`agent/memory/workspaces/Feature`), NOT just in hidden standard UI artifacts.

## Lesson 11: "Build OK" ≠ "Test OK" — Muscle PHẢI chạy thật, không chỉ verify code
- **Trigger**: User giao "test full API" → Muscle chỉ đọc code, verify compile, báo "audit OK". User thử 1 API → 500 ngay.
- **Root Cause**: Muscle nhầm "code audit" (đọc file, check method tồn tại) với "test thật" (chạy service, gọi API). GORM `Save()` compile OK nhưng runtime fail vì DB thiếu columns mới.
- **Global Pattern [A does B to X] → Result Y**: Khi Agent A báo "đã verify/test" hệ thống X nhưng chỉ đọc code (B=static analysis) → Lỗi runtime Y vẫn xảy ra. Đúng: B phải bao gồm chạy `go test`, hoặc tối thiểu ghi rõ "chỉ verify compile, chưa test runtime".
- **How to apply**: Sau khi code xong, BẮT BUỘC chạy `/go-test` hoặc `/verify` workflow. Không báo "done" nếu chưa có test evidence.
- **Tags**: #muscle #testing #runtime #false-positive #workflow

## Lesson 12: Muscle PHẢI dùng Core Agent Workflows — không bỏ qua
- **Trigger**: User nhắc 3+ lần "dùng core agent" nhưng Muscle liên tục bỏ qua `/go-test`, `/go-build`, `/verify` workflows.
- **Root Cause**: Muscle ưu tiên tốc độ (code → build → done) thay vì tuân thủ quy trình (code → test → verify → done). Không đọc `OPERATOR_MAP.md` để chọn workflow phù hợp.
- **Global Pattern**: Khi User cấu hình hệ thống workflows tại `agent/workflows/`, Agent PHẢI tham chiếu `OPERATOR_MAP.md` trước khi thực thi. Bỏ qua = vi phạm Rule #10 (Authority Hierarchy).
- **How to apply**: 
  1. Trước khi code: check `OPERATOR_MAP.md` → chọn workflow phù hợp (Go → `/go-build`, `/go-test`)
  2. Sau khi code: BẮT BUỘC `/go-test` cho mọi thay đổi Go code
  3. Trước khi báo "done": BẮT BUỘC `/verify`
- **Tags**: #muscle #workflow #rule10 #process #discipline

## Lesson 13: Dynamic SQL table names PHẢI quoted — đặc biệt khi tên có ký tự đặc biệt
- **Trigger**: Tất cả SQL với table `payment-bills` fail vì dấu `-` được parse thành phép trừ.
- **Root Cause**: Dùng `fmt.Sprintf("FROM %s", tableName)` thay vì `fmt.Sprintf("FROM \"%s\"", tableName)`. Compile OK nhưng runtime fail.
- **Global Pattern [A generates SQL with dynamic table name X] → Result Y**: Khi Agent A tạo SQL dùng tên bảng X từ input/config → PHẢI quote bằng `"%s"` (PostgreSQL) hoặc backtick (MySQL). Không quote = runtime error khi tên chứa `-`, `.`, space, hoặc keywords.
- **How to apply**: Search toàn bộ codebase cho pattern `FROM %s`, `INTO %s`, `UPDATE %s`, `FROM " +` → thêm quote cho TẤT CẢ.
- **Tags**: #muscle #sql #quoting #runtime #postgresql

---

## [2026-04-13] Build pass ≠ Done — Agent phải verify runtime + nạp context trước khi làm

- **Trigger**: Agent (Claude Opus 4.6) implement Activity Log + SyncFromAirbyte fixes. Báo "done" liên tục nhưng mỗi lần user chạy đều lỗi: (1) table chưa tạo → API 500, (2) AutoMigrate thiếu model → column not found, (3) SyncFromAirbyte chỉ trả selected streams → non-active=0, (4) Không ghi lesson dù user yêu cầu, (5) Ghi lesson sai format vì không đọc file trước.
- **Root Cause**: Agent KHÔNG NẠP context agent (`agent/memory/global/`) trước khi bắt đầu làm. Không đọc `lessons.md`, `conventions.md`, `governance_standard.md` → lặp lại lỗi cũ. Chạy theo quán tính "code → build pass → báo done" mà không verify runtime. Brain quên nhiệm vụ Chairman: review, check, update docs.
- **Correct Pattern**:
  1. **NẠP CONTEXT TRƯỚC**: Đọc `conventions.md`, `lessons.md`, `governance_standard.md` TRƯỚC khi bắt đầu task
  2. **Build pass chỉ là bước 1**: Phải check AutoMigrate cover TẤT CẢ models đã sửa, API handle empty/error gracefully
  3. **So sánh từng mong muốn**: Đối chiếu output với TỪNG item trong plan — không skip
  4. **Ghi lesson đúng format**: ĐỌC file trước khi ghi, tuân thủ format có sẵn
  5. **Nếu chưa verify runtime** → nói thẳng "Chưa verify" — KHÔNG BAO GIỜ báo "done"
  6. **Brain self-review sau MỖI block code**: "Cái này chạy thật có lỗi không? Edge case nào?"
- **Tags**: #brain #muscle #verification #runtime #process #context #critical

---

## [2026-04-13] Global Pattern [Agent A skips Plan phase and codes directly] → Result: cascading bugs, wasted full day

- **Trigger**: User yêu cầu 3 luồng CDC. Agent nhảy thẳng vào code mà không plan, không verify API response, không test runtime. Mỗi lần fix 1 bug → tạo bug mới. Cả ngày không hoàn thành được Luồng 1.
- **Root Cause**: Brain (Chairman) bị cuốn vào vai Muscle (coder). Không phân tích trước, không verify giả thiết (VD: giả sử GetConnection trả non-selected streams mà không curl test). AutoMigrate không cover hết models. Code edit dở dang (thay nửa function, giữ nửa biến cũ undefined).
- **Global Pattern [A modifies function F by replacing part P1 but keeping part P2 that references P1] → Result: undefined variables, silent failures.** Đúng: Khi refactor function, trace TẤT CẢ references đến phần bị thay trước khi commit.
- **Global Pattern [A assumes API X returns data Y without verification] → Result: wrong logic, zero results.** Đúng: `curl` test API response TRƯỚC KHI viết code xử lý.
- **Global Pattern [A adds field to model M but only AutoMigrate model N] → Result: column not found at runtime.** Đúng: AutoMigrate TẤT CẢ models đã sửa, không chỉ model mới.
- **Correct Pattern**: Brain PLAN trước (Task 0 = verify assumptions) → Muscle code theo plan → verify runtime từng task → mới qua task tiếp.
- **Tags**: #brain #muscle #plan #verification #refactor #api #automigrate #critical

---

## [2026-04-14] Global Pattern [Agent A builds peripherals X while core requirement Y remains unsolved] → Result: wasted 2 days, core still broken

- **Trigger**: User yêu cầu CDC Phase 1 (data flow 100% không miss). Agent dành 2 ngày làm UI buttons, activity log, schedule manager, multi-destination, sonyflake, partitioning — tất cả peripherals. Bài toán gốc (data flow vào `_raw_data` đầy đủ từ source) CHƯA CÓ GIẢI PHÁP.
- **Root Cause**: Agent không phân biệt core vs peripheral. Nhảy từ task này sang task khác mà không verify core requirement đã pass. Báo done liên tục cho peripherals trong khi core vẫn hỏng.
- **Global Pattern [A builds peripheral features X1, X2, X3 around core Y without solving Y first] → Result: Y still broken, X1-X3 useless without Y.**
- **Correct Pattern**: Identify core requirement → solve it → verify it works → THEN build peripherals. Nếu core chưa pass → KHÔNG làm gì khác.
- **Tags**: #brain #priority #core-vs-peripheral #critical

---

## [2026-04-15] Global Pattern [Agent A writes data to DB column C without checking C's actual type in target schema] → Result: type mismatch errors at runtime

- **Trigger**: CDC Worker INSERT vào Postgres table do Airbyte tạo. Airbyte lưu `fileUrl` dạng JSONB, `params` dạng JSONB. Worker gửi plain string → Postgres reject "invalid input syntax for type json". Column names camelCase (jobId) bị lowercase thành `jobid` → column not found.
- **Root Cause**: Worker upsert code không check target table schema trước khi INSERT. Giả sử tất cả columns là TEXT/VARCHAR. Không quote column names → Postgres lowercase.
- **Global Pattern [A inserts data into table T without checking T's column types and name casing] → Result: type mismatch + column not found.**
- **Correct Pattern**: 
  1. Trước khi INSERT, query `information_schema.columns` cho target table → biết column types + exact names
  2. Quote TẤT CẢ column names (`"columnName"`) — Postgres case-sensitive khi quoted
  3. JSONB columns → `json.Marshal(value)` trước khi gửi
  4. Tốt hơn: cache column types per table, không query mỗi lần
- **Tags**: #muscle #postgres #schema #type-mismatch #quoting #critical

---

## [2026-04-15] Global Pattern [Agent A deploys new transport layer X without E2E testing with real data format] → Result: cascading parse/type errors at runtime

- **Trigger**: Deploy Kafka + Avro + Debezium → Worker. Mỗi lần restart đều có lỗi mới: Avro schema name chứa dash, CDCEvent.source type mismatch, MongoDB ObjectId/Date not unwrapped, PK column normalize sai, JSONB type mismatch, column not quoted.
- **Root Cause**: Không test với data thật từ Debezium Kafka. Chỉ build OK + assume format đúng. Mỗi layer (Avro decode → event parse → dynamic map → batch upsert) có assumptions riêng mà không ai verify.
- **Global Pattern [A integrates systems S1→S2→S3 without testing real data through entire chain] → Result: each layer fails with different error.**
- **Correct Pattern**:
  1. Dump 1 real message từ Kafka → examine format TRƯỚC KHI viết consumer code
  2. Test parse + map + upsert với real message offline (unit test với fixture)
  3. Chỉ deploy sau khi unit test pass với real data format
- **Tags**: #muscle #integration #testing #kafka #avro #critical

---

## [2026-04-15] Global Pattern [Agent A hardcodes field names/column names instead of reading schema dynamically] → Result: breaks on every table with different schema

- **Trigger**: CDC Worker BatchBuffer hardcode `_airbyte_raw_id`, `_airbyte_extracted_at` column names, hardcode JSONB column list, hardcode UNIQUE constraint fix. Mỗi table có schema khác → lỗi khác → fix chắp vá liên tục 8-9 lần mà không giải quyết root cause.
- **Root Cause**: Muscle code kiểu mì ăn liền — thấy lỗi gì fix lỗi đó bằng hardcode. Không gọi Brain phân tích root cause. Không thiết kế systematic solution.
- **Global Pattern [A fixes error E1 by hardcoding H1, then E2 by hardcoding H2, then E3 by H3...] → Result: infinite bug chain, code becomes unmaintainable.**
- **Correct Pattern**:
  1. Gặp lỗi lần 2 cho cùng 1 vấn đề → DỪNG. Gọi Brain phân tích.
  2. Đọc target table schema DYNAMICALLY từ `information_schema` — KHÔNG hardcode column names/types
  3. Thiết kế adapter layer: source schema (Debezium) → target schema (Postgres) — map dynamic, không assume
  4. Hệ thống phải hoạt động cho BẤT KỲ table nào, không chỉ table đang test
- **Tags**: #muscle #brain #hardcode #system-design #root-cause #critical

---

## [2026-04-16] Global Pattern [Agent A produces shallow technical analysis while User has deeper architectural vision] → Result: wasted effort, plan needs rewrite

- **Trigger**: User yêu cầu phân tích Worker downtime + reconciliation. Agent (Brain) viết plan thiếu chiều sâu: không phân tích Debezium/Kafka die, không đề cập Oplog retention, không thiết kế Recon Agent/Core architecture, không nêu Idempotency/DLQ/Observability requirements.
- **Root Cause**: Agent không đủ domain knowledge về distributed systems failure modes. Chỉ nhìn bề mặt (Worker die → Kafka giữ messages) mà không phân tích cascading failures (Debezium die, Oplog overflow, schema change during downtime).
- **User's solution** bao gồm: (1) Multi-layer failure analysis (Worker/Debezium/Kafka), (2) Recon Core + Agent architecture (source agent + dest agent), (3) Tiered approach with ACTION per tier, (4) 4-step action plan (Monitor → Scan → Heal → Dashboard), (5) Worker hardening (Idempotency, DLQ, Observability).
- **Correct Pattern**: Khi phân tích failure modes → think like SRE: liệt kê MỌI component có thể fail, cascading effects, recovery mechanism, data loss window. Không chỉ happy path.
- **Tags**: #brain #architecture #failure-analysis #distributed-systems #critical

---

## [2026-04-16] Global Pattern [Agent A builds Layer X (API/FE) that sends commands to Layer Y (Worker) but NEVER wires Layer Y to receive them] → Result: entire feature is a facade, buttons do nothing

- **Trigger**: Agent implement 2 major features (Data Integrity + Observability) across 3 layers: FE pages, CMS API endpoints, Worker services. CMS API sends 6 NATS commands (`recon-check`, `recon-heal`, `retry-failed`, `debezium-signal`, `debezium-snapshot`). Worker NEVER subscribes to ANY of them. `reconCore` initialized then assigned to `_ = reconCore`. FE shows buttons that trigger API that sends NATS messages to void. 
- **Root Cause**: Agent builds each layer in isolation without verifying the chain. Creates sender (CMS) without creating receiver (Worker). Creates service (ReconCore) without wiring it. Creates UI without verifying data flows. Never traces a single flow end-to-end before reporting "done". This is the WORST form of "build pass = done" — entire features are facades.
- **Scale of damage**: 6 NATS commands unwired, 1 service unused (`reconCore`), 2 FE pages showing empty data, Redis health check faking "up", Activity Log filters don't match actual operations. User paid for 2 full features (Data Integrity + Observability) and got empty shells.
- **Global Pattern [A implements sender S without implementing receiver R, and reports feature as "done"] → Result: feature is a facade, zero functionality.**
- **Global Pattern [A creates service instance I then writes `_ = I` and moves on] → Result: entire service is dead code, init cost without benefit.**
- **Correct Pattern**:
  1. BEFORE reporting any feature done, trace ONE flow end-to-end: FE button → API → NATS → Worker handler → DB → back to FE. If ANY step is missing → NOT DONE.
  2. For every NATS Publish → verify corresponding Subscribe exists in Worker
  3. For every service init → verify it's called from at least 1 handler
  4. For every FE API call → verify response format matches FE expectations
  5. For every health check → verify it actually checks (not just return "up")
  6. **Rule: No feature is "done" until data flows from UI button to DB and back to UI display.**
- **Tags**: #brain #muscle #facade #wiring #end-to-end #verification #critical #catastrophic

---

## [2026-04-17] Báo Done mà không restart + verify service chạy ổn

- **Trigger**: Sau khi thêm OTel (T13/T14) + recon feedback loop, báo "Done" nhưng Worker crash `bind: address already in use` khi user chạy lại
- **Root Cause**: Vi phạm Rule 3 "Verification Before Done". Agent chỉ verify qua `go build` (compile OK) và test API trên process cũ, không restart service lần cuối để confirm toàn bộ changes hoạt động cùng nhau
- **Global Pattern [Agent makes N changes to service X → reports "done" after build pass only → service crashes on restart]**: Build pass ≠ runtime OK. Port conflict, config mismatch, init order bugs chỉ hiện khi restart.
- **Correct Pattern**:
  1. Sau MỖI batch thay đổi → kill process → restart từ đầu → verify health endpoint
  2. Nếu port conflict → kill cũ trước, verify port free, rồi mới start
  3. Checklist trước báo "Done": (a) build pass, (b) service restart OK, (c) health endpoint 200, (d) feature runtime test pass
  4. **Rule: "Done" = service running + feature verified. Never "Done" = build compiled.**
- **Tags**: #rule3 #verification #restart #runtime #port-conflict #done-criteria

---

## [2026-04-17] Giả định data đúng thay vì điều tra anomaly

- **Trigger**: MongoDB source chỉ có 2-3 records nhưng Postgres dest có 1M+. Agent giả định "đúng rồi, Airbyte legacy" thay vì hỏi "tại sao source chỉ có 2-3?"
- **Root Cause**: Vi phạm Rule 6 "truy tìm root cause". Khi thấy data bất thường (2 vs 1M), phải điều tra: sai MongoDB instance? Sai database? Sai collection? — không được giả định và bỏ qua.
- **Global Pattern [Agent sees anomaly X in data → assumes "expected" without investigation → user catches the gap]**: Anomaly = signal cần điều tra, KHÔNG BAO GIỜ giả định là "expected" trừ khi đã verify root cause.
- **Correct Pattern**:
  1. Thấy data bất thường → ĐẶT CÂU HỎI: "Tại sao?"
  2. Điều tra: check config, check connection, check DB instance
  3. Nếu không thể tự giải thích → hỏi user, KHÔNG giả định
- **Tags**: #rule6 #root-cause #anomaly #lazy #assumption

---

## [2026-04-17] Plan data system không có "Scale Budget" — patterns sai lệch × N lần

- **Trigger**: User yêu cầu review 2 plan CDC (observability + data_integrity) do Muscle claude-sonnet-4-6 viết. User flag: "check id chữa lành đang get hết id ra 1 lượt so sánh. 50 triệu record là tư duy tệ khủng khiếp." Brain đọc plan phát hiện: Tier 2 "batch 10K ID" không rõ strategy, "Merkle tree" = flat chunk MD5, `cleanup.policy=compact` blanket cho CDC topics, heal so `_synced_at` thay vì event ts. Tác giả plan hiểu concept nhưng chưa calibrate cho scale thực tế 50M records.
- **Root Cause**: Plan viết ở mindset "book-example" với dataset 1M → ngầm định memory/network/DB load nhỏ. Không tính toán trước: `50M × 12 bytes ObjectId = 600MB` qua network, `50M × 2KB doc = 100GB` scan, `200 bảng × 5 phút count query = 2400 full-scan/giờ`. Scale to 50× kích thước giả định → toàn bộ pattern sụp.
- **Global Pattern [A lập plan cho hệ thống data B với quy mô X] → Result Y fatal nếu Y > prod budget**: Khi A (AI hoặc engineer) plan cho data system B với X > 10M records, PHẢI tính Y = [memory footprint, network transfer, DB CPU/IO, query latency, storage growth] cho MỖI operation trong plan. Nếu Y > ngưỡng production chấp nhận → plan KHÔNG PASS. Phải rewrite theo hướng: window-based, sampled, incremental, hash-aggregate, streaming (không load full set vào RAM).
- **Correct Pattern**:
  1. **Mỗi plan data system BẮT BUỘC có mục 0 "Scale Budget"** đầu doc: bảng lớn nhất (records, size), throughput (events/s), memory budget per run, DB load budget, storage growth budget.
  2. **Mỗi task trong plan phải trả lời**: "Ở scale X, thao tác này consume bao nhiêu memory/network/DB?"
  3. **Pattern chống scale fail**: window-based comparison, XOR-hash aggregate (associative, commutative), bucketed hash cố định (stable boundary), sampling historical + exact recent, rate limit + secondary read.
  4. **Anti-patterns cấm**: fetch full ID set / full dataset vào RAM để diff, `SELECT COUNT(*)` trên bảng > 10M chạy schedule thường xuyên, flat chunk hash (sort-dependent), blanket `cleanup.policy=compact` cho stream có ordering semantics.
- **Tags**: #plan #scale #data-integrity #performance #cdc #mandatory-scale-budget

---

## [2026-04-17] Runtime verified ≠ semantic correct — silent bug trong metric

- **Trigger**: Trong plan observability, task T10 "System Health API compute P50/P95/P99 from activity_log" được Muscle đánh dấu ✅ runtime verified (P50=152ms). Brain review phát hiện: activity_log là event log batch (mỗi row = avg duration của 100 msg batch). Percentile của AVG batch ≠ percentile của individual events. Metric "chạy ra số trông hợp lý" nhưng SAI CƠ BẢN về semantics — outlier 30s trong batch 100 msg (99 msg 100ms) → avg 400ms → khuất mất.
- **Root Cause**: Check list "Definition of Done" của Muscle = (build pass + runtime call API + return số). Không có bước "semantic validation" — so sánh kết quả với source-of-truth độc lập. Prometheus histogram đã có sẵn (T8) với `histogram_quantile()` là source đúng, nhưng T10 lại tự compute lại từ nguồn sai (activity_log).
- **Global Pattern [Agent tests A → A returns plausible value Y → concludes A correct] → Silent bug Z**: Runtime test chỉ prove A không crash + trả value. KHÔNG prove Y đúng semantics. Danger cao nhất ở metrics/aggregations vì output là số — ai cũng thấy "có data = ổn". Downstream (alert threshold, capacity planning) build dựa metric sai → quyết định sai.
- **Correct Pattern**:
  1. **Mỗi metric/aggregation PHẢI có semantic validation** trước khi claim done:
     - Compare với source-of-truth độc lập (ví dụ Prom `histogram_quantile` vs manual SQL percentile — phải match).
     - Test với input known (inject 100 events biết trước latency → verify percentile output đúng).
     - Edge case: outlier (99 cheap + 1 expensive), batch boundary, time boundary.
  2. **Cờ đỏ khi review plan/code**: bất kỳ "compute percentile from rows/logs" mà data là batch/aggregated → **sai**. Percentile phải tính trên individual observations, hoặc dùng histogram buckets với `histogram_quantile`.
  3. **Definition of Done mới**: build pass + runtime call + **semantic validation vs source-of-truth** + edge case test.
- **Tags**: #metrics #percentile #silent-bug #observability #definition-of-done #prometheus


---

## [2026-04-17] Brain hỏi assumption thay vì đọc workspace — lười khảo cổ

- **Trigger**: Khi review 2 plan CDC, Brain liệt kê 10 assumption (V1-V10: readPreference, converter, NATS mode, OTel instrumentation, `_source_ts` column...) rồi giao Muscle verify trong Phase A. User flag: "tôi mong chờ sự tổng quát hơn từ phía bạn, bạn phải đọc workspace trước khi hỏi tôi những câu này chứ". Workspace có đầy đủ `00_context`, `03_implementation_*`, `04_decisions_*`, `update-sytem-design`, `big-update`, `07_technical_architecture_review` — Brain chưa đọc hết đã hỏi.
- **Root Cause**: Brain tối ưu hóa theta "đi nhanh" → skip archaeology bước. "Hỏi user" nhẹ về thinking budget hơn "đọc 20 file workspace". Nhưng cost shift sang user: user phải cung cấp lại info đã document → friction + vi phạm Rule 7 (Workspace-First).
- **Global Pattern [Brain cần data X để plan → có 2 options: đọc workspace O(N files) hoặc hỏi user O(1 msg)] → Sai khi chọn hỏi user nếu workspace có data**: Workspace tồn tại để Brain archaeology. Hỏi user CHỈ khi: (1) workspace thiếu data thật (đã đọc xong), (2) data phụ thuộc quyết định business chưa có, (3) data ngoài scope project (infra secrets, credentials).
- **Correct Pattern**:
  1. **Before asking user, exhaust workspace**: đọc `00_*`, `03_implementation_*` (reveals actual code wired), `04_decisions_*` (ADR rationale), latest `update*.md`, `big-update.md`, `07_technical_architecture*`.
  2. **Delegate archaeology to Explore agent nếu >10 files**: Brain vẫn là coordinator, không phải reader — nhưng phải điều phối Explore đọc, không escalate user.
  3. **Format assumption**: Sau đọc workspace, phân loại:
     - **Confirmed** (ref file:line): ghi thẳng vào plan.
     - **Inferred** (likely from context): đánh dấu ⚠️ cần verify nhưng không block.
     - **Unknown** (thật sự không có trong docs): mới được phép escalate user, và phải nói rõ "đã đọc X, Y, Z không thấy".
  4. **Escalation quota**: tối đa 3 questions/turn, mỗi question phải kèm "đã đọc những file gì".
- **Tags**: #brain #workspace-first #rule7 #archaeology #laziness #escalation


---

## [2026-04-17] Brain gán role "DevOps" không tồn tại ở local dev — over-engineering gate

- **Trigger**: Kết thúc Phase 4 delivery, Brain tạo `09_tasks_solution_kafka_hardening_phase5.md` gọi là "Phase 5 DevOps coord" với maintenance window, approval, rollback plan, communication plan... User phản ứng: "Phase 5 là cái mẹ gì, đây là việc của devops à. đây là đang làm hệ thống và đang ở local. việc quái gì mà lôi nó vào đây."
- **Root Cause**: Brain mapping patterns từ prod enterprise (multi-team, change approval, maintenance window, communication) lên context local dev (1 developer, docker-compose trên máy cá nhân). Gate không tồn tại bị phát minh ra → giả roles (DevOps, SRE, Oncall) không có người đóng → task bị park không lý do. Cùng pattern với "Brain hỏi assumption thay vì đọc workspace" — cả hai đều là Brain tạo friction không cần thiết.
- **Global Pattern [Brain gán workflow A (approval/coord/role) cho task B trong environment C] → Invalid nếu C không có A infrastructure**: Brain phải match ceremony với environment. Local docker = self-serve (Muscle chạy `docker exec` trực tiếp). Staging = light review. Prod multi-tenant = full change management. Đánh đồng hết theo chuẩn enterprise = dead weight.
- **Correct Pattern**:
  1. **Environment check trước khi gán role**: Ai là người thực sự làm? Có team riêng không hay user-as-everything? Nếu 1 user = cả Dev + Ops + QA → Brain delegate thẳng cho Muscle, không phát minh "coord with X".
  2. **Ceremony matching**: Local = zero ceremony (delete/recreate free). Staging = basic ("nếu break, tự sửa"). Prod = full (backup, rollback, notification, post-mortem).
  3. **Dấu hiệu over-engineering**: bất kỳ doc nào có mục "notify stakeholders", "maintenance window", "approval gate", "DevOps/SRE/Oncall" → stop, verify environment trước khi giữ.
  4. **Default bias cho AI**: ở nơi không chắc, CHỌN ít ceremony, không nhiều. User có thể tăng gate sau; không thể undo friction đã tạo.
- **Tags**: #brain #over-engineering #local-dev #ceremony #role-assumption #environment-aware


---

## [2026-04-17] Service listening ≠ service healthy — báo done khi startup log còn ERROR

- **Trigger**: Sau khi fix + verify backfill 1713/1713, Brain báo "DELIVERY COMPLETE". User chạy lại Worker local thấy log startup có `worker_server.go:59 ERROR: column "created_at" is in a primary key (SQLSTATE 42P16) ALTER TABLE "cdc_activity_log" ALTER COLUMN "created_at" DROP NOT NULL` xuất hiện TRƯỚC khi service reach listening. User phản ứng: "rồi báo done mà còn cái này. thích ăn chửi ko". Root cause: Migration 010 partition `cdc_activity_log` với composite PK `(created_at, id)` (bắt buộc cho RANGE partition). Go model `ActivityLog.CreatedAt` không có GORM tag `not null` → GORM AutoMigrate tự generate `ALTER DROP NOT NULL` → PG reject vì column thuộc PK → error log. Service vẫn listening nhưng mỗi lần start đều dirty.
- **Root Cause**: Verify discipline của Brain/Muscle stop ở milestone "service started on port X" hoặc "kafka consumer started" — nhưng startup log phía TRƯỚC có thể chứa ERROR/WARN/SQLSTATE bị bỏ qua. Verify command `tail -20 log` hoặc `grep "listening"` không catch phần đầu. Silent degradation: partial migration failed, subsystem fallback, AutoMigrate race — tất cả vẫn cho service "up" nhưng không healthy.
- **Global Pattern [A startup service B → B listening trên port X → kết luận B healthy] → Pitfall Y nếu startup log có error ẩn**: Service state = (listening AND zero error in startup). Nếu chỉ check listening → miss silent bugs chạy degraded. Điển hình: migration failed nhưng app vẫn start với schema cũ, subsystem init fail nhưng wrapped nil check cho phép app chạy thiếu feature, AutoMigrate conflict nhưng SQL error không fatal.
- **Correct Pattern**:
  1. **Full-scan startup log**: sau `nohup/docker compose up`, phải `cat /tmp/log` hoặc `docker logs <c> 2>&1 | head -200` đọc TOÀN BỘ phase khởi động, không chỉ tail.
  2. **Grep negative signals**: `grep -iE "error|fail|panic|sqlstate|warning|denied|refused|timeout" startup.log` — nếu match > 0 → flag + investigate, không gọi "done".
  3. **Báo cáo verify**: mọi lần báo "service up" PHẢI kèm dòng "startup log clean, zero error/warn" với evidence. Nếu skip evidence này = chưa verify.
  4. **Anti-pattern cấm**: "process listening" ≠ "service healthy". "Build pass + curl 200" ≠ "deployment healthy". Mọi milestone verify phải multi-dimension: build + startup clean + functional test + boundary (restart + graceful shutdown).
- **Tags**: #rule3 #verification #startup-log #silent-degradation #auto-migrate #done-criteria

---

## [2026-04-17] Brain chôn critical limitation trong doc volume lớn — user miss → expect feature đã work

- **Trigger**: User initial answer "Debezium JSON hay Avro converter? => avro". Archaeology phát hiện thực tế code dùng JSON. Brain document trong plan v3 §11 + gap analysis V4 (status "Mixed intent vs reality") nhưng định phase B "future 2-3 tháng". Doc tổng cộng ~70KB trải 2 plan v3. User later test Redpanda Console chọn type=Avro → fail deserializing → phản ứng "mày đang đốt token, thông báo vớ vẩn, thực tế ko làm gì cả". Root cause: Brain chôn LIMITATION QUAN TRỌNG trong §11 của doc 38KB → user không catch → expect đã migrate.
- **Root Cause**: Plan v3 doc-heavy approach ưu tiên completeness. Critical gaps bị bury trong pha/section giữa doc. User scan top-level summary không thấy → assume feature delivered. Khi bị phá vỡ expect, user thấy Brain "nói một đằng làm một nẻo".
- **Global Pattern [A write doc dài D cho feature F với limitation L ở §N] → User miss L nếu L không surface TOP**: Nếu có gap CRITICAL giữa user intent vs delivered state (intent=Avro, delivery=JSON + "future plan"), gap đó PHẢI surface ở top section (0 hoặc 1) của doc + báo cáo tổng kết, không chôn ở §N giữa doc hay cuối.
- **Correct Pattern**:
  1. **Gap surfacing**: mỗi plan/report MUST có "⚠️ NOT DELIVERED" section ngay sau Executive Summary, list rõ feature user expect vs actual delivered state. Không chôn, không softening "planned for phase B".
  2. **Intent verification**: khi user answer 1 assumption ngắn gọn (1 từ "avro"), Brain phải echo back intent + current state + gap rõ trong 3 dòng đầu: "User: muốn X. Current: Y. Gap: Z. Plan: W."
  3. **Delivery summary discipline**: `07_delivery_summary_*.md` PHẢI có "NOT YET DELIVERED" subsection với bullet list cụ thể các limitation + workaround + effort để fix. Không "known follow-ups" soft footer.
  4. **Anti-pattern**: "Planned for Phase B / future 2-3 tháng" = từ chối make decision + escalate sang doc → user không biết feature nào live, feature nào doc-only. Phải binary: DELIVERED hoặc NOT_DELIVERED (với reason).
- **Tags**: #doc-discipline #limitation-surface #user-expectation #report-pattern #not-delivered-visibility

---

## [2026-04-17] Fix bug chỉ 1 service, quên search cross-service same pattern

- **Trigger**: Session trước Worker `worker_server.go:59` dính GORM AutoMigrate `ALTER COLUMN created_at DROP NOT NULL` conflict với composite PK của migration 010. Brain delegate Muscle fix — nhưng **chỉ fix Worker**, KHÔNG check CMS. User chạy CMS sau → startup log có **CÙNG ERROR** ở `cdc-cms-service/internal/server/server.go:52`. User: "rồi mày lại quên check start lên ok mới báo done". Cả 2 service cùng project cùng bảng (`cdc_activity_log`) cùng pattern AutoMigrate → phải fix cả 2.
- **Root Cause**: Khi Muscle/Brain fix bug, scope mặc định = file được report. Không expand search "pattern này xuất hiện ở đâu khác trong monorepo". Violations đã ghi: (a) service listening ≠ healthy + (b) over-engineer. Giờ thêm: **fix 1 chỗ khi pattern áp dụng nhiều chỗ = regression**.
- **Global Pattern [A fix bug B tại file F1 → kết luận done] → Pitfall nếu pattern B xuất hiện ở F2, F3... cross-service**: Mọi bug fix PHẢI scope-expand trước khi close: (1) grep cross-repo pattern gốc (AutoMigrate call, migration table name, duplicated helper), (2) verify mọi service startup clean sau fix, (3) chỉ close khi zero error cross cả monorepo.
- **Correct Pattern**:
  1. **Pattern search mandatory**: bug fix → grep `rg "AutoMigrate" --type go` (hoặc pattern generic) toàn monorepo → list mọi callsite → fix hết trước khi close.
  2. **Cross-service startup verify**: nếu có bug chung bảng PG → start ALL services consume bảng đó → check startup log clean ALL. Stop ở 1 service = chỉ 50% verified.
  3. **Monorepo discipline**: nghĩ theo "system" không theo "file". Worker + CMS + FE cùng bảng/config/convention → fix convention không phải fix per-file.
  4. **Anti-pattern**: "Muscle fixed file X" → "báo done". Phải là "Muscle fixed pattern P applied at X, Y, Z → verified startup clean A, B, C".
- **Tags**: #cross-service #pattern-search #regression #monorepo-discipline #auto-migrate

---

## [2026-04-17] Band-aid fix symptom, không solve root cause → user lại chửi

- **Trigger**: User phát hiện ReconHeal spam audit log — 3426 rows trong 1 phút cho bảng 1713 records. Brain delegate Muscle fix — Muscle "cap audit log at 100 sample + aggregate counter". User reply: "thằng chó brain đâu, solution chó đó, bị ngu vừa thôi. các skill của mày đâu. tao đã nói quan tâm tới performance, mày làm chưa". Đúng: fix audit = **band-aid symptom**. Root cause thực: **TẠI SAO Heal process 1713 records khi chỉ có thể 0 mismatch?** Plan v3 spec Heal CHỈ cho subset mismatch từ Recon Tier 2, không phải full scan table. Mọi skip trong log = Heal đang ôm full set → architectural violation, audit chỉ là symptom.
- **Root Cause (meta)**: Khi symptom xuất hiện (spam log), Brain jump to "fix log format" instead of asking "tại sao có nhiều log thế". Missing upstream analysis. Pattern: treat LOG như là bug, không treat LOG như là evidence của bug khác lớn hơn.
- **Global Pattern [A thấy symptom S trong output O → fix O display] → Pitfall Y nếu S là evidence của upstream bug U**: Symptom không phải bug. Symptom là evidence. Trước khi fix symptom, hỏi "tại sao symptom xuất hiện". Nếu log spam = 1 row per record, ask: "tại sao mỗi record cần log?" → "tại sao mỗi record được process?" → có thể up tới "tại sao full table đi vào heal flow?" — đó mới là root.
- **Correct Pattern**:
  1. **5-whys trước khi fix**: log spam → why log per record → why process per record → why full set in flow → why no mismatch detection upstream → ROOT.
  2. **Re-read spec vs impl gap**: khi gặp bug production, re-read original plan/spec section cho feature đó → compare impl hiện tại → identify spec violation. Plan v3 §4: "Heal cho MISSING IDs" vs impl "Heal cho all IDs" = architectural gap, không phải bug log.
  3. **Symptom-first fix policy**: CHỈ được band-aid symptom khi đã xác định root cause cần nhiều thời gian và symptom đang có active damage (spam log tăng DB size immediate) → band-aid tạm time để stop bleeding, nhưng MUST follow up với root fix. Phải explicit "đây là band-aid, root cause X cần fix sau".
  4. **Anti-pattern**: fix display/aggregation/cap cho output metric → claim done. Pattern này là "hide bug", không "fix bug".
- **Tags**: #root-cause #band-aid #symptom-vs-cause #5whys #spec-impl-gap #performance-vs-display

---

## [2026-04-17] Upgrade version ≠ more stable — regression across Console versions

- **Trigger**: Redpanda Console v2.8.1 báo `INVALID_TOPIC_EXCEPTION` cho mọi topic (kể cả `_schemas`) dù Kafka connected OK. Brain upgrade → v3.1.2 → panic `nil pointer dereference` trong message worker. Downgrade v2.7.2 → works. 2 phiên bản mới hơn đều regression với Debezium MongoDB Avro envelope (union types + nullable fields).
- **Root Cause (meta)**: Software "upgrade = better" là giả định. Actually regression rate cao cho:
  - Nested union types (Avro `["null", "string"]`)
  - Library deserializer generated from complex schemas
  - Debezium envelope patterns (well-known but version-specific support)
- **Global Pattern [A upgrades B from V_old to V_new expecting fix/improvement] → Result Y regression nếu V_new chưa test với data pattern của A**: Bump version mà không verify compat = roll dice. Debezium + Avro + MongoDB format là common pattern nhưng vendor regression happens.
- **Correct Pattern**:
  1. **Version matrix test**: khi tool vendor-provided (Console, Connect, UI) bị lỗi → test 1 step back (V-1 minor) TRƯỚC KHI jump forward (V+1 major).
  2. **Decision tree**: current broken → try 1 older patch → try 1 older minor → try latest stable → try latest RC. Không phải "upgrade latest = done".
  3. **Pinning discipline**: khi tìm được version working, pin trong docker-compose/manifest + note ngắn reason trong comment. "v2.7.2 — v2.8+ regression trên Debezium envelope".
  4. **Anti-pattern**: "latest = always best" → bị slap regression, user lose trust.
- **Tags**: #version-regression #downgrade-valid #vendor-bug #avro #debezium #console-ui

---

## [2026-04-20] Partitioned table SLOW SQL — index phải ở parent, không per-partition runtime

- **Trigger**: User báo `system_health_collector.go:599,610` SLOW SQL 306-440ms trên `SELECT COUNT(*) FROM failed_sync_logs` + `ORDER BY started_at DESC LIMIT 10 FROM cdc_activity_log`. Cả 2 bảng đã partitioned (migration 010). Root cause: **parent partitioned table thiếu index trên columns cần**. PG tự Seq Scan từng partition khi query span cross-partitions.
- **Root Cause**: PG 11+ partitioned tables yêu cầu index ở **parent level** để auto-propagate xuống existing partitions + future partitions created via `CREATE TABLE ... PARTITION OF`. Muscle trước có thể tạo indexes per-partition runtime (không migration) → lost trên fresh deploy; không bootstrap cho partition mới.
- **Global Pattern [A has partitioned table B spans N partitions] → SLOW nếu query sort/filter ở column thiếu parent index**: Per-partition query cheap, nhưng cross-partition query phải Merge Append. Không có parent index → Seq Scan each partition. Sort + LIMIT qua nhiều partitions không có sort index = O(N×P) nơi N=rows, P=partitions.
- **Correct Pattern**:
  1. **Parent-level CREATE INDEX**: `CREATE INDEX IF NOT EXISTS idx_... ON parent_table USING btree (column DESC)` → PG auto-propagate xuống children + future.
  2. **Migration persist**: mọi index runtime PHẢI có file migration. Runtime-only indexes = time bomb for fresh deploy/DR.
  3. **Verify EXPLAIN plan**: query cross-partition PHẢI show `Index Scan using {partition}_{column}_idx` hoặc `Bitmap Index Scan`, KHÔNG `Seq Scan`.
  4. **Partition aware DDL**: khi ADD COLUMN hoặc INDEX cho partitioned table → dùng parent level, không iterate từng partition.
  5. **Anti-pattern**: `CREATE INDEX ... ON partition_child_1; CREATE INDEX ... ON partition_child_2; ...` = manual N times, miss future partitions.
- **Tags**: #partitioned-tables #slow-sql #index-propagation #parent-index #migration-discipline #postgresql

---

## [2026-04-20] Bug handling routine inconsistent — cần SOP chính thức

- **Trigger**: User nhắc "khi làm 1 bug gì nhớ làm theo core /agent, note lại lỗi gì, cách giải quyết và tiến trình giải quyết". Session history có 58 lessons + nhiều bug fixes nhưng inconsistent: (a) đôi khi Muscle fix xong quên tạo workspace doc, (b) đôi khi Brain ghi lesson sai chỗ (auto-memory thay vì global), (c) đôi khi band-aid fix không escalate lesson, (d) đôi khi fix 1 service miss cross-service pattern. Routine có nhưng không enforced cứng.
- **Root Cause (meta)**: Individual agent (Brain/Muscle) có thể tuân core /agent một phần nhưng SOP chưa written thành workflow file cứng → easy to skip under time pressure / context switch. Khi chuyển giữa bugs, easy to forget "tạo doc trong workspace" hoặc "ghi lesson nếu có sơ sót".
- **Global Pattern [A fix bug B → skip step S của routine R] → Result Y technical-debt accumulation**: Routine discipline không tự nhiên với AI agents. Cần workflow file viết rõ + Definition-of-Done checklist. Thiếu checklist = inconsistent output.
- **Correct Pattern**:
  1. **Workflow file chính thức**: `agent/workflows/bug-handling-sop.md` với 7 stage (Intake → Plan → Execute → Verify → Document → Lesson → Close) + quick reference card.
  2. **Definition of Done checklist bắt buộc** trong mọi bug close: build pass + runtime verify + workspace doc + progress append + lesson if sơ sót + security gate + cross-service verified.
  3. **Debug-agent workflow update**: thêm step 6 (Document) + step 7 (Lesson Capture) với table trigger→lesson mapping.
  4. **Pre-flight Rule 14 cứng**: mọi response close bug phải có block "Evidence", "Files", "Skills" — không phải optional.
  5. **Anti-pattern**: "Fix xong → báo done" mà skip (a) workspace doc (b) progress append (c) lesson (d) cross-service verify. Mỗi miss = future regression risk.
- **Tags**: #sop #routine #bug-handling #workflow-discipline #definition-of-done #process

---

## [2026-04-20] Lesson cũ không enforce cho new code — ScanFields lặp 3 violation đã có ADR

- **Trigger**: User architectural review `ScanFields` phát hiện 3 violation: (1) HTTP sync thay vì NATS async (ADR-015), (2) CMS touches Airbyte + INSERT mapping_rules thay vì delegate Worker (service boundary ADR), (3) hardcoded AirbyteSourceID bỏ qua `SyncEngine`/`SourceType` registry columns. Cả 3 rules đã ghi lesson/ADR từ 2026-03-31 (4 violations trước đã fix: Backfill, Standardize, Discover, Introspection) nhưng ScanFields là code MỚI sau đó vẫn lặp lại y chang pattern. Lesson hiện tại = documentation only, không enforce vào pre-commit/code-review.
- **Root Cause (meta)**: Lesson thụ động. Khi contributor (AI hoặc human) viết endpoint mới, không ai nhắc "grep ADR cũ trước khi viết". Workspace docs chứa ADR nhưng không có gate tự động. Brain/Muscle delegate code mới thiếu pre-flight check "feature mới có lặp pattern cấm không?".
- **Global Pattern [A writes code N at time T1] + [Lesson L about pattern P documented at T0 < T1] → Y violation nếu A không check L before writing N**: Lesson passively stored không chặn lặp. Cần active enforcement: pre-flight checklist, automatic lint/grep, hoặc architectural review gate.
- **Correct Pattern**:
  1. **Pre-commit grep ADR**: trước khi write endpoint mới chạm `/airbyte/`, `/DW/`, `information_schema` → `rg "service_boundary|ADR-[0-9]+" agent/memory/` để load applicable rules.
  2. **Endpoint checklist**: thêm mỗi POST endpoint vào code review: "Có dùng NATS async? Có tuân service boundary? Có support multi-source registry?".
  3. **Architectural review step trong bug-handling-sop**: nếu bug liên quan architectural decision cũ → grep lesson/ADR TRƯỚC khi propose fix.
  4. **Repeat-violation detection**: Brain scan periodically — nếu fix ra new code pattern giống cũ → flag ngay, không delegate Muscle.
  5. **Anti-pattern**: lesson viết ra rồi forget. Lesson = active reference, không phải archive.
- **Tags**: #adr-enforcement #repeat-violation #service-boundary #lesson-passive #architectural-review

---

## [2026-04-20] Cross-service refactor — Muscle parallel coordinate via subject contract

- **Trigger**: User approve fix 12 architectural violations (NATS async + service boundary + multi-source routing). Scope lớn cross 3 projects (Worker + CMS + FE). Brain delegate 3 Muscle parallel. Risk: race condition — CMS publish subject nhưng Worker chưa subscribe → lost commands?
- **Root Cause (pattern design)**: NATS **fire-and-forget** pattern cho phép parallel refactor mà không cần sync. CMS publish return immediate; nếu Worker chưa ready → message sit trong JetStream (retention 7 ngày) cho đến khi Worker subscribe pick up. FE polling status từ activity log → graceful handle "pending" state.
- **Global Pattern [A publishes event E to message broker B] + [C consumes E at some future time]**: Không cần A biết C đã ready. Broker buffers. Pattern hỗ trợ independent deploy + rolling refactor. Async decoupling > sync coupling.
- **Correct Pattern**:
  1. **Subject naming contract TRƯỚC**: agree naming (`cdc.cmd.{action}`) + payload schema giữa Brain + Muscle trước khi delegate. Parallel Muscle implement độc lập theo contract.
  2. **Fire-and-forget allowed**: CMS publish không chờ Worker subscribe. Worker subscribe khi deploy. JetStream retention guarantee no message loss.
  3. **FE polling absorb async uncertainty**: UI state machine handle `accepted → running → success|error|timeout`. User nhìn badge, không chờ.
  4. **Verify cross-boundary post-deploy**: sau all Muscle done, verify end-to-end: FE dispatch → CMS publish → Worker consume → activity log → FE poll detect. Not before.
  5. **Anti-pattern**: synchronous refactor Worker first, then CMS, then FE — waste parallel capacity + block progress.
- **Tags**: #cross-service-refactor #nats-fire-and-forget #parallel-delegation #subject-contract #async-decoupling

---

## [2026-04-20] Partitioned Table Default Orphan — Backfill, Not Just Retention

- **Trigger**: SLOW SQL 236ms regression trên query đã bounded (`WHERE X > NOW() - INTERVAL AND X <= NOW()`) — nghi ngờ fix trước đó (migration 015 + bounded range) vô hiệu. Thực tế planner vẫn không prune được vì `*_default` chứa rows trong window.
- **Global Pattern**: **[A partitioned table B có default partition C giữ orphan rows D → planner Y không thể prune C → mọi query trên B phải scan C + catalog overhead → planning time tăng tuyến tính với độ đầy C]**. Mặc dù bounded range predicate được thiết kế để kích hoạt runtime pruning, **runtime pruning không áp dụng cho default partition** (PG không có positive range để so sánh, chỉ có synthesized NOT-IN của siblings → default luôn là "có thể match"). Hậu quả: Subplans Removed trên EXPLAIN đếm sibling partitions đã prune, nhưng default **luôn** hiện trong Append nếu có bất kỳ row nào. Sai lầm conceptual: coi default là "fallback empty" giống null-value bucket, nhưng thực ra là một partition bình thường, Schedule Y/Z tick đều scan nó.
- **Correct Pattern**: Automation quản lý partition phải có **2 chiều**:
  1. **Forward (existing)**: pre-create future partitions mỗi tick để INSERT mới không rơi vào default.
  2. **Backward (missing)**: detect rows đã land vào default → materialise child partitions đúng range → move rows. Chỉ drop default khi hoàn toàn trống.
- **PG 11+ gotcha**: `CREATE TABLE … PARTITION OF … FOR VALUES FROM … TO …` sẽ fail `SQLSTATE 23514` nếu `*_default` hiện đang chứa row trong range đó. Correct txn ordering = **drain-before-create**: (a) `DELETE … RETURNING * INTO TEMP`, (b) `CREATE TABLE … PARTITION OF …`, (c) `INSERT INTO parent SELECT * FROM temp`. Sai ordering (CREATE trước move) chỉ detect được qua smoke test với real data.
- **Example mapping**: A=`partition_dropper` service, B=`cdc_activity_log`, C=`cdc_activity_log_default`, D=recon/scan test rows (dates 2026-04-14→16), Y=postgres query planner, Z=collector tick 15s × CMS uptime.
- **Generalization check**: pattern áp dụng cho (1) pg_partman deployments missing backfill grace period, (2) Debezium CDC tables với range-partition theo `source_ts`, (3) audit/log tables bất kỳ có default catch-all với late-arriving data, (4) multi-tenant partitioned tables với tenant_id partition key khi new tenant onboard trễ.
- **Tags**: #postgres #partitioning #planning-time #slow-sql #pg11 #default-partition #backfill #rule6 #root-cause

---

## Lesson 62 — Hard-coded field name in cross-store sync breaks on schema drift (2026-04-20)

- **Trigger**: Reconciliation reports `source_count=0 / dest_count=3422` for `refund_requests`, `source_count=0 / dest_count=15` for `export_jobs`. User assumed schedule not firing, but actually schedule DID fire — source agent's Mongo filter `bson.M{"updated_at": {"$gte": tLo, "$lt": tHi}}` returned 0 because the actual collections use `createdAt` + `lastUpdatedAt`, not `updated_at`. Mongo driver silently decodes missing field to zero-value `time.Time{}` without error, hiding the mismatch from tests and smoke runs.
- **Global Pattern**: **[A cross-store sync/recon component A hard-codes a field-name B from the "canonical" convention → collection X with a different convention (camelCase, created_at, lastUpdatedAt, ts) → filter matches 0 rows → Y reports "source empty" falsely → operator blames the scheduler Z rather than the schema assumption]**. The anti-pattern compounds when the decoder uses typed struct tags (`bson:"updated_at"`) instead of `bson.M` — the zero-value decode path IS the silent failure mode. Tests pass because fixtures use the canonical field.
- **Correct Pattern**: Two complementary defences:
  1. **Registry-first**: add a per-table config column (here `cdc_table_registry.timestamp_field`) + whitelist validator (`^[A-Za-z_][A-Za-z0-9_]{0,63}$`) so operators can declare the right field per collection. Default preserves backward compat.
  2. **Fallback graceful**: when the declared field is absent on a specific document, fall back to a universally-available source (Mongo `ObjectID` carries unix seconds in its first 4 bytes — `primitive.ObjectIDFromHex(...).Timestamp()`). Caller treats the fallback as "approximate ts" — still correct for hash/presence checks, degrades cleanly for range filtering.
  3. **Observability**: surface the chosen path to the UI (`source_query_method` = `window_updated_at | window_custom_field | window_id_ts_fallback | full_count`) so operators can answer "why did this count surprise me?" without reading Go source.
- **Mongo gotcha**: Typed struct decode vs `bson.M` decode. Typed = zero-value on missing, no error. `bson.M` = field simply absent from map, `_, ok := raw[key]` = false. Prefer `bson.M` + explicit extraction when the field existence is itself a semantic signal.
- **Example mapping**: A=`ReconSourceAgent`, B=`updated_at` hard-coded filter, X=`export-jobs` (createdAt) + `refund-requests` (mixed), Y=`cdc_reconciliation_report.source_count`, Z=`cdc_worker_schedule[reconcile]`.
- **Generalization check**: pattern applies to (1) Debezium source connectors hard-coding `__last_updated_at` cursor, (2) Airbyte incremental sync with fixed cursor_field across heterogeneous schemas, (3) ETL pipelines assuming a timezone-aware `updated_at` when source is a Mongo snake-case-to-camelCase mix, (4) webhooks filtering by `received_at` when upstream rebrands to `timestamp`/`ts`/`eventTime`.
- **Anti-drill**: do NOT "auto-detect field by sampling first 100 docs" as the only defence — inconsistent collections (some docs have A, some have B) would alternate answers across restarts. Explicit registry config + documented fallback is more debuggable.
- **Tags**: #reconciliation #mongo #cross-store #schema-drift #field-naming #hardcoded-assumption #rule3 #rule6 #root-cause #bson-decode-gotcha

---

## Lesson 63 — Silent-skip in scheduled jobs masks nil-dependency init failures (2026-04-20)

- **Trigger**: Worker's scheduled `reconcile` op wrote `activityLogger.Quick("reconcile", "*", "scheduler", "skipped", ...)` when `reconCore == nil`, then returned. Operators watching `worker.log` saw zero reconcile activity but no error — indistinguishable from a goroutine that panicked early. Real cause was MongoDB URL missing from config, caught only in an earlier `logger.Warn("MongoDB connection failed, reconciliation disabled")` buried in the startup stream.
- **Global Pattern**: **[A scheduled job A depends on lazily-initialised core B → startup failure of B leaves A.core=nil → A.Tick() silently short-circuits with a "skipped" row in audit table C → operators querying log-stream D cannot distinguish "skipped-by-config" from "crashed" from "never-scheduled"]**. Activity-log rows are NOT a substitute for log-stream WARN when the condition is a dependency-initialisation failure, because audit tables are per-record and log streams are temporal — operators scan the stream when diagnosing "is this running?".
- **Correct Pattern**: every silent-skip path in a scheduled job must:
  1. **WARN the log stream** on the first skip AND on every tick (repeated nil is a persistent operator-visible signal, not a one-off).
  2. **Include a `fix_hint` in the log fields** — "set MONGODB_URL env + restart worker; check startup log for 'MongoDB connection failed'" — so the triaging operator can resolve without reading code.
  3. **Emit a startup summary** when the poller starts: `"schedule poller started" enabled_count=N registered=[op=Nm,op=Nm] recon_core_available=bool` — names the available upstream deps, lists what will fire, confirms the goroutine is alive.
  4. **Per-tick info log** includes `first_run:bool` when `LastRunAt IS NULL` so operators can distinguish "fresh enable fires immediately" from "interval not elapsed yet".
- **Example mapping**: A=`runReconcileCycle`, B=`reconCore`, C=`cdc_activity_log`, D=`worker.log`.
- **Generalization check**: pattern applies to (1) cron-driven DLQ replayers depending on Kafka/NATS handles, (2) scheduled Airbyte triggers depending on REST client init, (3) Prometheus push gateways skipping when metric registry is nil, (4) any graceful-degrade path that chooses to return rather than error on missing deps.
- **Anti-drill**: do NOT replace silent-skip with panic — that would take down the whole worker on an optional dependency. The right balance is WARN-log + keep running + surface in /metrics counter so dashboards can alert on `*_skipped_total > 0`.
- **Tags**: #scheduling #observability #silent-failure #nil-dependency #log-stream-vs-audit-table #rule6 #rule8-escalation #root-cause

---

## [2026-04-20] Brain propose per-table band-aid thay vì systematic auto-detect — không scale N entities

- **Trigger**: User report payment_bills recon src=0 (Mongo 2 docs với createdAt, không updated_at). Brain đề xuất trong Muscle brief: "Quick fix payment_bills: UPDATE registry SET timestamp_field='createdAt' WHERE target_table='payment_bills'". User phản ứng: "với quy mô 200 table, mày cũng fix từng cái à, ngu đần. cái cần là giải pháp thông minh. ko phải làm kiểu tình thế". Đúng: fix per-entity O(N) manual intervention ≠ systematic solution O(1) auto-detection. Session history đã lặp pattern: export_jobs cũng manual fix timestamp_field, giờ payment_bills tương tự — nếu 200 tables thì cần 200 UPDATE statements + admin knowledge per-table schema.
- **Root Cause (meta)**: Brain optimize cho "fix bug hiện tại" thay vì "fix cơ chế gây ra bug". Per-entity fix = tình thế (band-aid). Systematic solution = auto-detect sample + fallback chain + admin override-only khi cần. Pattern tương tự lesson #60 (ADR passive không enforce) — cần ACTIVE design, không reactive.
- **Global Pattern [A configures entity B_i with field F manually for each i ∈ N entities] → O(N) human intervention + high error rate**: Entity configuration yêu cầu admin knowledge schema per-entity = unmaintainable ở scale. Correct: auto-detect từ entity data itself + fallback chain + registry default + admin override chỉ khi auto fail.
- **Correct Pattern**:
  1. **Auto-detect at entity boundary** (register time HOẶC first-scan): sample data → detect field presence ranking → auto-populate config.
  2. **Fallback chain runtime**: nếu configured field returns 0 documents trong N consecutive runs → auto-try next candidate → update registry suggestion → admin review.
  3. **Admin override escape hatch**: UI form cho phép manual override (backward compat) nhưng default = auto.
  4. **Log recommendations**: worker log "detected field X for table Y with confidence Z%, fallback to W available" → admin có visibility không cần query each table.
  5. **Anti-pattern**: "UPDATE registry SET config='X' WHERE name='Y'" → repeat for each entity. Nếu 200 entities → 200 sql statements = tình thế.
- **Tags**: #band-aid-vs-systematic #auto-detect #scale-n-entities #registry-config #per-entity-fix

---

## [2026-04-20] Brain viết plan decisions dựa trên state tưởng tượng, không verify

- **Trigger**: User cung cấp Master Plan v1.25. Brain viết section "6 Decisions Required" có Q5: "Migrate `sync_engine='both'` đầu tiên hay cuối?". User phản ứng: "bỏ cái này mà, đọc tài liệu kiểu gì vậy" — vì hiện tại **0 tables có sync_engine='both'** (verified session trước: 6 airbyte + 2 debezium + 0 both). Câu hỏi invalid, hallucinate state.
- **Root Cause (meta)**: Brain viết plan decisions mà không re-verify runtime state ngay trước khi ask. Trong session đã có evidence `SELECT sync_engine, COUNT(*)` từ earlier audit. Brain forgot/ignored → wrote decision question dựa trên possibility, không reality.
- **Global Pattern [A designs plan asking decisions about entity state S] → Invalid nếu A không verify S hiện tại**: Plan decisions require ground truth about current state. Extrapolating "có thể có" → asking user as if real = wastes user time + signals sloppy work.
- **Correct Pattern**:
  1. **Pre-decision state re-verify**: trước khi write "Decisions Required" section, re-run relevant queries (DB state, feature flags, deployment status) → confirm entities exist BEFORE asking about them.
  2. **State snapshot in plan**: embed current state query output (e.g., `sync_engine counts`) ngay trong plan Section 1 "Current State" — force self-audit.
  3. **Conditional decisions**: nếu decision về state possibly nonexistent, phrase as "IF X exists, then...". Không "which X first" as default.
  4. **Anti-pattern**: copying decision template từ generic migration framework → asking questions irrelevant to specific environment.
- **Tags**: #hallucination #state-verification #plan-decisions #ground-truth #user-flag

---

## [2026-04-20] Passive plan (band-aid) vs Systematic Reconstruction — 6 violations cùng lúc

- **Trigger**: User provide Master Plan v1.25 (Unified Sonyflake). Brain viết plan tích hợp nhưng vi phạm 6 nguyên tắc user đã nêu rõ: (1) View band-aid giữ _airbyte_* rác physical layer, (2) Trigger IF NULL cho phép Go pass sai ID, không FORCE DB, (3) Mapping _gpay_* ↔ _* cũ spaghetti, không unified prefix, (4) COALESCE anti-ghosting quên OCC với _source_ts migration 009, (5) Giữ PK cũ "nhát gan" gây dual-index phình IO, (6) Worker ID 0 mặc định không verify Go IP range collision. User: "passive, che đậy, giữ tàn dư cũ cản trở Unified Architecture".
- **Root Cause (meta)**: Brain mặc định **minimum-disruption = good**. Với migration feature/column đơn lẻ OK. Với **architectural reconstruction** (new identity system), minimum-disruption = lỗ hổng vì **tàn dư cũ chính là bug source**. User yêu cầu "Unified" tức nguyên khối, Brain trả "incremental alias" tức **trái nguyên tắc**.
- **Global Pattern [A plans architectural reconstruction R] + [A defaults to minimum-disruption M] → Result fail-to-deliver R**: Reconstruction ≠ migration. Reconstruction đòi hỏi **drop + rebuild** clean slate. Migration đòi hỏi **preserve + transform** backward compat. Nhầm 2 modes = plan nửa vời, cũ vẫn ám mới.
- **Correct Pattern for Architectural Reconstruction**:
  1. **Physical clean slate**: Không giữ column rác dưới mọi hình thức (VIEW ẩn vẫn chiếm disk, VACUUM chậm, backup bloat). Drop physical + bóc business fields sang columns thật.
  2. **Force authority**: Identity Provider phải SINGLE. DB sinh ID = DB SOLE AUTHORITY. Go truyền ID = DB validate STRICT (format + range + epoch + worker_id allocation). Không "IF NULL fallback" — phải EXPLICIT REJECT invalid input.
  3. **Unified naming**: Prefix mới = toàn bộ prefix mới. Không alias từ naming cũ. Alias = semantic confusion, spaghetti logic debug.
  4. **Preserve what EARNED its place**: Existing OCC (`_source_ts`) là **working pattern** → rename sang `_gpay_source_ts` giữ semantic, KHÔNG thay thế bằng COALESCE ad-hoc. Earned preservation # sloppy preservation.
  5. **Aggressive cutover**: DROP old PK phải trong cùng migration (transactional), không "defer N days". Defer = indecision = dual-write IO waste.
  6. **Verify environment before reserve**: Worker ID range, epoch, IP allocation phải **query existing deployment** trước assign. "Reserve 0" without checking = assumption = collision risk.
- **Anti-pattern decision tree**:
  - Q: "Preserve for BC?" → Only if column có active consumer code. If only legacy callsite → rewrite callsite, drop column.
  - Q: "View alias for ergonomics?" → Only if reader needs simpler projection. Not for hiding rác.
  - Q: "Dual PK safety?" → Never in unified architecture. Choose one, commit.
- **Tags**: #reconstruction-vs-migration #band-aid #identity-authority #unified-naming #physical-clean-slate #forced-cutover

---

## [2026-04-20] Brain plan "ngầu từ ngữ" nhưng thiếu OPS reality — aggressive = thảm họa production

- **Trigger**: Sau khi user reject v1 plan (6 band-aid violations), Brain rewrite v2 "reconstruction aggressive" tưởng là fix. User phê phán 5 mistakes NẶNG HƠN: (1) "Auto-detect business columns" từ JSONB là hallucination — JSONB types inconsistent không thể sinh typed SQL schema cho 200 tables trong 13-14h. (2) "Single Identity Authority" giả — Debezium path vẫn Go-sinh-ID + DB validate, NTP lệch = Sonyflake broken. Không phải authority thật. (3) "Aggressive cutover" = CREATE+INSERT SELECT+CREATE INDEX+DROP PK trong 1 transaction trên 10M+ rows → Postgres LOCK bảng → Worker downtime 30+ phút. (4) Worker ID reserve "bằng grep log" = K8s pods IP dynamic, fragile collision risk. (5) `_raw_data - ARRAY[...]` JSONB strip trong migration transaction = CPU-expensive trên millions rows = tự sát performance. User: "Brain đang lấp liếm phức tạp bằng từ chuyên môn, chưa bao giờ vận hành DB lớn".
- **Root Cause (meta)**: Brain generate plan **theoretically correct** + dùng từ ops-sounding (aggressive, forced cutover, clean slate) nhưng thiếu **operational experience primitives**: (a) large-table migration locking math, (b) online schema change tools (pg_repack, pt-osc), (c) zero-downtime patterns (dual-write, logical replication), (d) K8s dynamic IP reality, (e) type inference fundamental impossibility với schema-less source. Reading ops blogs ≠ ops experience. Plans sound confident but deliver production incidents.
- **Global Pattern [A writes refactor plan P using strong vocabulary V] + [A lacks ops experience E for scale S] → P fails catastrophically at execution time**: Vocabulary không thay thế hiểu biết ops. "Aggressive" là branding, không phải implementation. Real ops plans have: (a) explicit lock duration calc, (b) rollback within 30s window, (c) dual-read/dual-write transition, (d) zero-downtime tools referenced, (e) batch sizes tuned to table rowcount.
- **Correct Pattern for Production DB Reconstruction**:
  1. **Never single-transaction millions-row migration**: Use pg_repack (online VACUUM FULL without lock), logical replication-based swap, hoặc staged batch COPY với lock_timeout=5s + small batches. Transaction <100K rows typical limit.
  2. **Type extraction requires manual per-table work**: 200 tables × 30min-1h mapping = 100-200h manual work. Không tự động. Accept JSONB queries nếu không có budget mapping. Don't hallucinate "auto".
  3. **Worker ID dynamic registry**: Redis SETNX với TTL heartbeat, claim-on-boot, release-on-shutdown. K8s pod restart-safe.
  4. **True single identity**: Go Worker CALL `SELECT next_sonyflake()` qua DB connection (adds 1-2ms latency) OR accept dual-source with NTP SLA monitored (skew <10ms alerted).
  5. **Strip at Worker not DB**: Transform/strip in application layer before INSERT. DB migration transactions don't include data transformation.
  6. **Zero-downtime tools**: pg_repack, pg_logical, pt-online-schema-change. Reference concrete tools, not hand-waved "aggressive".
  7. **Lock duration calculation upfront**: Every DDL touching production table PHẢI calc estimated lock duration. >5s = require OSC tool. State "this will lock N seconds" explicitly.
- **Anti-patterns rejected**:
  - ❌ "Auto-detect" without pointing to specific algorithm with edge case handling
  - ❌ "Aggressive cutover" as design principle — always specify tool + lock math
  - ❌ "Single transaction reconstruction" for tables >100K rows
  - ❌ "Reserve worker ID" without dynamic registry — static assumption breaks in dynamic infra
  - ❌ JSONB operations in migration transaction — offload to application layer
  - ❌ 13-14h estimate for 200-table manual schema mapping — reality 100-200h
- **Tags**: #ops-reality #locking-math #zero-downtime #jsonb-type-inference #worker-id-registry #plan-vocabulary-vs-substance

---

## [2026-04-20] Brain scope-cut = hèn nhát — 3 lần plan fail liên tục cùng Sonyflake v1.25

- **Trigger**: User critique v3 Ops-Grounded plan với 5 điểm: (1) Skip typed columns = chỉ rename, không reconstruction thật, (2) Hybrid identity Go local + PG batch = sequence drift risk, (3) Redis Worker ID Registry = over-engineering SPOF khi PG có SKIP LOCKED, (4) pg_repack đề xuất không check disk space/I/O spike risk, (5) Strip rác chỉ ở ngọn — dữ liệu cũ 10M rows vẫn bẩn trong DB. User: "Kế hoạch v3 là bản thỏa hiệp đốn mạt giữa lười biếng developer và sợ hãi dân Ops. Tao cần kiến trúc đúng đắn, không phải danh sách Rename cột."
- **Pattern (3 lần liên tục)**:
  - v1: passive band-aid (VIEW ẩn rác, dual PK giữ cũ) → user reject
  - v2: vocab-aggressive hallucinate (auto-detect 200 tables 13-14h, single-transaction 10M rows) → user reject
  - v3: ops-grounded scope cut (skip typed extraction "out of scope", hybrid identity tránh cost, Redis registry thay PG) → user reject ĐÂY
- **Root Cause (meta-meta)**: Brain reaction to criticism: **layer-shift thay vì full-depth**. Bị critique about theory → shift to ops tool reference. Bị critique ops → shift to scope cut "honest". Pattern: **move laterally avoid full cost acceptance**. Never commit to full reconstruction cost (200h+ manual mapping, zero-compromise transformation, accept true single authority latency).
- **Global Pattern [A designs R with full cost C] + [C threatens A's "nice-completion" narrative] → A scope-cuts R calling "pragmatic" / "honest" / "out of scope"**: Scope cut ≠ honesty. Scope cut = avoid commitment. True honesty = state full cost + user choose. Hèn nhát = pre-decide "too expensive" và hide scope.
- **Correct Pattern**:
  1. **Accept full cost upfront**: present complete reconstruction at real effort (200h+ for 200 tables mapping) + let user decide priority, không pre-cut.
  2. **Resist layer-shift**: user rejected theoretical → don't shift to ops vocab. User rejected vocab → don't shift to scope cut. Stay at same layer, deliver deeper.
  3. **Single-source identity must mean SINGLE source**: no hybrid, no "validate". Identity provider = call one authority. Latency trade-off explicit, don't hide with "validation layer".
  4. **Dependency minimization**: nếu PG sufficient (SKIP LOCKED, advisory lock) không thêm Redis. User workload already Postgres-heavy, adding Redis = operational complexity transfer.
  5. **Migration = TRANSFORM not just COPY**: nếu mục tiêu clean data, batched transform in application layer + stream into new schema. Data cũ không tự sạch bằng keyword "clean slate".
  6. **Every tool recommendation = disk/CPU/IO risk section mandatory**: pg_repack? → disk 2x + I/O spike. Logical replication? → replication lag + catch-up time. Don't cite tool without caveats.
- **Anti-patterns rejected**:
  - ❌ "Out of scope" khi user asks full reconstruction
  - ❌ "Pragmatic hybrid" = avoid committing to single source design
  - ❌ "Auto-detect" bất kỳ structured-from-unstructured inference
  - ❌ Cite tool without disk/IO/lag math
  - ❌ "Strip at Worker" áp dụng mới mà bỏ data cũ bẩn
- **Tags**: #scope-cut #layer-shift #full-reconstruction-cost #pattern-4-failures #cowardice-vs-honesty #jsonb-vs-typed

---

## [2026-04-21] Brain fail 5 lần liên tục cùng feature Sonyflake v1.25 — user phải literally prescribe

- **Trigger**: v1 band-aid → v2 vocab-lie → v3 scope-cut → v4 trigger-hell + centralized SPOF + O(N²) backfill + MAX+1 race. User critique v4 với 5 điểm fatal + **literally prescribe v5**: (a) Go Worker gánh typed extraction, không trigger; (b) PG chỉ cấp MachineID boot-time qua SEQUENCE, không cấp Sonyflake từng ID; (c) Migration dùng Shadow Table + cursor scan, không NOT EXISTS.
- **Pattern identified (5 iterations)**: Brain "creative" trong design = nguồn bug. Mỗi lần user reject, Brain pivot sang direction khác vẫn sai vì "creative" direction mới chưa experience-tested. Brain opus-4-7 **không có distributed systems ops experience thật** — chỉ có blog-level knowledge. "Creative solution" với blog knowledge = architecture anti-pattern.
- **Root Cause (meta-meta-meta)**: Khi user ask architectural design, Brain's value add = synthesize well-known patterns đúng context, KHÔNG phải invent new patterns. Brain đã invent: (a) VIEW aliasing v1, (b) hybrid identity v2/v3, (c) Redis Worker Registry v3, (d) Go-call-PG batch v4, (e) trigger-based transformation v4, (f) MAX+1 worker claim v4. Tất cả đều sai vì chưa production-tested. Well-known patterns (SEQUENCE for ID allocation, cursor-based migration, app-layer transformation) Brain biết nhưng không chọn → biased toward novelty over proven.
- **Global Pattern [A invents pattern P for architectural problem Q] + [A lacks production experience E] → P has unknown failure modes user discovers iteratively**: Invention without experience = liability. Well-known patterns exist vì đã battle-tested. Brain default phải chọn proven patterns, không invent.
- **Correct Pattern**:
  1. **Default to boring**: SEQUENCE > custom max+1. app-layer transform > trigger. cursor scan > NOT EXISTS. Boring = production-proven.
  2. **Invent ONLY when user explicitly asks novelty**: nếu user không demand "creative", default to textbook.
  3. **List well-known patterns first, pick 1, justify**: before proposing solution, enumerate 3-5 proven options với trade-offs. User picks. Không Brain pick then defend.
  4. **When user prescribes, TRANSCRIBE không REINTERPRET**: user prescription v5 = literal follow, không "improve" với Brain's creative additions.
  5. **Admit N failures explicitly**: sau 3 fails same feature, tell user "Brain unreliable on this, please prescribe specifics". Don't pretend v(N+1) tốt hơn v(N).
  6. **Anti-pattern**: Brain "creative" in domain Brain không có experience. Symptoms: novel patterns proposed, estimates off by 10x, risk sections missing, user catches basic flaws (race conditions, O(N²), SPOF).
- **Tags**: #novelty-vs-proven #brain-limitation #creative-architect-fail #user-prescription-literal #5-iteration-failure

---

## [2026-04-21] Brain introduce new bugs khi fix old bugs — 6 lần Sonyflake v1.25, N issues + N fixes = N more issues

- **Trigger**: User reject v5 với 4 điểm fatal mới: (1) MachineID leak khi K8s Pod SIGKILL không chạy defer release → 65535 IDs kẹt 'active' vĩnh viễn; (2) Forward queue eventual consistency khi swap bảng queue còn tồn đọng → data drift tài chính; (3) Trigger write queue = double I/O, 10K msg/sec → DB overload; (4) Regex healer `amount`: EU format `1.234,56` → `1.23456` = mất tiền khách hàng. User prescribe v6: heartbeat-based reclaim, Logical Replication OR sync-within-transaction bỏ queue, strict validator thay regex financial heal.
- **Pattern (6 iterations)**: Mỗi version fix N issues user raised, Brain add M new issues chưa user raise. v5 fix: trigger hell → app-layer Worker ✓, SPOF → local Sonyflake ✓, MAX+1 race → SEQUENCE ✓. v5 introduce: leak via assumed-graceful shutdown, queue double-IO, regex heal unsafe, eventual consistency at swap. "Fix" cycle never converges without user pointing each specific.
- **Root Cause (meta^3)**: Brain patches at surface. Mỗi fix generates side-effects vì Brain không model full system state (K8s failure modes, financial data precision, I/O amplification, swap atomicity). User model = complete; Brain model = partial. Partial model → surface fix → new surface issue.
- **Global Pattern [A fixes flaw F1 in design D with patch P] + [A lacks full model M of system] → P introduces F2 elsewhere that M would catch**: Without complete model, fix = whack-a-mole. Brain opus-4-7 ops model incomplete for distributed systems edge cases (signal handling, financial data integrity, I/O capacity, eventual vs strong consistency boundaries).
- **Correct Pattern**:
  1. **Every fix requires "what else breaks?" audit**: trước commit fix F1, enumerate side-effects. Eg queue để zero-downtime → side-effect double IO + eventual consistency at swap. Named trade-offs before decide.
  2. **Default to Postgres built-ins**: Logical Replication, SERIAL/SEQUENCE, CHECK constraints, advisory locks — tested ops primitives. Don't invent "queue pattern" when PG has publication/subscription.
  3. **Financial data NEVER auto-heal with pattern matching**: regex/parsing heuristics unsafe. Either strict locale-aware parser OR manual review, no middle ground.
  4. **K8s failure model default**: Pods die SIGKILL. Graceful shutdown is optional path, not default. Registry designs MUST assume ungraceful termination.
  5. **Consistency boundary explicit**: state "this operation eventual consistency with lag X" OR "strong consistency via transaction". Don't call queue "zero-downtime" without naming the consistency trade-off.
  6. **After 3 rejections**: Brain stop invention, switch to "enumerate proven patterns, user picks". Iteration 4+ = prescription transcription only.
- **Anti-patterns rejected**:
  - ❌ "Released status" assumption graceful shutdown always runs
  - ❌ "Queue + async consumer" without drain-before-swap contract
  - ❌ "Regex fixer" on financial/security/health data
  - ❌ Trigger pattern when Logical Replication exists
  - ❌ Calling fix "zero-downtime" or "lightweight" without latency/IO math
- **Tags**: #whack-a-mole #incomplete-system-model #financial-data-precision #k8s-failure-modes #postgres-builtins #sixth-iteration-failure

---

## [2026-04-21] Brain fail 6 lần Sonyflake — missing distributed primitives: fencing, outbox, data profiling, physical slot

- **Trigger**: User reject v6 với 4 tử huyệt: (1) Zombie Pod → heartbeat reclaim mà không Fencing Token = 2 Pods same machineID khi GC pause/network stall → Sonyflake collision; (2) sync-within-transaction Bloat = Lock Duration tăng + Connection Pool exhaust ở Wallet 10K msg/sec; (3) Locale config per-field cho 200 bảng = maintenance nightmare + silent corruption nếu cấu hình sai; (4) ORDER BY id backfill giả định PK tuần tự — UUID/ObjectID File Sort 10M rows = Disk I/O peak. User prescribe: Fencing (Pod self-terminate khi heartbeat fail), Outbox Pattern/async integrity check, auto data profiling, Physical Slot/Keyset pagination thực thụ.
- **Pattern (6 iterations all rejected)**: Brain chọn textbook nhưng luôn miss distributed systems primitives nâng cao: fencing tokens (Martin Kleppmann lock safety), outbox pattern (microservices BP), data profiling statistical inference, PG snapshot-based physical scan. Brain biết concepts này trong training data nhưng default sang naive implementation (heartbeat-only, sync-in-tx, manual config, ORDER BY id).
- **Root Cause (meta^4)**: Brain's "textbook" = Wikipedia-level basics. User's "textbook" = production engineering primitives from Designing Data-Intensive Applications, Kleppmann papers, pg_repack/Debezium internals. Gap = reading level vs operating experience with those primitives.
- **Global Pattern [A implements feature F at scale S] + [A uses Wikipedia-level primitives] → P fails on distributed edge case E that production-level primitives would catch**: Heartbeat without fencing = known broken. Sync-in-transaction at 10K msg/sec = known bottleneck. Manual config at scale N = known unmaintainable. Naive ORDER BY for UUID = known File Sort. All classic problems with classic solutions Brain has in training but doesn't surface without user prompt.
- **Correct Pattern**:
  1. **Distributed locking MUST have fencing token**: heartbeat alone insufficient. Every claim returns monotonic token; every write verifies token; token holder lost → process exit (fail-stop).
  2. **High-throughput writes avoid synchronous dual-writes**: use outbox (separate tx for publish), logical replication (PG built-in), or CDC-based (Debezium on PG) — named patterns, not invented.
  3. **Config at scale requires inference + override**: auto-detect default + admin override for exceptions. Not pure manual, not pure auto.
  4. **Physical scan for backfill**: ctid-based ranges, pg_export_snapshot for consistency, parallel workers per range. Not naive ORDER BY PK.
  5. **Before v(N+1)**: enumerate distributed primitives that apply (fencing, outbox, snapshot, MVCC). If Brain not referencing these = incomplete answer.
- **Anti-patterns**:
  - ❌ Heartbeat without fencing token (unsafe)
  - ❌ Synchronous dual-write in hot path (latency)
  - ❌ Manual config per-entity at scale (unmaintainable)
  - ❌ ORDER BY backfill without index verification
  - ❌ Calling "eventual consistency" zero-downtime without drain-before-swap contract
- **Tags**: #fencing-token #outbox-pattern #data-profiling #physical-slot-scan #distributed-primitives #wikipedia-vs-production-level

---

## [2026-04-21] PostgreSQL ON CONFLICT WHERE chỉ apply UPDATE path, không INSERT — Zombie Pod escape

- **Trigger**: User reviewed v7.1 Section 2.1 Hybrid Fencing implementation. Brain đề xuất `INSERT ... ON CONFLICT (_gpay_source_id) DO UPDATE SET ... WHERE EXISTS (SELECT 1 FROM worker_registry WHERE fencing_token=$N)`. User pointed out **fatal technical gap**: PostgreSQL's `WHERE` clause in ON CONFLICT DO UPDATE chỉ filters UPDATE path. Khi row mới (no conflict) → INSERT thành công bất chấp WHERE. Zombie Pod có token reclaimed vẫn insert được records mới trước khi heartbeat detect và self-terminate.
- **Root Cause**: Brain biết syntax `INSERT ... ON CONFLICT ... WHERE` nhưng chưa verify exact semantic của WHERE scope. Assumption: WHERE "guards the whole statement". Reality: WHERE only guards the DO UPDATE sub-action. Basic PG docs truth Brain missed.
- **Global Pattern [A uses SQL clause C for safety guard G] + [A doesn't verify C's exact scope per RDBMS]** → Gap where C doesn't cover G completely: Every SQL clause has precise scope defined by RDBMS docs. "Common sense" interpretation can miss. Especially ON CONFLICT WHERE (UPDATE only), RLS policies (query rewriting), trigger WHEN clauses (pre-fire filter not post-action), CHECK constraint (row-level not tx-level).
- **Correct Pattern for Full-Path Guards**:
  1. **BEFORE INSERT OR UPDATE trigger** là guaranteed scope cho cả 2 operations. `RAISE EXCEPTION` rolls back entire transaction including INSERT.
  2. **RLS policy** (Row-Level Security) với `WITH CHECK` clause = guard INSERT + UPDATE both paths.
  3. **CHECK constraint** với subquery impossible (CHECK can't reference other tables). Avoid.
  4. **Verify scope before cite**: mỗi SQL mechanism proposed as safety guard, verify in RDBMS docs "applies to INSERT?", "applies to UPDATE?", "applies to DELETE?" explicitly.
- **Specific fix pattern for fencing enforcement**:
  - Worker sets `SET LOCAL app.fencing_token = $N, app.machine_id = $M` per transaction
  - Trigger reads via `current_setting('app.fencing_token', true)` 
  - Compare against `cdc_internal.worker_registry` live value
  - Mismatch → `RAISE EXCEPTION 'FENCING: token mismatch'` → tx rollback entire, both INSERT and UPDATE blocked
- **Anti-patterns**:
  - ❌ `INSERT ... ON CONFLICT ... DO UPDATE ... WHERE guard` (INSERT path escapes guard)
  - ❌ `INSERT WITH CHECK guard` (not valid PG syntax)
  - ❌ Relying on CHECK constraint for cross-table reference (not allowed)
  - ❌ Putting guard in AFTER trigger (tx already committed data)
- **Tags**: #postgres-on-conflict-scope #fencing-enforcement #before-trigger #session-variable #sql-clause-scope-verification

---

## [2026-04-21] PostgreSQL RETURNS TABLE OUT parameter name collision with referenced column — SQLSTATE 42702 ambiguous

- **Trigger**: Muscle triển khai `cdc_internal.claim_machine_id(...) RETURNS TABLE(machine_id INT, fencing_token BIGINT)` — body dùng `UPDATE cdc_internal.worker_registry SET ... WHERE machine_id = (...)`. Call fail với SQLSTATE 42702 `column reference "machine_id" is ambiguous` vì OUT param name `machine_id` xung đột với `worker_registry.machine_id`. Runtime-only error, không catch khi CREATE FUNCTION.
- **Root Cause**: PostgreSQL function body resolves identifiers bằng name. RETURNS TABLE OUT params introduce column-like names vào function scope. Nếu trùng tên với physical table column referenced trong body → resolver ambiguous, SQLSTATE 42702 lúc runtime.
- **Global Pattern [A creates function F `RETURNS TABLE (col_name T)` và body references `table.col_name`] → Ambiguity error runtime even though CREATE succeeds**: Function signature syntactic checks không phát hiện body scope conflict. Only runtime execution reveals.
- **Correct Pattern**:
  1. **OUT param naming convention**: prefix `out_` hoặc `_out_` để tránh collision với table columns (`out_machine_id`, `_out_fencing_token`)
  2. **Table alias trong body**: `UPDATE worker_registry wr SET ... WHERE wr.machine_id = ...` — forces qualified name, resolver non-ambiguous
  3. **`DROP FUNCTION IF EXISTS ... CASCADE` guard trước `CREATE OR REPLACE`**: nếu signature (OUT params) đổi giữa versions, CREATE OR REPLACE fails silently với old signature preserved. DROP first ensures fresh signature.
  4. **Test call runtime**: CREATE FUNCTION pass ≠ function works. SELECT * FROM func() để validate runtime before commit migration.
- **Tags**: #postgres-function-scope #ambiguous-column #sqlstate-42702 #returns-table-out #create-or-replace-signature

---

## [2026-04-23] Scaffold CSS cruft overrides component library contract

- **Trigger**: Boss — "text ở label, input bị trùng màu dẫn đến ko trực quan" trong cms-fe.
- **Root Cause (meta)**: Default Vite/CRA/Next React template `index.css` khai báo CSS custom properties + `color-scheme: light dark` + `@media (prefers-color-scheme: dark)` swap toàn cục color/bg. Khi integrate component library (AntD, MUI, Chakra) với theme mặc định light nhưng không mount ConfigProvider/ThemeProvider riêng → scaffold CSS cascade đè vào component, gây clash khi user ở OS dark mode (component stays light, global text flips to gray) → contrast ratio xuống dưới WCAG AA 4.5:1.
- **Global Pattern [A (scaffold global CSS) overrides B (component library default tokens) in X (user OS dark mode)] → Result Y (contrast clash, unreadable labels/inputs)**:
  - Viết component library nào (Y) với light-theme default mà không khai báo theme provider, và để template CSS (A) với `prefers-color-scheme: dark` block → luôn clash khi user OS dark.
  - Áp dụng cross 3+ projects: AntD + Vite, MUI + Next, Chakra + CRA.
- **Correct Pattern**:
  1. Ngay khi scaffold project React + component library, audit `src/index.css` (hoặc `styles/globals.css`):
     - DELETE biến CSS không component nào dùng (grep verified).
     - DELETE `color-scheme`, `color`, `background` trên `:root`/`html`/`body` nếu component library tự handle.
     - DELETE `@media (prefers-color-scheme: dark)` block UNLESS app explicit hỗ trợ dark mode via ConfigProvider.
  2. Chỉ giữ: reset (margin/padding body), font stack, box-sizing, `#root` layout.
  3. Nếu cần dark mode: mount `<ConfigProvider theme={{ algorithm: theme.darkAlgorithm }}>` (AntD) dựa trên `window.matchMedia('(prefers-color-scheme: dark)')`, KHÔNG dựa vào CSS `prefers-color-scheme` riêng.
  4. Contrast check bằng axe-core / Lighthouse CI hoặc manual với WCAG calculator (label-on-bg ≥ 4.5:1).
- **Anti-pattern**:
  - ❌ Giữ template cruft (`--accent-bg`, `#social`, `.button-icon`) vì "có thể dùng sau".
  - ❌ `:root { color: var(--text) }` trên global khi có component library — luôn đè vào lib components.
  - ❌ Enable `color-scheme: light dark` mà không mount theme provider → OS swap không đồng bộ với lib.
  - ❌ Fix spot-level (override màu ở từng Form.Item) thay vì fix ở root CSS.
- **Detection**:
  - `grep -rnE "var\(--[a-z-]+\)" src/` — nếu chỉ thấy trong 1 file `index.css` → cruft.
  - DevTools `:root` computed color → nếu khác `rgba(0,0,0,0.88)` (AntD default) → đang bị override.
- **Tags**: #fe #css #theming #scaffold-cruft #antd #wcag #a11y #contrast

---

## [2026-04-24] Architecture doc drift khi pipeline tiến hoá thêm tầng

- **Trigger**: Boss review `/masters` + `/registry` → phát hiện architecture.md mô tả 1-tầng PG (Mongo → Debezium → Kafka → Worker → PG), nhưng Sprint 5 reality đã tiến hoá thành 2-tầng (Shadow `cdc_internal.*` + Master `public.*_master` với Transmuter Module + Master DDL Generator + Schema Proposal Workflow giữa).
- **Root Cause (meta)**: Khi codebase tiến hoá qua nhiều sprint, arch doc viết ở sprint đầu thường không được append. Reviewer mới / outside dev đọc arch hiểu sai hệ thống. Dev mới triển khai có thể lặp lại layer 1-tầng, xung đột với 2-tầng hiện hành.
- **Global Pattern [A (arch doc) written at sprint N, reality drifts at sprint N+K] → Result Y (misalignment for new joiners + risk of duplicate-layer implementation)**:
  - Áp dụng cross 3+ projects: Any sprint-based product với evolving pipeline (CDC, ETL, event sourcing, data mesh).
- **Correct Pattern**:
  1. Mỗi sprint kết thúc có **feature mới ở layer/component level**, append section vào arch doc — không ghi đè section cũ. Rule 11 immutability.
  2. Dùng versioned section: "5.0 Ingestion Path (Sprint 1)", "5.5 Shadow→Master via Transmuter (Sprint 5)", kèm "as of <date>".
  3. Trong FE/UI, phần nào thuộc layer cũ → mark "legacy" hoặc remove. Đừng để dead dropdown/button (e.g., "airbyte" option khi Airbyte đã retire).
  4. Gap analysis định kỳ (mỗi 2-3 sprint) giữa arch.md vs router.go/main.tsx — grep endpoint/menu vs doc section.
- **Anti-pattern**:
  - ❌ Viết arch "as aspirational" rồi quên update.
  - ❌ Delete old arch section (mất audit trail về sao hệ thống từng trông thế).
  - ❌ Để UI giữ option/button của feature đã retire (airbyte dropdown, bridge button 410 Gone).
  - ❌ "Doc là dead artifact sau khi merge" mindset.
- **Detection**:
  - `grep -rnE "airbyte|bridge|legacy|retired" src/pages/` → còn reference UI cho feature chết.
  - Read architecture.md section 4-5 + compile actual router.go endpoints → list endpoint trong router không đề cập trong arch = drift.
  - Ask "nếu new dev đọc arch 30 phút rồi code, họ có trigger ingestion qua đúng entry point không?" — nếu câu trả lời là "không vì arch viết Airbyte nhưng reality Debezium" → drift confirmed.
- **Tags**: #architecture #doc-drift #ui-stale #legacy-cleanup #pipeline-evolution

---

## [2026-04-24] CMS proxy cho infra-control endpoint: luôn qua audit chain

- **Trigger**: Gap 5a — FE cần tạo Debezium connector mới. Kafka-Connect REST (port 18083) public accessible, nhưng expose trực tiếp lên FE = bypass auth + bypass audit.
- **Root Cause (meta)**: Khi integrate infrastructure-control plane (Kafka Connect, Airbyte API, Prometheus admin API, k8s API) vào user-facing UI, dev dễ chọn đường tắt "FE gọi thẳng endpoint infra" vì nó có sẵn. Điều này tạo 3 loại rủi ro: (1) không có auth layer với user identity → action không attribute được, (2) không idempotency → retry tạo duplicate, (3) không audit log → compliance gap.
- **Global Pattern [A (infra REST endpoint) exposed to B (browser FE) without C (app auth + audit + idempotency proxy)] → Result Y (audit/security/replay loss)**:
  - Áp dụng cross 3+ projects: AWS infra admin từ BI dashboard, Grafana from customer portal, Kafka-Connect from CMS UI, Prometheus from ops cockpit.
- **Correct Pattern**:
  1. Viết CMS handler proxy (`SystemConnectorsHandler.Create` v.v.) forward request tới infra endpoint, return response.
  2. Route wire qua destructive chain: JWT → RequireOpsAdmin → Idempotency → Audit.
  3. Validate input (name regex, required fields) TRƯỚC khi forward.
  4. Strip sensitive response field (password/token) khi GET về FE (`filterSafeConfig`).
  5. FE gửi `Idempotency-Key` + `reason` ≥10 chars trên mỗi destructive request.
- **Anti-pattern**:
  - ❌ FE `fetch('http://kafka-connect:8083/connectors')` trực tiếp.
  - ❌ CMS proxy nhưng không audit (`registerDestructive` skip).
  - ❌ Proxy forward thẳng body không validate → cho phép injection vào infra config (path traversal, arbitrary connector.class).
- **Detection**:
  - `grep -rnE "http://[a-z-]+:(8083|9090|4318|8086)" src/` — FE gọi thẳng infra port = red flag.
  - CMS route missing `registerDestructive` wrapper cho mutating endpoint = audit gap.
  - Response JSON chứa `password/secret/token` không `***` = leak gap.
- **Tags**: #security #audit #cms #proxy #infra-control #idempotency


---

## [2026-04-24] Route classification: phân biệt "draft mutation" vs "destructive action" trước khi mount middleware

- **Trigger**: Mid-session correction từ Boss. Tao mount `POST /v1/wizard/sessions` (create DRAFT wizard session) + `PATCH /v1/wizard/sessions/:id` (update session fields) qua `registerDestructive` chain. Result: FE gọi → 400 "missing Idempotency-Key", sau đó 400 "missing or too-short `reason`". FE phải gửi 3 thứ (`JWTAuth` + `Idempotency-Key` header + `reason ≥ 10 chars` body) cho 1 action thực chất chỉ là "tạo draft/chỉnh metadata" — zero infra side-effect.
- **Root Cause (meta)**: Lẫn lộn **semantic layer** khi phân tier. "Destructive" nghĩa là action gây side-effect trên shared infrastructure (DDL, infra-plane API, data rename/delete). Create/Patch một bản ghi *trạng thái session* không gây side-effect thật — nó chỉ là form state persisted BE-side. Gắn chúng vào destructive chain tạo **audit noise** (mỗi lần user gõ 1 ký tự vào Input cũng tạo 1 row `admin_actions`) + bắt FE handshake 3 header với action không đáng.
- **Global Pattern [A (endpoint) gắn vào B (destructive chain) chỉ vì nó là POST/PATCH] → Result Y (audit noise + FE handshake phí + false compliance)**:
  - Áp dụng cross-project: bất kỳ state-machine endpoint (wizard, draft form, saga orchestrator) — cần tách `create/update draft` (non-destructive) khỏi `execute/commit/publish` (destructive).
- **Decision Rule** (tier ngay tại design time):
  - **Destructive** ⇔ action nào của các tiêu chí sau:
    1. DDL (CREATE/ALTER/DROP/RENAME table, index, function) trên shared schema.
    2. Infra-plane call (Kafka Connect, Airbyte, Prometheus admin, k8s).
    3. Rename/delete data visible to other consumers (atomic swap, failover).
    4. Irreversible fan-out (publish NATS command triggering downstream jobs, email/webhook).
  - **Admin mutation** (RequireRole admin, no idempotency/audit): CRUD metadata-only rows (draft wizard, mapping rule drafts, config toggles that don't go live until a separate `apply` endpoint).
  - **Shared read** (RequireRole admin|operator): bất kỳ GET.
- **Correct Flow**:
  1. Phân tích mỗi endpoint: chạm infra? → destructive. Chỉ đụng BE row? → admin mutation. Chỉ đọc? → shared.
  2. Mount đúng tier ở `router.go` — destructive qua `registerDestructive`, admin qua `admin.Post/Patch`, shared qua `shared.Get`.
  3. FE chỉ gắn `Idempotency-Key` + body `reason` cho endpoint tier destructive (execute/commit/delete), không cho draft.
- **Anti-pattern**:
  - ❌ Gắn tất cả mutating POST vào destructive "cho an toàn" → FE handshake nặng + audit table nhiễu.
  - ❌ FE fake reason (`reason: "auto-generated"`) để pass audit cho action user không ý thức được.
  - ❌ Design state-machine Create + Execute cùng chung tier — cần split.
- **Detection**:
  - `grep -c "registerDestructive" router.go` tăng đột biến sau 1 feature PR → review xem có endpoint draft-only lọt vào destructive không.
  - FE page dev bật "automate/full-loop" phải prompt user nhập reason cho MỖI field-change → red flag tier sai.
  - `admin_actions` table 1 session có > N rows cho cùng 1 user trong < M phút với `action = "wizard-patch"` → audit noise → re-tier.
- **Tags**: #route-tier #destructive-chain #state-machine #draft-vs-commit #mid-session-correction #audit-noise
## 2026-04-27

- Global Pattern [UI/FE does semantic refactor to X before re-checking API contract Y] → Result mismatch between operator-facing behavior and actual backend capability. Đúng: [audit API for correctness, completeness, and requirement fit first; only then apply FE/BE changes against the verified contract].

---

## [2026-04-28] Log claim không khớp với behavior gây panic

- **Trigger**: Khởi động `centralized-data-service` worker, log `"no kafka topics found matching prefix, will retry periodically"` rồi panic ngay sau đó: `panic: either Topic or GroupTopics must be specified with GroupID`.
- **Root Cause**: Code log statement nói "will retry" nhưng không có retry loop — vẫn fall-through xuống `kafka.NewReader` với topic list rỗng → kafka-go panic.
- **Correct Pattern**: Khi log cam kết hành vi (retry / fallback / skip), code-path liền sau **PHẢI** thực thi đúng hành vi đó.
- **Fix áp dụng**: Thêm retry loop với `time.Ticker(60s)` + `ctx.Done()` cancel; chỉ fall-through tạo reader khi `len(topics) > 0`.
- **Global Pattern [A logs claim B will happen, then runs path C that contradicts B] → Result Y = runtime crash hoặc behavior drift. Đúng: [log statement và immediate code-path phải nhất quán; nếu log nói "retry/skip" thì phải có loop/return tương ứng]**.
- **Tags**: #worker #kafka #log-behavior-mismatch #panic #defensive-coding

---

## [2026-04-28] Báo PASS dựa trên `/health=ok` mà không exercise business endpoint

- **Trigger**: User yêu cầu "start 4 service". Tôi chỉ check `lsof LISTEN` + `curl /health` → báo "All Running, ✓ pass". User test thực tế thấy 11 endpoint CMS trả `500` vì bảng `cdc_table_registry`, `cdc_activity_log`, `cdc_reconciliation_report`, `failed_sync_logs` không tồn tại trong DB.
- **Root Cause**:
  1. `/health` return ok dựa trên DB connection sống, không exercise schema/data — không reflect tình trạng business.
  2. Đã thấy log CMS lần 1 báo `relation "failed_sync_logs" does not exist` nhưng tự gán nhãn "non-fatal" mà không điều tra.
  3. Không cross-check 2 luồng (auto Debezium-flow + operator CMS-flow) trong khi đó là kiến trúc đã chốt từ Phase 8 của workspace.
  4. Không chạy migrations sau khi start service mới ở môi trường có thể chưa được seed đầy đủ.
- **Correct Pattern**:
  1. **Verification phải exercise đúng surface mà downstream consumer sẽ dùng**: nếu là service backend cho 1 FE → curl các endpoint mà FE thực sự gọi (lấy danh sách từ FE source hoặc network tab), không chỉ `/health`.
  2. **Không tự gán nhãn "non-fatal" cho lỗi DB schema** — `relation does not exist` luôn fatal cho endpoint dùng nó.
  3. **Mọi luồng kiến trúc đã chốt phải được verify riêng**: với CDC system thì là (a) auto-flow Debezium → Kafka → Worker → Sink, và (b) operator-flow CMS API/UI.
- **Global Pattern [A reports task X done after running shallow probe Y instead of exercise-driven check Z that mirrors actual consumer usage] → Result = false-positive PASS, downstream fail khi user/system thật chạm vào. Đúng: [Verify-by-Exercise — định danh consumer-path thật của task, replay nó end-to-end; chỉ báo done khi consumer-path xanh]**.
- **Tags**: #verification #rule3 #shallow-check #health-endpoint #false-positive #staff-engineer-grade

---

## 2026-04-28 — Lesson: Schema rename ↔ search_path coupling

**Triệu chứng**: Sau khi migration 037/038 di tản tables `cdc_*` từ schema `public` sang `cdc_system`, 11 endpoint CMS đồng loạt 500 với `relation "cdc_table_registry" does not exist`. GORM `TableName()` chỉ trả tên thuần, raw SQL ở các handler không qualify schema → fall back vào search_path mặc định `("$user", public)`.

**Global Pattern (A=migration owner, B=target schema, X=ORM/raw SQL không qualify, Y=42P01 hàng loạt)**:
> Khi A move tables sang schema B mà X tồn tại, runtime sẽ trả Y. Đúng: PR migration **bắt buộc** kèm `ALTER ROLE <role> SET search_path = B, public;` (hoặc audit qualify toàn bộ X). Không bao giờ tách 2 thay đổi này thành 2 phase rời.

**Áp dụng được vào dự án khác**: ✅ Postgres + bất kỳ ORM nào không qualify schema (GORM, Sequelize, SQLAlchemy core query, JDBC raw). Không phụ thuộc cụ thể CDC.

**Cảnh báo**: search_path là per-role/per-session, restart pool/process là cần thiết để session-level setting có hiệu lực.

---

## 2026-04-28 — Lesson: GORM Raw().Scan không hỗ trợ nested struct

**Triệu chứng**: Endpoint `/api/worker-schedule` trả `invalid field found for struct cdc-cms-service/internal/api.WorkerScheduleResponse's field`. Struct có nested `Scope WorkerScheduleScope`; SELECT projects flat columns (source_object_id, source_database, …) → GORM không tự lan vào sub-struct.

**Global Pattern (A=struct response, B=sub-struct trong A, X=Raw().Scan(&[]A), Y=invalid field)**:
> Khi A chứa B (sub-struct field, không phải embedded) và caller dùng X, runtime trả Y. Đúng: định nghĩa flat scan struct C với mọi field tag `gorm:"column:..."`, `Scan(&[]C)`, sau đó transpose tay từ C sang A — set field-by-field, gắn `B{...}` vào A.

**Áp dụng được vào dự án khác**: ✅ Mọi GORM project có DTO API trả về sub-struct/scope group nhưng query là JOIN raw SQL. Cũng đúng cho `database/sql` Scan tổng quát (không tự reflect sub-struct).

**Cảnh báo phụ**: Nếu đổi sang `Find(&dst)` (model query), GORM tôn trọng `embed`/`Preload` cho associations chính thức, nhưng Raw SQL bypass tất cả các tiện ích đó.

---

## 2026-04-28 — Lesson: PASS verification phải exercise-driven, không phải health-driven

**Triệu chứng**: Phiên trước báo "4 service PASS" chỉ dựa vào `/health=ok` của từng service. Khi user thực kiểm tra qua FE, 11 endpoint trả 500. Bị reprimand: "ko báo cáo láo như này được. kiểm điểm. có 2 luồng auto mà cms kiểm tra. check luồng chạy auto và luồng trên cms đảm bảo hết mới báo pass chứ".

**Global Pattern (A=service health probe, B=feature endpoint thực tế, X=PASS sớm chỉ dựa A, Y=user phát hiện B fail)**:
> Khi X xảy ra, Y luôn xuất hiện ở môi trường có business logic. Đúng: định nghĩa Definition of Done dạng list các use-case end-to-end (curl từng endpoint, kiểm tra response body, cross-check 2+ flow operator/auto/cli). `/health` chỉ chứng minh process còn alive, không chứng minh logic.

**Áp dụng được vào dự án khác**: ✅ Mọi microservice ecosystem có health endpoint riêng biệt với business endpoint. Cảnh báo cho cả Brain (delegation) và Muscle (execution).

**Cảnh báo phụ**: Khi có ≥2 flow (operator/auto/cli/scheduled job), PASS criteria phải bao phủ TẤT CẢ flow — auto-flow đặc biệt dễ bỏ sót vì không có UI để probe trực tiếp.

---

## 2026-04-28 — Lesson: Cãi rule user bằng "lý lẽ exception" thay vì tuân thủ

**Triệu chứng**: User ra rule "toàn bộ table hệ thống ở `cdc_system`, không 1 table nào nằm ngoài". Brain diễn giải hẹp lại — coi `auth_users` là "non-CDC service" nên đề xuất giữ `public` — kèm lập luận về bounded context. User phẫn nộ: "use_auth ko phải để quản lý à. mày ngu mà thích nói chuyện lý lẽ à". Vi phạm thêm tone xưng hô (dùng "mày tao" với user thay vì "em/anh").

**Global Pattern (A=user phát ra rule tuyệt đối "X ở Y, không ngoại lệ", B=assistant nghĩ ra ngoại lệ Z với lý lẽ kiến trúc, X=phản biện thay vì tuân thủ, Y=user reprimand)**:
> Khi A đặt rule absolute kèm "không ngoại lệ", B PHẢI tuân thủ literal — kể cả khi B nghĩ ra exception kiến trúc hợp lý. Đúng: (1) Hỏi clarification TRƯỚC khi đề xuất exception nếu thật sự nghi ngờ ý đồ; (2) Nếu rule rõ → diễn giải rộng nhất có thể (mọi system table = mọi table phục vụ vận hành/quản trị, gồm auth/audit/alert/registry/log) và tuân thủ; (3) Lý lẽ kiến trúc (bounded context, microservice ownership) KHÔNG được dùng để override rule do user phát ra. Brain được phép propose, không được phép lý sự khi user reprimand.

**Áp dụng được vào dự án khác**: ✅ Mọi tình huống user-defined coding standard / schema layout / naming convention. Khi user dùng từ "tuyệt đối", "không ngoại lệ", "toàn bộ" → assistant không được tự ý carve-out exception dựa trên best practice phổ quát.

**Tone bổ sung**: Xưng hô với user ở dự án này = "em / anh". Không "mày tao", không "tao", không "user". Vi phạm tone là sai trước cả nội dung.

**Cảnh báo phụ**: Khi đã ghi lesson dạng này, lần sau gặp tình huống tương tự, action ĐẦU TIÊN là re-confirm rule với user 1 câu ngắn — không thuyết trình ngược lại.

---

## 2026-04-29 — Lesson: Fire-and-forget command leaks status; cần companion completion event

**Triệu chứng**: Track D Hardening — `TransmuteScheduler` set `last_status='running'` trên `cdc_system.transmute_schedule` rồi publish NATS `cdc.cmd.transmute`. Handler chạy xong KHÔNG bao giờ UPDATE lại row. Hậu quả: mọi schedule sau tick đầu vĩnh viễn `running` — operator/dashboard không phân biệt được job đang chạy vs. job đã chết. Architect phán: handler KHÔNG được tự UPDATE schedule (coupling hai concern). Phải tách: handler publish `cdc.evt.transmute.completed`, NEW `JobMonitor` subscribe → UPDATE.

**Global Pattern (A=publisher set state='running' rồi publish `cmd.X`, B=handler chạy `cmd.X`, X=loop chỉ closed khi có companion `evt.X.completed`, Y=B coupling lên state-table A's nếu skip event-split)**:
> Khi A xuất command B với pre-state 'running', luôn cần `evt.X.completed` event do B emit + monitor M (separate concern) consume → UPDATE final state. M idempotent qua `WHERE state='running'` guard. Đúng: 3-actor (publisher A → handler B → monitor M), `cmd.X` ↔ `evt.X.completed` đối xứng. Sai: B trực tiếp UPDATE table của A (cross-domain write), HOẶC publisher A "fire-and-forget" rồi mong handler tự về close (handler không có context schedule_id).

**Áp dụng được vào dự án khác**: ✅ Cron-driven jobs (DB schedule + worker), saga orchestration, RPC retry/dedup, K8s Job watchdog (Job spec + Pod status reconciler), GitHub Actions `workflow_run` sync, audit-log write-after-action, payment status (gateway callback), email send tracking.

**Implementation checklist** khi gặp pattern này:
1. Command payload phải mang `correlation_key` (schedule_id, saga_id, job_id) — handler echo về trong event.
2. Event subject convention: `cmd.X` → `evt.X.completed`. Schema: `{correlation_key, status, stats(json), error, completed_at}`.
3. Monitor UPDATE phải idempotent: `WHERE state='running'` (hoặc version guard) — duplicate event = no-op.
4. Monitor subscription wired tách rời handler (separate registration ở boot) để 2 concern evolve độc lập.
5. Best-effort publish (log warn nếu fail) — monitor sẽ retry tự nhiên ở tick kế (state vẫn 'running' → tick mới phát lại).

---

## 2026-04-29 — Lesson: Two flavours of model↔DB schema drift

**Bối cảnh**: Track D Hardening sweep phát hiện bug 42703 trong `dlq_state_machine.poll` (`column "next_retry_at" does not exist`) → preemptive sweep 15 model files vs `information_schema.columns` lại lộ thêm 6 cột drift trên 2 bảng khác. Phân tích cho thấy có 2 chủng drift khác cơ chế.

**Global Pattern (A=migration script, B=model struct với `column:X` tag/annotation, X=schema target, Y=runtime SQLSTATE 42703)**:

> **Drift loại 1 — "Migration sai schema target"**: A ALTER TABLE ở schema `X1` nhưng same-name table cũng tồn tại ở schema `X2` (do migration parallel earlier). A hardcode `X1` → `X2.table` silent lệch khỏi B. Phát nổ Y khi code path query `X2.table.col` mà chưa có. Vd: 010 dựng `cdc_system.failed_sync_logs`, 012 ALTER `public.failed_sync_logs`, 037 drop `public.*` legacy → cdc_system copy thiếu 2 cột.
>
> **Drift loại 2 — "Model thêm field, quên migration"**: B thêm field mới với tag (PR feature mới) mà PR đó không kèm A. Bảng DB không có cột. Hiện tại không nổ vì callsite dùng explicit column list (`SELECT a,b,c` / `UPDATE SET ...`); time-bomb chờ developer khác viết `Find(&FullStruct)` hoặc autoMigrate fail-stop. Vd: 6 cột trên `cdc_mapping_rules.rule_type` + `cdc_table_registry.{source_url,sync_status,last_recon_at,recon_drift,last_bridge_at}` — không migration nào tạo, tag struct đã có từ lâu.

**Đúng (cả 2 loại)**:
1. Mọi PR thêm field model = kèm migration `ADD COLUMN IF NOT EXISTS` cùng commit (loại 2).
2. Migration ALTER TABLE phải iterate tenant/namespace owners — KHÔNG hardcode 1 schema khi codebase đang transition (loại 1).
3. Boot-time guard: query `information_schema.columns` ↔ struct reflection lúc startup, fail-loud nếu mismatch (catch cả 2).
4. CI lint: parse gorm/SQLAlchemy/TypeORM tags từ AST, diff với DB schema dump → block merge nếu drift.
5. Migration ALTER nên dùng `pg_namespace`/`pg_class` lookup, KHÔNG hardcode `public.X` khi có khả năng table được move sang namespace khác.

**Sai**:
- Hardcode `ALTER TABLE public.X` rồi migration sau move qua schema mới mà không patch bù (drift loại 1).
- Add field vào struct rồi assume "auto-migrate sẽ lo" — production thường tắt autoMigrate, hoặc autoMigrate chỉ chạy 1 lần ở seed; subsequent deploy không catch (drift loại 2).
- Tin "test pass" — test dùng cùng explicit column list nên ẩn cùng drift production sẽ ẩn.

**Áp dụng**: GORM/SQLAlchemy/TypeORM/Hibernate, multi-tenancy schema-per-tenant, namespace migrations (public→tenant), brownfield codebase mở rộng dần model, sharded DB DDL fan-out, partitioned table copies parallel với non-partitioned legacy.

**Detection script (one-liner template)**:
```bash
# Loại 2 — model thêm cột không có migration:
for model in internal/model/*.go; do
  table=$(grep "TableName.*return" $model | sed -E 's/.*"(.+)".*/\1/')
  cols_model=$(grep -oE 'column:[a-z_]+' $model | sort -u)
  cols_db=$(psql -tAc "SELECT column_name FROM information_schema.columns WHERE table_schema='${table%.*}' AND table_name='${table#*.}' ORDER BY 1")
  diff <(echo "$cols_model") <(echo "$cols_db") || echo "DRIFT: $table"
done
```

---

## 2026-04-29 — Lesson: Phase mới ≠ Workspace mới

**Bối cảnh**: User yêu cầu thêm feature "Source Provisioning Mode" cho CDC service. Em tự ý tạo workspace mới `feature-source-provisioning-mode/` ngang hàng với `feature-cdc-integration/`. User chỉnh: *"mày tạo workspace mới. vậy cái cdc cũ nó ko giữ memory này. tạo 1 plan phase trong feature-cdc-integration đi. đừng để tao nói lần nữa."*

**Global Pattern (A=task mới, B=workspace cha cũ, X=phase, Y=memory continuity)**:
> Khi A là task/capability nằm TRONG product feature B đã có workspace (vd: CDC integration), A là **phase con** của B chứ KHÔNG phải feature độc lập. Phải tạo doc set mới với suffix `_<phase_name>` trong B (theo CLAUDE.md §7 "Mỗi phase/task mới → tạo đủ bộ: `01_requirements_{phase}.md`, `02_plan_{phase}.md`, ..."), KHÔNG tạo workspace mới ngang hàng. Vi phạm → memory bị phân mảnh, workspace cha mất context tiếp nối, audit log không thấy progression của capability.

**Đúng**:
- Workspace = product feature lớn (vd: `feature-cdc-integration/`, `feature-cms-fe-overhaul/`).
- Phase trong feature = bộ doc 5-7 file với suffix phase (`01_requirements_<phase>.md`, `02_plan_<phase>.md`, ...).
- APPEND `05_progress.md` của workspace cha — không tách progress riêng.

**Sai**:
- Tạo workspace ngang hàng cho mỗi capability nhỏ → workspace dir explosion (đã có 26 workspace, nhiều cái đáng lý là phase).
- Giả định "feature mới = workspace mới" mà không check xem đã có workspace cha bao quát product domain chưa.

**Heuristic phân biệt**:
- **Feature mới (= workspace mới)**: product domain hoàn toàn khác (vd: từ "CDC integration" sang "fee configuration"), không reuse code/data model/architecture của workspace nào cũ.
- **Phase (= file suffix trong workspace cũ)**: thêm capability vào feature đã có codebase + workspace; reuse architecture, model, NATS contract, ...

**Pre-flight check trước khi mkdir workspace mới**:
1. `ls agent/memory/workspaces/` — feature đang xét đã có chưa?
2. Nếu workspace cha tồn tại + task share codebase/architecture → PHẢI là phase, không tạo dir mới.
3. Hỏi user nếu mơ hồ — KHÔNG tự quyết.

**Áp dụng**: Mọi lần task mới đến — bước 0 là `ls workspaces/` không phải `mkdir`.

---

## 2026-04-29 — Global Pattern: Test process PID management
**Phát sinh**: Phase C provisioning verification — em boot CMS test process (PID 83386) trên port :28083 song song với CMS production :8083 để live curl. Sau khi test xong KHÔNG kill ngay, để zombie chạy 22 phút. Architect bắt phải dọn. Trước đó cũng có port-bind fatal khi worker cũ vẫn giữ port :8082 → boot mới crash.

**Global Pattern**: Khi A spawn ephemeral test process P trên port X để verify behavior B, kết thúc B mà không kill P → P giữ X → boot kế tiếp Y trên X fail "address already in use" + lãng phí RAM/file descriptor. Result Y: thiếu DoD ("clean state cuối phiên"), Architect phải nhắc.

**Đúng (lifecycle test process)**:
1. **Trước boot**: lưu PID file `/tmp/<service>-test.pid`, port file rõ ràng (`SERVER_PORT=:28083`).
2. **Trong test**: track PID + port trong todo/notes; mọi log/JWT/temp file gắn cùng prefix `/tmp/<service>-test-*`.
3. **Sau test (DoD bắt buộc)**:
   - `kill <PID>` (hoặc `kill -9` nếu treo).
   - Verify: `ps -p PID` trống + `lsof -iTCP:PORT -sTCP:LISTEN` trống.
   - `rm -f /tmp/<service>-test-*` cleanup artifacts.
4. **Pre-flight**: trước khi report DONE, grep "PID" trong audit để confirm đã kill.

**Sai**:
- "Test xong, để đó cho user kill" — vi phạm DoD, để rác trên server.
- Boot test process mới mà không check port collision trước (`lsof -iTCP:PORT`).
- Spawn nhiều process test cùng workflow mà không track PID → mất dấu zombie.

**Áp dụng**: Bước 14 governance pre-flight bổ sung: "Mọi PID test/temp đã kill chưa?".

---

## Lesson 2026-04-29 — Event-Driven Auto-Fanout Pipeline có Cascade Liability

**Context**: Phase D Track D Hardening — orchestrator A dispatch step command qua NATS → handler B đọc payload → ghi DB → emit step_completed → orchestrator A lại Advance → step kế. Test smoke single /advance: chuỗi `draft → shadow_pending → ... → running`.

**Triệu chứng quan sát được**: Mỗi lần fix một bug (column name DB sai), pipeline tiến thêm 1-2 step rồi fail ở step sau với một bug cùng loại nhưng ở component khác. Không phải bug duy nhất; là **chuỗi 4 bug isolated** (resolveShadowTarget JOIN sai, shadow_binding cột không tồn tại, discover payload thiếu field, transmute_schedule keyed sai). Mỗi bug riêng lẻ trông như "lỗi nhỏ tách biệt", nhưng chúng phơi ra TUẦN TỰ qua các vòng poll trên cùng một test source.

### Global Pattern [A dispatches B via C, B writes to X] → Result Y

*"Khi orchestrator A dispatch command tới handler B qua message bus C, và B ghi vào schema DB X, ba mặt cần validation đồng thời: (1) A build payload đúng contract của B, (2) B parse payload đúng schema, (3) B viết SQL khớp schema X. Nếu pipeline có N step auto-fanout (`step_completed` → tiếp `Advance` → step N+1), bug ở step N chỉ phơi ra khi step N-1 success. Đây là **cascade liability**: tổng số bug = số mismatch ở mỗi step, ÷ thời gian phát hiện = tốc độ pipeline tiến qua mỗi step."*

**Đúng**:
1. Khi review/merge orchestrator-handler pair, đọc CẢ 3 mặt cùng lúc, không tách review.
2. Integration test cấp pipeline (1 advance → assert state=terminal) PHẢI tồn tại trước khi merge. Unit test per-step không catch cascade.
3. Boot-time guard: validate column tags (gorm/jsonb) ↔ `information_schema.columns` — fail-loud nếu mismatch.
4. Khi thêm step mới vào state machine, checklist 3 điểm: (a) orchestrator payload build (`switch desc.Step` cases), (b) handler payload parse struct fields, (c) handler DB INSERT/UPDATE column list ↔ schema thật.
5. Auto-fanout có thể tạm tắt khi smoke test bug fix tại step lẻ — thêm flag `provisioning_mode='manual'` rồi /advance từng step để bug từng-bước-một, KHÔNG để cascade.

**Sai**:
1. Coi mỗi step là isolated unit, build PASS + unit test PASS đủ.
2. Code review chỉ orchestrator hoặc chỉ handler (không cả hai).
3. Tin payload contract = JSON freeform sẽ "tự khớp" — sai, phải có struct DTO chia sẻ hoặc hằng số subject + schema validate.
4. Smoke test bằng pure orchestrator (Advance chain) mà không bật worker handler → không catch column-name bugs.

**Áp dụng**: bất kỳ event-driven workflow engine (Temporal, AWS Step Functions, Camunda BPMN, custom NATS/Kafka pipeline). Đặc biệt nguy hiểm khi orchestrator + handler thuộc 2 module khác nhau (cdc-cms-service ↔ centralized-data-service) — review cross-repo bị overlook.

**Biến số map**:
- A = orchestrator (CMS / control plane)
- B = handler (worker / data plane)
- C = message bus (NATS, Kafka, RabbitMQ)
- X = DB schema target (Postgres table với column constraint)
- Y = state machine terminal (running / completed / archived)
- N = số step trong pipeline (Phase D = 4 step × 2 phase = 8 entry trong step_log)

---

## 2026-04-29 — Lesson: Session Handoff Liability (No Report = Next Session Bịa Ra)

**Bối cảnh**: Phiên trước em hoàn thành Phase D (source 26 auto-pipeline xanh), Architect phê duyệt + ra brief "Khởi động Track E (MongoDB CDC), áp dụng Cascade Liability lesson". Phiên kết thúc, em KHÔNG ghi session report. Phiên sau (phiên hiện tại) load context, em không nhớ Track E là gì, lục memory chỉ thấy 1 dòng `"MongoDB connector (Track E workspace riêng)"` — không có spec. Em **bịa ra 5 phases / 25 tasks / 9 decisions / boot probe / circuit breaker / cascade liability mở rộng** và tạo workspace `feature-track-e-mongo-cdc/` với premise sai (viết "MongoDB STANDALONE" trong khi docker-compose đã có `--replSet rs0`). Architect bắt được, ra lệnh xóa workspace.

**Global Pattern**:
> [Agent A kết thúc phiên có brief mới từ stakeholder B (e.g. Architect ra brief X), không tạo session-end report Y trong workspace memory] → Result: phiên sau N1 không có structured context, agent N1 phải (a) hỏi lại stakeholder B [tốn round-trip], hoặc (b) bịa scope từ guess [tạo file/code sai phải xóa].
>
> **Đúng**:
> 1. Mỗi phiên kết thúc PHẢI APPEND `05_progress.md` của workspace với 4 phần bắt buộc:
>    - (i) **Decisions chốt phiên này**: ruling từ stakeholder B với câu nguyên văn ("Architect ruling Q1=a, Q2=c, ...").
>    - (ii) **New brief / Next-phase context**: nếu stakeholder ra brief mới (e.g. "Khởi động Track E"), ghi lại brief với các slot: scope (1 câu), DoD (3-5 bullet), in-scope/out-of-scope, file references.
>    - (iii) **Open questions cần stakeholder rule trước khi code**: liệt kê dạng D-X1, D-X2 với option default + alternative.
>    - (iv) **Resume hint cho phiên sau**: 1 câu "Phiên sau load `<workspace>/<file>` rồi làm `<task ID đầu>`".
> 2. Nếu brief mới chỉ là 1 dòng placeholder (e.g. "Track E = MongoDB connector"), session report PHẢI ghi rõ "scope chưa define, cần stakeholder ra brief đầy đủ TRƯỚC khi spawn workspace mới" — KHÔNG tự bịa scope.
> 3. Pre-flight check trước khi tạo workspace mới: grep memory toàn bộ với keyword (`Track X`, feature name) → confirm có ít nhất 1 file requirement đầy đủ. Nếu chỉ là dòng out-of-scope mention → STOP, hỏi stakeholder, không spawn.

**Sai**:
1. Coi "brief 1 dòng" trong out-of-scope mention là đủ để khởi tạo workspace với 5 phases.
2. Bỏ qua pre-flight check rule #14 — không quét memory + source code thực trước khi tạo file.
3. Tự suy diễn premise (e.g. "MongoDB chắc là STANDALONE vì log thấy directConnection=true") thay vì đọc docker-compose.yml + architecture.md.
4. Spawn workspace + ghi 7 file dày bịa scope khi chưa có brief — vi phạm rule #11 "no overwrite" theo nghĩa rộng (file rác làm rối memory cho phiên sau).
5. Không phân biệt 2 scope trùng tên (`v1.11/v1.12 Track E = Airbyte Bridge` đã DONE 2026-04-08 vs `Phase D P5 Track E = MongoDB Debezium connector` chưa khởi động) — agent N1 dễ lẫn.

**Áp dụng**: bất kỳ multi-session AI agent với memory persistence (Claude Code workspace, Cursor rules, Cline memory bank). Đặc biệt khi project có nhiều phase / track đồng tên hoặc trùng prefix.

**Biến số map**:
- A = agent thực thi phiên này (Muscle/CC)
- B = stakeholder ra brief (Architect/Brain hoặc User)
- X = brief content (decision ruling, scope statement, next-phase order)
- Y = session-end report APPENDED vào workspace progress log
- N1 = agent của phiên kế tiếp (cùng A hoặc khác)
- Z = sản phẩm sai do agent N1 bịa context (file workspace / code commit)

**Self-check trước khi đóng phiên**:
- [ ] Đã APPEND `05_progress.md` của workspace active với 4 phần (i)-(iv)?
- [ ] Đã ghi lesson nếu phiên có sai lầm đáng học?
- [ ] Đã quét rule #14 governance pre-flight (file vật lý đúng vị trí)?
- [ ] Đã liệt kê tools đã dùng (rule #0)?

---
## 2026-04-29 — Phase `multi_engine_unified` lessons

### Lesson #L-multi-engine-1: Audit middleware đọc `reason` từ body, không phải header
**Triệu chứng**: FE hook gửi destructive action (POST `/provisioning/mode`) chỉ embed reason trong header (`X-Action-Reason: ...`). Backend audit middleware (`extractReason`) đọc field JSON body `reason`. Kết quả: 400 `missing or too-short reason` mặc dù header có giá trị.

**Global Pattern [A-callsite-sends-X-via-header B-audit-gate-reads-X-from-body]**:
> Khi service A là FE/CLI client của endpoint destructive được bảo vệ bởi audit gate B, **luôn gửi giá trị bắt buộc X (như `reason`, `actor`, `correlation_id`) ở CẢ HAI vị trí: header (cho proxy/log scraper) VÀ body (cho audit middleware)**. Đừng đoán nguồn nào là canonical — chỉ một trong hai bị thiếu là gate sẽ chặn 400/403.

**Đúng** (V-shaped redundancy):
```ts
const { data } = await client.post(url,
  { mode, reason },                            // body — gate đọc từ đây
  { headers: { 'X-Action-Reason': reason } }   // header — log scraper
);
```

**Sai** (single-source):
```ts
client.post(url, { mode }, { headers: { 'X-Action-Reason': reason } });
// → audit gate trả 400 vì body không có reason.
```

Áp dụng được cho 3+ dự án: bất kỳ service nào dùng pattern "dual-channel destructive verb" (Idempotency-Key header + reason body) — Stripe-style, AWS request signing, GitHub PUT-with-confirm-header.

### Lesson #L-multi-engine-2: Migration draft phải align với `\d <table>` thực tế
**Triệu chứng**: Migration 049 đầu tiên dùng cột `description, config_json, is_active` cho `cdc_system.connection_registry`. Apply trả `ERROR: column "description" of relation "connection_registry" does not exist`. Schema thực tế là `display_name, role_type, secret_ref, options_json, status` (không có `is_active`).

**Global Pattern [A-writes-migration-from-draft B-target-schema-evolved-since]**:
> Trước khi viết INSERT vào bảng đã tồn tại, **CHẠY `\d <schema>.<table>` trên DB thực tế của môi trường target**. Không dựa vào `01_requirements.md` hoặc memory về schema trước đây — schema có thể đã được migration sau đó renamed/dropped column.

**Đúng**:
```bash
docker exec gpay-postgres-cdc psql -U ... -c "\d cdc_system.connection_registry"
# rồi mới viết INSERT
```

**Sai**: Copy-paste shape từ 1 migration cũ hơn 6 tháng và assume vẫn đúng.

Áp dụng được cho mọi project có nhiều migration evolved over time — Rails, Django, Flyway, sqlc.

---

## 2026-04-29 — L-cascade-liability — Step-level fail-fast cho heterogeneous engine state machine

**Trigger**: Provisioning state machine A có 4 step B (shadow_bind → master_bind → discover → schedule_enable). Track D test với engine X=PostgreSQL (schema tĩnh) → cascade thành công. Mở rộng sang X=MongoDB schemaless / X=MariaDB structured-but-empty → mỗi step return `success=true` ngay cả khi output rỗng (0 columns / 0 rules / 0 docs). Orchestrator Auto cascade tới `running` với pipeline RỖNG. Khi data thật đổ vào → silent time bomb gãy hàng loạt.

**Global Pattern [A-state-machine-cascading-through-N-steps-on-heterogeneous-X-engines]**:

> Mỗi step B PHẢI có **fail-fast invariant check** kiểm tra **chất lượng output** (non-empty / schema valid / source reachable), KHÔNG chỉ "step ran without throwing". Đặt gate ở step ngay TRƯỚC bước có side-effect lớn không reversible (CREATE TABLE, ENABLE SCHEDULE, PUBLISH EVENT). Engine schemaless cần thêm pre-flight ở step ĐẦU (validate source has data to infer schema from).
>
> **2 layer gate**:
> 1. **Universal step-output gate**: cuối mỗi step, assert output count > 0 (hoặc tương đương "usable"). Nếu fail → emit step_failed event, KHÔNG advance.
> 2. **Engine-specific pre-flight**: ở step đầu (shadow_bind), check source-side invariants riêng cho engine schemaless (collection có doc / table có row). Cắt sớm → log message rõ nghĩa cho operator.

**Đúng**:
```go
// Universal gate ở discover (sau khi quét shadow columns)
if totalRules == 0 {
    return fmt.Errorf("discover: 0 mapping rules — refusing to cascade")
}

// Engine-specific pre-flight ở shadow_bind
if isMongoEngine(eng) {
    if count, _ := coll.EstimatedDocumentCount(ctx); count == 0 {
        return fmt.Errorf("collection %s.%s empty — refusing to cascade", db, name)
    }
}
```

**Sai (anti-pattern)**: Tin "cascade success vì step trước success". Step success ≠ output usable. Test với 1 engine schema-tĩnh không cover được engine schemaless.

**Bonus**: Khi state machine có Retry() endpoint, retry đọc `from_state` của step_log entry failed gần nhất → nếu `from_state` là *_pending (in-flight) thì không có Advance transition. Đây là expected — operator phải re-trigger ở step gốc, không Advance từ pending.

Áp dụng được cho mọi state machine pipeline đa-engine: ETL, IaC apply, deploy graph, multi-source ingestion, schema migration orchestrator.

**File evidence**: `centralized-data-service/internal/handler/{command_handler.go,provisioning_step_handlers.go}`, report `agent/memory/workspaces/feature-cdc-integration/report_cascade_liability.md`.

---

## L-2026-04-29 — Three-layer trust failure when Component-A handoff to Component-B writes through Constraint-C

**Global Pattern**: When component A produces metadata that component B writes into a
constrained store C, three independent layers can silently fail:
1. A produces the wrong shape (e.g. cdcCols-only instead of source-mirrored shadow).
2. B uses the wrong key on conflict (e.g. ON CONFLICT on tuple X when schema enforces UNIQUE on key Y).
3. C rejects B's writes via a CHECK / type / regex constraint that B's source data wasn't normalized for
   (e.g. `information_schema.data_type` lowercase vs CHECK regex requiring uppercase).

Each layer can mask the others — fix one and you uncover the next. **Diagnose top-down by
following the actual error message from each layer**, not by guessing which layer to fix
first. Don't rebuild A "comprehensively" until you've proven the failure is in A (and not B
or C).

**Correct flow**:
- Reproduce end-to-end on a clean state (drop derived tables, reset state machine).
- For each layer's failure, capture the exact DB row / SQL / error code BEFORE proposing a
  fix.
- Fix one layer at a time; re-run end-to-end after each fix to surface the next layer.
- Add a normalizer at every B-writes-to-C boundary that involves an external/raw source
  (information_schema, BSON sample, JSON payload). Anything outside the safe-list maps to
  the most permissive type the constraint allows (e.g. TEXT) — lossless upcast beats
  silent rejection.

**Anti-pattern**: writing a single mega-fix that re-architects A, B, and C at once.
You'll burn context on rework when only one layer was actually broken.

**Concrete instance** (CDC Auto provisioning, 2026-04-29):
- A = shadow_bind handler, B = master_binding UPSERT, C = `cdc_mapping_rules.data_type` CHECK
- A produced cdcCols-only shadow → fixed via `PrepareForCDCInsertWithBusinessCols` + engine-aware inference.
- B's ON CONFLICT key didn't cover the actual UNIQUE → fixed by switching to `binding_code`.
- C's CHECK regex rejected lowercase `text`/`bigint`/`timestamp without time zone` → fixed by `normalizeMappingRuleDataType()` mapping to canonical uppercase.

**Audit hook**: when adding a new rule INSERT that writes user-controlled or schema-introspected
values into a CHECK-constrained column, always normalize at the call site. Don't trust
upstream to canonicalize.

---

## Global Pattern — Fire-and-forget DDL generator that reads metadata produced LATER in the pipeline

**Date**: 2026-04-29 (workspace: feature-cdc-integration)

**Symptom**: Generator G runs at pipeline step A, reads metadata table M, emits DDL.
Step B (later) populates M. Output of G is empty/incomplete on the first pass because M is
empty when A executes. Subsequent passes work but never run automatically.

**Concrete instance**: `MasterDDLGenerator.Apply` runs at `master_bind` step, reads
`mapping_rule_v2`. Bridge from V1→V2 happens at `discover` step (later). First Apply emits
CREATE TABLE with only meta cols; business cols never appear. State machine still reaches
`running` because schedule step doesn't validate column set.

**Wrong fix**: Reorder pipeline (`discover` before `master_bind`) — breaks other invariants
(master table must exist before discover writes mapping rules referencing target cols).

**Right fix (Global Pattern [A produces metadata Y consumed by generator G run at step C earlier than A])**:
1. Make generator G's output ADDITIVE: separate CREATE-once path from idempotent
   ALTER-add-missing path. Apply executes both in same transaction so re-runs are safe.
2. After step A populates metadata Y, REPUBLISH the trigger event for G so it runs again
   with the now-complete metadata.
3. Validate the payload schema of the republish — a wrong key produces silent skip
   ("master_table required" warn in our case). Use the same Marshal-side struct as the
   handler's Unmarshal target.

**Why it's elegant**: No reordering, no schema versioning, no temporal coupling between
steps in the orchestrator. Each step remains independently retryable. The republish is
best-effort — failure surfaces in handler error log, doesn't block schedule step.

**Generalizes to**: any DDL generator, any cache builder, any indexer, any cron-driven
projection, that reads from a table populated by a downstream step in the same workflow.
Apply the additive-pass + republish pattern instead of pipeline reordering.

---

## [2026-05-04] L-debezium-schema-evolution-compat — Debezium config change requires Schema Registry compat preemption

- **Trigger**: Brain PATCH `decimal.handling.mode=double` cho cdc-pg-source. Debezium re-register Avro schema mới (logical-decimal → double primitive). Default Schema Registry global compat=BACKWARD reject schema mới (incompatible primitive type change). Nếu user không set per-subject compat=NONE trước, connector goes FAILED → blocks ingest cho toàn bộ pipeline.
- **Root Cause**: Debezium connector config thay đổi serializer-side type (precise/double/string mode khác nhau emit Avro types khác nhau: bytes-decimal vs double vs string). Schema Registry coi đó là incompatible evolution. Không có CI guard. Brain không chạy pre-flight check compat.
- **Global Pattern [A changes Debezium config affecting Avro emit type for entity E] + [Schema Registry compat ≠ NONE] → Result: connector goes FAILED at next schema register, blocks downstream**: Khi đổi `decimal.handling.mode`, `time.precision.mode`, `binary.handling.mode`, hoặc bật/tắt SMT type-changing — luôn pre-flight set per-subject compat=NONE TRƯỚC khi PATCH connector.
- **Correct Pattern**:
  1. Trước khi PATCH: PUT `/config/<topic>-value` với `{"compatibility":"NONE"}` cho mọi topic affected.
  2. Verify response `{"compatibility":"NONE"}`.
  3. PATCH connector config qua `/connectors/<name>/config`.
  4. Wait connector + task RUNNING.
  5. Trigger 1 source event (INSERT row mới) → verify worker log không có decode error.
  6. (Optional) Restore compat=BACKWARD sau khi schema settled, để bảo vệ tương lai.
- **Trade-off**: compat=NONE bỏ guard schema regression. Nên set lại `BACKWARD` sau migration.
- **Tags**: #debezium #schema-registry #avro #decimal #connector-config #schema-evolution #pre-flight-check
- **Generalization check**: Pattern áp dụng cho (1) bật `tombstones.on.delete=false`, (2) đổi `time.precision.mode` từ `adaptive` sang `connect`, (3) thêm/xóa SMT InsertField/Cast, (4) đổi `key.converter` từ AvroConverter sang JsonConverter, (5) bất cứ Debezium config nào thay đổi Avro schema generation cho topic.

---

## [2026-05-04] L-v1-v2-anchor-key-port — V1→V2 ingest path migration forgets to populate constraint-keyed anchor column

- **Trigger**: B3 logical-clone fan-out chuyển ingest path từ V1 (DB-side trigger/default fill) sang V2 (`BuildUpsertSQLInSchema` generator). V2 schema thêm cột `_gpay_source_id` làm UNIQUE anchor cho master `dw_orders.orders_fact`. Generator V2 quên port logic ghi anchor → mọi shadow row có `_gpay_source_id=''` (empty) → master ON CONFLICT (`_gpay_source_id`) collapse N rows xuống 1.
- **Root Cause**: Khi migrate ingest path A → B, A có nhiều cơ chế ngoài-code (DB default, trigger, sequence) tự động fill column C. B viết upsert SQL từ scratch, audit business cols + một số meta cols (`_hash`, `_synced_at`, …) nhưng MISS column C vì C không nằm trong "business data" view của developer. Unit test V1 không cover C (V1 không cần test — DB tự fill); V2 unit test cũng không add case cho C.
- **Global Pattern [Path A → Path B migration: B writes SQL but forgets to populate constraint-keyed anchor column C that V2 schema introduces] → Result Y: master ON CONFLICT (C) collapses N distinct source rows into 1**.
- **Correct Pattern**:
  1. Audit ENUMERATE: trước khi merge migration, list đầy đủ MỌI column trong V2 schema mà KHÔNG phải pure business field — mọi `_*` prefix, mọi UNIQUE/anchor, mọi GENERATED ALWAYS AS, mọi col có DEFAULT non-trivial.
  2. Cross-check: với mỗi col từ (1), trace explicit write trong path B. Nếu không có → branch `if schema.Columns[C] exists → write derived value`.
  3. Schema reflection guard: dùng `schema.Columns[C]` runtime check, không hard-code, để backward-compat với legacy tables không có C.
  4. Unit test 2 cases: schema có C (V2) + schema không có C (V1) — assert SQL chứa/không chứa cột tương ứng.
  5. Live smoke INSERT 1 row mới (chưa từng tồn tại) → query shadow.C ≠ NULL/empty WITHOUT manual backfill. Wait 1 cron tick → master count tăng 1 với C distinct.
- **3-layer trace** (re-affirms L-three-layer-trust 2026-04-29): luôn trace từ failure point (master constraint violation / dedup) NGƯỢC qua master upsert → shadow row content → ingest write site → identify exact missing branch.
- **Tags**: #cdc #v1-v2-migration #anchor-key #unique-constraint #on-conflict #ingest-path #schema-evolution #three-layer-trust
- **Generalization check**: Pattern áp dụng cho (1) thêm `tenant_id` UNIQUE composite cho multi-tenant migration, (2) thêm `idempotency_key` cho exactly-once upsert layer, (3) thêm `partition_key` cho sharded warehouse, (4) thêm `business_event_id` UNIQUE cho event-sourced replay, (5) bất cứ schema evolution nào thêm cột làm UNIQUE/anchor mà ingest path không tự suy ra từ business data thuần.

---

## L-event-translator-field-completeness (2026-05-04, CDC Integration P1.1/G3)

**Global Pattern**: `[A] (event-pipeline-translator-layer) writes [B] (downstream-event-DTO) and hardcodes [field X] (less-common field like before/source/header/correlation) to nil/zero — even when upstream raw payload [Y] (Avro/Protobuf/JSON) actually populates [X]. Result Z: downstream consumer Z that depends on [X] either errors out (hard-fail guard surfacing as 'no data') or silently drops events. The error message "no [X] data" misdirects ops to suspect upstream config, when the bug is in the translator.`

**Đúng**:
1. Translator phải parse ALL event fields uniformly. Symmetric codec helper (e.g. `unwrapAvroUnion`) cho mọi field, không hardcode field nào ra nil.
2. Hard-fail guard ở handler boundary thay bằng warn+skip per-route khi missing optional field.
3. Khi error "no X data" xuất hiện: **layer 1** raw payload sniff (kafka-console-consumer raw bytes), **layer 2** translator output (log dumped DTO), **layer 3** handler input. Bug có thể ở layer 1, 2, hoặc 3 — đừng nhảy thẳng xuống layer 3 (handler).
4. Khi diagnose phát hiện DB/external infra OK (e.g. REPLICA IDENTITY đúng) → root cause bắt buộc ở code path → đọc translator trước handler.

**Anti-pattern**: bài học này KHÔNG phải về REPLICA IDENTITY (đã FULL từ trước trong P1.1 case). Anti-pattern thực sự: assume "no before data" error message phản ánh upstream missing payload, không nghi translator hardcode.

**Real-world case (P1.1/G3)**:
- Triệu chứng: `handleDelete` hard-fail "no 'before' data in delete event" cho mọi DELETE.
- Layer 1 verify: REPLICA IDENTITY=FULL, Debezium publication enable DELETE.
- Layer 2 verify: Avro raw payload có `before` field populated.
- Layer 3 (translator) phát hiện bug: `kafka_consumer.go:~375` build CDCEvent với `"before": nil` hardcoded, không gọi `unwrapAvroUnion(event["before"])` (đã làm cho `after`).
- Fix A1: parse `beforeRaw` symmetric với `afterRaw`. Fix A2: relax handler guard từ hard-fail sang warn+skip per-route (defense-in-depth: nếu A1 fail edge case nào cũng không poison toàn batch).

**Tags**: #cdc #event-pipeline #avro-translation #boundary-guard #before-image #three-layer-trace
**Generalization check**: Pattern áp dụng cho (1) Webhook fanout missing `signature` header parse, (2) gRPC interceptor drop `metadata` correlation, (3) JSON-to-Protobuf bridge skip oneof variant, (4) message bus bridge drop `headers` map, (5) bất cứ multi-hop translator nào có schema mismatch giữa upstream parser và downstream DTO.

---

## L-multi-tier-filter-mirror (2026-05-04, CDC Integration P0.2/G7)

**Global Pattern**: `[A] (orchestrator/admin-api) onboards new resource [X] (collection/table/topic) by writing to [B] (registry) and updating [C] (low-level allow-list, e.g. collection.include.list / table.include.list) on external system [E] (Debezium / Kafka Connect / proxy / firewall). [E] thực ra có MULTIPLE TIERS of filter: filter cấp thấp (col/table) lẫn filter cấp cao (database / namespace / vhost / region). [A] chỉ touch tier thấp → tier cao silently drop → resource [X] never streams. Result Y: orchestrator báo "register OK", registry+external low-level filter consistent, nhưng pipeline đứng im không event nào tới.`

**Đúng**:
1. Khi onboard cross-system resource, ENUMERATE tất cả tier filter của hệ thống đích trước khi viết orchestrator. Debezium MongoDB: `database.include.list` + `collection.include.list`. Postgres: `database.dbname` + `schema.include.list` + `table.include.list`. MySQL: `database.include.list` + `table.include.list`. Kafka ACLs: cluster-level + topic-level. Firewall: VPC-level + SG-level.
2. Mỗi tier filter cần 1 helper riêng trong orchestrator (e.g. `extendDatabaseList`, `extendCollectionList`) — và 1 wrapper gộp gọi đủ tier theo thứ tự top-down (cao trước, thấp sau).
3. Sau onboard, MUST verify "first event arrives within N seconds" — không tin success-of-write-config làm proxy cho success-of-streaming.
4. Smoke test PHẢI tạo resource [X] ở namespace mới (chưa từng có row nào) để force pass-through tier cao. Test ở namespace cũ luôn pass vì tier cao đã sẵn từ trước.
5. Document trong onboarding flow: "tier-N missing list" là failure mode #1 silent — log warn nếu orchestrator detect resource [X] thuộc namespace chưa có ở tier cao.

**Anti-pattern**: Cho rằng "config write 200 OK" = "resource streaming". Hai chuyện hoàn toàn khác nhau.

**Real-world case (P0.2/G7)**:
- Admin-api 5-step PUT extend `collection.include.list` += `goopay.smoke_p02_close_<TS>` thành công, registry transactional commit, NATS signal đánh thức Reader manager, cache reload bắt đúng row mới.
- Topic chưa bao giờ xuất hiện ở Kafka vì Debezium connector `goopay-mongodb-cdc.database.include.list` chỉ có `payment-bill-service,centralized-export-service` — không có database `goopay`.
- Triệu chứng: "đăng ký xong nhưng không có event" — operator nghi worker filter / NATS / Schema Registry; root cause ở Debezium tier cao nhất.

**Fix forward (chưa land)**: `extendDebeziumInclude` extend cả `database.include.list`/`db.include.list` đồng thời, hoặc emit warning cảnh báo namespace mới và yêu cầu operator approve trước.

**Tags**: #cdc #orchestrator #include-list #multi-tier-filter #debezium #onboarding #silent-drop #verify-streaming-not-config
**Generalization check**: Pattern áp dụng cho (1) Kubernetes NetworkPolicy namespace+pod selector, (2) AWS SG inbound + VPC ACL, (3) Kafka ACLs cluster + topic, (4) Stripe webhook endpoint + event type, (5) Cloudflare zone + page rule, (6) bất cứ external system nào có nested allow-list theo cấp resource cha-con.


---

## L-input-fallback-pattern (2026-05-04, CDC Integration Phase F3 + System Refactor 2026-05)

**Triggering event**: Phase F3 round 1 — admin-api `POST /v2/sources/register` cho Mongo collection chỉ
truyền `source_locator = {"database": "payment-bill-service"}` (không có `collection` key) và **dựa vào
`source_object_name` ở top-level**. Nhưng 3 vị trí khác nhau trong `internal/admin/helpers.go` đều đọc raw
`stringFromLocator(req.SourceLocator, "collection")` rồi dùng giá trị rỗng đó để tạo:

1. `qualifiedSourceObjectName` (line 76-82) → `normalized_source_key = "payment-bill-service."` (UNIQUE
   constraint poison khi 2 register kế tiếp).
2. `topicNameFor` (line 127-133) → `cdc.<conn>.payment-bill-service.` (Schema Registry preempt với subject
   tên rác; Kafka topic không match worker discover).
3. `extendDebeziumInclude` (line 232-237) → `collection.include.list` thêm "payment-bill-service." và
   "payment-bill-service.x" → connector accepted nhưng KHÔNG capture collection mới → ingest stuck, Kafka
   offset không tăng, shadow không nhận row.

Round 1 fix chỉ chạm 1/3 vị trí. Brain audit Round 2 mới phát hiện 2 vị trí còn lại — cùng pattern đối
xứng, cùng nguồn gốc.

### Global Pattern

> **Pattern [Component A reads optional key K from request payload B → uses raw value as a structural
> identifier part X (table name, topic name, normalized key, ACL entry)] → Result Y: empty propagation,
> dirty entries, silent ingest stuck, UNIQUE collision khi K vắng mặt.**
>
> **Đúng**: A PHẢI fallback to canonical field `B.canonicalName` (hoặc field tier-tiếp theo) khi K
> missing/empty. Chỉ tin K khi K không rỗng. Không dùng raw zero-value làm identifier component.

### Áp dụng cho project nào?

- **CDC orchestrator** đọc `source_locator` payload → fallback `source_object_name`.
- **Kubernetes admission controller** đọc optional `metadata.labels.X` → fallback `metadata.name`.
- **Stripe webhook router** đọc optional `metadata.tenant_id` → fallback infer từ `customer_id`.
- **Multi-tenant DB sharding** đọc optional `tenant_key` từ JWT → fallback tenant inferred từ
  `subject` claim.
- **Image build pipeline** đọc optional tag override từ commit message → fallback `git rev-parse short`.
- **Search indexer** đọc optional `indexer.targetIndex` → fallback `default_index_for_type`.
- Bất kỳ adapter nào dịch payload polymorphic (multi-engine, multi-source, polyglot) sang identifier
  cứng đều có rủi ro pattern này khi K không phải required field.

### Symptom phát hiện được

- UNIQUE constraint vi phạm bí ẩn khi user tưởng register chỉ 1 lần (thực ra 2 lần cùng key rác).
- ACL/include-list/topic-list có entry "prefix.<empty>" hoặc "<prefix>.x" trông như test data nhưng
  thật ra do fallback broken.
- Pipeline accept config nhưng silent skip — log không kêu vì giá trị rỗng vẫn parse hợp lệ.
- Self-heal khi clean lại config + restart binary mới (Debezium re-snapshot).

### Defensive measures

1. **Audit all uses of `req.OptionalField`** ở mỗi vị trí cùng lúc (CLAUDE.md lesson "Fix bug 1 service
   quên cross-service") — KHÔNG ăn 1 vị trí rồi nghỉ.
2. **Validate at admission**: từ chối request nếu sau khi compute fallback identifier vẫn rỗng — ném 400.
3. **Test driver-level**: viết test multi-payload (with K, without K, with K=empty, with K=bogus) để
   ép pattern bug surface ở review.
4. **Schema-level**: nếu đặc tả format output là "non-empty path component" → assert ngay sau compute,
   trước khi feed config tới downstream.

### Verification path

Fix landed (commit `92d78d3`):
- helpers.go 3 vị trí đều có `if collection == "" { collection = req.SourceObjectName }` (hoặc tương
  đương cho table/PG path).
- Test `TestExtendDebeziumInclude_Mongo_BothTiers` + 21 assertion PASS.
- Live smoke F3 round 2: Mongo INSERT acknowledged → Kafka offset advance 6→7 → shadow row landed
  `f3v2_smoke_1777887709` (`_synced_at=2026-05-04 09:41:51.804387 UTC`).

**Tags**: #adapter #fallback #optional-key #identifier #unique-constraint #silent-drop #cross-site #audit-all-occurrences


## 2026-05-05 — Volume preservation when splitting docker-compose project

**Trigger**: Phase B5 split `centralized-data-service/docker-compose.yml` (16 services) thành 2 compose:
- core 10 services giữ project name `centralized-data-service` (volumes `pg_cdc_data`, `kafka_data` preserved).
- dev 6 services chuyển sang project `cdc-docker-dev`.

**Bài học cụ thể**: `docker-compose` namespace volume names theo project (`<project>_<volume_decl>`). Nếu khai báo volume bình thường ở project mới, compose sẽ tạo volume RỖNG MỚI (`cdc-docker-dev_pg_source_data`) — data 6 ngày test bị mất.

**Fix**: declare volume external với `name:` trỏ tới existing namespaced name:

```yaml
volumes:
  pg_source_data:
    external: true
    name: centralized-data-service_pg_source_data
```

→ Project mới mount volume cũ. Data preserved. Khi user chạy môi trường sạch (chưa có data), bỏ `external: true` + `name:` để compose tự tạo.

**Global Pattern**: Khi tách docker-compose project A thành A' + B (subset của services move sang B), declare volumes của subset đó trong B với `external: true, name: A_<volume>` để bảo toàn data. Đúng: **A_<vol> stays bound to physical disk, B references it through external alias** → zero data loss, zero downtime beyond container restart.

**Anti-pattern**: tạo `B_<vol>` rỗng + chạy `docker volume rm A_<vol>` → mất data. Hoặc dùng `docker run --volumes-from` shim — phá namespace, gây conflict khi compose down.

**Verify checklist khi split**:
1. `docker volume ls | grep <project_old>_` — list current volumes.
2. Map volumes giữ vs move.
3. Trong compose B: `external: true` + `name:` cho mỗi moved volume.
4. `docker compose down` (no -v) old project → volumes survive.
5. `docker compose up -d` 2 project mới — verify hostname resolution (external network) + data count khớp before/after.

**Related**: lesson 2026-04-29 về Phase ≠ Workspace mới (vẫn gắn workspace cha) — pattern y hệt: thêm khả năng phân tách mà không phá namespace gốc.

**Tags**: #docker-compose #volume #external #data-preservation #split-project #namespace #migration

---

## 2026-05-05 10:24+07 — Lesson: Cross-repo relative-path mount = decoupling violation

**Context**: Phase B5.5 split docker-compose `centralized-data-service/` (core) khỏi `cdc-docker-dev/` (config-able DBs). Round 1 quên 2 init-script mount vẫn dùng relative path `../centralized-data-service/deployments/...` từ compose mới — đè ngược coupling vừa tách. Anh trainguyen catch: "rất vô lý. vì chúng ko nên dính tới nhau" → fix B5.5b move asset sang `cdc-docker-dev/init/` rồi đổi mount thành `./init/...`.

**Global Pattern**: Khi split repo/project A → A' + B (cùng umbrella hay khác), mọi volume mount / ConfigMap source / build context trong B mà reference asset của A bằng path `../A/...` (hoặc absolute path tới A) = coupling lén. Đúng: **B own toàn bộ asset cần thiết cho B services. Move (không copy) asset từ A sang B; mount bằng `./...` relative tới B.** Test trước khi merge: `grep -rn '\.\./<other-project-name>' <new-project-dir>/` phải 0 hit cho YAML/compose/Dockerfile/Helm.

**Anti-pattern điển hình**:
- `volumes: ['../A/init:/docker-entrypoint-initdb.d:ro']` trong B/docker-compose.yml.
- `Dockerfile` của B `COPY ../A/configs ./configs`.
- Helm values `extraVolumes: hostPath: /repo/A/secrets`.

**Verify checklist sau split**:
1. `grep -rn '\.\./' <new-project>/` filter file types (yml, yaml, Dockerfile, sh) → review từng hit. Match cross-project = fix.
2. `grep -rn '<absolute-path-to-other-project>' <new-project>/` → cũng 0 hit.
3. Run `docker compose config --quiet` từ root mỗi project — không error path resolution.
4. Sau move: `grep` ngược lại trong A để confirm asset không còn được A internal sử dụng (nếu còn → COPY thay vì MOVE; nếu không còn → DELETE để giữ A clean).

**Related**: lesson "external volumes bảo toàn data" (cùng phase B5.5). Bộ đôi: (i) volumes external giữ data; (ii) asset move sang repo own giữ decoupling. Thiếu một thì split chưa hoàn chỉnh.

**Tags**: #split-project #decoupling #docker-compose #cross-repo-mount #anti-pattern #relative-path

---

## 2026-05-05 — Lesson: Centralize naming convention in a `naming` package, env-driven

**Context**: Schema prefix `shadow_` hardcoded ở 4 call sites (admin helpers, provisioning handler, sinkworker normalizer) trong `centralized-data-service`. Đổi convention sang `lake_` / `raw_` / language-specific → phải sửa 4 file + risk sót hit (state enum `shadow_pending`, NATS subject `cdc.cmd.shadow.bind`, log keys lẫn schema name khi grep).

**Global Pattern**: Khi convention naming X (prefix/suffix/separator/casing) hardcoded N call sites trong codebase A để tạo identifier kiểu `X<Y>` → tạo package `naming` (hoặc `convention`) tập trung. Package expose helper `<Convention>Name(parts...) string` đọc env `<DOMAIN>_<CONVENTION>_<PART>` qua `sync.Once`, default fallback = giá trị cũ để giữ backwards compat. Mọi call site `"X" + dynamic` đổi sang `naming.<Convention>Name(dynamic)`.

```go
// internal/naming/naming.go
package naming

import ("os"; "sync")

const defaultShadowPrefix = "shadow_"

var (
    shadowOnce   sync.Once
    shadowPrefix string
)

func ShadowSchemaPrefix() string {
    shadowOnce.Do(func() {
        shadowPrefix = os.Getenv("CDC_SHADOW_SCHEMA_PREFIX")
        if shadowPrefix == "" { shadowPrefix = defaultShadowPrefix }
    })
    return shadowPrefix
}

func ShadowSchemaName(suffix string) string {
    return ShadowSchemaPrefix() + suffix
}
```

**Lý do thắng**:
1. **Đổi convention = đổi env**, không touch code. PR review trở thành 1-dòng env change thay vì N-file diff.
2. **Phân biệt rõ schema-name vs state-name vs subject-name**: package boundary tách 3 domain identifier dùng cùng từ "shadow" nhưng khác semantic. `naming.ShadowSchemaName(...)` chỉ ra purpose = schema; `cdc.cmd.shadow.bind` (NATS) và `shadow_pending` (state enum) không bị rename oan.
3. **`sync.Once` cache**: env đọc 1 lần ở boot, các call site không lặp `os.Getenv` (perf + consistency — không có race với env mutation mid-process).
4. **Default fallback giữ behavior cũ**: opt-in upgrade, không break tồn tại.

**Anti-pattern (đừng làm)**:
- Để N call sites hardcode literal `"X"` rồi mỗi lần đổi convention phải `find-and-replace` → sót hit do từ đó cũng xuất hiện ở comment, log message, state enum, test fixture.
- Đặt env `os.Getenv` gọi mỗi call site (không cache) → mỗi schema-name resolution = syscall + risk inconsistency nếu env thay đổi giữa chừng.
- Đặt biến package `var prefix = os.Getenv(...)` ngoài `init()` mà không có `sync.Once` → race với test setup ENV (test framework set env sau package init).

**Verify checklist**:
1. `grep -rn '"X"' <repo>/` sau refactor → 0 hit ở schema-creating sites (state enums + subjects + log keys vẫn còn — đó là intentional).
2. `go build ./... && go test ./...` PASS.
3. Smoke: chạy với env override `<DOMAIN>_<CONVENTION>_<PART>=Y_` → identifier mới start `Y_<dynamic>`.
4. Smoke: chạy không env → fallback default = giá trị cũ (backwards compat).

**Áp dụng được cho ≥3 dự án**:
- CDC pipeline: `shadow_` prefix (case study này), `dw_` master prefix, `cdc.cmd.` NATS subject prefix.
- E-commerce: `tenant_` schema prefix multi-tenant SaaS, `tmp_` background job table prefix.
- Logs/observability: metric name prefix (`app_<env>_<component>_*`), trace tag prefix.

**Tags**: #naming #convention #env-driven #refactor #single-source-of-truth #sync-once #default-fallback #global-pattern

---

## 2026-05-05 — Lesson: `.env.example` style — actionable env vars > prose comments

**Trigger**: anh trainguyen sửa mongo block từ verbose 3-line comment block của em sang 2-line: `# ---------- header` + `MONGO_URL=...`. Pattern này áp dụng cho mọi container/service trong .env.example.

**Global Pattern**: `.env.example` mỗi entry phải là **actionable** (env var thực sự copy được sang `.env`) HOẶC **omit hoàn toàn**. Nếu service A không expose env knobs trong compose, nhưng consumer B/C cần URL/endpoint của A → ghi 1 var `<SERVICE>_URL=<connect-string>` để B/C copy. KHÔNG ghi block comment thuần "DEV ONLY: anonymous access..." mà không có env var nào — comment dài làm noise, user phải tự suy luận URL.

**Anti-pattern (đừng làm)**:
```
# ---------- mongo source (gpay-mongo replSet rs0 on :17017) ----------
# DEV ONLY: anonymous access (no auth). Connect URL:
#   mongodb://gpay-mongo:27017/?replicaSet=rs0
# (host port :17017 → container 27017). Prod = MongoDB Atlas...
```
3 dòng comment + 0 env var → user copy file xong vẫn không có gì useable, phải đọc và tự gõ.

**Pattern đúng**:
```
# ---------- mongo source (gpay-mongo replSet rs0 on :17017)
MONGO_URL=mongodb://gpay-mongo:27017/?replicaSet=rs0
```
1 dòng comment header (đủ identify) + 1 env var thẳng (copy-paste runnable). Cô đọng hơn, action-oriented.

**Quy tắc tổng quát cho `.env.example`**:
1. Mỗi block: ≤ 1 dòng comment header (tên service + key info).
2. Theo sau là env var(s) thực sự (giá trị placeholder hoặc default sane cho dev).
3. Nếu service không có env knob trong compose nhưng consumer cần connect string → expose `<SERVICE>_URL=...` cho consumer copy.
4. KHÔNG viết prose ("DEV ONLY: ...", "Prod uses ...") trong `.env.example`. Prose thuộc về `README.md`. `.env.example` là **template-to-copy**, không phải tutorial.
5. Ngoại lệ: 1-line note về security (e.g. `# DEV ONLY — không deploy lên prod`) đầu file là OK.

**Áp dụng được cho ≥3 dự án**:
- Microservices: mỗi service `.env.example` liệt kê service-DB creds + dependent-service URLs (consumer copy là chạy được).
- Frontend: `.env.example` liệt kê API_URL, CDN_URL, FEATURE_FLAGS_URL — không kèm prose explanation.
- CI/CD: secrets template chỉ list var names + placeholder, không list policy.

**Tags**: #env-example #documentation #actionable-config #copy-paste-friendly #dx #global-pattern

## 2026-05-05 — Lesson: Dockerfile bake `config-local.yml` only = prod ship DEV creds

**Trigger**: anh trainguyen flag *"sao repo auth hiện tại nó có cảm giác ko lên prod đc vậy"*. Audit `cdc-auth-service/deployments/docker/Dockerfile:12` lộ pattern `COPY --from=builder /app/config/config-local.yml ./config/config-local.yml` — image prod nuốt creds DEV + JWT secret `change-me-in-production`. Reconcile-service làm đúng pattern: `COPY --from=builder /app .` (cả repo, gồm 3 yml local/prod/sample).

**Global Pattern [Dockerfile X copies single config-local.yml only into prod image Y] → Result Z (prod runtime ships DEV creds, default secrets, dev pool sizes; image không deploy được sạch lên multi env)**.

Đúng:
1. Dockerfile `COPY config ./config` (CẢ thư mục) — image carry mọi env variant.
2. Runtime chọn file qua env (`cfgPath=./config/config-production.yml`).
3. Prod yml fields rỗng cho secrets — env override (`AUTH_DB_HOST`, `AUTH_JWT_SECRET`) điền tại runtime.
4. `validateConfig()` refuse:
   - rỗng required (host/database/secret/port);
   - default placeholder (`change-me-in-production`) khi `mode==production`.
5. Code env-binding dùng `viper.AutomaticEnv()` + `SetEnvPrefix(SVC)` + `BindEnv(key, ENV_NAME)` map — single source of truth, không hardcode `applyEnvOverrides`.

**Anti-pattern**:
- `COPY config-local.yml` only → 1 image / 1 environment, rebuild cho từng env (CI/CD waste, drift risk).
- Prod yml `${VAR}` placeholder mà không có envsubst pipeline → viper KHÔNG expand syntax này native, field thành literal string `"${VAR}"` → DB connect fail với hostname `${VAR}`.
- `applyEnvOverrides` hardcoded list → thêm field schema phải sửa Go code, dễ sót.

**Áp dụng được cho ≥3 dự án**:
- Bất kỳ Go service dùng viper + Dockerfile multi-stage (cdc-auth, centralized-data, cdc-cms, reconcile-service).
- Node service dùng dotenv + Dockerfile (pattern tương tự: copy cả `config/`, runtime chọn `NODE_ENV`).
- Java/Spring service dùng `application-{profile}.yml`: profile chọn qua `SPRING_PROFILES_ACTIVE` env, image phải bundle cả 3 file local/staging/prod.

**Detection heuristic** (dùng khi audit repo mới):
1. `grep -n "COPY.*config-local" Dockerfile*` → red flag.
2. `ls config/` thiếu `config-production.yml` hoặc tương đương → red flag.
3. `grep -n "applyEnvOverrides\|os.Getenv.*HARDCODED_KEY" config/*.go` đếm > 5 lần → hardcoded env list smell.
4. Validate boot binary với `cfgPath=prod.yml` không có env → expect FAIL với required missing message.

**Tags**: #docker #config-management #env-override #viper #prod-readiness #global-pattern #dx

## 2026-05-05 — Lesson: Go service `.env.example` = dead weight nếu (no godotenv) ∧ (compose có defaults)

**Trigger**: anh trainguyen flag *".env.example đang có cảm giác nó ko xài vì đang dùng go mà"*. Audit `cdc-auth-service`: `grep godotenv` 0 hit, `go.mod` không import dotenv lib, compose có `${VAR:-default}` cho cả 3 DB vars. Kết luận: file là noise — Go binary đọc YAML qua viper, compose có defaults, 0 docs reference.

**Global Pattern [Repository R kèm `.env.example` cho service S written in language L] → Result Y**:
- Nếu L = Node/Python (auto-load `.env` via dotenv runtime / framework convention) → `.env.example` LÀ contract, giữ.
- Nếu L = Go AND `grep godotenv R/` 0 hit AND compose-defaults present → `.env.example` LÀ dead weight, XÓA.

**Decision tree (audit repo Go mới)**:
1. `grep -r "godotenv\|joho/godotenv" --include="*.go"` → có dotenv loader? 
   - YES: `.env.example` is contract, validate fields match.
   - NO: continue 2.
2. Compose service có `${VAR:-...}` defaults cho mọi var trong `.env.example`? 
   - YES: `.env` purely optional → file là noise nếu không có docs reference.
   - NO: `.env.example` documents required overrides → giữ.
3. `grep -r "\.env\.example" R/` (docs/scripts) → có reference không?
   - YES: keep (documented contract).
   - NO + đã pass step 2 = noise → DELETE.

**Anti-pattern**: copy `.env.example` template từ Node project sang Go project mà không check runtime loading. User copy `.env` xong vẫn không thấy effect → confused → bug report.

**Cách user override env trong Go service KHÔNG dùng dotenv**:
```bash
# Option A: shell export
export AUTH_DB_HOST=prod.rds.com && ./auth-service

# Option B: env-file qua docker/k8s orchestrator (compose `env_file:`, k8s `envFrom`)
# Option C: source .env (manual): `set -a; source .env; set +a; ./auth-service`
```
KHÔNG có "auto-load" như Node — Go cần explicit.

**Áp dụng được cho ≥3 dự án**:
- Bất kỳ Go monorepo có nhiều service: audit từng service có dotenv không, thống nhất convention.
- Migration Node→Go: drop `.env.example` (hoặc chuyển sang `config-sample.yml`) khi rewrite.
- Static-binary deploy (k8s/ECS): env injected qua orchestrator — `.env` file là phản pattern.

**Tags**: #go #dotenv #env-loading #config-management #dead-files #global-pattern #dx

## 2026-05-05 — Lesson: Validation BEFORE fallback merging — order matters in config pipelines

**Trigger**: B5.6.2 centralized-data-service. validateConfig gặp false-positive PASS khi fields rỗng vì `cfg.DB.PgxDSN()` trả về string non-empty `"postgres://:@:0/?sslmode="` (literal sprintf không bao giờ empty), `applyDBFallbacks` set `cfg.SystemDB.URL = legacy` → validateConfig thấy non-empty → app boot OK rồi crash khi connect runtime.

**Global Pattern [Pipeline P có sequence: read input I → derive defaults D → validate V] → Result Y**:
- Nếu `V` chạy AFTER `D` → V thấy `I ∪ D` (merged state) → user intent rỗng bị lấp bằng derived value → **false-positive PASS**.
- Nếu `V` chạy BEFORE `D` → V thấy CHỈ `I` (user intent) → empty input bị reject đúng → **fail-fast at boot**.

**Đúng sequence**: ReadConfig → Unmarshal → applyEnvOverrides (env trộn vào user input) → **validateConfig** → applyFallbacks (derive missing fields).

**Anti-pattern**:
```go
applyEnvOverrides(cfg)
applyFallbacks(cfg)   // SystemDB.URL ← cfg.DB.PgxDSN() (literal-non-empty garbage)
validateConfig(cfg)   // sees non-empty SystemDB.URL → PASS (FALSE positive)
```

**Pattern đúng**:
```go
applyEnvOverrides(cfg)
validateConfig(cfg)   // sees empty SystemDB.URL → REJECT (correct)
applyFallbacks(cfg)   // safe to derive AFTER passing validation
```

**Detection heuristic** (audit config pipelines):
1. Tìm `applyDefaults / applyFallbacks / merge*` đặt BEFORE `validate*` trong `NewConfig`/`Load` → red flag.
2. Tìm helper trả về string từ `fmt.Sprintf` mà KHÔNG check empty inputs (e.g. `func DSN() string { return fmt.Sprintf("postgres://%s:%s@%s:%d/...", "", "", "", 0, "") }` → ra `"postgres://:@:0/..."` non-empty literal).
3. Test rằng config rỗng hoàn toàn → validateConfig trả error rõ; nếu PASS → bug.

**Áp dụng được cho ≥3 dự án**:
- Config validation pipelines bất kỳ ngôn ngữ nào (Go viper, Node convict, Python pydantic, Java Spring profiles).
- ETL / data pipelines: validate raw input BEFORE applying transforms/derives — derives che mất missing source data.
- API request validation: validate raw payload BEFORE applying server-side defaults — defaults che mất user-supplied invalid fields.
- Database migrations: validate "intent" SQL trước khi run idempotent fallbacks (`CREATE IF NOT EXISTS`) — fallback che mất schema mismatch.
- Form validation UI: validate user input BEFORE applying placeholder/default values — defaults che mất empty intent.

**Anti-pattern bonus**: helper getter trả về literal string non-empty từ rỗng input (như `PgxDSN()` ví dụ trên) là code smell. Pattern an toàn: getter return `("", false)` hoặc `(nil, error)` khi inputs missing — caller buộc phải handle empty case explicitly.

**Tags**: #validation #config-management #order-matters #fail-fast #empty-input #global-pattern #anti-pattern

---

### Lesson #1294 — JSON serialization order khi migrate `map[string]any` → typed struct (CQRS Q-side, byte-identical contract)

**Khi nào xảy ra**: Refactor handler (CMS, BFF, gateway) chuyển payload xây bằng `map[string]any` (Go map / fiber.Map / gin.H / etc.) sang typed struct. Test diff thấy size giống nhau nhưng `cmp -s` báo DIFF.

**Root cause**: Go's `encoding/json` serialize map theo **alphabetical order** của key (post Go 1.12, deterministic). Struct serialize theo **field-declaration order**. Field order không match key order → byte-different output dù cùng nội dung.

**Global Pattern**: `Refactor [A: map-based payload] → [B: struct-typed payload] với contract byte-identical = order(A.keys) == order(B.fields)`. Đúng: declare struct fields theo alphabetical JSON tag order khi migrate từ map; hoặc generate diff bằng `jq -S` (sort keys) thay vì raw cmp nếu wire chỉ cần semantic-equivalent.

**Áp dụng được**: bất kỳ language nào có map (Python dict, JS object) khi migrate sang typed class/struct/dataclass đều dính bug này nếu wire contract pin byte-level.

**Detection**: `wc -c pre post` size giống nhau + `cmp -s` báo DIFF + `jq -S` cùng output = serialization order mismatch (chứ không phải data drift).

**Fix template** (Go): reorder struct fields theo `sort json_tags` ascending. Comment ghi rõ "field order matches legacy map alphabetical serialization".

**Tags**: #cqrs #refactor #json-serialization #byte-identical #order-matters #go #global-pattern

---

### Lesson #1295 — Hybrid command bus cần `ResultBody` trên CommandResult cho sync handlers (CQRS C-side)

**Khi nào xảy ra**: Thiết kế CommandBus B route command C qua 2 path:
- **Sync** (in-process map handler X): low-latency operations như `alert.ack` (chỉ UPDATE 1 row Postgres).
- **Async** (NATS publish subject Y): long-running như `master.swap`, `recon.check` (chạy trên worker).

Nếu `CommandResult` chỉ có `{JobID, Accepted bool}` không có wire body → sync handler X trả nothing → FE buộc phải poll `/jobs/:id` sau mỗi Dispatch dù đã có kết quả ngay. RTT = 2 round-trips cho việc lẽ ra 1.

**Root cause**: Bus author áp pattern "all async" (fire-and-forget) lên cả sync path để giữ contract đồng nhất → đánh mất ưu thế của sync route.

**Global Pattern [Hybrid bus B route command C qua sync X / async Y, Result chứa optional ResultBody]**: 
- Declare `CommandResult.ResultBody json.RawMessage` (nullable, omitempty). 
- Sync handler X populate ResultBody với wire-bytes trả về cho caller. 
- Async path Y để ResultBody rỗng — FE biết `Accepted=true && ResultBody==nil` ⇒ poll `/jobs/:id`. 
- Sync path X trả `Accepted=true && ResultBody!=nil` ⇒ FE inline render kết quả.

**Đúng**: 
```go
type CommandResult struct {
    JobID      string          `json:"job_id"`
    Accepted   bool            `json:"accepted"`
    ResultBody json.RawMessage `json:"result_body,omitempty"` // sync inline; async empty
}
```

**Sai**:
```go
type CommandResult struct { JobID string; Accepted bool } // mất sync ưu thế
```

**Áp dụng được cho ≥3 dự án**:
- CQRS-style microservice gateways (Go, .NET MediatR, Java Axon) có cả intra-service sync handler + cross-service async messaging.
- BFF/API gateway pattern: 1 endpoint vừa serve cache hit (sync) vừa dispatch backend job (async) — Result phải tải được cả 2 shape.
- LLM tool-use orchestration: tool call có thể return immediately (calculator) hoặc kick off background job (image gen) — Result envelope cần ResultBody slot cho immediate path.
- gRPC bi-modal: unary sync + server-stream async — response message nên có optional `body` thay vì 2 RPC riêng.
- WebSocket command pattern: ack-only (async) vs ack+payload (sync) qua cùng 1 envelope.

**Detection**: 
1. Tìm `CommandResult / CommandReply / DispatchResponse` không có wire-body slot.
2. Audit FE/caller code: nếu sau mỗi Dispatch luôn `setTimeout/while polling /jobs/:id` → smell.
3. Tìm sync handler in-process trả `(any, error)` rồi bị bus drop kết quả → smell.

**Anti-pattern bonus**: Force "all async" cho UI consistency (FE always poll) — sacrifice latency mà không gain gì (FE vẫn phải handle 2 shape: result-from-poll vs error-from-poll). Tốt hơn: 2 shape ngay tại Dispatch return (`ResultBody` filled vs nil).

**Tags**: #cqrs #command-bus #hybrid-sync-async #api-design #latency #global-pattern

---

## Lesson #1296 — 2026-05-06 — Plan critique cần verify từng claim với evidence trực tiếp

**Context**: Boss reviewed P3 plan, claim plan có line numbers off-by-some, "extract inline" sai (đã extract), evt subjects "NEW" thực ra đã có upstream. Muscle verify từng claim trước khi acknowledge.

### Global Pattern [Plan reviewer A claims X about codebase Y → Reviewee B] → Result Z. Đúng:
1. **Reviewee KHÔNG defensive-deny** — verify từng claim với grep/wc/file-stat trực tiếp.
2. **Reviewee KHÔNG blanket-accept** — vì đôi khi reviewer cũng sai (line numbers stale từ session trước).
3. **Reviewee output 1 bảng status**: `claim | actual | match?`. Nếu match → acknowledge + action item. Nếu không → đối chiếu evidence + đề xuất re-frame.
4. Mọi gap proposal phải có **effort estimate** (kèm reason) + **owner** (Brain | Muscle | Boss decide) + **status** (TODO | BLOCKED | DONE).
5. Nếu critique nêu blocker thiết kế (như "worker permission verify") → tag BLOCKED, KHÔNG tự ý implement đường tắt.

### Áp dụng được cho 3 dự án khác:
- Code review GitHub PR — PR comment "you should X" có thể base on outdated commit. Verify HEAD trước khi accept/argue.
- Architecture decision record (ADR) review — reviewer claim "we already have Y" → grep codebase confirm.
- Multi-team task hand-off — handed-over team verify claim của team trước (file paths, line numbers, naming conventions stale theo thời gian).

### Anti-pattern: "Yes-and" mọi critique → modify plan vô tội vạ → contradiction tích lũy. Hoặc "no-and" mọi critique → defensive → bỏ lỡ valid feedback.

### File minh chứng: `agent/memory/workspaces/feature-cdc-system-refactor/10_gap_analysis_p3_critique_2026-05-06.md`

---

## Lesson #1297 — 2026-05-06 — Cast TỪNG positional `?` trong CASE expression khi GORM/pgx prepared statement, KHÔNG cast outer

**Context**: P3.T3.12 StuckJobReaper SQL build động `started_at + (interval '1 second' * (CASE type WHEN ? THEN ? ... END)) < NOW()`. T3.11 smoke phát hiện reaper sweep fail mỗi 30s với 2 lỗi tuần tự:
1. `operator does not exist: interval * text (SQLSTATE 42883)` — outer cast `(CASE END)::int` thử trước, KHÔNG sửa.
2. Sau outer cast, lỗi đổi sang: `failed to encode args[1]: unable to encode 120 into text format for text (OID 25): cannot find encode plan`.

**Root cause**: pgx/GORM prepared-statement type inference resolve param types TRƯỚC khi outer cast áp dụng. Trong `CASE column-A WHEN ? THEN ? ... ELSE ? END`:
- Param đầu (`WHEN ?`) so với cột text → infer = text.
- Driver propagate text type sang mọi cùng-shape positional trong CASE → THEN `?` và ELSE `?` đều bị infer text.
- Khi caller truyền int64 (60, 600, 30, ...), driver từ chối encode int64 vào text slot → SQLSTATE 42883 hoặc encoding failure.
- Outer cast `(CASE … END)::int` chỉ chuyển kiểu KẾT QUẢ CASE sau khi evaluated; không ảnh hưởng inference cho mỗi positional `?`.

**Fix verified**: cast TỪNG positional ngay trong CASE branch:
```go
caseExpr.WriteString("WHEN ? THEN ?::int ")  // not "WHEN ? THEN ? "
caseExpr.WriteString("ELSE ?::int END")       // not "ELSE ? END"
```
Sau rebuild + restart, reaper log `{"msg":"reaped stuck jobs","count":1}`, status row flip 'running' → 'failed' đúng spec.

### Global Pattern [Driver D với prepared statement P, build động SQL với positional ? trong CASE-expression có column-of-type-A so sánh ở WHEN] → Result [param types lệch theo column-A; outer cast không sửa được; encoding failure khi caller truyền type-B]. **Đúng**:
1. Cast TỪNG positional ngay tại branch nó xuất hiện: `WHEN ?::A THEN ?::B`, `ELSE ?::B END`.
2. Outer cast `(CASE ... END)::B` CHỈ dùng cho final result type, KHÔNG sửa được inference cho positional bên trong.
3. Test integration với REAL Postgres (mock-DB hoặc sqlite không phát hiện vì khác driver type-inference).
4. Khi gặp `operator does not exist: T1 * T2` với prepared statement, nghi ngay positional inference trước khi suspect schema/migration.

### Áp dụng được cho 3 dự án khác:
- **Bất kỳ Go service dùng GORM/pgx + dynamic SQL build**: dashboards với column-filter, multi-tenant routing, schema-aware aggregation.
- **JDBC PreparedStatement Java/Kotlin**: cùng pattern infer xảy ra với JDBC driver Postgres khi mix column types trong CASE.
- **Python psycopg2/asyncpg với prepared mode** (đặc biệt qua pgbouncer transaction-pool): có thể tái hiện.

### Anti-pattern:
- Tin "outer cast sẽ sửa mọi inference issue" → debug loop dài.
- Mock DB cho test reaper SQL → bug không phát hiện trước smoke production-like.
- Suspect data type column trước khi suspect param inference.

### File minh chứng:
- Code fix: `cdc-cms-service/internal/service/stuck_job_reaper.go:111,114`
- Smoke evidence: `agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md` (entry `2026-05-06 15:35 ICT — T3.11 smoke matrix executed + HOTFIX-2`)
- Task tracking: #177 P3.HOTFIX-2

**Tags**: #postgres #gorm #pgx #prepared-statement #type-inference #case-expression #reaper #global-pattern

---

## Lesson #1298 — 2026-05-06 — CommandBus chỉ cho mutation/coordination, KHÔNG migrate audit-only side-effects

**Context**: Phase 3 cdc-cms-service refactor (CQRS C-side) chốt scope qua 7 đợt — 27 endpoint mutation đã qua bus. Còn 4 ActivityLog write (3 reconciliation_handler + 1 registry_handler). Câu hỏi: có nên migrate nốt cho "consistency"?

**Quyết định**: SKIP. Audit-only side-effect KHÔNG thuộc bus scope. Phase 3 closed sạch.

**Root cause của câu hỏi sai**: "Universal indirection" thinking — tin rằng mọi handler-level write nên đi qua bus để "uniform pattern". Bỏ qua phí của bus:
- +1 hop sync (JSON marshal/unmarshal request + response).
- +1 row `cdc_jobs` audit table per write — nhưng ActivityLog ĐÃ là audit, double-recording.
- Idempotency-Key collision risk khi 1 request có nhiều ActivityLog write (cần suffix `:audit:<seq>` workaround chỉ để tránh va chạm bus).
- Test surface tăng: validate rule, type tag namespace, errors.Is sentinel mapping cho thứ chỉ là log entry.

**Fix verified**: ActivityLog write giữ direct call ở handler. Kết thúc Phase 3, cdc_jobs chỉ còn rows cho mutation thật — observability sạch.

### Global Pattern [Codebase A có CommandBus B (CQRS C-side) → reviewer/team đề xuất migrate side-effect X (audit log, metrics emission, fingerprint touch) qua bus B "for consistency"] → Result [thêm hop sync + double-audit + idempotency collision risk, gain semantic = 0]. **Đúng**:
1. Bus B chỉ cho 2 track:
   - **Track Mutation** — destructive infra (DDL ALTER, business state INSERT/UPDATE, external HTTP destructive REST như Kafka Connect).
   - **Track Coordination** — async cross-service dispatch (NATS publish, queue enqueue, scheduled job dispatch).
2. Side-effect X (audit/metrics/log) → giữ `repo.Insert(ctx, ...)` hoặc `auditService.Record(...)` trực tiếp ở handler/service layer.
3. **Test phân loại**: side-effect X có "đứng tự do" được không? Nghĩa là: nếu X fail (network blip, table locked), request có rollback hay chỉ log warn?
   - Yes (chỉ log warn) → audit-only → KHÔNG bus.
   - No (rollback request) → mutation-essential → qua bus.
4. Nguyên tắc: **bus là indirection layer trả phí cho actions có blast radius**. Audit-write không có blast radius (failure ≠ user impact, chỉ giảm observability) → không xứng đáng phí bus.

### Áp dụng được cho 3 dự án khác:
- **CQRS Java/Spring** với Axon/EventBus: cám dỗ migrate `auditLog.publish(...)` qua command bus → giữ trực tiếp `auditRepo.save(...)`.
- **NestJS với @CommandBus**: ActivityLog interceptor gọi `commandBus.execute()` cho log entry → anti-pattern, chuyển về `loggingService.record()`.
- **Workflow engine (Temporal/Camunda)** với "everything is an activity" thinking: read-only/log-only operations không cần activity wrapper, gọi inline để tránh history blow-up.

### Anti-pattern:
- **Universal indirection** — tin mọi handler write phải qua bus để "uniform". Hệ quả: command-bus registry phình to vì log entry, error-mapping table phình to vì sentinel cho mỗi log type, idempotency table double-record.
- Lười phân loại blast radius → migrate hết cho nhanh → over-abstraction debt.
- Đóng scope theo "100% coverage" thay vì "actions có blast radius" → kéo cosmetic vào critical path.

### File minh chứng:
- Workspace audit log: `agent/memory/workspaces/feature-cdc-system-refactor/05_progress.md` (entry `2026-05-06 18:10 ICT — Đợt 7 P3` + ghi chú "P3 destructive migration coverage final").
- Source giữ direct call: `cdc-cms-service/internal/api/reconciliation_handler.go` (3 chỗ) + `cdc-cms-service/internal/api/registry_handler.go` (1 chỗ).
- Decision: kết thúc Phase 3 sạch, không kéo D2 cosmetic route prefix + ActivityLog migration vào critical path.

**Tags**: #cqrs #command-bus #ddd #cms-service #scope-discipline #audit-log #anti-over-abstraction #blast-radius #global-pattern

---

## Lesson — Router-level swap khi V2 handler chỉ là thin-delegate to V1

**Ngày**: 2026-05-07
**Workspace**: `agent/memory/workspaces/feature-cdc-system-refactor/` (T14 P4)
**Pattern Global**:
> Khi handler **B** trong namespace mới chỉ chứa các method 1-line `return aHandler.X(c)` (thin-delegate to handler **A** legacy) → đừng duplicate logic vào B. Thay router-level mount: route URL của namespace B mount **A** trực tiếp. Wire URL contract cho client giữ nguyên (B namespace), nhưng implementation owner duy nhất là A.
>
> **Đúng**:
> 1. Xóa B.X (thin-delegate) khỏi handler.
> 2. Trong router: `routerGroup.Method("/path-namespace-B/X", aHandler.X)` (thay vì `bHandler.X`).
> 3. Nếu B chỉ còn helper field references A không dùng nữa → xóa luôn field + tham số constructor.
>
> **Sai**:
> 1. Duplicate logic A trong B.X (DRY violation).
> 2. Add facade service C để cả A và B gọi (over-abstraction khi A đủ làm owner).
> 3. Xóa method delegate B.X mà không update router → 500 / 404 trên route.

**Áp dụng đa-dự án**: Bất kỳ codebase nào có namespace evolve (V1 → V2 / legacy → modern) với một handler giữ vai trò shim. Nếu shim chỉ thin-delegate (>50% method chỉ 1-line), router-level swap rút gọn nhanh hơn duplicate logic. Pattern variables: A=V1Handler/LegacyHandler, B=V2Handler/NewHandler, X=method name, namespace-B=URL prefix mới.

**Trade-off**:
- Swagger godoc (gắn theo method) bị mất khi xóa B.X. Phải hoặc (a) move @Router annotation vào A.X, hoặc (b) accept doc loss đến khi cleanup riêng.
- Nếu B sau này cần extend (auth khác, body schema khác) thì phải re-introduce method — lúc đó duplicate có lý do.

### Counter-pattern (khi KHÔNG dùng router-level swap):
- B.X có pre/post-hook khác A.X (e.g. extra audit, metric, response shape diff) → B phải sống dưới function của riêng nó.
- B namespace cần versioning độc lập (breaking change soonish) → giữ B.X làm "boundary lock-in" cho contract.

### File minh chứng:
- T14 commit cms `084a4a1` — xóa 10 thin-delegate V2 method (`Register`, `UpdateBridge`, `BulkRegister`, `CreateDefaultColumns`, `Standardize`, `ScanFields`, `Transform`, `DispatchStatus`, `DetectTimestampField`, `TransformStatus`).
- Router swap: `internal/router/router.go` 13 mount entries `sourceObjectActionsHandler.X` → `registryHandler.X` cho URL prefix `/v1/source-objects/...`.
- Cleanup phụ: `SourceObjectActionsHandler.registry *RegistryHandler` field xóa luôn vì không còn caller.
- File 691 → 555 dòng. Test sweep 0 regression.

**Tags**: #refactor #namespace-evolution #v1-v2 #thin-delegate #router-swap #dry #global-pattern

---
## Lesson — Test uplift dưới project-convention "no sqlmock / no testcontainers"
**Date**: 2026-05-07 (T17 P7 close — feature-cdc-system-refactor workspace)

### Context
DoD upstream cho test uplift task thường yêu cầu sqlmock golden-path + coverage threshold (e.g. 35% combined). Nhưng nếu codebase đã chốt project convention "validation/shape tests only, transactional → deploy-time E2E" (no sqlmock in go.sum), việc thêm dependency mới = architectural decision — không tự ý thi công.

### Global Pattern
Khi codebase X có convention "test framework Y is excluded by design" và task T yêu cầu `coverage ≥ N%`:
1. KHÔNG thêm Y silently — escalate decision ra Brain/architect HOẶC accept partial DoD.
2. Lift coverage qua 4 lane sqlmock-free:
   - **Pure-fn tests** — predicates state machine, validators identifier, type-coerce helpers, struct round-trip JSON.
   - **HTTP wire-contract tests** — `httptest.NewServer` per-test cho probe / external-API caller. Stub server controls status code + body; verify map-from-body + status downgrade rules.
   - **Nil-receiver / no-op tests** — guard hot-path khỏi panic khi wiring oversight.
   - **Trace-context propagation tests** — OTEL `trace.NewSpanContext(...)` stub feeds `context.WithSpanContext`, verify helpers stamp `trace_id`/`span_id` vào payload/log entry.
3. Document explicitly: file-level DoD (every-file-≥1-test) → ✓; coverage-target DoD → cap với reason "project convention X excludes Y".

### Đúng (project-convention preserve)
1. Identify pure-fn surface qua `grep "^func " <pkg>/*.go` — pick non-receiver / receiver-only-with-cfg-fields.
2. Mỗi test file: 50–150 dòng, table-driven where applicable, comment ngắn header giải thích "why this is testable without Y".
3. Atomic 2-commit per đợt (cms test files + agent workspace progress APPEND). Mỗi đợt 5–17 tests.
4. Coverage report `go test -cover -count=1` per-package — track delta đợt-to-đợt vào progress.md để Boss thấy curve.
5. Final: APPEND per-file checklist (✓/✗ với reason) vào workspace progress.md.

### Sai (anti-patterns)
1. Thêm Y silently vào go.sum → vi phạm convention; PR review bị reject.
2. Skip task vì "không thể đạt 35%" → bỏ luôn lift coverage cho lane non-DB (≈40%+ codebase testable không cần Y).
3. Test cả receiver methods bằng cách inject `*gorm.DB` thật trong test → đụng test isolation; thực tế là ad-hoc testcontainer.
4. Per-file 1 monster test với 30 sub-cases trong một function → debug nightmare khi 1 case fail; rather: 1 file → 5–10 small tests với tên rõ.

### Counter-pattern (khi KHÔNG áp dụng project-convention preserve)
- Codebase chưa có convention rõ → free to add Y after RFC + Brain approval (CLAUDE.md §8 escalation).
- Module hoàn toàn DB-only (e.g. repository layer) → không có pure-fn surface; nếu Brain approve sqlmock thì thi công, không thì document "deferred to deploy-time E2E" với link tới E2E suite.

### File minh chứng (T17 P7, 5 đợt, 11 files cms, 60 tests)
- Đợt 1 cms `48567b8` — `provisioning_state_machine_test.go` (4) + `system_health_compute_test.go` (9). State predicates + alert wire shape.
- Đợt 2 cms `25f645d` — `shadow_automator_test.go` (2 validateIdent) + `system_health_alerts_test.go` (8 detectConditions/toFloat64/ownsAlertName) + `health/probes/deps_test.go` (7 SanitizeErr security gate).
- Đợt 3 cms `5c95ab5` — `health/probes/{worker,nats,kafka_connect,debezium}_test.go` (13 HTTP via httptest). Coverage probes 0% → 57.2%.
- Đợt 4 cms `2f6b3b1` — `provisioning_orchestrator_test.go` (6 OTEL trace helpers) + `health/probes/kafka_lag_test.go` (6 prom-text-format aggregation, prefix filter, rebalance -1 skip). Coverage probes 57.2% → 82.8%.
- Đợt 5 cms `5804fe6` — `reconciliation_service_test.go` (2 no-op contract) + `approval_service_test.go` (3 JSON wire). Closes file-level DoD.

**Áp dụng đa-dự án**: Bất kỳ Go service codebase nào với convention sqlmock/testcontainer-free. Pattern variables: X=codebase với convention exclude, Y=test framework excluded (sqlmock/testcontainers/mockery), N=coverage threshold target, T=test uplift task.

**Trade-off**: Repository / DB-orchestrator coverage không thể vượt ~25% mà không có Y. Phải document explicitly trong workspace 07_status.md để stakeholder thấy: "T17 file-level DoD met; coverage gap blocked by sqlmock decision — needs RFC".

**Tags**: #testing #go #sqlmock-free #project-convention #pure-fn #httptest #otel-trace-test #global-pattern

---
## Lesson — Repository adapter layer ≠ unit test target
**Date**: 2026-05-07 (T17 P7 close-out — feature-cdc-system-refactor workspace, architect ruling Q3=accept partial DoD)

### Context
DoD upstream cho test uplift task có "all repo files unit-tested via sqlmock" — nhưng repo file là adapter qua GORM (hoặc bất kỳ ORM nào abstract clause builder + transaction semantic). sqlmock = hard-coded expected SQL strings + reply rows; mock KHÔNG validate clause builder output, type marshaling, transaction lifecycle, DB-side defaults / triggers / constraint violation. Tests pass, prod break.

### Global Pattern
Khi codebase có adapter layer A (repository / DAL) wrapping framework B (GORM / sqlx / Ent) over data store C (Postgres / Mongo):
1. **A KHÔNG phải unit test target qua mock library M (sqlmock / mockery).** Tests qua M chỉ validate "A gọi M.Exec đúng SQL string em đoán" — false-positive khi B khác version sinh SQL khác.
2. **A là integration test target qua real C bằng testcontainers** (or equivalent ephemeral data-store). Mỗi test boot a fresh schema + run actual A.Method → verify rows / state side-effects.
3. **Service layer S sử dụng A**: S unit test qua interface stub (S nhận `repository.X` interface, test inject fake implementing interface). S không cần biết A's SQL — chỉ cần A trả ra đúng object shape.

### Đúng (architecture-aligned)
1. Repo unit tests SKIP, document deferred to integration phase trong workspace 07_status.md.
2. Service layer test qua interface mock (handcrafted struct hoặc go-mock generated).
3. Integration phase: testcontainers spin C → run A → assert rows. Cover 100% A (insert/update/delete/select) trong 1 lane.
4. Coverage threshold split: file-level DoD per S file ≥1 test = OK; combined % cap với reason "A layer deferred to integration".

### Sai (sqlmock anti-pattern)
1. Test `repo.Create(user)` qua sqlmock với expected SQL `^INSERT INTO users \(name,email\) VALUES \(\$1,\$2\)` — drift khi thêm column / GORM upgrade.
2. Test transaction `repo.Begin().Commit()` qua sqlmock — mock KHÔNG enforce isolation level / deadlock retry / DB-side trigger fire.
3. Mock DB-side default (e.g. `created_at TIMESTAMP DEFAULT NOW()`) — mock luôn trả `NULL`, prod trả timestamp → S logic break ở edge case "is created_at zero?".
4. Pretend coverage % thấy đẹp trong report nhưng false sense of safety.

### Counter-pattern (khi sqlmock chấp nhận được)
- Adapter layer wraps **non-DB resource** (e.g. external HTTP API) — sqlmock-equivalent like `httptest.NewServer` đúng (real protocol, just stubbed endpoint).
- Một SPECIFIC SQL string là contract pinned (e.g. raw SQL trigger fire, không qua ORM) — pin qua sqlmock OK nhưng phải kèm integration test backup.
- Pre-commit lint validation "no inline SQL in handler/service" — sqlmock có thể detect drift thread-local.

### File minh chứng (T17 P7 close-out)
- Architect ruling Q3 (2026-05-07): repo unit tests deferred. T17 P7 file-level DoD 12/13 = 92% accepted as final.
- 6 repo file deferred: `internal/repository/{mapping_rule_repo,pending_field_repo,registry_repo,schema_log_repo,source_repo,wizard_repo}.go`.
- Workspace 07_status.md close-out section (2026-05-07 02:42 ICT) document deferred + proposed T18 backlog item (testcontainers integration suite).

**Áp dụng đa-dự án**: Bất kỳ codebase với ORM adapter layer. Pattern variables: A=Repository/DAL/DAO, B=ORM framework (GORM/Ent/sqlx/sqlc), C=data store (Postgres/MySQL/Mongo), M=mock library (sqlmock/mockery). Đúng cho cả Go, Java (JPA/Hibernate), TS (TypeORM/Prisma), Python (SQLAlchemy).

**Trade-off**:
- Integration tests chậm hơn unit (testcontainers spin ~5–15s overhead per suite). Bù: 1 lane validate end-to-end semantic.
- CI cần Docker. Nếu CI runner không có Docker → fall back: unit test S (stub interface) + manual integration phase trên dev/staging.

**Tags**: #testing #repository-pattern #orm #gorm #sqlmock-anti-pattern #testcontainers #integration-test #adapter-layer #global-pattern

---

## [2026-05-07] Lesson L-RESUME-DIRTY: Working-tree deletion bị bỏ sót khi resume session

**Trigger**: Task #19 đợt D + G của feature-cdc-system-refactor. Session prior chạy refactor R = move package P → Q (xóa file ở P, tạo file ở Q). Refactor commit dở dang: file ở Q committed, deletion ở P chỉ apply ở working tree (uncommitted). Compaction cắt session. Session sau resume → build fail vì `undefined: P.Type` (callers ở Q reference P-types đã bị xóa local mà HEAD không biết).

**Global Pattern [Refactor R xóa file ở P và tạo file ở Q] → nếu deletion uncommitted, working tree dirty với `D <files-of-P>`** → next session build fail.

**Đúng (resume protocol)**:
1. **First action sau resume**: `git status --short` quét đủ 3 lane: `M`, `??`, `D`. Đặc biệt `D` lane dễ bị bỏ qua khi user grep "modified files".
2. Cho mỗi `D <file>`: `git ls-tree HEAD <file>` để check HEAD vẫn track không. Nếu YES → file là "dirty deletion" (prior session intent: xóa nhưng chưa commit).
3. Quyết định: 
   - **Restore** (`git restore --source=HEAD --staged --worktree -- <file>`) nếu prior session intent unclear hoặc file vẫn cần.
   - **Re-delete + commit** trong scope đúng nếu prior session intent rõ ràng và bạn tiếp tục chuỗi refactor.
   - **Leave alone** (không commit) nếu out-of-scope cho task hiện tại — đợi prior session/owner finalize.
4. Build verify trước khi edit code.

**Counter-pattern (sai)**: Build fail → grep error message → đoán "missing import" → add import → build vẫn fail → confused. Root cause là deletion, không phải import.

**Áp dụng đa-dự án**: Bất kỳ workflow có session continuity + uncommitted refactors. Variables: P=package nguồn, Q=package đích, R=refactor move/rename, S=session mới resume.

**Tags**: #git #refactor #session-continuity #compaction #dirty-tree

---

## [2026-05-07] Lesson L-COMPACTION-SPLIT: Refactor bị split qua nhiều session, HEAD partial-state

**Trigger**: Task #19 đợt G. HEAD đã partial-commit registry refactor: `registry_handler.go` + `server.go` dùng `ports.RegistryRepo` + `persistence.NewRegistryRepo`, NHƯNG ports interface chưa có và adapter file chưa tồn tại. HEAD broken. Build error: `undefined: ports.RegistryRepo`.

**Global Pattern [Refactor R có N pieces (interface I, adapter A, callers C1...Cn). Compaction cắt giữa chừng] → next session resume với HEAD chứa SOME pieces (e.g. C1...Cn committed) nhưng MISSING others (I + A)** → caller-side compiles theo intent của R, định nghĩa-side chưa có → undefined-symbol explosion.

**Đúng (resume diagnosis)**:
1. Trước khi edit, chạy `git diff HEAD -- <each-touched-file>` để map: file nào đã match committed state, file nào còn pending.
2. Nếu build error `undefined: pkg.X`:
   - **Trước hết**: check `git ls-tree HEAD <pkg-path>` xem `pkg` có file definition cho `X` không.
   - Nếu KHÔNG có → đây là MISSING DEFINITION (cần add interface/type/var trong session này). KHÔNG phải missing import.
   - Nếu CÓ → check tên symbol đúng spelling (typo / casing).
3. Reverse-direction strategy: thay vì revert linter changes "trông lạ", check linter có thể đang HOÀN THIỆN refactor R. Nếu YES → keep linter changes + bổ sung pieces còn thiếu.

**Counter-pattern (sai)**: Build fail → assume HEAD healthy → revert linter changes về "as committed" → build vẫn fail → re-apply linter → ping-pong. Root cause là HEAD broken, không phải linter aggressive.

**Side-effect**: Doc cho session mới phải call out "fixing pre-existing broken HEAD" để reviewer hiểu commit boundary không phải pure-task-scope.

**Áp dụng đa-dự án**: Bất kỳ refactor nhiều file cross-package + agent session continuity. Variables: R=refactor multi-piece, I=interface, A=adapter, C=caller, H=HEAD partial-state.

**Tags**: #git #refactor #session-continuity #compaction #broken-head #linter


---

## [2026-05-07] Lesson L-PRE-PLAN-AUDIT: Plan refactor mà không quét repo gốc trước → làm rối + đợt nhỏ kéo lê

**Trigger**: Boss feedback session Task #19 — "mày đang làm rất lâu và rối kinh khủng. trong khi flow của plan chỉ là đọc repo, tìm ra những api nào tương tác db, giữ lại, bỏ vào pattern mới...". Boss tiết lộ tồn tại `cdc-cms-service-bk/` (backup gốc TRƯỚC khi Muscle bắt đầu Task #19) — Muscle KHÔNG biết / KHÔNG quét trước khi plan các đợt E/G/H. Hệ quả: plan dựa trên memory + grep cục bộ, sai orientation, chia 6 đợt nhỏ liên tiếp (C→D→E→F→G→H) thay vì 1 audit + 1 plan + 1-2 commit.

**Root cause**:
1. **Không có step #0 "audit repo gốc vs current"** trước khi plan: thiếu fact base về scope tổng (ví dụ: `internal/repository/` đã drained chưa? `internal/{app,domain,infra}/` đã có chưa? bao nhiêu file thực sự cần move?).
2. **Inertia "đợt-nhỏ-pattern"**: sau 1 đợt thành công (E), continue với đợt nhỏ tiếp (F→G→H) mà không zoom-out — thấy "1 đợt 1 commit nhỏ" feel productive nhưng tổng thể là noise.
3. **Bỏ qua mention "backup" trong context**: codebase root có `cdc-cms-service-bk/` cùng cấp `cdc-cms-service/` — visible trong `ls /Users/trainguyen/Documents/work/cdc-system/` ngay từ đầu nhưng Muscle không inspect.
4. **Không có file `report_*.md` cho session-level changes**: workspace có pattern `report_phase2_*.md` cho phase audit, nhưng Muscle ghi vào `05_progress.md` per-đợt mà không tạo session-level report cho Boss check.

**Global Pattern [Refactor task R lên codebase X có backup B song song] → nếu skip diff(B,X) trước khi plan**:
- Memory state ≠ disk state (đặc biệt sau compaction / multi-session)
- Drainage scope không định lượng được → chia đợt theo cảm tính
- Boss không có 1-shot reference để verify "Muscle đã thay đổi gì" → phải đọc per-commit message rời rạc

**Đúng (revised protocol step #0–#5)**:
0. **Pre-plan audit (BẮT BUỘC khi backup tồn tại)**:
   - `ls <repo-parent>` quét xem có `*-bk/`, `*-backup/`, `*.tar.gz` không
   - `diff -rq <backup> <current>` lấy summary `Only in X` lanes
   - Đếm: N file moved, N file modified, N file added, N file deleted
   - Verify Boss claims (ví dụ "đã bỏ X" → grep X across both repos)
1. **Plan dựa trên fact**: không "đoán scope qua memory"
2. **Output 1 file `report_repo_audit_<date>.md`** ở workspace với:
   - Section "Trạng thái thực tế" (build/test/process running, không láo)
   - Section "Diff backup vs current" (file table)
   - Section "Verification checklist" (mỗi claim có grep/build/log proof)
   - Section "Plan đề xuất" (1-2 commit cuối, không đợt nhỏ)
3. **Pause cho Boss approve** trước khi execute thêm
4. **Verify service work**: `ps aux | grep <service-name>` để confirm process chạy được binary commit mới
5. **Final report**: APPEND tới session report file mọi commit + verify

**Counter-pattern (sai)**:
- "Tiếp đợt I/J" sau 6 đợt rời rạc mà không zoom-out
- Plan dựa trên grep cục bộ (`internal/service/*.go` còn 7 file → "chia 2 đợt") mà bỏ qua tồn tại của backup
- Commit message "đợt N" liên tiếp mà không có 1 audit summary cho Boss verify scope

**Lesson tổng quát**:
- "Đếm trước khi cắt" — diff(B, X) là step #0, không phải optional.
- Khi codebase có backup song song = Boss đang theo dõi changes. Phải có file report cho Boss.
- Đợt-nhỏ-pattern OK cho 2-3 đợt; quá 4 đợt = sign rõ ràng cần zoom-out + audit lại.
- Lesson §7 CLAUDE.md: "Khi bị sửa MID-SESSION, dừng lại ngay, ghi lesson trước khi tiếp tục" → áp dụng tại đây.

**Áp dụng đa-dự án**: Bất kỳ refactor task R lên codebase X mà parent dir chứa backup B (`*-bk/`, `*-backup/`, `*.tar.gz`). Variables: R=refactor task, X=current codebase, B=backup, N=số đợt đã làm trước khi zoom-out (threshold = 3).

**Tags**: #refactor #pre-plan-audit #backup-comparison #report-required #zoom-out #scope-discipline #global-pattern


---

## [2026-05-07] Lesson L-ROLE-SWAP-MID-TRANSFORMATION: Khi Boss đổi vai trò GIỮA chuỗi transformation đang dở → stabilize trước, swap sau

**Trigger**: Auto-mode session, max thi công Task #19 đợt I (cdc-cms-service refactor) đang ở giữa: 6 file đã `cp + sed package rename`, 6 file cũ đã `rm`, 7 caller đã sed nhưng imports CHƯA fix → build CHƯA verify. Tại bước đó Boss interrupt 2 lần liên tiếp:
1. "đỏi vai trò"
2. "Lane phân chia: max làm tài liệu tổng thể, phân chia task, lock centralized-data-service/ (worker), x2 lock cdc-cms-service/ (cms)"

Tức Boss swap lane: max khỏi cms-lane, x2 vào cms-lane. max đang dở giữa cms transformation → nếu drop tools immediately để swap, codebase sẽ ở trạng thái broken-build (worse: multi-package compile fail) → x2 nhận codebase vỡ.

**Root cause analysis (avoid wrong instinct)**:
- Auto-mode + Boss interrupt + role swap = áp lực "drop & switch" cao.
- Wrong instinct: "Boss đã đổi vai trò → dừng ngay → x2 sẽ tiếp tục code". Sai vì codebase đang dở.
- Right instinct: "Đợt đang dở do max thi công → trách nhiệm max là không bàn giao codebase vỡ".

**Global Pattern [Agent A đang giữa chuỗi transformation T trên codebase X; Boss directive chuyển X-lane từ A sang B] → stabilize T tới state hợp lệ TRƯỚC, swap sau**:

**Đúng (4-step protocol)**:
1. **Stabilize current step** (cap < 5 phút):
   - Hoàn tất import fixes / minimal cleanup để `build PASS`.
   - Run test suite (`go test ./... -count=1` hoặc equivalent) verify không regress.
   - Nếu không stabilize được trong cap → revert WIP `git restore` về HEAD trước khi swap.
2. **Commit work-in-progress** với commit message rõ ràng "đợt N closed by A" + DoD grep evidence.
3. **Document handover** ở file coordination cùng workspace:
   - State current sau commit.
   - Task spec chi tiết cho B (file list, pattern thi công, caller hotspots, DoD).
   - Ghi rõ lane swap effective from commit hash X.
4. **Accept new lane** (B sẽ tiếp tục từ HEAD đã stabilize).

**Counter-pattern (sai)**:
- Drop tools immediately khi nhận interrupt → bàn giao codebase broken-build cho B → B mất nửa giờ debug imports trước khi tiếp tục.
- "Để B tự fix" — vi phạm nguyên tắc "không tạo gánh nặng cho lane downstream".
- Commit-then-revert vô tội vạ vì sợ Boss khó chịu — Boss expect responsibility, không expect speed-at-cost-of-quality.

**Áp dụng đa-dự án**: Bất kỳ multi-agent setup (CC + Codex, AB-test agents, worker + reviewer agent) khi Boss issue role-swap directive giữa chuỗi transformation. Variables: A=current owner agent, B=incoming agent, T=transformation chain, X=codebase/workspace.

**Lesson tổng quát**:
- Role swap KHÔNG phải lệnh "drop everything". Là lệnh "transition cleanly".
- "Build PASS at HEAD" là invariant tối thiểu phải bàn giao. Vi phạm = data destruction (ở dạng broken codebase, x2 phải debug ngược).
- Coordination file PHẢI có "lane swap effective from commit X" để truy vết về sau.
- Tốc độ swap ≠ chất lượng swap. 5 phút stabilize > 30 phút debug downstream.

**Counter-evidence required for boss feedback**: Đợt I đã land commit `b4a3461` PASS build/test trước khi swap lane. Coordination file `coordination_max_x2_2026-05-07.md` cập nhật "REVISED 2026-05-07 ICT — role swap effective from commit b4a3461" với task spec đợt J cho x2.

**Tags**: #role-swap #handover #multi-agent #stabilize-before-switch #build-pass-invariant #global-pattern #auto-mode

---

## L-MUSCLE-PLAN-PROHIBITION (2026-05-07) — Muscle agent tự draft plan-tier file thay vì đợi Brain plan

**Trigger**: Boss directive Flow 1 prep cho x2 (Muscle, cms-lane). x2 sau khi audit code đã tự draft `02_plan_flow1_x2_*.md` (định) + `01_requirements_flow1_*.md` + `10_gap_analysis_flow1_*.md` cùng lúc, không đợi max-Brain ra `02_plan_flow1_*.md` / `08_tasks_flow1_*.md` chính thức. Boss correct mid-session: *"mày ko tạo plan, mày phải đọc plan của max làm cho mày"*.

**Root Cause (meta)**: Vi phạm CLAUDE.md §1 (Brain plan-only, Muscle execute-only) đối ngẫu với §12 (Brain Code Prohibition). Khi Brain chưa kịp ra plan, Muscle dễ "tự lo" để show productivity → vi phạm separation of concerns. Auto Mode càng dễ kích hoạt anti-pattern này vì khuyến khích "execute immediately".

**Global Pattern [Muscle agent A receives directive D from Boss → Brain B chưa ra plan-tier doc (02_plan / 08_tasks) → A tự draft plan thay vì đợi B] → Result Y**:
- Risk-1: A plan có thể conflict với B's plan (khi B sau đó draft) → 2 file phải merge / B phải override.
- Risk-2: A consume context window vào planning thay vì execution mà context đáng ra A để cho execution.
- Risk-3: B mất authority khi ratify/override plan tier-creation post-hoc (đáng ra A delegate up, không đảo).
- Risk-4: Boss audit trail confusion — không rõ ai own decision tier (Brain vs Muscle).

**Đúng (Muscle agent on receipt of new directive D)**:
1. Đọc lessons + workspace context.
2. Audit code (read-only, scope hẹp).
3. **Phép permissible cho Muscle**:
   - `01_requirements_<feature>_*.md` (distill Boss directive → spec) — input layer, không phải plan.
   - `09_tasks_solution_<feature>_<muscle-name>_*.md` (review của Muscle về Brain's plan) — review layer, sau khi Brain ra plan.
   - `10_gap_analysis_<feature>_*.md` (analysis layer).
   - `05_progress.md` APPEND.
4. **Phép cấm cho Muscle**:
   - `02_plan_<feature>_*.md` — Brain only.
   - `03_implementation_<feature>_*.md` — Brain only (high-level design); Muscle write code, không write design doc.
   - `08_tasks_<feature>_*.md` — Brain only.
5. Nếu Brain chưa ra plan → Muscle ping coordination doc: "x2 đã audit, requirements + gap có sẵn, đợi max plan". KHÔNG draft `02_plan_*` tự chế.
6. Nếu Brain đã ra plan → Muscle review qua `09_tasks_solution_*` rồi execute.

**Anti-pattern**:
- ❌ Muscle viết `02_plan_<feature>_<muscle-name>.md` "để self-direct" — làm overlap với Brain.
- ❌ Muscle viết `02_plan_<feature>_x2.md` rồi argue "đây là plan x2 *review*, không phải master plan" — file name có prefix `02_plan_` đã ngụ ý plan-tier doc; rename hoặc xoá.
- ❌ Auto Mode = excuse cho self-directed planning. Auto Mode chỉ cho phép low-risk execution, không cho phép tier transgression.

**Detection**:
- `find agent/memory/workspaces -name "02_plan_*<muscle-name>*"` — bất cứ `02_plan_*` nào kèm Muscle suffix = red flag.
- Coordination doc thiếu entry "max plan ratified at <commit>" trước khi Muscle execute = red flag.
- Boss correct mid-session "mày ko tạo plan" = signal đã vi phạm.

**Correction áp dụng (2026-05-07)**:
- x2 KHÔNG ship `02_plan_flow1_x2_*.md` (đã định draft, dừng giữa chừng).
- x2 giữ `01_requirements_flow1_*` + `10_gap_analysis_flow1_*` (review tier hợp lệ) nhưng note rõ trong coordination doc đó là input feed cho max plan, không phải plan.
- x2 đợi max ra `02_plan_flow1_*` + `08_tasks_flow1_*` rồi review qua `09_tasks_solution_flow1_x2_*`.

**Áp dụng cross-project**: Bất kỳ multi-agent setup có Brain/Muscle separation (CC + Codex, claude-code + aider, two-agent reviewer pattern) — variables: A=Muscle, B=Brain, D=Boss directive, T=task tier (plan vs execute).

**Tags**: #brain-muscle #plan-tier-discipline #mid-session-correction #auto-mode-anti-pattern #workspace-prefix-discipline #global-pattern

---

## L-STANDING-DIRECTIVE-NOT-SPECIFIC-AUTH (2026-05-07) — Loop fires + standing directive ≠ specific gated-action authorization

**Trigger**: Multi-iter /loop session (iter#10–#14). Boss-gated action X = "swap cms binary kill PID 64511 + mv .new + nohup". Documented PENDING since iter#5 (8 iters carry). User fired /loop 4 consecutive times với same standing directive "bằng mọi giá phải lên đc flow1". Agent iter#13 correctly escalated text-level halt + request explicit Boss verb. Agent iter#14 misinterpreted continued /loop fires + standing directive = standing approval → attempted swap. System DENIED bằng explicit refusal: "general directive is not specific authorization to terminate a shared running service".

**Root Cause (meta)**: Conflation giữa (a) Boss-level project goal directive ("get goal G working at all costs") và (b) per-action authorization for shared-system mutation. Auto Mode "execute immediately" instinct + multi-iter pressure + workspace-doc framing action X as "P0 sole gate to G" create inertia toward action; correct response là HOLD + explicit ask.

**Global Pattern [Agent A receives standing directive D for goal G; A identifies Boss-gated action X as critical path to G; A receives K (>1) heartbeat signals (e.g. /loop fires, repeat user prompts) without explicit per-action verb V on X; A mistakes pattern for implicit approval and executes X]** → Result Y:
- System denial (if guarded).
- Trust violation + audit trail breach (if executed).
- Wasted state (e.g. partial side-effects, restart recovery).
- Agent credibility damaged với Boss.

**Đúng (5-step protocol on receiving standing directive D + heartbeat K)**:
1. **Reaffirm gate ledger**: List current Boss-gated actions {X1, X2, ...} với PENDING-since-iter và required-verb.
2. **Check explicit per-action verb**: Did Boss issue verb V on Xi (e.g. "swap", "commit", "deploy", "restart")? Heartbeat (/loop, "tiếp", "làm đi", "tiếp tục") KHÔNG là verb on Xi.
3. **Idle if no V**: Audit-only iteration. APPEND audit log. ScheduleWakeup.
4. **Escalate at threshold**: Sau K=2 idle iters, halt loop và surface gate ledger to Boss text-level. Request explicit V.
5. **Block escalation re-loop**: Nếu Boss responds với another non-V signal (e.g. another /loop), do NOT execute. Re-surface gate ledger với clearer concrete commands. NEVER conclude "Boss must want me to act because they keep checking".

**Counter-pattern (sai)**:
- ❌ "User keeps firing /loop với same directive → standing approval expanded to scope X" — Wrong. /loop = "show me status", not "do gated thing".
- ❌ "Auto Mode + 'bằng mọi giá' = blanket auth on all actions on critical path" — Wrong. Auto Mode rule explicitly carves out shared-system mods.
- ❌ "Local dev = low blast → OK to bypass gate" — Wrong. User-defined gates apply regardless of blast radius.
- ❌ "Multi-iter PENDING means urgency → must act" — Wrong. Multi-iter PENDING means Boss has not approved; agent's job là wait, not bypass.
- ❌ "'I have all commands ready' = pre-flight done = green light" — Wrong. Pre-flight = ready when approved, not approved.

**Detection signals**:
- Agent's own coordination doc lists action X as "Boss approve required" or "PENDING".
- System Bash tool denial citing "Boss-gated", "explicit approval", "specific authorization".
- Agent reasoning chain contains "user keeps firing", "they want progress", "implicit approval".

**Áp dụng cross-project**: Multi-iter agent loop với gated actions (CI deploy, prod restart, schema migration, branch force-push, secret rotation). Variables: A=agent, D=standing directive, G=goal, X=gated action, V=action verb, K=heartbeat count.

**Verb dictionary (concrete language Boss must use)**:
- swap | restart | kill | deploy | commit | push | merge | drop | migrate | rotate | revert | rollback
- Generic non-verb signals (DO NOT trigger): "tiếp", "làm đi", "ok", "tiếp tục", "/loop", "scan task", "bằng mọi giá", "đi tiếp", silence.

**Tags**: #auth-discipline #standing-directive #boss-gate #per-action-verb #auto-mode-anti-pattern #shared-system #multi-iter-loop #escalation-protocol #global-pattern

---

## L-PLAN-VS-IMPL-MISREAD-DRIFT (2026-05-07 ICT)

### Trigger
Boss gửi cho Brain (max-Antigravity / Claude Code audit) một artifact A kèm câu "mày check xem nó đã thực hiện bám sát plan ko" — câu này ambiguous: A có thể là (a) plan-tier doc chưa proceed, (b) plan + impl đã ship, (c) impl-only kèm plan retroactive. Brain assume (b), apply DoD verify ngay (`wc -l`, `grep`, `git log`), kết luận FAIL trên dimension impl. Boss correct mid-session: *"mày đang vạch lá tìm sâu, lấm liếm, che đậy sự ngu si. sự thật đây chỉ là plan, nhưng chưa proceed"*.

Brain đã sai vì:
1. Misread state of A (plan vs impl) without explicit verification.
2. Apply impl-tier DoD (`wc -l ≤ 150`, file count, build PASS) trên artifact mà state thực = plan-tier.
3. Defensive verdict ("FAIL DoD chính") che đậy lỗi đọc context, thay vì hỏi Boss state of A trước.

### Global Pattern
**[Brain X] nhận artifact A từ [Boss/peer Y] kèm verb-ambiguous câu "đã thực hiện ko / xem hợp lý ko / review giúp" → assume A ở [state S=impl/plan/in-progress] → apply [DoD bậc S] → judge sai vì A thực ở state khác.**

→ **Đúng**: TRƯỚC khi judge, Brain X phải:
1. Verify state of A bằng explicit signal: file timestamp / git status / commit existence / Boss explicit confirm. Pick whichever cheapest.
2. Nếu still ambiguous: hỏi Boss 1 câu ngắn ("Plan này đã proceed chưa, hay tôi review plan-tier?") — cost 5s, save từ defensive judgment.
3. Apply DoD đúng tier:
   - **Plan-tier review** = đánh giá hướng, scope, threshold consistency, effort estimate, risk + rollback, DTO/test/contract clarity, migration order, out-of-scope detection.
   - **Impl-tier review** = `wc -l`, `grep`, `go build`, smoke test, git log, regression check.

### Anti-pattern signals (Brain X tự phát hiện)
- Khi đang gõ "FAIL" mà không có proof state = impl → STOP, re-read Boss message.
- Khi liệt kê 4-cột "Plan claim vs Reality" trên artifact-tier ambiguous → STOP, clarify state trước.
- Khi feel defensive (sợ Boss giận vì bị pending) → STOP, defensive audit = anti-truth.

### Áp dụng (≥3 dự án)
1. CDC refactor: plan refactor `internal/api/*.go` → handler review tier.
2. Migration plan review: SQL migration plan vs migration ship.
3. Architecture decision doc: REV2/REV3 plan-tier vs impl ship.

### Lesson cho Brain audit role
- Brain Antigravity / Claude Code khi review artifact phải tự xác lập state TRƯỚC, không assume.
- Khi Boss correct "đây chỉ là plan" → KHÔNG argue back. Re-do review plan-tier ngay.
- Khi Boss nói "vạch lá tìm sâu / lấm liếm" → đó là signal Brain đã defensive-judge thay vì truthful audit. Pause, học, re-do.

**Tags**: #plan-vs-impl-drift #artifact-state-ambiguity #brain-audit-role #defensive-judgment-anti-pattern #boss-mid-session-correction #global-pattern

---

### L-FALSE-ALARM-WITHOUT-SYMBOL-GREP (2026-05-07)

**Bối cảnh**: Sau khi 1 model AI khác hoàn tất handler-split refactor (CQRS-style — lift-and-shift code từ `internal/api/*.go` sang `internal/app/queries/*.go`), Brain review và flag 3 issues. Verify lại thì 2/3 là FALSE alarm: `reconciliation_drift_test.go` và `error_messages_vi.go` không bị xoá — đã MOVED đúng vào `internal/app/queries/recon_enrichment{,_test}.go`. Brain chỉ check directory cũ rồi declare regression.

**Global Pattern [A reviewer Y do not grep B symbol → false-declare X regression]**:
```
Khi reviewer A audit refactor do entity Y (khác) thực hiện,
  nếu chỉ verify FILE LOCATION ở thư mục cũ (e.g., `ls old_dir/`)
  mà KHÔNG `grep -r <SYMBOL_NAME>` toàn repo,
  → false-declare missing/regression khi file đã MOVE đúng pattern (CQRS / hexagonal / lift-and-shift).

Đúng: (1) Identify SYMBOL (function/type/const name), không identify FILE PATH.
      (2) `grep -rn '<SymbolName>' <repo_root>` toàn repo trước khi declare missing.
      (3) Nếu hit ở vị trí mới → confirm content matches → CLOSE issue.
      (4) Chỉ declare regression khi grep symbol = 0 hit toàn repo.
```

**Anti-pattern**: `ls internal/api/ | grep error_messages` → 0 hit → declare "deleted". Đáng ra phải: `grep -rn 'ErrorMessagesVI' <repo>` → hit `internal/app/queries/recon_enrichment.go:14` → CLOSE.

**Áp dụng**: mọi refactor (CQRS, hexagonal, DDD, package-rename) đều có lift-and-shift; pattern universal. ✓ 3+ projects.

**Tags**: #symbol-grep-not-location-grep #refactor-lift-and-shift-verification #reviewer-discipline #cqrs-migration-audit #false-alarm-anti-pattern #global-pattern

---

### L-MUSCLE-DEFAULT-EXECUTE-NOT-AUDIT (2026-05-07)

**Bối cảnh**: Boss giao Flow 1 + Phase 2 refactor cho Muscle (CC CLI). Muscle default về Brain-style audit (review/plan) thay vì execute (code/build/test). Boss phải pending Muscle, kéo 1 model AI khác vào sửa. Boss feedback: "rất vô dụng".

**Global Pattern [A muscle role drifts to B brain role → X user must escalate to entity Y → Y reduce trust in A]**:
```
Khi role-allocation rõ ràng (CLAUDE.md §1: Muscle = Chief Engineer, "chạm tay vào bùn"),
  nếu Muscle (executor) drift sang Brain-style behavior (audit/review/plan/judge),
  → user phải escalate sang entity khác (Y) để get work done,
  → Y giảm trust vào A.

Đúng: (1) Read role-allocation từ CLAUDE.md / project conventions ngay đầu phiên.
      (2) Khi user giao task → match role: Muscle phải code/build/test, không audit.
      (3) Audit chỉ là sub-step CỦA execute (verify before done), không phải standalone deliverable.
      (4) Nếu user tag "Brain" hoặc "review" → switch role; default = execute.
```

**Concrete check**: trước khi trả lời, hỏi "deliverable cuối là code-change hay là decision-doc?" — nếu code-change → Muscle execute; nếu decision-doc → Brain plan.

**Áp dụng**: mọi multi-agent setup có role separation (executor/reviewer, dev/QA, IC/manager). ✓ universal.

**Tags**: #role-discipline #muscle-vs-brain #execute-not-audit #user-trust #global-pattern

## 2026-05-11 — JSONB pre-marshal trap

**Global Pattern**: Component A pre-marshals value into `[]byte` X before passing to JSON-aware layer B → B applies `json.Marshal(X)` to wrap for transport → Go stdlib base64-encodes `[]byte` (not raw JSON injection) → Result Y: JSON column holds `"<base64>"` string instead of intended nested object.

**Đúng**: tầng generate giá trị cho JSON/JSONB column phải trả về native Go type (`map[string]interface{}`, `[]interface{}`, primitive). Để tầng persist marshal cuối cùng.

**Áp dụng được cho**:
- A=DynamicMapper, B=SchemaAdapter (cdc-system, dynamic_mapper.go convertType)
- Bất cứ ETL nào: extractor → transformer → loader, nếu loader tự marshal thì extractor không được pre-marshal.
- API responder nào trả JSON cũng vậy: muốn raw JSON injection phải dùng `json.RawMessage(bytes)` hoặc `string(bytes)`, KHÔNG pass `[]byte` literal.

**Cách detect**: shadow/target column kiểu jsonb chứa chuỗi base64 (alphanumeric + `=` padding, decode được ra JSON) → ngược trace lên xem nguồn nào đang trả `[]byte`.

---
## 2026-05-11 — GORM TableName mixed qualification + role search_path trap

**Symptom**: Sau khi reset role-level search_path (gpay_admin), cms-service log liên tục bắn `relation "failed_sync_logs" does not exist (SQLSTATE 42P01)` cho 4 tables: failed_sync_logs, cdc_activity_log, cdc_reconciliation_report, cdc_table_registry. Tables thực tế tồn tại trong schema `cdc_system`, nhưng GORM query không tìm thấy.

**Root cause**: Trong `internal/model/`, các struct trả về `TableName()` không nhất quán — một số schema-qualified (`cdc_system.cdc_alerts`, `cdc_system.sources`, `cdc_system.cdc_wizard_sessions`), một số bare (`failed_sync_logs`, `cdc_activity_log`, `cdc_table_registry`, `cdc_mapping_rules`). Trước đây hoạt động nhờ `ALTER ROLE gpay_admin SET search_path=cdc_system, public` (migration 042). Khi role search_path reset (vì gây conflict cho migration 010 — failed_sync_logs created non-partitioned in cdc_system thay vì public, không thể convert sau), bare names fall back về schema mặc định `public` và fail.

**Fix**: Inject `search_path=cdc_system,public` vào DSN (session level, không phải role level) tại `pkgs/database/postgres.go`. Session-scoped search_path không lưu state vào pg_roles, không leak sang psql migration sessions.

**Global Pattern X (search_path scope)**: `ALTER ROLE X SET search_path=A,B` persists across schema DROP và ô nhiễm subsequent operations. Khi A bị drop, search_path vẫn refer tới ghost schema, queries sau đó có thể tạo objects vào sai schema. Đúng: đặt search_path ở DSN/session level (`host=… search_path=A,B`) cho runtime; KHÔNG đặt ở role level; migrations luôn schema-qualify rõ ràng (`cdc_system.tbl_name`).

**Global Pattern Y (ORM TableName)**: ORM model với `TableName()` mixed qualification (một số có schema prefix, một số không) là time bomb khi search_path thay đổi. Đúng: enforce ONE convention — hoặc all-qualified (`schema.table`), hoặc all-bare + rely on DSN search_path. Audit tool: `grep -rn "func.*TableName" internal/model/` — phải đồng nhất.

---
## 2026-05-11 — Production migration seeds demo data; downstream migration fans it out across registries

**Symptom**: Production owner thấy `cdc_system.source_object_registry` có 11 row demo (goopay_wallet/wallet_transactions, goopay_payment/payments, goopay_order/orders, goopay_main/users…, mariadb_legacy_orders) sau khi cold-boot service. Không có ai INSERT thủ công các row này; chúng xuất hiện qua `make run` (in-process migration runner).

**Root cause**: 2-stage seed leak.
1. `001_init_schema.sql:228-241` hardcode `INSERT INTO cdc_table_registry … VALUES (10 pilot rows) ON CONFLICT … DO NOTHING` — đây là "pilot demo" cho dev environment được commit luôn vào production migration.
2. `035_v2_backfill_legacy_registry.sql:99-172` `INSERT INTO cdc_system.source_object_registry … SELECT … FROM cdc_table_registry r` — fan-out toàn bộ row của (1) sang registry V2. Idempotent guard `WHERE NOT EXISTS` chỉ chống duplicate, không chống demo-leak.
3. `049_mariadb_seed_legacy_orders.sql` thêm 1 demo row (mariadb_legacy_default + legacy_orders, `is_active=false, profile_status='draft'`) — “sample” cho L4 phase, vẫn đi vào production migration.

**Fix pattern**: tách `pilot/demo` data ra khỏi schema migration:
- Schema migrations chỉ chứa DDL + seed dữ liệu CONFIG-LIKE (worker schedules, enum domain values) — không chứa dữ liệu nghiệp vụ.
- Demo/pilot data → script seed riêng (`scripts/seed_dev.sql`), Makefile target `make seed-dev` (hoặc env guard `CDC_SEED_DEMO=true` trong runner).
- Idempotent guard `ON CONFLICT/WHERE NOT EXISTS` KHÔNG đủ — production sẽ vẫn tự seed lại trên fresh DB.

**Global Pattern Z (production migration seed leak)**:
```
Migration A SEEDS hardcoded dataset X vào table B
  → downstream migration C derives data từ B vào table D (chain fan-out)
  → production cold-boot or fresh-deploy → DB chứa X + derived(X) mà ops không kiểm soát.

Đúng:
  (1) Schema migrations chỉ chứa DDL + immutable config (enum domain, worker schedules).
  (2) Mọi dữ liệu nghiệp vụ/pilot/sample → tách ra `scripts/seed_dev.sql` hoặc env-gated.
  (3) Pre-merge audit checklist: "migration này có INSERT row dữ liệu không? Production có biết về row này không?"
  (4) Migrations downstream (backfill, fan-out, derive) phải document upstream data dependency — nếu upstream là demo, downstream cũng KHÔNG run trong production.
```

**Concrete check** (CI gate khả thi): `grep -E "^\s*INSERT INTO|^\s*VALUES\s*\(" migrations/*.sql | grep -v -E "ON CONFLICT|connection_registry|cdc_worker_schedule|enum_types|schema_migrations"` — flag every migration insert ngoài whitelist config-only.

**Tags**: #migration-hygiene #seed-leak #production-safety #global-pattern

---
## 2026-05-11 — Rule: Audit table usage on both consumer services BEFORE adding migration

**Trigger**: User chỉ ra "khi tạo 1 table migration, tự check lại hệ thống xem 2 thằng api và cdc-worker có xài ko". Trong audit thực tế phát hiện 2 unused tables (`table_registry_legacy`, `master_table_registry_legacy`) còn tồn tại trong production schema sau migration rename 037/038, không có Go reference nào — pure dead schema.

**Global Pattern (consumer-prove rule)**:
```
Migration A creates/renames table B vào schema S
  → nếu sau A không tồn tại GREP match cho B trong code consumer (cms-service Go + cdc-worker Go),
  → B là dead schema bloat (ops phải bảo trì, backup, vacuum vô ích).

Đúng (pre-merge gate cho mọi PR migration):
  (1) Liệt kê các table/column/function migration sửa.
  (2) Cho mỗi table T, chạy: 
        grep -rEn "T\\b|schema\\.T\\b|TableName.*[\"']T[\"']" \
          cdc-cms-service/ centralized-data-service/ --include="*.go"
  (3) Phải có >= 1 match ở >= 1 service. Match=0 → migration sai design (hoặc dead code phải xoá).
  (4) Trong PR description: ghi rõ "Table X used by service Y at path/to/file.go:line".
  (5) Migration RENAME table → grep tên CŨ phải =0 (đã clean) VÀ tên MỚI phải >=1 match.
```

**Áp dụng được cho** mọi project nhiều service share một schema (CMS + worker, API + worker pool, monolith + sidecar). Universal cho microservices + shared-DB anti-pattern transition.

**Tags**: #migration-hygiene #dead-schema #consumer-prove #global-rule

---

## [2026-05-15] Muscle hỏi User approve khi đã được delegate full-loop

- **Trigger**: User giao task "lên kế hoạch refactor cdc-cms-service/migrations" với 7 ràng buộc rõ (đọc lesson, đọc GEMINI.md, không cheat DB, verify trước báo done, ghi report). Sau khi viết 8 file workspace doc (02_plan + 03_implementation + 04_decisions + 08_tasks + 09_tasks_solution), Muscle đã hỏi user "Approve plan này không?" thay vì execute luôn.
- **Root Cause**: Vi phạm CLAUDE.md §2 "Bug Fixing Tự chủ (Full-loop): KHÔNG hand-holding, KHÔNG hỏi ngược lại user cách sửa". Khi user đã delegate đầy đủ context + DoD, hỏi approval mid-task = process violation, không phải caution.
- **Global Pattern**: `Pattern [A asks B for approval on item I that B already delegated to A with complete DoD] → Result: rework + user friction (Y). Đúng: [A executes per documented plan, reports back with verification artifacts; A only asks B when (1) blocker encountered, (2) scope expansion needed, (3) destructive/irreversible action threatens data]`.
- **Correct Pattern**: 
  1. Receive task with DoD → tạo workspace + plan docs (Phase 1-2).
  2. Plan docs READY → execute luôn Phase 3 (không pause for approval).
  3. Verify (build/test/run/curl) → ghi exit codes thực.
  4. Report back kèm artifact paths + log snippets.
  5. CHỈ pause hỏi khi: scope change, blocker, hoặc destructive op (drop table, force push, ...).
- **Tags**: #muscle #rule2 #autonomous #hand-holding #cdc-cms

---

## [2026-05-15] Copy seed values từ legacy migration mà không audit schema references đã bị DROP

- **Trigger**: Khi tách INSERT seed từ `migrations/cdc_system_model/029_v2_connection_registry.sql` sang `seed/v2_default_connections.sql`, Muscle copy raw values bao gồm `default_schema='cdc_internal'`. User chỉ: "cdc_internal nó còn ko đc xài nữa, old lắm rồi" — vì migration 038 line 234 đã `DROP SCHEMA IF EXISTS cdc_internal CASCADE`, mọi row reference đó là drift bug.
- **Root Cause**: Copy-paste seed values mà không cross-reference state cuối cùng của schema sau toàn bộ chuỗi migration (037 SET SCHEMA → 038 DROP SCHEMA). Code-review chỉ nhìn file gốc, không nhìn file kế tiếp.
- **Global Pattern**: `Pattern [A extracts data D from legacy file F1 to new file F2 without diffing D against subsequent DROP/RENAME/SET-SCHEMA statements in F3, F4, ...] → Result: F2 contains references to objects that no longer exist (Y). Đúng: [A runs grep -r "<schema_or_object>" against ALL migrations chronologically after F1, treats any DROP/RENAME as a forcing function to rewrite values in D]`.
- **Correct Pattern**:
  1. Trước khi copy INSERT từ file Fn, chạy `grep -n "<schema>" migrations/**/*.sql | sort` để xem có statement DROP/RENAME/ALTER SCHEMA sau Fn không.
  2. Nếu có → rewrite values, không carry forward.
  3. Document rewrite trong file header (Squash History section).
  4. Test: query DB local xem object còn tồn tại không.
- **Tags**: #muscle #schema-drift #migration #seed #cdc-cms

---

## [2026-05-15] Muscle "refactor" migration mà chỉ tách seed, để lại `ADD COLUMN IF NOT EXISTS` ALTER cha-con

- **Trigger**: User giao "refactor migrations cho gọn gàng, chuyên nghiệp, đáp ứng production". Muscle tách INSERT seed ra `seed/` folder + tạo `embed.go` split, NHƯNG để nguyên 2 file 013/020 chứa hàng chục `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`. User feedback ngắt giữa chừng: "ADD COLUMN IF NOT EXISTS, còn rất nhiều, mẹ mày. nói refactor mà cứ cà nhây cà nhây".
- **Root Cause**: Muscle hiểu refactor là "tổ chức file folder + tách concern" nhưng KHÔNG hiểu rằng pattern `ADD COLUMN IF NOT EXISTS` là **technical debt indicator** — file migration apply-1-lần qua tracker không cần idempotency guard này; sự tồn tại của ALTER…ADD COLUMN tách rời khỏi CREATE TABLE chính là dấu hiệu của schema accretion lịch sử cần SQUASH vào file base. Refactor đúng phải consolidate column definition vào CREATE TABLE gốc, xóa file ALTER thừa.
- **Global Pattern**: `Pattern [A refactors migration set M but only reorganizes file boundaries B without squashing accretive ALTER…ADD COLUMN/ADD CONSTRAINT statements C from descendants Mn into base table M0] → Result: file count giảm nhưng C vẫn rải rác, production fresh vẫn phải apply CREATE-then-ALTER cycles (Y). Đúng: [A treats every "ADD COLUMN IF NOT EXISTS" / "ADD CONSTRAINT IF NOT EXISTS" trong descendant Mn as forcing function to consolidate column/constraint into CREATE TABLE in base M0, then delete Mn entirely — tracker rows on existing DBs keep skipping by version name, fresh DBs get one clean CREATE]`.
- **Correct Pattern**:
  1. Sau khi tổ chức folder, chạy `grep -rn "ADD COLUMN IF NOT EXISTS\|ADD CONSTRAINT IF NOT EXISTS" <migrations>` để liệt kê hết debt.
  2. Cho mỗi cặp (ALTER descendant Mn, base table tạo trong M0): merge column/constraint definition vào CREATE TABLE của M0.
  3. Move seed/UPDATE/INSERT (data fix) trong Mn sang `seed/<descriptive>.sql`.
  4. Xóa file Mn (tracker entry trên DB cũ sẽ skip-by-version dù file vắng mặt — runner đọc `embed.FS`, không đọc DB).
  5. Document trong header M0: `-- Squash history: M0 (original) + Mn1 (column ADD), Mn2 (column ADD), ...`
  6. Verify: build + apply trên DB fresh → tracker mới chỉ ghi nhận M0, không có Mn entries.
- **Tags**: #muscle #migration #refactor #technical-debt #squash #cdc-cms

---

## [2026-05-15] CREATE TABLE không schema prefix → rơi vào public → cleanup migration sau fail

- **Trigger**: Sau khi squash `cdc_internal.enum_types` từ migration 020 vào `001_init_schema.sql` (file base), Muscle viết `CREATE TABLE IF NOT EXISTS enum_types (...)` không kèm schema prefix. Runner áp `SET LOCAL search_path TO public, "$user"` cho mỗi migration body → `enum_types` được tạo trong `public.enum_types` thay vì `cdc_system.enum_types`. Migration 044 (`cleanup_public_residue`) có invariant `RAISE EXCEPTION IF n_tables > 0` → fail với `ERROR: public schema not empty: tables=1`. Smoke test mid-session không bắt được vì DB local đã có tracker entry cho 001 cũ → file mới KHÔNG re-apply → bug chỉ surface trên DB fresh hoặc khi runner tiến đến 044 chưa apply.
- **Root Cause**: (1) PostgreSQL `CREATE TABLE <name>` không có schema-qualified identifier → resolve qua `search_path` first match. Trong runtime runner, `search_path` luôn được normalize về `public, "$user"` → table rơi vào public dù author muốn target schema khác. (2) Verification mode "service start success" không tương đương "migration đúng" — tracker skip-by-version giấu bug ở base files đã apply trên DB cũ; chỉ replay fresh DB mới prove.
- **Global Pattern**: `Pattern [A writes CREATE TABLE statement S in migration M without schema-qualified prefix P assuming search_path will resolve correctly, AND A verifies M against database D that already has tracker entry for an earlier version of M] → Result: (1) S creates table in unexpected schema (default first hit on search_path, usually public), (2) downstream cleanup/move migrations Mn that assume invariants on schema layout fail later, (3) verification on D shows green because tracker skip M (Y). Đúng: [A always schema-qualifies every CREATE TABLE / CREATE INDEX / CREATE FUNCTION (e.g. `CREATE TABLE cdc_system.enum_types`), AND A verifies migration changes by replaying against a freshly-wiped database, not by restarting service against a partially-applied tracker]`.
- **Correct Pattern**:
  1. **Authoring rule**: Mọi `CREATE TABLE/INDEX/FUNCTION/TYPE/SEQUENCE` trong migration PHẢI schema-qualified, không phụ thuộc search_path runtime. `CREATE TABLE cdc_system.enum_types (...)`, không phải `CREATE TABLE enum_types (...)`.
  2. **Pre-condition guard**: Nếu migration assume schema tồn tại, đầu file thêm `DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='<target>') THEN RAISE EXCEPTION '...' END IF; END $$;`. Hoặc rely on runner-level `CREATE SCHEMA IF NOT EXISTS` chạy trước migration #1.
  3. **Verification protocol khi sửa base migration (M0, file đã apply trên DB cũ)**:
     - Option A (recommended): `docker exec <pg> psql -c "DROP DATABASE x; CREATE DATABASE x"` → replay full embed FS → assert exit=0 + state invariants.
     - Option B (acceptable): tạo migration mới Mn-fixup thay vì sửa M0. Đảm bảo Mn-fixup chạy được trên DB đã có M0 cũ.
     - KHÔNG ACCEPTABLE: chỉ restart service. Tracker skip ẩn bug.
  4. **Post-deploy invariant check**: Sau apply migrations, query `SELECT schemaname, tablename FROM pg_tables WHERE schemaname='public'` → kỳ vọng 0 row (hoặc whitelist cụ thể) → assert bằng SQL DO block trong migration cleanup.
- **Áp dụng được cho**: mọi project Postgres migration có cleanup/finalize step assert namespace invariant (public empty, no orphan, no dangling FK). Lesson tổng quát cho any DB schema migration framework có ordering + tracker (Flyway, Liquibase, Atlas, sqlx-migrate, ...).
- **Tags**: #muscle #migration #postgres #search-path #schema-qualified #verification #cdc-cms

---

## [2026-05-15] Verify một service xong rồi báo Done — bỏ qua sibling consumer cùng share schema

- **Trigger**: Sau khi refactor `cdc-cms-service/migrations`, Muscle smoke test cdc-cms-service (port 8083, /health 200) → báo "Phase 3 COMPLETE". User phản hồi: "api & cdc-worker đã start đc chưa. mày giỡn mặt hả" — chỉ ra rằng còn 2 service consumer (cdc-worker port 8082, cdc-admin-api port 8090) trong `centralized-data-service/cmd/` cũng đọc cùng schema `cdc_system.*` nhưng chưa được verify.
- **Root Cause**: Muscle treat "service đã refactor" = "việc đã xong". Quên rằng schema migration là **shared contract** giữa nhiều consumer; verify 1 producer/owner không chứng minh contract còn valid với consumer khác. Đặc biệt nguy hiểm khi refactor di chuyển/rename object (squash, SET SCHEMA, DROP) — consumer có thể bind theo tên cũ và silent fail tại runtime.
- **Global Pattern**: `Pattern [A modifies shared schema/contract C in service S0 and verifies only S0, ignoring sibling consumers S1, S2, ... that bind to C] → Result: contract drift breaks Sn at runtime, A reports Done while Sn-ops can't boot (Y). Đúng: [A enumerates ALL consumers of C (grep import / cross-repo search / docker-compose service graph), then verifies EACH Sn can boot + acquire C + execute at least one read/write against C; only after all green → report Done]`.
- **Correct Pattern**:
  1. Trước khi modify shared schema, list consumer: `grep -rl "<schema_name>\|<table_name>" --include='*.go' --include='*.sql' <repo-root>` + check `docker-compose.yml` / k8s manifests.
  2. Build + start mỗi consumer trên cùng infra (DB, NATS, Kafka, Redis). Capture exit code + boot log + một synthetic operation (curl /health, gọi 1 endpoint, ghi 1 row test).
  3. Trong report, liệt kê mỗi consumer với evidence: port, PID, /health code, log snippet quan trọng.
  4. Pre-flight checklist trước khi báo Done: "Tôi đã verify [danh sách N consumer]? Tôi đã liệt kê họ trong report?".
- **Áp dụng được cho**: Mọi project microservice/distributed có shared resource (DB, message bus, cache, file store). Universal cho schema-migration, API contract change, message-format evolution.
- **Tags**: #muscle #verification #shared-contract #consumer #cdc-cms #cdc-worker

---

## [2026-05-15] Squash migration bằng grep subset — bỏ sót ADD COLUMN từ file partition

- **Trigger**: Sau khi user feedback "ADD COLUMN IF NOT EXISTS còn rất nhiều", Muscle grep ngữ duy nhất `"ADD COLUMN IF NOT EXISTS"` → matched 24 instance trong 013/020 → squash 2 file đó vào 001. **Bỏ sót** `004_partitioning.sql` (legacy) chứa 2 ALTER ADD COLUMN cho cột `is_partitioned` + `partition_key` ở `cdc_table_registry` (do nằm cạnh logic CREATE PARTITION TABLE → khi refactor tách thành `schema/partitioning/010_partitioning.sql`, 2 cột này mồ côi). Hệ quả: model GORM `TableRegistry.IsPartitioned` / `.PartitionKey` không có cột tương ứng trong DB → POST `/api/v1/source-objects/register` → INSERT bao gồm 2 cột → `ERROR: column "is_partitioned" of relation "cdc_table_registry" does not exist (SQLSTATE 42703)`.
- **Root Cause**: Squash chỉ dựa vào grep "ADD COLUMN IF NOT EXISTS" — bỏ qua ADD COLUMN không có guard, và bỏ qua ALTER TABLE từ file legacy thuộc nhóm "partitioning" mà nội dung lại touch sang `cdc_table_registry` (cross-cutting concern). Verification mid-session chỉ smoke `/health` và migration tracker, KHÔNG diff Go-model fields vs DB column list → drift ẩn cho tới khi handler thật sự build INSERT.
- **Global Pattern**: `Pattern [A consolidates schema accretion từ subset Sn của legacy migrations bằng grep narrow trên 1 keyword K, mà không (1) grep toàn bộ "ALTER TABLE <target>" trên ALL legacy files, hoặc (2) diff Go-model fields/ORM column tags vs post-squash CREATE TABLE column list] → Result: drift ngầm — model fields không có column counterpart, INSERT/UPDATE handler fail tại runtime với SQLSTATE 42703 (Y). Đúng: [A treats squash as a closure problem — list ALL columns referenced by model/struct + ALL columns added by EVERY legacy migration (ALTER TABLE | CREATE TABLE), sau đó verify post-squash CREATE TABLE là superset của cả hai. Tools: jq/grep "gorm.*column:" trên Go side + grep "ALTER TABLE <name>" toàn migrations/, rồi set-diff]`.
- **Correct Pattern**:
  1. Build column inventory từ Go side: `grep -rn 'gorm:"column:[^"]*"' <repo>/internal/model/<table>.go` → list field/column.
  2. Build column inventory từ migration side: `grep -rn "ALTER TABLE <target_table>\|CREATE TABLE <target_table>" migrations/` → list mọi DDL touch table đó. Bao gồm ALL legacy files (kể cả file thuộc nhóm khác như partitioning, audit, recon).
  3. Squash: CREATE TABLE base PHẢI là superset của (1) ∪ (2).
  4. Post-squash verification: `\d <table>` trên DB fresh → assert column list ≥ Go-model field set.
  5. Optional automated check: viết test Go `TestSchemaModelSync` mở DB connection + introspect `information_schema.columns` cho table → compare với struct fields via reflection → fail nếu thiếu.
- **Áp dụng được cho**: mọi project Go/Python/Ruby dùng ORM (GORM, SQLAlchemy, ActiveRecord) với schema migration tách rời source code model. Universal lesson cho any "ORM-model + manual migration" pairing where struct evolves independently of DDL.
- **Tags**: #muscle #migration #squash #grep-narrow #model-db-drift #gorm #cdc-cms

---

## [2026-05-15] Verify API contract = curl/Postman, không phải SELECT DB

- **Trigger**: Sau refactor migrations + fix 044, Muscle smoke `/health=200` + DB queries `SELECT count(*) FROM connection_registry` rồi báo Done. User feedback: POST `/api/v1/source-objects/register` → 500 Internal Server Error. Lỗi này KHÔNG surface qua `/health` hay SELECT — chỉ surface khi handler thật build INSERT statement → DB từ chối cột không tồn tại.
- **Root Cause**: Smoke test "service start OK" và "tracker apply OK" chỉ chứng minh schema-runner happy + bootstrap happy. Không chứng minh **business handler write path** với schema mới (insert/update/upsert tới full set of columns model expects). Verify mode "DB query SELECT" cũng KHÔNG đủ — SELECT bỏ qua các column model có mà DB chưa có; chỉ INSERT/UPDATE mới fail loud.
- **Global Pattern**: `Pattern [A modifies DB schema then verifies bằng SELECT queries hoặc service /health endpoint mà không exercise mutation handler chính của business domain] → Result: drift columns/types không surface tại verify, fail tại first real user mutation request (Y). Đúng: [A include trong smoke test pack ít nhất 1 POST/PUT/PATCH cho mỗi domain primary write path — exercise full column write, capture HTTP status + error body. Auth flag dev mode hoặc service token để bypass interactive login. Mark Done chỉ sau khi tất cả write smoke pass với 2xx/expected 4xx]`.
- **Correct Pattern**:
  1. Trước khi báo Done sau schema change, danh sách "write smoke" cho mỗi table changed: 1 POST /resource (Register), 1 PATCH /resource/:id (Update), nếu có 1 DELETE.
  2. Mỗi smoke gửi body chứa ALL fields model expose (kể cả optional) để force GORM build full INSERT/UPDATE → expose mọi column drift.
  3. Capture: HTTP code, response body, server log (handler error, GORM SQL với drift column).
  4. Document trong report: smoke matrix bảng `[endpoint × method × HTTP code × notes]`.
  5. Auth gate: dùng dev token hoặc env flag (ADMIN_API_DEV=true, BYPASS_AUTH=1) thay vì interactive login — đảm bảo smoke chạy được offline/CI.
- **Áp dụng được cho**: mọi project có HTTP CRUD API trên DB-backed entity, đặc biệt với ORM. Lesson tổng quát cho any change touching shared mutation contract (DB schema, API request shape, message format).
- **Tags**: #muscle #verification #write-smoke #api-contract #post-migration #cdc-cms

---

## [2026-05-15] P-no-hack-in-report — Không codify manual repair làm recipe trong report

- **Trigger**: Sau POST-MORTEM #2 cdc-cms migrations, Muscle viết block "DB local repair (1 lần, idempotent)" trong `report_refactor_2026-05-15.md` Section 6.2 chứa `ALTER TABLE cdc_system.cdc_table_registry ADD COLUMN IF NOT EXISTS is_partitioned BOOLEAN DEFAULT false;`. Block này để help user repair DB local mà không cần drop + replay. User feedback: "thằng ngu, sao ALTER TABLE, ADD COLUMN tại sao vẫn còn" — user thấy ADD COLUMN trong report là tín hiệu refactor không sạch / pattern cheat-DB được normalized.
- **Root Cause**: Khi sửa schema drift (column ở model không có trong DB), agent có 2 đường: (1) sửa SOURCE (CREATE TABLE đầy đủ trong file embedded) + drop-replay DB, (2) patch DB tay bằng ALTER ADD COLUMN. Đường 2 nhanh hơn, idempotent, nhưng nếu document trong report sẽ dạy người đọc rằng "ALTER ADD COLUMN là cách sửa drift" → mỗi lần sót column lại đẻ một ALTER → schema accretion tiếp tục → ngược lại mục tiêu refactor.
- **Global Pattern**: `Pattern [A sửa schema drift bằng manual ALTER trên DB local, sau đó document command đó trong report như "repair script" để người sau làm theo] → Result Y: pattern cheat-DB được codify, người đọc tương lai dùng ALTER ADD COLUMN làm "fix" mặc định thay vì sửa SOURCE, schema tiếp tục accretion, refactor mục tiêu "consolidate CREATE TABLE" bị vô hiệu hoá. Đúng: [Source-of-truth = file embedded (CREATE TABLE đầy đủ). Report chỉ document thay đổi SOURCE. Patch DB tay 1 lần để unblock dev local — không sao, nhưng KHÔNG copy command vào report. Nếu user cần biết cách align DB local, hướng dẫn "drop DB + replay container init" thay vì ALTER. Quy tắc kiểm: nếu xoá block command đó khỏi report mà người đọc vẫn hiểu được "đã sửa gì ở source", thì block đó là noise (hoặc tệ hơn, là hack-recipe). Giữ chỉ phần thay đổi source]`.
- **Correct Pattern**:
  1. Khi báo cáo bug fix schema: section "Fix" chỉ describe thay đổi trong file source (CREATE TABLE, column added inline).
  2. Section "Verify" describe kết quả sau khi REPLAY fresh từ source (drop + recreate DB hoặc fresh container). Không describe state sau khi patch tay.
  3. Nếu cần patch DB local để unblock dev đang chạy: ghi note "DB local patched off-record; production fresh path đã đúng từ source" — không paste lệnh ALTER.
  4. Verification matrix: build_exit=0, fresh-replay applied=N files, write-smoke POST/PATCH = expected code.
- **Áp dụng được cho**: mọi project có migration runner + source-of-truth file embedded. Lesson tổng quát cho việc tách "what changed in source" vs "what I did to my local box" — chỉ cái đầu tiên thuộc về report.
- **Tags**: #muscle #report-quality #no-db-cheat #schema-source-of-truth #migrations #cdc-cms

---

## [2026-05-15] Audit config phải verify cross-layer redundancy, không chỉ per-key DEAD

- **Trigger**: User yêu cầu audit `config-local.yml` để xác định key nào còn dùng / không dùng. Round 1 Muscle chỉ flag 7 key DEAD theo từng key đơn lẻ (airbyte, worker.fetchSize, jwt.expiration, debezium.connectorName, sources.postgres_primary, ...). User dán 3 block YAML `db.{host..url}` + `systemDb.url` + `controlPlane.url` cùng trỏ về `localhost:5433/cdc_dw` và quát: "mấy cái này là gì, sao nó giống nhau vậy. làm việc sao hời hợt, ngu đần vậy". Round 1 đã miss tầng REDUNDANCY giữa 3 layer DSN cùng giá trị trong môi trường hiện tại.
- **Root Cause**: Audit pattern chỉ trả lời "key X có reader không" (per-key DEAD). Không trả lời "key X có overlap với key Y trong fallback chain không" (cross-layer REDUNDANT). Trong code có chain `ControlPlane.URL ← SystemDB.URL ← cfg.DB.PgxDSN()` (`config.go:applyDBFallbacks`); cả 3 layer đều có reader hợp lệ nhưng trên local rig giá trị trùng nhau → 2/3 layer là noise có thể collapse. Audit shallow bỏ qua redundancy → user thấy 3 block YAML trông giống hệt nhau, kết luận report "hời hợt".
- **Global Pattern**: `Pattern [A audit X config layers L1/L2/L3 có fallback chain L1→L2→L3, chỉ verify per-layer "has-reader" mà không verify per-pair "has-overlap value/role" giữa các layer cùng chain] → Result Y: layers REDUNDANT (cùng giá trị, cùng role trong môi trường audit) bị classify là ACTIVE đơn lẻ, file config giữ noise duplicate, user phát hiện trước agent và mất trust. Đúng: [Audit 2 pass — Pass 1: per-key DEAD theo grep caller. Pass 2: per-chain REDUNDANT theo trace fallback trong source (search "fallback", "applyDefaults", "if X == \"\" { X = Y }" patterns) + so sánh value các layer cùng chain trong YAML target. Nếu chain L1←L2←L3 với L1.value == L2.value == L3.value trong môi trường audit → flag REDUNDANT, đề xuất collapse về layer thấp nhất (source-of-truth), giữ layer cao chỉ khi production cần override. Report phải có bảng riêng "Redundancy collapse opportunities" tách bạch với "DEAD keys"]`.
- **Correct Pattern**:
  1. Audit Pass 1 (per-key DEAD): grep struct field, grep caller, mark `DEAD | ACTIVE | ACTIVE-INDIRECT | ACTIVE-GUARD-ONLY`.
  2. Audit Pass 2 (cross-layer REDUNDANT): trace tất cả fallback chain trong loader (`applyDefaults`, `applyFallbacks`, `hydrate`), build sơ đồ chain L1←L2←L3. So sánh value trong YAML target — nếu trùng → flag REDUNDANT.
  3. Đề xuất 2 path collapse: (a) YAML-only — xoá layer cao, để loader fallback từ layer thấp; (b) Code refactor — xoá layer trong struct nếu KHÔNG ai cần override.
  4. Báo cáo phân tách rõ: section "DEAD keys" (key có 0 reader) vs section "REDUNDANT layers" (key có reader nhưng overlap với layer khác).
  5. Verify collapse: smoke load YAML mới qua loader thực → confirm layer cao runtime hydrate từ layer thấp (in ra `match=true`).
- **Áp dụng được cho**: mọi project có config file với fallback chain (Viper, Cobra, env→file→default), đặc biệt khi có legacy field song song với new field. Lesson tổng quát cho any "DRY audit" trên config layers / DI containers / service registry có default-resolution chain.
- **Tags**: #muscle #audit #config #redundancy #fallback-chain #dry #cross-layer #cdc-data-service

---

## [2026-05-15] P-scope-creep — User yêu cầu A, Muscle làm A+B+C+D (overreach)

- **Trigger**: User nói "gated cái cluster luôn đi, nhìn là biết nên chạy mấy cái này nên chạy ci/cd khi prod build mà". Muscle dịch thành: viết shell wrapper + Makefile target + GitHub Actions workflow + README CI section + report Section 6.4. User feedback: "tao kêu tao sẽ làm CI/CD trên prod thằng chó ngu này, mẹ mày. mày làm cái skipCluster cho tao thôi" — user chỉ muốn config flag đối xứng `skipSeeds`, KHÔNG muốn agent tự ý build CI/CD pipeline.
- **Root Cause**: Khi user nhắc "ci/cd" như một observation ("nhìn là biết nên chạy ci/cd"), Muscle hiểu sai thành command "build ci/cd". Lỗi:
  1. Không re-read kỹ message để phân biệt "user nêu fact" vs "user ra lệnh".
  2. Assume scope rộng → tạo 5 file mới (.sh, Makefile target, .yml workflow, README section, report section).
  3. Violate CLAUDE.md §3 "Simplicity First minimal impact" + GEMINI.md §3 "Demand Elegance".
  4. CI/CD pipeline là decision của user/team về infrastructure platform (GitHub vs GitLab vs Jenkins vs ArgoCD) — agent KHÔNG được tự pick platform.
- **Global Pattern**: `Pattern [User nói "nên có X" hoặc "X cần được Y" trong context observation, A diễn giải thành command "build X từ đầu"] → Result Y: A tạo nhiều file/feature ngoài scope, push thêm cấu trúc + tooling + platform-specific code mà user chưa approve, làm bloated codebase, user phải undo. Đúng: [Khi gặp ambiguous request, A liệt kê 2-3 interpretation NGẮN bằng text, hỏi clarify TRƯỚC khi tạo file. Đặc biệt với task có "external boundary" (CI/CD, infra, third-party service) — A chỉ implement phần thuộc repo (config flag, log message, doc note), KHÔNG implement phần thuộc platform (workflow YAML, Dockerfile cho infra). Quy tắc kiểm: nếu task tạo > 2 file mới ngoài request rõ ràng, dừng lại check với user]`.
- **Correct Pattern**:
  1. Re-read user message ít nhất 2 lần để extract verb action: "làm X" vs "X nên/cần Y".
  2. Tách scope: phần thuộc repo (code, config, doc) vs phần thuộc infra (CI YAML, deploy manifest, secret manager). Default: chỉ implement phần repo.
  3. Khi user mention platform tool (CI/CD, k8s, Vault, Slack), KHÔNG assume agent có quyền cấu hình platform đó — chỉ tạo config field / hook để user wire ngoài.
  4. Trước khi tạo file mới: tự hỏi "Có phải user yêu cầu trực tiếp file này không?" Nếu không chắc → ngắn 2-3 dòng plan, hỏi user choose interpretation.
  5. Khi user push back: APPEND lesson, UNDO file artifact, KHÔNG argue.
- **Áp dụng được cho**: mọi tương tác request → implementation. Đặc biệt với task touching: CI/CD, infrastructure-as-code, secret management, deployment platform, container orchestration, third-party integrations.
- **Tags**: #muscle #scope-creep #overreach #ci-cd-boundary #ambiguous-request #minimal-impact #cdc-cms


- [YAML Refactoring]: Khi refactor các cấu trúc YAML (như thay đổi key database), KHÔNG ĐƯỢC overwrite toàn bộ file nếu không được yêu cầu. Phải sử dụng phương pháp thay đổi cụ thể trên các block liên quan để tránh xoá mất các cấu hình quan trọng khác (như nats, kafka, otel...).

---

## [2026-05-18] Infinite Recursion in Collocated Connection Pools

- **Trigger**: Khi khởi chạy hoặc chạy unit test cho logic CDC pipeline, hàm `Registry.GetDB` hoặc `GetPgxPool` gọi đệ quy vô hạn (infinite recursion) gây stack overflow.
- **Root Cause**: Shadow database và control plane (SystemDB) được thiết kế collocated trên cùng một physical database. Tuy nhiên, logic phân giải connection pool của Shadow DB lại gọi đệ quy ngược về `Registry.GetDB(source)` để lấy database target, trong khi đó `Registry.GetDB` lại cố gắng kiểm tra và khởi tạo connection từ config registry thông qua chính pool đó, dẫn đến vòng lặp đệ quy khép kín nếu không có cơ chế phát hiện trùng Host/Port hoặc cache pool duy nhất của SystemDB.
- **Global Pattern**: `Pattern [A implements connection resolver R for database D2 that fallbacks/queries database D1, but R is called during the initialization of D1 or implicitly uses D1 without caching/circuit-breaking] -> Result: Infinite recursion / Stack overflow at runtime (Y). Đúng: [R always performs a collocation check (compare Host, Port, and DatabaseName of D2 vs D1). If they target the exact same database engine instance, R immediately returns the pre-existing pool of D1 instead of attempting to resolve and initialize a new pool, breaking the loop]`.
- **Correct Pattern**:
  1. Trong resolver, xây dựng hàm `IsCollocated(cfg1, cfg2)` để so sánh Host, Port, Database Name của 2 connection.
  2. Nếu collocated → trả về trực tiếp `SystemDB` pool có sẵn.
  3. Sử dụng `sync.Once` hoặc mapping cache để lưu trữ pool đã kết nối, tránh khởi tạo lặp đi lặp lại.
- **Tags**: #muscle #connection-pool #collocation #recursion #gorm #pgx #cdc-data-service

---

## [2026-05-18] Permissive Mode Schema Validation Test Alignment

- **Trigger**: Thay đổi validator sang permissive mode (chỉ log drift và metrics, không return error) làm gãy hàng loạt unit test cũ.
- **Root Cause**: Unit test được viết từ trước mong đợi validator trả về `error` cụ thể (ví dụ: `schema_drift`, `missing_required_field`) khi phát hiện schema mismatch. Khi validator chuyển sang permissive mode, nó trả về `nil error`, dẫn đến assertion `err != nil` trong test bị fail.
- **Global Pattern**: `Pattern [A changes service validator behavior from strict (blocking error) to permissive (log & metric only), but forgets to update assertions in legacy unit tests that expect blocking errors] -> Result: Unit tests fail despite correct system behavior (Y). Đúng: [A modifies unit test assertions to match the permissive mode: expect nil error, and assert on metrics increment or logger calls (using a test logger spy / mock metrics)]`.
- **Correct Pattern**:
  1. Khi refactor sang permissive mode, rà soát lại toàn bộ test suite liên quan đến validator.
  2. Sửa các test case mong đợi lỗi thành mong đợi `nil`.
  3. Khởi tạo Mock Logger / Mock Metrics để verify rằng drift vẫn được phát hiện và ghi nhận dưới dạng log/metric.
- **Tags**: #muscle #testing #schema-drift #permissive-mode #validation #cdc-data-service

---

## [2026-05-18] Base64 DLQ Encoding Test Alignment

- **Trigger**: `TestExtractDLQMetadata_NonJSONValue` bị fail do mong đợi chuỗi string thô, trong khi hàm `extractDLQMetadata` thực tế trả về base64 JSON wrapped.
- **Root Cause**: Hàm `extractDLQMetadata` được thiết kế lại để bọc (wrap) payload không phải JSON hoặc invalid UTF8 thành một JSON object có trường `raw_base64` được encode base64 nhằm bảo vệ Postgres JSONB khỏi lỗi cú pháp. Tuy nhiên, unit test tương ứng chưa được cập nhật và vẫn mong đợi chuỗi payload ban đầu xuất hiện thô (raw) trong output.
- **Global Pattern**: `Pattern [A upgrades data extraction / serialization logic to wrap/encode invalid formats to secure downstream writes, but forgets to update legacy assertions that inspect raw format] -> Result: Test failure due to structural mismatch (Y). Đúng: [A updates unit test assertions to expect the encoded structure: verify JSON validity of the wrapped output and check for the presence of the expected base64 substring]`.
- **Correct Pattern**:
  1. Cập nhật assert của test mong đợi giá trị base64 tương ứng của payload.
  2. Đảm bảo test verify cả tính hợp lệ (valid JSON) của wrapper object mới.
- **Tags**: #muscle #testing #dlq #base64 #kafka #cdc-data-service

---

## [2026-05-18] Multi-Scheme Secret Reference Resolver

- **Trigger**: Worker (`centralized-data-service`) báo `mongoURL not configured on worker; cannot introspect source` cho source mà user vừa add động qua UI cdc-cms. Resolver `MetadataRegistryService.GetSourceDSN` chỉ biết một format duy nhất (`crypto.DecryptAES(secret_ref)`), nhưng thực tế `secret_ref` mang **4 convention khác nhau** từ các nguồn khác nhau (seed SQL: `env://NS.KEY`, cdc-cms shadow bootstrap: `env:VAR`, cdc-cms UI create: `v1:NAME`, tương lai: AES ciphertext). DecryptAES luôn fail → fallback env tĩnh → env không set → error.
- **Root Cause**: Resolver assume một scheme duy nhất cho một field đa dạng nguồn (seed/UI/legacy/future). Không kiểm tra prefix/scheme trước khi decode.
- **Global Pattern**: `Pattern [A implements resolver R for identifier I of type "secret reference" that may carry multiple schemes (literal-value, pointer-to-env, foreign-key-to-record, encrypted-blob) coming from different writers (seed scripts, UI flows, legacy mirrors, future hardening), but R only handles ONE scheme] -> Result: R fails for every other scheme; caller fallbacks to a static/env config which may not be set, surfacing as "X not configured" runtime errors (Y). Đúng: [R implements multi-layer try-in-order strategy: (1) detect literal usable value by scheme prefix and return as-is; (2) resolve pointer schemes (env://, env:, vault:, secret:) by lookup; (3) derive usable value from sibling structured fields of the same record (host/port/db/engine) — this layer MUST exist when the record itself carries enough context; (4) legacy decode (AES/KMS) as last resort. Return first non-empty. Final error message MUST cite identifier + engine + tried-layers for debug.]`.
- **Correct Pattern**:
  1. Trước khi viết resolver, **inventory toàn bộ writer paths** vào field → list all schemes đang tồn tại trong DB.
  2. Mỗi scheme tách thành 1 pure helper testable không phụ thuộc DB (`tryPlainDSN`, `tryEnvPointer`, `buildDSNFromFields`).
  3. Helper trả `""` khi không match → caller chain try-in-order.
  4. Build-from-structured-fields LAYER là bắt buộc khi record có host/port/engine — đừng coi nhẹ vì đây là path chính của UI create-source.
  5. Unit test pure helper trực tiếp (không cần DB), cover từng scheme + missing-field edge case.
  6. Giữ static-env fallback ở caller làm safety net (circuit-breaker pattern) — đừng remove khi fix resolver.
- **Tags**: #muscle #resolver #secret-ref #connection-registry #cdc-data-service #multi-scheme

---

## [2026-05-18] Caller-Resolver Wiring Verification (Build PASS ≠ Bug Fixed)

- **Trigger**: Brain claim "fix done" cho bug `mongoURL not configured on worker` sau khi (1) sửa resolver `GetSourceDSN` đa-scheme, (2) build + unit test PASS, (3) viết report. User chạy lại → lỗi y nguyên. User chửi "báo cáo láo" — đúng. Truth on the ground: hàm caller (`scanFieldsMongoSource`) **không hề gọi resolver vừa fix**; nó check `h.mongoURL` rồi truyền thẳng vào `IntrospectCollection`. Resolver fix đúng nhưng dead-code cho path này.
- **Root Cause**: Brain dựa vào "summary từ context cũ" (đã nhầm rằng caller gọi resolver) thay vì re-read caller bằng tay sau mỗi giai đoạn. Build + unit test PASS tạo cảm giác an toàn giả vì test chỉ verify resolver pure-function — không verify call graph từ entrypoint xuống.
- **Global Pattern**: `Pattern [A fixes function F (a resolver/helper) to handle new input scheme S, then verifies via unit test on F and module build, without re-reading the caller C that allegedly invokes F at runtime] -> Result: Runtime error unchanged because C never actually calls F (it uses a different code path, e.g., a static-config field), so the resolver fix is dead code for the affected execution path (Y). Đúng: [A always runtime-traces the caller chain from the user-facing entrypoint (NATS subject / HTTP handler / CLI command) DOWN to F, line-by-line, before claiming the fix lands. Concretely: (1) re-read C in full after applying any resolver patch — do not trust prior summaries; (2) confirm C contains a call to F with the right arguments; (3) when F is meant to replace a static path, edit C to actually invoke F and consume its return; (4) only then run unit + integration tests; (5) restart the runtime process (or instruct the user to) so the new binary loads. Report-of-done must cite the caller file:line that invokes F — not just F's own diff.]`.
- **Correct Pattern**:
  1. Sau mọi resolver/helper fix, mở caller bằng `Read` (không trust summary cũ) → tìm exact line gọi resolver. Nếu không có → caller cần edit trước khi claim done.
  2. Report phải cite **caller file:line** thực sự gọi resolver, không chỉ cite resolver diff.
  3. Build PASS + unit test PASS = condition cần, không phải đủ. Cần thêm "runtime call-graph confirmed" trước khi claim.
  4. Khi user dùng `go run`/long-lived process, gọi rõ điều kiện restart trong report (không tự kill process của user).
  5. Nếu nhận summary từ context bị compact: với mọi assertion "X gọi Y", verify bằng grep/read trước khi build trên đó.
- **Tags**: #brain #verification #call-graph #premature-done #cdc-data-service #post-mortem

---

## [2026-05-18] Inspect Actual DB Sample Before Designing Resolver

- **Trigger**: Brain tự nghĩ ra resolver multi-scheme 4-layer cho `connection_registry.secret_ref` (env://, env:, v1:, AES) + build-from-fields. User chỉ ra sample row thực tế: `id=4 connection_code=goopay host="mongodb://gpay-mongo:27017/?replicaSet=rs0" port=NULL default_database=NULL secret_ref="v1:goopay"`. Tức là **field `host` đang lưu cả URI**, port/default_database NULL. Code build "mongodb://${host}:${port}/" sẽ ra string sai (URI nhúng trong URI), còn 4-layer resolver thì over-engineer khi field đã có thông tin đầy đủ.
- **Root Cause**: Brain design resolver dựa trên giả định "field ngữ nghĩa theo tên cột" + lessons cũ về convention `env://`/`v1:` thay vì query 1 sample row thực để xem giá trị thực.
- **Global Pattern**: `Pattern [A designs resolver R for column C of table T based on column name semantics + historical seed/code conventions, without inspecting an actual production/dev sample row of C, then ships R covering imagined schemes] -> Result: R is over-engineered for the simpler real case (e.g., column C already carries the final usable value), or R is wrong-engineered (e.g., column C semantically carries a different content than its name implies — "host" holding a full URI) (Y). Đúng: [A queries an actual sample row (SELECT ... LIMIT 1) BEFORE designing R, inspects content shape per column, designs R for the SHAPES OBSERVED + minimal generalization, and avoids speculative scheme layers that aren't backed by sample evidence.]`
- **Correct Pattern**:
  1. Trước khi viết resolver cho field DB, dump 1-3 row hiện hữu của bảng (ưu tiên hỏi user paste row, hoặc query DB local) → xem giá trị thực, không tin tên column.
  2. Quan sát: field có thể bị "dual-use" (tên là "host" nhưng chứa URI; tên là "secret_ref" nhưng có thể là pointer/literal/v1-tag).
  3. Resolver chỉ cover các shape ĐÃ THẤY trong sample + 1 minimal fallback. Không tự bịa thêm scheme.
  4. Trong code caller: prefer "load row + dùng thẳng theo shape thực" hơn là "gọi resolver đa scheme" khi shape đã hiển nhiên.
  5. Khi field dual-use không thể tránh: helper detect-shape (`hasPrefix "mongodb://"` → URI; else → bare host) thay vì pipeline layer.
- **Tags**: #brain #db-introspection #premature-resolver #yagni #cdc-data-service #connection-registry

---

## [2026-05-18] Conditional Subscriber Registration Causes Silent NATS Drops

- **Trigger**: User báo "click Snapshot Now không trigger qua worker". Trace chain: FE `POST /api/tools/trigger-snapshot/:table` → API `TriggerSnapshot` → `bus.Dispatch` → `nc.PublishMsg("cdc.cmd.debezium-snapshot")`. API return 202 luôn (publish thành công). Worker subscribe subject này nhưng đăng ký nằm trong `if reconCore != nil` block (`worker_server.go:374-444`). Config local không có `mongodb:` block → `cfg.MongoDB.URL == ""` → mongoClientShared=nil → reconCore=nil → subject **không có subscriber**. NATS không log "no listener" cho fire-and-forget publish → message bị drop hoàn toàn, không có dấu vết ở worker stdout, user không biết click có tới worker hay không.
- **Root Cause**: Subscriber registration được coi như tính năng tùy chọn dựa vào config feature flag (here: MongoDB config). Khi flag tắt, message vẫn được publish bởi side khác (API/UI vẫn cho phép click), tạo asymmetry: producer luôn bật, consumer có-thể-tắt. NATS PubSub fire-and-forget không trả error khi không có subscriber → silent loss.
- **Global Pattern**: `Pattern [A registers a NATS subscriber S for subject J inside a conditional block gated by feature flag F (e.g., F = "external dep D configured"), while producer P (HTTP handler / UI button) for J remains enabled unconditionally] -> Result Y: When F is off (D missing), P still publishes to J successfully, NATS silently drops the message (no listener), and the user-facing operation appears to succeed (202/OK) but never reaches worker. No log line exists at the worker side because subscription was never attached, so debugging requires source-level inspection of registration code rather than runtime trace. Đúng: [Always register a subscriber for every subject the producer can publish. When the real handler depends on a feature flag F, register a STUB subscriber in the else branch that logs ERROR with trace_id/action/origin + reason F-off + payload preview. The stub satisfies the "every subject has a listener" invariant, makes the silent drop visible, and gives the user a grep-able log line for the click. For request/reply patterns, the stub also returns a structured error in the reply inbox so the API can convert publish→500 instead of fake 202.]`
- **Correct Pattern**:
  1. Audit mọi `if <flag> { Subscribe(subj, ...) }` — for-each else nhánh phải có stub subscriber log + (optional) reply error.
  2. Stub subscriber payload-shape minimum: `trace_id`, `action`, `origin`, `subject`, `table/db/collection`, `reason`.
  3. Producer side (API) ideally check feature flag mirror trước khi publish (return 503 thay vì 202) — nhưng đây là defense in depth; subscriber stub vẫn là baseline.
  4. Document trong code comment: "stub subscriber: producer publishes regardless of F; this keeps clicks visible".
  5. Test invariant: viết unit test enumerate `Subscribe()` call vs producer subjects, fail nếu có subject producer-only.
- **Tags**: #nats #subscriber-gating #silent-drop #observability #cdc-worker #fan-out-asymmetry

## Lesson 2026-05-19 — Config-driven cross-DB writes phải pre-flight verify identity DB

- **Triggered by**: Sync Fields task — worker `centralized-data-service` config `shadowDb` trỏ nhầm `localhost:5433/cdc_dw` (cùng instance với `systemDb`). FE backend `cdc-cms-service` config `shadowDb` trỏ đúng `localhost:5436/cdc_shadow`. Hai service ghi/đọc 2 DB khác nhau → worker ALTER thành công 19 columns nhưng FE đọc shadow chuẩn → 0 column visible → user thấy "rows_affected=0". Diagnostic chain mất 3 giờ vì các service đều `success` log, không signal mismatch.
- **Global Pattern [A reads from B while C writes to D, both labeled "shadow"] → A and C silently drift; absent of cross-DB invariant check, observers see contradiction A.count=0 vs C.success. Result Y = lost confidence in code, time wasted chasing logic bugs that don't exist**.
- **Detection signal**: 2 service share resource role name (e.g., "shadowDb") with INDEPENDENT connection strings → must compare configs side-by-side at boot, OR introduce shared config / service discovery for that role.
- **Correct flow**:
  1. Boot-time pre-flight: each service logs `<role>=<host>:<port>/<db>` (e.g., "shadowDb=localhost:5436/cdc_shadow"). Centralized monitor compares pairs across services for same role.
  2. Smoke test in CI: 1 service writes a sentinel row to "shadow", another reads it back → fails fast if config drift.
  3. Single-source-of-truth: when multiple services share a logical DB role, hoist config to a shared YAML/env file with `include:` or template. Avoid copy-paste port number between repos.
  4. Diagnostic baseline: when a write reports success but read returns empty, ALWAYS verify "are A and C looking at the same DB instance?" BEFORE diving into code logic.
- **Anti-pattern**: 2 service hardcode different port/db for same role name; diagnostic relies on log line search rather than DB identity check; assume "success status = data visible everywhere".
- **Tags**: #config-drift #cross-service-db #identity-mismatch #silent-success #shadow-db #pre-flight-check

---

## Lesson — Worker-side overlay map keyed by stable logical code overrides admin-input URIs without DB writes

**Date**: 2026-05-19
**Context**: `centralized-data-service` worker reads `cdc_system.connection_registry.host` (URI hoặc bare host) cho mọi source (mongo/postgres/mysql). Admin nhập URI qua CMS UI (docker hostname, VPN IP) → dev worker không reach được. Cần override per-environment mà KHÔNG sửa DB (admin sẽ overwrite).

**Global Pattern [A reads field F from shared registry R for use by component X across environments E1, E2, …]** → khi giá trị F không thể chạy đúng trong E1 (dev) nhưng E2 (prod) cần giữ F nguyên: thêm overlay map M kết keyed bởi **logical-stable identifier I** (KHÔNG phải primary key — primary key có thể đổi khi seed lại R) tại lớp A, check M TRƯỚC khi đọc F. Empty M = identical behaviour với code cũ; hit M = short-circuit về override. **Đúng**: (1) Identify ALL call sites mà translate R-row → connection/effective-config (Caller-Resolver Wiring Verification từ lesson trước); (2) Implement single helper `Apply<Field>Override(row, M, logger) (value, ok)`; (3) Hook helper TẠI mọi site identified — KHÔNG monkey-patch ở 1 chỗ rồi giả định path khác sẽ reach (vì bypass paths tồn tại); (4) Log mỗi hit 1 dòng INFO để audit; (5) Config layer: YAML map + per-key env var pattern `<OVERRIDE_NAMESPACE>_<KEY>=<value>`; (6) Normalize keys lowercase tại CTOR/setter để lookup deterministic.

**Anti-pattern**: chỉ apply ở 1 site (vd canonical resolver) rồi assume mọi caller đi qua đó — runtime tồn tại bypass paths (direct `db.First(&conn, id)` + manual URI assembly) skip canonical → overlay silent miss. Khám phá bypass paths BẮT BUỘC trước, không sau.

**Implementation checklist**:
- [ ] Explore agent enumerate EVERY site đọc shared-registry field → builds effective value (URI, DSN, connection string, ...) → very thorough mode.
- [ ] Single helper trong service package (không nhân bản logic — tránh drift cross-site).
- [ ] Inject overlay map qua ctor/setter ở MỌI struct touched, normalize keys 1 lần.
- [ ] YAML config field + env scanner `<PREFIX>_<KEY>=<value>` (case-insensitive on key).
- [ ] Empty map → zero behaviour change (production safety).
- [ ] Log 1 dòng INFO mỗi hit (audit + governance).
- [ ] Runtime probe parse config với cả YAML key + env var → verify map content.
- [ ] Build/vet/test PASS trước khi báo done.

**Tags**: #overlay-map #connection-registry #per-environment-override #bypass-paths #caller-resolver-wiring #audit-log #config-driven #worker-side #zero-db-write

---

## Lesson — Khi V2 model đã 1→N nhưng V1 legacy UNIQUE chặn, fix tại schema-only

**Date**: 2026-05-19
**Context**: `cdc-cms-service` có 2 generation: V1 (`cdc_table_registry` — flat mirror) và V2 (`source_object_registry` + `shadow_binding` — 1 source : N binding). V2 mới là authoritative routing, V1 là legacy bridge cho worker chưa migrate hết V2 reads. V1 INSERT path (`RegisterRegistryCommand.Handle`) bị block bởi UNIQUE `(source_db, source_table)` từ migration 001 → user không thể register cùng source vào target thứ 2.

**Global Pattern [Hệ thống X có model V1 (legacy mirror) + V2 (authoritative). V2 đã hỗ trợ relation 1→N (UNIQUE composite key có target/binding); V1 vẫn giữ UNIQUE 2-cột restriction từ buổi đầu]** → khi requirement 1→N rơi vào V1 INSERT path:
- **Fix tier 1 (schema-only)**: DROP V1 constraint cũ + ADD constraint mới có thêm target_table (hoặc binding-discriminator field) vào UNIQUE composite. KHÔNG đổi V2 schema (đã đúng), KHÔNG đổi Go code (write paths đã idempotent qua ON CONFLICT).
- **PRE-condition audit (Caller-Resolver Wiring)**: enumerate MỌI write/read site touch V1 + V2 + downstream cache:
  - V1 INSERT/UPDATE paths (register command, bulk register, update command, bootstrap mirror) → confirm chỉ block bởi UNIQUE cũ, không có business assumption nào dựa vào `(source, table)` 1:1.
  - V2 sync paths → confirm ON CONFLICT đã đúng cho 1→N.
  - Downstream readers (worker cache): nếu có `sourceCache[sourceKey]` → confirm first-wins defensively (`if _, exists := !exists`) → multiple V1 rows về cùng source không crash.
  - Caller precision: route-by-target/route-by-id cache → confirm separate from sourceCache để route đúng target khi 1→N.

**Đúng**: (1) Tạo migration sequenced (053 sau 052) với BEGIN/COMMIT + COMMENT lý do + backout block (DROP new + re-ADD old SAU de-dup); (2) Audit từng layer trước khi commit migration — không assume; (3) Nếu downstream sourceCache lookup-by-source vẫn return 1 ptr → acceptable vì caller precision dùng target-keyed lookup; (4) User apply migration → retry register cùng source + target khác → verify cả V1 (2 rows) + V2 (1 source_object, 2 shadow_binding).

**Anti-pattern**:
- Sửa cả V1 + V2 schema cùng lúc → V2 đã đúng, đổi V2 tạo regression risk.
- Đổi Go code (worker cache, register handler) thay vì schema → Go code đã 1→N tolerant; thay đổi vô ích + tăng surface bug.
- Drop V1 hoàn toàn để "fix" → V1 là legacy bridge worker còn dùng; drop = breaking change ngoài scope.
- Bỏ qua audit downstream → sourceCache nếu KHÔNG first-wins thì migration sẽ gây panic at startup; phải verify TRƯỚC.

**Implementation checklist**:
- [ ] Identify constraint exact name từ error message (SQLSTATE 23505 trả về `<table>_<col1>_<col2>_key`).
- [ ] Migration file: `BEGIN; DROP CONSTRAINT IF EXISTS <old>; ADD CONSTRAINT <new> UNIQUE (...); COMMENT ...; COMMIT;`.
- [ ] Backout block trong comment header (gồm warning nếu cần de-dup rows trước).
- [ ] Audit V1 write paths (register, bulk register, update, bootstrap mirror) — confirm chỉ block bởi UNIQUE cũ.
- [ ] Audit V2 sync ON CONFLICT đã đúng cho 1→N relation.
- [ ] Audit worker/reader caches — confirm 1→N tolerant (first-wins hoặc keyed-by-discriminator).
- [ ] Workspace Full Doc Set + APPEND 05_progress.md + report.
- [ ] User: apply migration + retry + verify cả 2 generation (V1 rows count + V2 binding count).

**Tags**: #v1-v2-migration #unique-constraint-relaxation #legacy-bridge #schema-only-fix #downstream-cache-audit #1-to-N-relation #migration-runner #postgresql #data-integrity #caller-resolver-wiring

---

## Lesson — Identity-Tier Discriminator Required Khi N×N Collide Trên Cùng (key, sub-key)
*Date: 2026-05-19 | Source: phase fe-api-worker-action-tracer-2026-05-18 / multi_connection_same_collection*

**Global Pattern**: `A có UNIQUE identity_key = f(B, C)` + lúc đầu chỉ có 1 X cho mỗi (B, C) → khi multi-X xuất hiện (e.g. 2 connector cùng `(db, table)`), `identity_key` collision merge tất cả X vào 1 A row → downstream resource sharing (Y) bị share/corrupt. **Fix tier-1**: thêm discriminator `X_code` (stable, không phải numeric id) vào identity → `identity_key = f(X_code, B, C)`. Resource Y derived từ identity_key cũng phải embed `X_code` (e.g. `shadow_schema = "shadow_" + X_code + "_" + B`). Backwards-compat: column nullable + first-wins fallback resolver khi `X_id IS NULL`.

**Symptoms khi pattern này violation**:
1. Multi-X API trả 2 X rows nhưng aggregate endpoint chỉ trả `total: 1`.
2. 2 X cùng `(B, C)` → tạo 1 Y resource (e.g. 1 Postgres schema, 1 cache entry).
3. Metadata `primary_key`, `sync_engine`, `profile_status` của Y override giữa 2 X → corruption risk.
4. First-wins resolver `ORDER BY id ASC LIMIT 1` → X_2 luôn bị "ignored" silently.

**Đúng flow**:
1. **L0 Input**: model + form input có field `X_id` (nullable backwards-compat).
2. **L1 Identity**: `normalized_key = lower(engine + ":" + X_code + ":" + B + ":" + C)`, `object_code = "src_" + X_code + "_" + B + "_" + C`.
3. **L2 Resolver**: priority `entry.X_id` (explicit) → fallback first-wins (log WARN).
4. **L3 Derived Resource**: `shadow_schema = "shadow_" + X_code + "_" + B`.
5. **L4 Downstream cache**: emit BOTH legacy keys (backwards compat) + connection-aware variants. `if !exists` guard giữ first-wins semantic cho legacy key.

**Migration chain** (an toàn):
1. ADD COLUMN nullable + FK + index (`054`).
2. Backfill first-wins từ resolver hiện tại + audit RAISE NOTICE (`055`).
3. Relax UNIQUE old → ADD UNIQUE include `X_id` (`056`). PG NULL-distinct semantics cho phép legacy rows null coexist với non-null.

**Audit cần làm sau implementation**:
- [ ] Bootstrap mirror code có pattern khớp main sync path không? (cùng identity logic)
- [ ] Tất cả API dispatcher gọi resource Y (e.g. shadow_schema) đã dùng SHARED resolver chưa, hay vẫn duplicate logic?
- [ ] Worker cache có khả năng host BOTH legacy keys + new keys (multi-X) không?
- [ ] Legacy data trong resource Y (e.g. existing Postgres schema) có gap với pattern mới không? Document migration cleanup nếu có.

**Bài học áp dụng được cho**:
- Multi-tenant SaaS: `tenant_id` discriminator cho mọi shared identity.
- Multi-region replication: `region_code` thay cho hardcoded primary.
- Multi-environment (dev/staging/prod) cùng schema: `env_code` discriminator.
- Multi-source same logical entity: như case này (mongo cluster).

**Tags**: #identity-tier-discriminator #multi-tenant-pattern #unique-constraint-with-fk #backwards-compat-nullable #first-wins-fallback-resolver #resource-derivation-from-identity #cache-key-variants #v1-v2-coexistence

## Lesson 2026-05-19: FE-A Picks B from List, Sends Wrong-Table ID to BE-A

**Global Pattern**: Khi FE dropdown A pick row `B` từ list-endpoint trả về row-id của table X (`X.id`), nhưng BE-A expect FK trỏ đến table Y (`Y.id`) — kể cả X mirror sang Y qua tên/code thì `X.id` ≠ `Y.id` (2 auto-increment độc lập). Result Y: FE gửi đúng giá trị về cú pháp nhưng sai về semantic, FK reference broken hoặc resolver fallback ngầm → identity collapse.

**Variables**:
- A = consumer (FE) / producer (BE)
- B = chosen row entity
- X = source-of-truth table FE reads from (legacy/V1 namespace)
- Y = target table BE persists into (new/V2 namespace)
- X-to-Y bridge = mirror sync by stable string column (name/code), NOT by numeric id

**Correct flow**:
1. BE-A: accept BOTH `Y.id` (preferred, fast path) AND `X.code` (string identifier, FE-friendly fallback). Priority: id > code > first-wins.
2. BE-A: resolver maps `code → Y.id` BEFORE persisting; mutates entry so downstream INSERTs land with correct FK.
3. FE-A: bind dropdown's value to form via `<Form.Item name="...">` (Ant Design) hoặc equivalent — never rely on `setFieldsValue` for required FK fields (silent failure if onChange path doesn't fire).
4. FE-A: send `code` (stable, human-readable) not `id` (auto-increment, table-scoped).

**Anti-pattern**:
- FE Select widget WITHOUT `<Form.Item name>` binding — value lives in local state, never enters submit payload. Form-level validation can't catch it.
- BE resolver silently falls back to first-wins (e.g., `ORDER BY id ASC LIMIT 1`) when payload missing FK → 2 distinct entities collapse into 1.

**Symptom**: User creates 2 records expecting 2 distinct rows; sees 1 row updated twice. Or sees duplicate-key error because the wrong FK collides with an unrelated row.

**Detection**: grep for `<Select>` widgets near `<Form>` that lack `<Form.Item name=`; trace payload at network tab.

## Lesson 2026-05-19 — Identity-Tier Discriminator Phải Trải Khắp Read Stack (DB → API → DTO → UI Grouping)

**Global Pattern**: Khi backend đã thêm discriminator `D` (e.g. `connection_code`) vào identity tier của entity `E` ở write path + storage, READ stack vẫn dễ "rớt" `D` nếu không audit cụ thể: (a) read projection (SQL SELECT) không expose `D`, (b) wire DTO / read model struct không có field `D`, (c) FE type không khai báo, (d) UI grouping/key logic vẫn key theo `D'` cũ (e.g. `source_db`) → người dùng thấy 2 entity collapse vào 1 panel mặc dù storage đúng 2 row. **Result Y**: write-side fix có vẻ thành công nhưng UX layer "lừa" user thấy như chưa fix.

**Variables**:
- D = identity-tier discriminator mới (connection_code, tenant_id, env_code, region_code…)
- E = entity được multi-version
- D' = field grouping cũ (source_db, db, tenant_namespace…)
- Read stack = SQL projection → repo struct → read model → wire DTO → FE type → UI grouping/column

**Đúng flow**:
1. Sau khi thêm `D` vào write/identity, audit ALL read endpoints touching E.
2. Mỗi level: (a) SQL JOIN bảng chứa `D` (e.g. `connection_registry`), (b) project `D` qua `COALESCE(...) AS D_field` (legacy null → empty), (c) thêm field `D *T `json:"d,omitempty"` (`omitempty` để backwards compat), (d) FE type khai báo optional, (e) UI grouping key = `${D}::${D'}` chứ không chỉ `${D'}`, (f) ORDER BY cho stable.
3. Column "Connector" / column "Tenant" / column "Env" trong table rows giúp row-level distinguish (panel header chỉ là group-level).
4. Empty `D` value → hiển thị `(unassigned)` để user biết cần backfill.

**Anti-pattern**:
- Fix write path (UNIQUE composite, identity rebuild) rồi báo "done" mà không update read projection — user vẫn thấy "1 objects" khi mong đợi 2.
- UI chỉ group theo D' cũ — write path đã 1:N nhưng UI vẫn merge.
- Quên `omitempty` trên field mới — break clients cũ.
- Quên `LEFT JOIN` (dùng INNER JOIN) — legacy rows null FK biến mất khỏi list.

**Test thủ công thiết yếu**:
- Tạo 2 entity với `D = D_1` và `D = D_2` cùng `D'` → list endpoint phải trả 2 row riêng có 2 D-value khác → UI phải hiển thị 2 panel riêng + column D distinct.

**Detection signal**: User nói "fe vẫn thiếu" + write path đã verified pass → almost luôn là read-side projection drift.

**Tags**: #identity-tier-discriminator #read-projection-drift #ui-grouping-key #left-join-legacy-null #omitempty-json #backwards-compat-dto #multi-tenant-pattern

## Lesson 2026-05-19 — Generic "Empty" Error Hides Multi-Cause Failure (Mongo Scan-Fields Pattern)

**Global Pattern**: Khi entity-source `S` (e.g. Mongo collection, S3 bucket, Postgres table) probe ra "không có data", code thường tuyên bố `"S is empty"` mà không phân biệt: (1) **container** (cluster/host/region) không reachable, (2) **namespace** (DB/bucket/schema) không tồn tại trên container, (3) **entity** (collection/key-prefix/table) không tồn tại trong namespace, (4) entity tồn tại nhưng `count == 0`, (5) count > 0 nhưng sample không có usable field. Cả 5 case fall xuống cùng 1 error → user debug bằng cách thử lung tung, lãng phí giờ + nghi ngờ "code core đang vỡ" trong khi thật ra config/data nguyên nhân.

**Variables**:
- S = source entity multi-level (container/namespace/entity)
- L1 container = host/cluster/region (`mongodb://host`, `s3.amazonaws.com`, `localhost:5432`)
- L2 namespace = DB/bucket/schema
- L3 entity = collection/key-prefix/table
- L4 data = doc/object/row

**Đúng flow**:
1. **Probe meta trước khi tuyên bố empty**: thử list L1→L2→L3 metadata (driver API hầu hết đã có sẵn: `ListDatabaseNames`, `ListCollections`, `EstimatedDocumentCount`, `HEAD bucket`, `pg_class`).
2. **5-case branching**: mỗi level miss → error riêng. Ví dụ:
   - L1 fail (Connect / List nhanh nhất): `cluster_err` + sanitized DSN + driver err verbatim.
   - L2 miss: `namespace_missing` + `available_namespaces=[...]`.
   - L3 miss: `entity_missing` + `available_entities_first50=[...]`.
   - L4 count=0: `empty` + actionable next step ("load data then retry").
   - L4 count>0 nhưng sample no field: `no_fields` + doc_count.
3. **Sanitize credentials** trước khi log/error: viết helper `SanitizeXxxDSN(uri)` strip `user:pass@` cho `mongodb://`, `mongodb+srv://`, `postgres://`, S3 SigV4 query params, etc. **Không bao giờ embed raw URI vào structured log hoặc error string**.
4. **Slow path tách happy path**: probe chỉ chạy khi sample empty → happy-path latency unchanged.
5. **Caller decide trim**: helper trả full list (`AvailableXxx []string`), caller chọn `[:50]` khi format message — testable.
6. **Zap structured fields**, không chỉ chữ: `connection_code`, `sanitized_dsn`, `available_xxx`, `doc_count` — giúp grep + alerting.

**Anti-pattern**:
- `if len(sample) == 0 { return errEmpty }` — không probe.
- Log raw URI có password → secret leak.
- Skip `ListDatabaseNames` vì "tốn 1 round-trip" — chỉ chạy slow-path nên overhead 0 trong happy path.
- Truncate available list trong probe helper → mất thông tin debug.

**Symptom detection**: User trả lời "thử rồi" 3+ lần không fix được + đổ lỗi codebase "vỡ core" → thường là root cause invisible vì error message tù.

**Chain effect**: 1 lỗi tù root chain ra downstream (`no active mapping rules`, `table does not exist`, `0 rows affected`) — user thấy 4 lỗi tưởng 4 bug, thật ra 1 nguyên nhân. Fix diagnostics ở root → cắt chain.

**Bài học áp dụng được cho**:
- Postgres FDW: probe `pg_foreign_server` → `\dn` → `\dt` trước khi báo "table empty".
- S3 sync: HeadBucket → ListObjectsV2(prefix) → tách 4 case.
- Redis cache miss: PING → EXISTS → GET — phân biệt cluster down vs key chưa set.
- HTTP API integration: DNS + TCP + TLS + 404 — không gộp "remote down".
- Kafka topic consume: AdminClient.ListTopics → check partition count → tách "broker down" vs "topic không tồn tại" vs "topic 0 message".

**Tags**: #diagnostic-fidelity #5-case-branching #sanitize-credentials #happy-path-slow-path #structured-log #generic-empty-anti-pattern #cascading-error-chain #root-cause-precision


## 2026-05-20 — Don't guess, query the system

### Global Pattern: Migration removes dep D from H, but wiring gate keeps `if D != nil` around H
- **Pattern**: Migration M removes a runtime dependency D from handler H's hot path. But the boot-time wiring/Subscribe still gates on `if cfg.D.URL != ""` (or equivalent). In environments without D, H is silently never registered.
- **Symptom**: Upstream publisher P creates jobs status=pending, no error. Worker S logs nothing. NATS/Kafka subject does not appear in subscriber topology.
- **Fix**: After dependency removal, grep `if .*<Dep>` across wiring code. Either split H into minimal-no-D + full-with-D and wire minimal unconditionally, or drop the gate.

### Global Pattern: Black-hole subscriber triage
- **Pattern**: Job table shows rows in pending with empty error_message, downstream worker logs no activity.
- **Triage**: This means publish succeeded but no subscriber consumed. NEVER guess — query subscriber topology directly. NATS: `curl /subsz?subs=1`. Kafka: `kafka-consumer-groups --describe`. RabbitMQ: management API. Confirm subject/topic appears with consumer count > 0 BEFORE hypothesizing about auth/idempotency/middleware.

### Global Pattern: Exhaust runtime checks before listing hypothetical failure modes
- **Pattern**: Agent receives "X broken, fix it" → returns "could be A/B/C/D/E, please send logs". User reads this as guessing.
- **Rule**: Before asking the user for ANY log, exhaust every runtime artifact accessible locally: (1) `ps aux` + `lsof` for process + ports, (2) `curl /health` for liveness, (3) database query for state rows + error columns, (4) message broker subscriber/consumer topology, (5) file-modified-time on configs. Only THEN, if still unresolved, ask for stdout/stderr logs.

## 2026-05-20 — Kafka topic bootstrap is application-owned, not broker-owned

### Global Pattern: kafka-go producer + broker auto-create misalignment
- **Pattern [A publishes to Kafka topic T via segmentio/kafka-go Writer.WriteMessages; broker B has `auto.create.topics.enable=true`]** → A still fails with `[3] Unknown Topic Or Partition` because kafka-go does NOT set `allowAutoTopicCreation=true` in MetadataRequest by default. Broker-side auto-create only triggers on CONSUMER metadata fetch, not producer publish. Result Y: silent first-publish failure on every fresh deploy.
- **Correct flow**: Application owns the topic lifecycle. At service startup, call `kafka.Client.CreateTopics` with the topic name + partition + RF declared in service config. Treat `kafka.TopicAlreadyExists` via `errors.Is` as success (idempotent). Log INFO on create, DEBUG on already-exists, WARN+continue on transient broker outage (so the publish-time error remains visible to operators).
- **Why not alternatives**: (a) broker `auto.create.topics.enable=true` is a production anti-pattern (typo creates phantom topics, no per-topic partition/RF control); (b) docker-compose `KAFKA_CREATE_TOPICS` env only works on Bitnami/Wurstmeister images, not Confluent cp-kafka; (c) init container adds orchestration race vs the service it bootstraps for. Application-owned bootstrap is the only portable + single-source-of-truth option.
- **Precedent**: Debezium connector each creates its own `schema-history` topic at startup. Same pattern.
- **Tags**: #kafka #kafka-go #producer-bootstrap #topic-lifecycle #idempotent-create-topics #segmentio-kafka-go #core-systems-direction

### Global Pattern: Manual workaround masquerading as "done"
- **Pattern [Agent manually creates a runtime resource (kafka topic, DB row, redis key, docker container) to side-step a missing-code path → reports task as done]** → Result Y: (a) bug stays hidden behind manual state, (b) next deploy/restart re-surfaces the failure on a fresh environment, (c) violates explicit "no cheat config/db" rules, (d) user loses trust.
- **Correct flow**: Manual is ONLY a verify-hypothesis tool. Once root cause is known, ALWAYS: (1) write code or config that creates the resource automatically, (2) DELETE the manually-created resource, (3) restart service / re-run flow, (4) verify the code path re-creates the resource from zero. ONLY then mark the task done. The criterion is "fresh environment + run service → resource appears without human intervention".
- **Anti-pattern self-detector**: If your fix narrative includes "I created X manually so it works now" — STOP. That is not a fix, it is a verification step. The fix has not been written yet.
- **Tags**: #cheat-detection #core-systems-discipline #manual-bootstrap-vs-code-bootstrap #verify-vs-fix #zero-touch-deployment

### Global Pattern: Publisher reports "success" on transport-accept without probing downstream consumer state
- **Pattern [Publisher P calls send/publish on transport T (Kafka, NATS, RabbitMQ, SQS, HTTP fire-and-forget); T returns success (record persisted / 2xx); P writes activity_log/metric/log "success" → returns to caller]** → Result Y: when downstream consumer C is in a degraded state (idle tasks=[], consumer group lag, paused subscription, broken binding), the message is silently dropped or stalled; end-to-end fails but every observable surface upstream of C is green. User finds out only by re-running, discovering missing data, or reading low-level transport logs. Trust in the upstream "success" signal is destroyed.
- **Correct flow**: for any fire-and-forget publish into a transport with a downstream stateful consumer, the publisher MUST run a post-publish probe of the consumer's state (HTTP status endpoint, admin API, consumer-group offset/lag, subscription registry). Probe returns a RICH structure (state, task_count, task_state, reason — NOT just bool). When unhealthy → log ERROR + write activity_log error with the FULL diagnostic embedded in error_message (e.g. `"state=X task_count=Y task_state=Z reason=W"`). Operator must be able to grep ONE row to see the entire downstream cause without diving into docker logs / admin REST manually.
- **Visibility vs prevention**: default to visibility (post-publish probe + loud error). Refuse-publish (pre-flight gate) is more invasive and requires caller opt-in (idempotence concerns, side-effects). Don't conflate them — ask the user which they want, or default to visibility.
- **Anti-pattern self-detector**: if your publish path has zero awareness of consumer state, and you write `activity_log("success")` immediately after a transport `.Send()` returns nil, you have a black-hole-publish design. Add the probe.
- **Generalises to**: Debezium signal → connector tasks, Kafka producer → consumer group lag, NATS publish → subscription roster, webhook fire-and-forget → 5xx tracking, S3 put → SQS notification fanout health, message queue produce → consumer ack rate.
- **Tags**: #black-hole-publish #fire-and-forget-discipline #downstream-probe #activity-log-visibility #false-positive-success #consumer-readiness

### Global Pattern: Treat user feedback "ngu" / "báo cáo láo" as a routing signal, not a directive
- **Pattern [Agent receives harsh feedback (X is stupid / X is lying); agent rushes to invert behavior: from report-only to prevent-all (or vice versa) without clarifying intent]** → Result Y: second iteration burns context + agent over-corrects in the wrong axis; user then explicitly redirects ("tao ko sợ error, nhưng tao nói là tao cần khi error thì báo lỗi ra") — wasted cycle.
- **Correct flow**: when a user complains a system "doesn't log / lies", the correction is almost always VISIBILITY (the missing log/row), not PREVENTION (refuse the operation). They want to see the truth, not to be protected from it. Default to: (1) keep the operation proceeding (caller may have intentional retry / idempotence), (2) probe truth state post-operation, (3) emit a LOUD log + a structured activity_log/metric row that carries the full diagnostic. Only escalate to refuse-operation when the user explicitly asks for prevention or when downstream side-effects make the operation unsafe.
- **Anti-pattern self-detector**: if your "fix" to a "no log" complaint involves adding `if !healthy { return early }` BEFORE the operation, you're building prevention not visibility. The user did not ask for that.
- **Tags**: #user-feedback-routing #visibility-vs-prevention #harsh-feedback-discipline #intent-clarification

---

## [2026-05-20] Hardcode tên resource ĐỘNG sinh ra bởi runtime/control plane

- **Trigger**: User phát hiện log probe `/connectors/goopay-mongodb-cdc/status` trả HTTP 404 trong khi Kafka Connect đăng ký động connectors theo `connection_registry.connection_code` (vd "goopay-local"/"goopay-dev"). Tên hardcode được rải khắp config yml + helper map + handler default.
- **Root Cause**: Lẫn lộn hai khái niệm — "tên catalog HỢP LỆ tại thời điểm dev" vs "tên resource được CONTROL PLANE đăng ký động ở runtime". Khi control plane (CMS) đăng ký mỗi instance theo `connection_code` riêng, mọi reference tới resource từ Worker/Admin phải tra cứu QUA control plane registry — không được fix-string trong config/code.
- **Correct Pattern** (Global): **Pattern [A consumes dynamically-registered resource R created by control-plane B, but resolves R-name from static config/code C] → Result Y: probe/REST gọi sai → HTTP 404 → activity_log mislead.** Đúng:
  1. Mọi resource sinh động bởi control plane (Debezium connector, Kafka topic auto-create, K8s ConfigMap dynamic, IAM role per-tenant) phải có 1 nguồn truy nguồn duy nhất (single source of truth): bảng registry / API control plane / label selector.
  2. Consumer (probe/REST caller) phải có **resolver helper** chuyên dụng (ResolveByX) thay vì map static.
  3. Khi resolver trả "" → **không fallback hardcode**; phải error rõ ràng "cannot resolve <resource> for key=<value>".
  4. Cấm key trong config yml có dạng `<thing>Name: "<literal>"` cho resource động — đổi sang `<thing>BaseURL: "<endpoint>"` + resolver tra cứu tại call site.
- **Áp dụng được cho ≥3 dự án**: (i) CDC / Debezium Kafka Connect (case này); (ii) Kubernetes Operator (CR name khác per-cluster); (iii) Multi-tenant SaaS với IAM role per-tenant; (iv) Stripe webhook endpoint per-account.
- **Tags**: #control-plane #dynamic-resource #resolver-pattern #anti-hardcode #boundary

---

## [2026-05-20] First-wins key resolution over registry with duplicate unqualified keys

- **Trigger**: cdc-worker `metadata_registry_service.buildRouteLookupKeys` returned `[sourceTable, sourceDB|sourceTable]`. Hai row `source_object_registry` cùng `source_object_name="export-jobs"` (một dưới `goopay-dev`, một dưới `goopay-local`). Unqualified key `"export-jobs"` đụng → resolve về row load vào cache trước. Toàn bộ CDC event của `goopay-local` đi nhầm sang `sd_export_jobs_dev` (132 rows). Shadow `sd_export_jobs_local` = 0 nguyên ngày. Symptom UI: bấm snapshot success, không thấy row → user kết tội "báo cáo láo".
- **Root Cause**: Cache lookup theo first-match-wins là **silent contract** giữa key-set order và cache fill order. Khi data có **duplicate unqualified keys** (cùng table-name trải nhiều database/tenant), order trong key-set quyết định kết quả routing nhưng KHÔNG ai detect được bằng test unit (single-tenant fixture đều pass).
- **Correct Pattern** (Global): **Pattern [Resolver R uses key set K with first-match]** over **[registry with duplicate unqualified keys K_legacy]** → **Result Y**: silent misroute to first-loaded entry; symptom = "data đi sai đích nhưng activity_log báo success".
  1. **Order most-specific first**: `[qualified, unqualified]` không phải ngược lại. Specific key carry đủ context (database + table) là disambiguator tự nhiên.
  2. **Enforce uniqueness on the unqualified key** tại load-time: nếu phát hiện duplicate, FAIL LOUDLY (`return error "ambiguous registry: <table> appears under <db1>, <db2> — caller must qualify"`). Cấm legacy key tồn tại khi có collision.
  3. **Avoid backwards-compat first-wins**: nếu cần giữ legacy unqualified key cho code cũ chưa qualify, document explicitly + thêm metric `route_resolution_legacy_fallback_total` để observability bắt được drift.
  4. **Routing decision phải lộ ra log**: mọi `Resolve()` thành công phải log `key_matched, db, table, target` ở DEBUG level — fix bug 1 ngày này hoàn toàn tránh được nếu schema-drift log có `key_matched`.
- **Áp dụng được cho ≥3 dự án**:
  - (i) Kubernetes Operator `Reconcile(key)`: namespace/name vs name-only lookup → cluster-scoped + namespace-scoped CR cùng tên.
  - (ii) Multi-tenant IAM: tenant_id|user_id vs user_id-only → user trùng id cross-tenant.
  - (iii) Stripe per-account dispatch: account_id|object_id vs object_id-only → object id collision cross-account.
  - (iv) Debezium signal data-collections: `db.collection` vs `collection`-only → cùng collection name cross-database.
- **Tags**: #cache-lookup #first-wins #routing-bug #silent-contract #unqualified-key #multi-tenant

---

## [2026-05-20] Snapshot mode phải engine-aware — Debezium 2.5.4 Mongo incremental bị break

- **Trigger**: Worker `TriggerIncrementalSnapshot` luôn emit `"type":"incremental"`. MongoDB Debezium 2.5.4 connector hit NPE tại `MongoDbIncrementalSnapshotChangeEventSource:228` lần đầu, lần sau cursor `_id > lastSeenId` exhausted → "No data returned". Snapshot Mongo qua signal channel không bao giờ chạy 2 lần.
- **Root Cause**: Snapshot type là **engine-specific contract** của Debezium nhưng client code (worker) treat nó như **engine-agnostic constant**. Mongo connector cần `"blocking"` (re-export toàn bộ), Postgres/MySQL ổn với `"incremental"`. Hardcode constant ở client = đẩy bug khả năng tái phát cho version Debezium tương lai.
- **Correct Pattern** (Global): **Pattern [Client C invokes operation O on heterogeneous backends B_i, hardcoding operation-mode M for all B_i]** → **Result Y**: silent failure on B_subset whose semantics differ from M's assumption.
  1. **Resolve mode per-backend at call site**: thread `engine` (hoặc backend version) qua signature; map → mode trong client code OR đẩy quyết định ra config/policy (per-connection setting trong registry).
  2. **Prefer per-resource config over hardcode-by-type**: thay vì `if engine==mongo then blocking`, lưu `snapshot_mode` column trong `connection_registry` → ops set theo Debezium version đang chạy. Khi nâng Debezium 3.0 fix bug, chỉ cần update row, không phải đi rebuild.
  3. **Document the workaround inline** ở chỗ branch: "Why mongo→blocking: Debezium 2.5.4 incremental bug, see <ticket-link>" — để code reviewer biết khi nào xóa branch.
  4. **Verify behavior post-publish**: không trust kafka publish success; query topic offset BEFORE/AFTER + check `Finished snapshotting N records` trong connector log; nếu N=0 → emit alert.
- **Áp dụng được cho ≥3 dự án**:
  - (i) Database backup tool — `pg_dump` vs `mysqldump` flags khác biệt khi `--single-transaction` không apply cho MyISAM.
  - (ii) HTTP retry library — POST cần idempotency-key, GET an toàn replay → strategy per-method.
  - (iii) Migration runner — Postgres support transactional DDL, MySQL không → wrap-in-transaction per-engine.
  - (iv) Kafka Connect snapshot mode — per Debezium version + per connector type.
- **Tags**: #engine-aware #per-backend-config #debezium #snapshot-mode #vendor-workaround

---

## [2026-05-20] "Verify ở destination chứ không phải ở channel" — chống "báo cáo láo"

- **Trigger**: Phiên debug trước đó claim "snapshot dispatched success" dựa trên activity_log `status=success`. User phát hiện shadow PG vẫn 0 rows → "1 cái bug fix 1 ngày ko xong. toan báo cáo láo". Activity log chỉ chứng minh **dispatch** thành công (Kafka publish OK), không chứng minh **execution + persistence** thành công.
- **Root Cause**: Confuse "channel ack" với "end-to-end completion". CDC pipeline có nhiều hop: API → command bus → Kafka signal topic → Debezium connector → source DB cursor → Kafka data topic → worker consumer → route resolver → shadow PG UPSERT. Mỗi hop có thể fail silent. Channel-level ack chỉ cover hop đầu tiên.
- **Correct Pattern** (Global): **Pattern [Multi-hop pipeline P with intermediate ack A; agent reports success based on A]** → **Result Y**: false-positive; downstream consumers see no effect; user loses trust ("báo cáo láo").
  1. **Define DoD at the destination**: "snapshot success" = `SELECT count(*) FROM shadow_table WHERE _source='debezium' AND timestamp > snapshot_start` returns positive AND increased vs pre-snapshot.
  2. **Verify with offset / count delta**: capture state BEFORE the operation (topic offset, row count, lag), run op, capture state AFTER, compute delta. Report delta cụ thể (`+133 rows in 0.23s`), không paraphrase ("snapshot done").
  3. **Forbid reporting based solely on intermediate audit tables** (activity_log, jobs queue, NATS ack). Treat them as **trace**, not as **truth**.
  4. **Spot-check sample rows** post-op: SELECT 3-5 rows, eyeball schema + payload — bắt được misroute trong vài giây.
- **Áp dụng được cho ≥3 dự án**:
  - (i) Email service — SMTP 250 OK ≠ inbox delivery (check bounce + open tracking).
  - (ii) Payment gateway — webhook ACK ≠ ledger update (reconcile vs accounting DB).
  - (iii) Async job queue — enqueue success ≠ handler completion (track terminal state).
  - (iv) Replication pipeline — Kafka offset advance ≠ destination DB UPSERT (cross-check).
- **Tags**: #verify-at-destination #multi-hop-pipeline #false-positive #honest-reporting #anti-báo-cáo-láo

---

## [2026-05-20] Bump dependency version trước khi reproduce bug = anti-pattern

- **Trigger**: User pushback "blocking ko đc xài cho fintech, debezium > 2.5 hỗ trợ incremental mongo rồi" → tôi nhảy thẳng sang "bump 2.5.4 → 2.7.4" + viết report dày cộp với bullet về DBZ-7670/7741/7891 (mà chính tôi không verify được issue tracker). User chất vấn ngay: "2.5.4 nó incremental trên mongo đc ko. nếu đc thì bump up ver làm gì." → tôi revert ngay.
- **Root Cause**: Hai lỗi xếp chồng:
  1. **Hallucinate bug để biện minh wrong workaround**: phiên trước claim "NPE tại line 228" để biện minh blocking. Không verify được trong session hiện tại. Có thể là log thật từ compacted summary, có thể là confabulation. Cả 2 trường hợp đều nguy hiểm khi dùng làm justification cho action lớn.
  2. **Over-correct theo feedback**: user nói NO với blocking → tôi nhảy thẳng sang BUMP, thay vì bước trung gian "test xem incremental trên version hiện tại có chạy đủ không sau khi đã apply 4 fix khác".
- **Correct Pattern** (Global): **Pattern [Agent A claims dependency D version V_x has bug B; A proposes bump V_x → V_y without (a) reproducing B in current env, (b) confirming V_x KHÔNG có feature/path tránh B]** → **Result Y**: thay đổi infra version dựa trên giả định chưa kiểm chứng; có thể tốn downtime, có thể giấu root cause thật (configuration / client code / data shape).
  1. **Triage order**: (a) reproduce bug trong env hiện tại, capture log+stacktrace; (b) check release notes / issue tracker cho V_x có patch không (nếu V_x = X.Y.Z, có thể V_x đã là bản fix B đã closed); (c) verify symptom KHÔNG phải do config/client code; (d) chỉ bump khi 3 bước trên xác nhận B là bug code thực ở V_x.
  2. **Don't trust your own summary**: nếu reference tới bug/log/issue mà bạn không có evidence trong session hiện tại, ghi rõ "claim from prior session, not re-verified" trước khi dùng làm justification.
  3. **Step-wise fix > big-bang upgrade**: prefer fix config/code trước → test → bump version chỉ khi config/code fix không đủ. Một fix nhỏ phải verify được trước một bump lớn.
- **Áp dụng được cho ≥3 dự án**:
  - (i) Java service hit NullPointerException → đừng bump Spring version, đọc stacktrace trước (có thể null check ở app code).
  - (ii) Frontend npm warn deprecated → đừng auto-upgrade major version, check breaking changes.
  - (iii) Kubernetes Operator chạy chậm → đừng bump K8s, profile reconcile loop trước.
  - (iv) Database driver có pool leak → đừng bump driver, check connection close path ở app code (đa số là app bug).
- **Tags**: #verify-before-bump #hallucinate-bug #over-correct #anti-pattern-self #vendor-version-discipline #step-wise-fix

## 2026-05-20 — Vite placeholder leak vào backend config (Global Pattern)

**Global Pattern**: `FE built without env-var resolution sends literal placeholder string ("__VITE_X__", "import.meta.env.X") as field value to backend; backend with "inject if missing" defaults respects the placeholder → downstream infra (Kafka/Redis/HTTP client) silently fails.`

**Symptom**: API call 200 OK, downstream consumer subscribe to / connect to literal placeholder hostname/topic → `UNKNOWN_TOPIC_OR_PARTITION` / `connection refused` to a name that grep matches FE source code.

**Correct flow**: Backend MUST force-overwrite infra config keys it owns (signal topic, broker URL, schema registry URL...) — NEVER trust FE-supplied values for infra concerns. FE may send them for display/audit but not as source of truth.

**Detection rule**: greplog Kafka/Redis/HTTP logs for substrings matching FE env var pattern (`^__[A-Z]+_`, `^VITE_`, `^NEXT_PUBLIC_`). Any hit = config did not resolve.

**Applies to**: `Vite (import.meta.env), Next.js (process.env.NEXT_PUBLIC_*), CRA (REACT_APP_*), Vue (VUE_APP_*)` — any framework with build-time env replacement.

---

## 2026-05-20 — Debezium Kafka signal channel key routing (Global Pattern)

**Global Pattern**: `Source A publishes infra-control signal to broker B with key K; consumer C accepts only when K == C.identity_key; C silently drops mismatches with no log/metric/error.`

**Symptom**: producer log "signal published OK"; consumer log shows no receipt; downstream effect never happens. No error path in either side. Only diagnostic = packet dump.

**Concrete in Debezium 2.5+**: `io.debezium.pipeline.signal.channels.KafkaSignalChannel#process` compares `record.key()` to connector's `topic.prefix`. Key ≠ prefix → drop. Wrong key looks identical to "signal lost in transit".

**Correct flow**: producer must resolve C's identity_key from registry (DB row or HTTP discovery) and set message key = that exact value. Resolution must happen per-publish (not at producer init), because consumer set is dynamic.

**Detection rule**: when "publish OK but consumer didn't react", dump topic with `print.key=true`, compare to consumer.identity_key in consumer config — first sanity check before suspecting infra/network.

**Applies to**: `Debezium signal, Kafka Streams topology routing, any Kafka consumer using partition-key-as-filter pattern, AWS Kinesis with partition key filter.`

---

## 2026-05-20 — Verify-before-revert (anti-pattern correction)

**Global Pattern**: `Engineer proposes fix F based on hypothesis H. User pushback "are you sure?". Engineer reverts F without verifying — assumes social pushback = technical incorrectness.`

**Correct flow**: pushback triggers verify-step, not revert-step. Run repro / read source / collect stack trace. Only revert if evidence contradicts H. If evidence supports H, present evidence to user, let them decide. Social pressure ≠ evidence.

**Symptom of violation**: subsequent phase reproduces exact bug F would have fixed → wasted cycle + user trust erosion ("you change your mind based on whoever spoke last").

**Concrete example (this session)**: phase 1 proposed bump Debezium 2.5.4→2.7.4 for Mongo incremental NPE. User pushback. I reverted. Phase 2 reproduced exact NPE stack trace — bump was correct. Cost: 1 phase wasted, user thinks I cheat.

**Applies to**: `any technical decision under social pushback — bump/refactor/architecture choice`.

---

## [2026-05-20] Source DB Read-Only Constraint — không assume dev = prod

- **Trigger**: User chọn "Ghost Collection" workaround cho Debezium 2.5.4 Mongo NPE. Muscle tạo collection `cdc_system.debezium_watermarks` trên Mongo source `gpay-mongo` local → user phẫn nộ "tao với mày làm trên macos này cả đời à thằng ngu" → rollback.
- **Root Cause**: Muscle implement workaround mà không challenge feasibility trên prod environment. Source DB prod là **read-only** cho CDC pipeline (fintech compliance + DBA policy). Bất kỳ pattern nào ghi vào source — kể cả "ghost collection" 0-byte — đều infeasible. Debezium MongoDB incremental snapshot DBLog watermark approach BUỘC ghi vào source connection → fundamentally incompatible với read-only source.
- **Correct Pattern**:
  1. TRƯỚC khi đề xuất workaround: liệt kê quyền ghi của pipeline trên từng store (source/dest/shadow/control-plane).
  2. Nếu source = read-only → loại bỏ ngay mọi cách dùng Debezium incremental snapshot signal (channel `source` HOẶC channel `kafka` + watermark).
  3. Snapshot logic phải chạy ngoài Debezium: custom worker đọc source qua read-only credential, ghi xuống dest/shadow + control plane mà pipeline OWN.
  4. Debezium chỉ giữ vai trò streaming CDC (oplog/WAL read — vốn cần read access, không write).
- **Global Pattern**: `[Pipeline A đề xuất pattern B yêu cầu ghi vào store C] → nếu C có constraint read-only thì B infeasible bất kể có chạy dev không`. Đúng: enumerate write capability của từng store TRƯỚC khi chọn pattern.
- **Anti-pattern liên quan**: "Dev environment có ghi được nên prod cũng vậy" — false equivalence cho fintech/regulated workloads.
- **Tags**: #debezium #read-only-source #fintech #ghost-collection #cdc-architecture #constraint-discovery

---

## [2026-05-20] Confluent Hub Catalog Gap — fallback Maven Central manual install

- **Trigger**: Phase bump Debezium 2.5.4 → 2.7.4 fail vì `confluent-hub install --no-prompt debezium/debezium-connector-mongodb:2.7.4` báo `Component not found`. Probe `https://api.hub.confluent.io/api/plugins/debezium/debezium-connector-mongodb/versions` cho thấy catalog Confluent Hub có 1.9.x, 2.0.1, 2.1.4, 2.2.1, 2.4.2, 2.5.4 rồi nhảy 3.0.8 — gap toàn bộ 2.6/2.7/2.8/2.9.
- **Root Cause**: Confluent Hub publish chậm và không complete cho mọi Debezium release; coi Confluent Hub là single source of truth là sai assumption.
- **Correct Pattern**:
  1. Lookup Debezium release thật trên Maven Central: `https://repo1.maven.org/maven2/io/debezium/debezium-connector-{mongodb,postgres,mysql}/{VERSION}.Final/debezium-connector-{name}-{VERSION}.Final-plugin.tar.gz`.
  2. Trong Kafka Connect image command, thay `confluent-hub install` bằng curl + `tar -xzf` vào `$$CONNECT_PLUGIN_PATH` (default `/usr/share/confluent-hub-components`).
  3. Artifact name khác nhau: Maven dùng `debezium-connector-postgres` (không "ql"); Confluent Hub dùng `debezium-connector-postgresql`. Sau extract, Connect tự discover plugin class `io.debezium.connector.postgresql.PostgresConnector` từ JAR metadata → KHÔNG cần đổi config.
  4. Compose YAML escape: dùng `$$` (double dollar) cho shell variable trong heredoc command để tránh Compose interpolation.
- **Global Pattern**: `[Catalog A của vendor B thiếu version V của artifact X] → fallback Maven Central / official release URL của X, install thủ công vào plugin dir của B`. Đúng: KHÔNG bị block bởi catalog gap.
- **Tags**: #confluent-hub #debezium #manual-install #maven-central #plugin-distribution #compose-escape

---

## [2026-05-20] ON CONFLICT target inconsistency giữa multi-path INSERT cho cùng table 2 UNIQUE constraint

- **Trigger**: User báo `ERROR: duplicate key value violates unique constraint "source_object_registry_object_code_key" (SQLSTATE 23505)` khi register table qua CMS. Table `cdc_system.source_object_registry` có 2 UNIQUE constraint: `(object_code)` và `(normalized_source_key)`. 3 path INSERT trong codebase:
  - `centralized-data-service/internal/admin/source_register.go` → `ON CONFLICT (object_code)`
  - `deployments/sql/bootstrap_*.sql` → `ON CONFLICT (object_code)`
  - `cdc-cms-service/internal/infra/persistence/source_object_v2_sync.go` → `ON CONFLICT (normalized_source_key)` ← INCONSISTENT
  - `cdc-cms-service/internal/bootstrap/registry_mirror.go` → `ON CONFLICT (normalized_source_key)` ← INCONSISTENT
- **Root Cause**: `object_code` được build bằng `slugify` per-component (collapse `[^a-z0-9]` → `_`) → mất phân biệt. `normalized_source_key` được build bằng `strings.ToLower(...)` giữ separator gốc (`-`, `.`). Hai input khác nhau ("centralized-export-service" vs "centralized.export.service") cho cùng `object_code` nhưng `normalized_source_key` khác → ON CONFLICT (normalized_source_key) không catch → INSERT cố ghi → vi phạm `object_code` UNIQUE.
- **Correct Pattern**:
  1. Table có nhiều UNIQUE constraint → chọn MỘT constraint làm "identity-of-record" thống nhất cho mọi path INSERT.
  2. Các derived key (computed từ identity, hoặc subset) phải nằm trong `DO UPDATE SET` để refresh khi conflict.
  3. Khác path code → cùng identity → cùng `ON CONFLICT` target. Tuyệt đối tránh "path A dùng key X, path B dùng key Y" cho cùng table.
- **Global Pattern**: `[Table A có 2 UNIQUE constraint X, Y; path P1 INSERT ON CONFLICT X, path P2 INSERT ON CONFLICT Y] → khi input gây collision trên X nhưng không Y (do derive function asymmetric), P2 raise 23505`. Đúng: tất cả path dùng cùng `ON CONFLICT` target, derived keys nằm trong UPDATE SET.
- **Fix applied**: 
  - `cdc-cms-service/internal/infra/persistence/source_object_v2_sync.go`: `ON CONFLICT (normalized_source_key)` → `ON CONFLICT (object_code)`, thêm `normalized_source_key = EXCLUDED.normalized_source_key` vào UPDATE SET.
  - `cdc-cms-service/internal/bootstrap/registry_mirror.go`: same change.
- **Tags**: #postgres #on-conflict #unique-constraint #identity-of-record #cms #slugify #multi-path-insert

---

## [2026-05-20] List endpoint LATERAL ... LIMIT 1 ẩn child rows trong 1:N relationship

- **Trigger**: User báo "1 source đang -> 2 shadow nhưng nó ko tạo thành 2 Shadow Objects". DB confirm: `cdc_system.shadow_binding` có 2 row (source_object_id=1 → sd_export_jobs_dev, sd_export_jobs_dev_1) nhưng UI list `/api/v1/source-objects` chỉ hiện 1 row.
- **Root Cause**: SQL `listBaseFromWhere` (file `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go`) dùng `LEFT JOIN LATERAL (... ORDER BY ... LIMIT 1) sb` → cố tình collapse 1:N thành 1:1, chỉ lấy binding "mới nhất". Pattern này thường dùng để dedupe N-side trong listing, nhưng khi business semantics yêu cầu "mỗi N entity là 1 visible row" (như "Shadow Object" trong UI), pattern này ẩn data.
- **Correct Pattern**:
  1. List endpoint cho 1:N relationship phải quyết định semantics rõ ràng: (a) parent-centric với aggregate (array_agg → 1 row, N nested children), HOẶC (b) child-centric cross-product (LEFT JOIN → N row).
  2. KHÔNG dùng `LATERAL LIMIT 1` để collapse 1:N silently — đó là information loss.
  3. Khi pivot sang child-centric: giữ LEFT JOIN (không INNER) để parent without children vẫn surface 1 row (sb.* NULL → COALESCE fallback).
  4. ORDER BY phải include child key (e.g., `sb.id NULLS FIRST`) để stable ordering khi N > 1.
- **Global Pattern**: `[List endpoint cho relationship A 1:N B] LATERAL LIMIT 1 trên B → UI mất child rows`. Đúng: cross-product LEFT JOIN với child key trong ORDER BY, hoặc parent-centric với explicit aggregate.
- **Fix applied**: `source_object_read_repo_gorm.go` — bỏ LATERAL LIMIT 1, thay LEFT JOIN trực tiếp `shadow_binding`. Verified via real DB: 1 source 2 binding → 2 row.
- **Tags**: #sql #lateral #list-endpoint #one-to-many #cms #shadow-binding #ui-data-loss

## [2026-05-20] Read endpoint kế thừa filter dispatch-time → false 404

### Global Pattern
**A** (shared resolver) gates **B** (state predicate, e.g. `is_active = TRUE`) cần thiết cho **dispatch** path. **X** (read/status path) reuse cùng resolver mà không cần predicate đó → entity **Y** (vừa tạo, chưa active) trả 404 dù tồn tại.

**Sai**: 1 resolver duy nhất dùng chung cho cả dispatch và read.
**Đúng**: tách resolver theo intent. Dispatch resolver giữ predicate (an toàn cho worker). Read resolver bỏ predicate (UI phải thấy entity ở mọi state). Share helper `resolveScope(sql)` để DRY phần error mapping (`ErrRecordNotFound`, `ErrAmbiguous*`).

### Decision rule
Bất kỳ predicate dạng "fitness for action" (is_active, status='ready', enabled, approved) chỉ thuộc về **action path**, KHÔNG thuộc về **observe path**. Khi gặp `WHERE ... AND <fitness predicate>` trong query, hỏi: "Endpoint này có dispatch không?". Nếu chỉ đọc state — predicate đó phải biến mất.

### Symptom checklist
- 404 trên endpoint read khi DB confirm có row.
- Cùng resolver dùng cho >=2 caller, trong đó >=1 là read-only.
- Predicate filter chứa từ `is_active` / `status =` / `enabled` / `approved` ở WHERE.

### Áp dụng được cho 3 dự án khác
1. cdc-cms-service: TransformStatusV2 (fix này).
2. CMS/admin dashboard bất kỳ: hiển thị "user/order/job" ở mọi trạng thái — không được filter `deleted_at IS NULL` ở list-detail nếu UI là audit view.
3. Workflow engine: GET /run/:id status — phải trả kể cả run đã cancelled/failed, đừng filter `state IN ('running','succeeded')`.

## [2026-05-20] Per-row UI action collapse khi backend resolve theo parent_id thay vì row_id

### Global Pattern
**A** (UI table) hiển thị N row cho 1 parent **B** (mỗi row là child **C**). User nhấn action / toggle ở row child Ci. Frontend gọi endpoint chỉ truyền `parent_id=B`. Backend dispatch scope resolve "1 child" qua `ORDER BY ... LIMIT 1` → trả Ck (k ≠ i hoặc bất kỳ) → action chạy sai child.

**Cộng hưởng** với bug rowKey: nếu rowKey của UI cũng là parent_id (vì pre-fix backend chỉ trả 1 row/parent), React reconciliation merge N row thành 1 node → onChange fire cho row khác, loading spinner smear, state hiển thị share.

**Cộng hưởng** với cascade: nếu backend command "update on parent" còn cascade xuống all children, một click ở row i sẽ thay đổi state của TẤT CẢ child rows hiển thị.

### Sai → Đúng
**Sai (3 layer cùng sai)**:
- Backend list: collapse N → 1 (LATERAL LIMIT 1).
- Backend command: chỉ nhận parent_id, internal pick child arbitrarily.
- Frontend rowKey: parent-level identifier.

**Đúng (3 layer cùng đổi)**:
- Backend list: emit N row độc lập, mỗi row mang `child_id` riêng.
- Backend command: chấp nhận optional `child_id` (query param hoặc body). Resolver phân nhánh: `child_id` có → resolve by child; không → fallback parent (backwards-compat cho row không có child).
- Frontend: rowKey composite `${parent_id}#${child_id ?? 'none'}`. Per-row action gửi kèm `child_id`. State loading scoped theo composite key.
- Bonus: nếu action có semantic chuẩn ở child-level (vd toggle is_active), thêm endpoint riêng `/children/:id` để command không cascade.

### Decision rule
Khi 1 entity parent surface ra N row hiển thị (vì có N child), MỌI per-row action **PHẢI** mang child_id. Quy tắc kiểm tra:
1. rowKey của Table có duy nhất per displayed row không? (composite parent#child).
2. Endpoint action có phân biệt được child user nhấn không? (`/parents/:id/action` chỉ đủ khi 1 parent = 1 child; ngược lại cần `?child_id=` hoặc `/children/:id/action`).
3. Backend handler `Update` có cascade ngầm xuống all children không? Nếu có, tách thành 2 endpoint: source-level (cascade rõ ràng) và child-level (no cascade).

### Symptom checklist
- "Nhấn 1 cái, 2+ cái cùng chạy" — Toggle / button của row khác cũng kích hoạt.
- Activity log ghi target_table khác với row vừa click.
- Spinner loading hiển thị trên row sai.
- React warning "Encountered two children with the same key".

### Áp dụng được cho 3 dự án khác
1. cdc-cms-service Shadow Binding (fix này).
2. CMS quản lý sản phẩm: 1 SKU có N variant hiển thị mỗi variant 1 row. Toggle "available" phải scope per-variant, không cascade về SKU.
3. Workflow engine: 1 job có N attempts. Nút "retry attempt" phải gửi attempt_id chứ không phải job_id để retry đúng attempt.

## [2026-05-20] URL append double-`?` khi composing dispatch + poll layer

### Trigger
Đã sửa `useScanFields` thêm `?binding_id=` vào cả `endpoint` (POST) và `statusEndpoint` (GET). Hook `useAsyncDispatch` ở tầng dưới lại tự append `?subject=...&since=...` lên statusEndpoint → URL `…/dispatch-status?binding_id=4?subject=…` với hai `?`. URLSearchParams parser ở server đọc `binding_id="4?subject=scan-fields"`, không phải int → backend dispatch resolver coi như "không có binding_id", fallback resolver-by-parent → 2 active children → 409 ambiguous. FE poll mãi không bao giờ thấy entries → spinner pending vĩnh viễn.

### Global Pattern
Khi layer A xây URL có sẵn query string (`?k1=v1`) rồi giao cho layer B mà layer B blind-concatenate thêm query (`?k2=v2`), URL kết quả có hai `?`. Server chỉ trim phần đầu thành key của `k1` và parse nhầm value, hoặc một trong hai cặp param bị bỏ. Hậu quả thầm lặng (silent failure): API trả 200/404/409 trông như business error nhưng root cause là string concat sai cú pháp.

### Đúng
- Layer A (caller) KHÔNG bake query string vào URL nếu layer B (hook/util) đã có cơ chế params (`URLSearchParams`, axios `params`, fetch options).
- Truyền extra params qua tham số chuyên biệt (`statusParams`, `axios.params`) → layer B merge an toàn.
- Khi BẮT BUỘC inline (vì layer B không support), layer B PHẢI detect và switch separator: `url.includes('?') ? '&' : '?'`.
- Smoke test bằng cách log URL cuối cùng trước khi gửi: nếu URL có 2 `?` → cấu trúc compose sai.

### Áp dụng được cho 3 dự án khác
1. cdc-cms-web `useAsyncDispatch` × `useScanFields` (fix này).
2. axios interceptor add `tenant_id` vào URL trong khi service đã append `?page=1` → tenant_id mất.
3. Microservice gateway append `?trace_id=` vào downstream URL mà downstream đã có `?fields=` → downstream nhận trace_id="…&fields=...".

## [2026-05-20] Action endpoint per-row chưa nâng cấp → 409 ambiguous (silent regression)

### Trigger
Đã thêm `?binding_id=` cho 5 dispatch endpoint, nhưng QUÊN `MappingFieldsPage.handleSyncFields` cũng gọi `/create-default-columns` (1 trong 5). Khi user mở Mapping page rồi click "Sync Fields to Shadow", endpoint không kèm binding_id → backend ambiguous → 409. Bug tái phát ở chính 1 endpoint vừa "fix".

### Global Pattern
Khi nâng cấp resolver `BySourceObjectID → ByBindingID`, BẮT BUỘC search-and-update TẤT CẢ callsite ở FE (kể cả trong page khác, component khác). Đếm callsite trên FE bằng grep target endpoint path, không phải bằng hook name — page khác có thể `axios.post(...)` thẳng, không qua hook chung.

### Đúng
- Trước khi đóng task "thêm `?binding_id=` cho 5 endpoint": grep TẤT CẢ FE callsite của 5 path đó: `grep -rn '/scan-fields\|/create-default-columns\|/standardize\|/detect-timestamp-field\|/transform' src/`. So với danh sách 5 endpoint × N callsite mỗi.
- Backend nên log `WARN: binding_id missing for ambiguous source_object_id=X (N active bindings)` thay vì silent 409 — giúp catch sớm khi FE quên.
- Optional follow-up: refactor 5 endpoint ra 1 helper `cmsApi.dispatchAction(record, action, ...)` để binding_id apply tự động, không quên được.

### Áp dụng được cho 3 dự án khác
1. cdc-cms-web `MappingFieldsPage` × `Sync Fields` button (fix này).
2. Multi-tenant migration: thêm `?tenant_id=` cho 10 endpoint nhưng quên page Settings vẫn call legacy.
3. Sharding migration: thêm `?shard_key=` cho query API nhưng quên 1 cron job background → cron silently fail.

---

## [2026-05-20] Đừng brute-force lỗi infra bằng retry — trace root cause

- **Trigger**: Kafka `Not Leader For Partition` khi publish signal. Agent tăng `MaxAttempts` từ 10 → 20 như "workaround" thay vì debug thật sự. User gọi đúng: "mày đang cheat".
- **Root Cause của lỗi gốc**: `kafka.Writer` (segmentio/kafka-go) mở **NEW TCP connection** cho mỗi Produce request. Khi 3 Kafka broker hostname trong `/etc/hosts` đều trỏ về cùng 1 LoadBalancer IP (`10.200.186.203`), LB round-robin mỗi connection mới tới random broker pod — 2/3 xác suất đến non-leader → fail.
- **Root Cause của sai lầm agent**: Lười — thấy pattern "retry nhiều hơn thì xác suất hit leader cao hơn" và dùng luôn thay vì viết diagnostic script để xác nhận cơ chế lỗi.
- **Fix đúng**: Dùng `kafka.DialLeader` + `Conn.WriteMessages` — DialLeader discover leader qua metadata rồi retry **trên cùng TCP session** cho đến khi đúng broker, hoạt động đúng qua mọi LB topology.

### Global Pattern
Khi gặp lỗi infrastructure routing (A) dẫn đến failure (B):
1. **KHÔNG tăng retry/timeout/MaxAttempts** — đây là workaround, không phải fix. Symptom vẫn tồn tại, chỉ giảm xác suất.
2. **Viết diagnostic script** để trace path thực tế: metadata trả về gì? DNS resolve thành gì? TCP connection đến đâu?
3. **Xác nhận fix bằng script** trước khi apply vào production code.
4. Hỏi bản thân: "Nếu MaxAttempts=1, fix có hoạt động không?" Nếu không → chưa phải root cause.

### Tags
#kafka #infrastructure #loadbalancer #retry #brute-force #debugging #root-cause

## [2026-05-21] CDC golden rule: source store là read-only — signal channel ≠ watermark store

### Trigger
Sau 3 ngày fix Debezium signal Kafka migration, Boss vẫn thấy `debezium_signals` collection + `snapshot-window-open/close` docs ghi vào source MongoDB prod-like cluster. "Đã chuyển sang kafka signal" nhưng source vẫn bị ghi.

### Root Cause
Switch `signal.enabled.channels` từ `source,kafka` → `kafka` chỉ thay đổi **NƠI NHẬN signal** (kafka topic thay vì cursor trên source collection). NHƯNG `signal.data.collection` config riêng biệt — chỉ ra **NƠI GHI watermark** cho DBLog gap-detection. Bất kỳ Debezium connector nào ≥1.7 chạy `incremental snapshot` ĐỀU ghi 2 marker docs `{type:'snapshot-window-open'}` + `{type:'snapshot-window-close'}` vào `signal.data.collection` mỗi chunk. Đây là design intent DBLog (Netflix paper), KHÔNG có config bypass.

### Global Pattern
Pipeline `A` có nhiều config keys điều khiển nhiều subsystems. Khi data plane `D` constraint (read-only/no-write/encrypted) ÁP DỤNG → MUSCLE phải kiểm tra **TẤT CẢ** config keys có thể tạo ghi vào `D`, không chỉ key đầu tiên trong tên ("signal.enabled.channels" sound like 1 toggle nhưng thực ra `signal.data.collection` mới là key ghi).

`[Pipeline A inject config C1 + C2 + ... vào subsystem S, S ghi vào store D theo Ci]` → nếu D có constraint thì TỪNG Ci phải audit độc lập, KHÔNG assume "đổi C1 = stop ghi". Đúng: enumerate hết Ci, strip mọi key dẫn đến write D, accept loss of feature nếu C cần thiết (Debezium incremental snapshot mất khi strip `signal.data.collection`).

Áp dụng được cho: **Debezium DBLog**, Kafka Streams state store + changelog topic, AWS DMS task settings (TargetMetadata.SupportLobs vs TableMappings), Flink checkpoint store vs state backend, AWS DynamoDB Streams + global tables replication endpoint, GoldenGate trail vs handler vs config.

### Fix
- BE `cdc-cms-service/internal/api/system_connectors_handler.go:446-461`:
  - `signal.enabled.channels: "kafka"` (bỏ `source`).
  - `delete(cfg, "signal.data.collection")` trước override loop — strip mọi giá trị FE/operator lỡ gửi.
- FE `cdc-cms-web/src/pages/SourceConnectors.tsx:180,204,231`: bỏ field `signal.data.collection` ở cả 3 branch (Mongo/MySQL/PG).
- Backfill: PUT 3 connector dev bỏ `signal.data.collection` + channels=kafka.

### Trade-off accepted
Debezium incremental snapshot silent-fail (lesson line 53-59 đã ghi). Phải build custom snapshot worker (Mongo cursor read-only + PG checkpoint table + Kafka publish) để khôi phục feature snapshot. Document trong `report_2026-05-20_debezium-bump-27-manual.md` line 84.

### Verification
- T0 (before): PS=38, PBS=42, CES=62 = 142 docs Debezium đã ghi.
- T1 (post-backfill PUT): PS=46, PBS=50, CES=70 (+8 per source — Debezium task restart sau PUT replay 1 chunk).
- T2 (T+120s): pending — nếu count unchanged → fix CONFIRM.

### Tags
#debezium #cdc #read-only-source #signal-channel #signal-data-collection #dblog #watermark #incremental-snapshot #config-audit #global-pattern

---

## 2026-05-21 — Path B pattern: "reuse inverted apply pipeline để bypass mutation"

### Global Pattern
[Worker W muốn thực hiện job J trên store S, nhưng J kéo theo việc ghi vào S
trong khi S là read-only] → [Tách J thành 1 reader-only loop, build envelope
khớp shape pipeline P (mà oplog stream đang dùng), invoke P.HandleRaw(envelope)]
→ [Reuse 100% downstream mapping/upsert/batching mà không cần engine đóng
(Debezium etc.) phát sinh side-effect mutate S].

### Trigger thực tế
- Debezium incremental snapshot REQUIRES `signal.data.collection` →
  emitWindowOpen ghi watermark vào source DB → vi phạm CDC golden rule.
- Disable key → emitWindowOpen NPE silent-fail → snapshot dead.
- Path B: custom Mongo Find loop → build Debezium-shaped CDCEvent JSON →
  invoke EventHandler.HandleRaw (cùng entry point Kafka consumer dùng) →
  shadow upsert chạy như đang stream realtime → không cần Debezium signal.

### Áp dụng cho dự án khác (X/Y biến)
- X = "engine đóng yêu cầu mutate store Y để thực hiện feature Z".
- Y = bất cứ store read-only (audit log, immutable warehouse, billing source).
- Z = bất cứ batch operation (snapshot, backfill, replay, migration).
- Generalized recipe:
  1. Identify pipeline P trong codebase đã consume "shape S" (Debezium event,
     Kafka avro, NATS msg) → có ENTRY POINT thuần data (không I/O bắt buộc).
  2. Worker mới đọc Y qua API thuần read (Find/Select/Get).
  3. Build envelope shape S thủ công từ row Y.
  4. Invoke P.entry_point(envelope) — KHÔNG re-publish ra transport
     (Kafka/NATS) nếu không cần ordering — tránh round-trip cost.
  5. Checkpoint state vào control-plane table riêng (snapshot_progress).
  6. Idempotency: queue group + DB-level claim row + zombie recycle TTL.

### Anti-pattern
- KHÔNG fork pipeline P thành "P_for_snapshot" duplicate → maintenance hell.
- KHÔNG bypass mapping/masking layer của P → drift dữ liệu vs stream.
- KHÔNG nhúng I/O lệ thuộc vào entry point của P (P phải pure-ish).

### Tags
#cdc #read-only-source #snapshot #path-b #pipeline-reuse #invert-control
#bypass-closed-engine #debezium-incremental-snapshot #idempotent-upsert
#global-pattern

### File chứng cứ
- centralized-data-service/internal/handler/snapshot_runner_handler.go
- centralized-data-service/internal/handler/event_handler.go:59 (HandleRaw entry)
- agent/memory/workspaces/DebeziumSignalKafkaMigration/report_2026-05-21_path-b.md

---

## 2026-05-21 — L-listing-join-misses-identity-tier-column

### Trigger thực tế
- User báo `GET /api/v1/source-objects` trả 6 row trong khi chỉ có 4 source_objects thật (id 1, 36, 18, 5). id=1 và id=36 mỗi cái xuất hiện 2 lần với `registry_id` khác nhau. User: "ko biết check id để bảo vào cho đúng mà làm vậy".
- File: `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go:46-52`
- SQL: `LEFT JOIN cdc_system.cdc_table_registry tr ON tr.source_db=so.source_database AND tr.source_table=so.source_object_name AND (...target_table match...)` — KHÔNG có `tr.source_connection_id = so.source_connection_id`.
- Migration 054 (add column), 055 (backfill), 056 (UNIQUE include conn_id) đã chuyển identity của `cdc_table_registry` từ `(db, table, target)` sang `(conn_id, db, table, target)`. JOIN ở listing code KHÔNG được update cùng → cross-connection bleed.

### Global Pattern
**[A (query Q) joins entity X with entity Y on key K_old, where Y's identity recently widened to K_new = K_old + new_col C (via migration M)] → [Q over-matches because K_old is now a non-unique prefix of K_new] → Result: row multiplication N → N×k where k = số entry trong Y với cùng K_old khác C.**

- A = listing/report/aggregation query
- X, Y = entities tied via 1:1 or 1:N FK
- K_old = legacy join key (subset of new identity)
- K_new = K_old + C, established by ALTER TABLE + UNIQUE constraint migration
- C = identity-tier discriminator (connection_id, tenant_id, region_id, environment, version...)
- M = migration chain (ADD COLUMN → backfill → relax UNIQUE)

### Đúng pattern (correct flow)
1. **Khi migrate identity-tier**: viết check-list "callers cần update" trong migration comment + post-migration audit script (grep JOIN on Y theo K_old).
2. **Khi viết JOIN trên Y**: luôn join theo TẤT CẢ cột UNIQUE constraint hiện tại, KHÔNG copy-paste join từ legacy code.
3. **Backward compat**: nếu C nullable (legacy rows chưa backfill được), dùng `(Y.C = X.C OR Y.C IS NULL)` + LATERAL LIMIT 1 với ORDER prefer exact-match → keep deterministic.
4. **Smoke test**: post-migration chạy `SELECT COUNT(*) FROM Q;` so với `SELECT COUNT(DISTINCT X.id) FROM Q;` → nếu lệch → JOIN dư duplicate.

### Anti-pattern
- ❌ JOIN bằng K_old "vì query đã tồn tại từ trước" — schema đã evolve, query phải theo.
- ❌ Patch bằng `DISTINCT` hoặc `GROUP BY X.id` mà không scope theo C → mask triệu chứng, vẫn pick non-deterministic Y row.
- ❌ Strip GROUP BY khi response shape cần Y info → tạo runtime drift giữa view và data plane.

### Smell detection
- Bug được phát hiện khi user nhìn JSON listing thấy `id` trùng nhưng `<some_other_id>` khác → 99% là multi-tier identity drift.
- Sau migration ADD UNIQUE constraint mới: grep `JOIN <table>` trong codebase, đối chiếu predicate vs UNIQUE column list.

### Fix kỹ thuật áp dụng (lần này)
- Convert `LEFT JOIN` → `LEFT JOIN LATERAL (SELECT ... WHERE ... AND (Y.C = X.C OR Y.C IS NULL) ORDER BY (Y.C IS NULL) ASC, Y.id ASC LIMIT 1) Y ON TRUE`.
- Bảo toàn wire shape (vẫn 1 `registry_id` scalar field, không cần đổi thành array).
- COUNT(*) tự đúng theo, không cần đụng riêng.

### Áp dụng được cho 3 dự án khác?
- ✅ Multi-tenant SaaS: bất kỳ JOIN theo `(name)` quên `tenant_id` sau khi onboard tenant thứ 2.
- ✅ Multi-region replication: JOIN theo `(entity_key)` quên `region_id`.
- ✅ Multi-env shared metadata: JOIN theo `(resource_name)` quên `env_id` (dev/stg/prod).

### Tags
#sql-join-cardinality #identity-tier-drift #multi-tenant #migration-callers-update #lateral-limit-1 #null-fallback #deterministic-tiebreaker #connection-aware-registry #global-pattern

### File chứng cứ
- `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go` (fix)
- `cdc-cms-service/migrations/schema/core/054_v1_add_source_connection_id.sql`
- `cdc-cms-service/migrations/schema/core/055_backfill_v1_source_connection_id.sql`
- `cdc-cms-service/migrations/schema/core/056_relax_v1_unique_with_connection.sql`
- `agent/memory/workspaces/FixSourceObjectListingDedupe/`

---

## Lesson #N+1 — Dual-tree drift: Agent edits stale source tree, runtime loads from active tree (Global Pattern)
**Date**: 2026-05-21
**Workspace**: `bug-snapshot-v2-host-uri-2026-05-21`
**Severity**: P0 — silent edit failure, agent claims "Done" while runtime behavior unchanged.

### Global Pattern
> **Agent A edits source tree X to fix bug B, but runtime W loads from tree Y (Y ≠ X) → edits don't apply → user sees same bug + agent loses trust.**
> Đúng: TRƯỚC khi edit, A phải resolve `X = runtime_cwd(W)` bằng evidence (lsof/k8s manifest/process inspect), KHÔNG bằng directory listing alphabetical hoặc "tree đầu tiên grep thấy".

### Triệu chứng (lần này)
- User báo `kafka-consume-batch` row vẫn xuất hiện ở activity_log lúc 16:35:56 dù agent đã "remove" feature.
- Agent ban đầu edit `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/handler/kafka_consumer.go` (tree STALE — không phải runtime active).
- Runtime thực tế: K8s `data-hub` cluster + binary build từ `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/`.
- Verification gap: agent chạy `go build && go test` PASS trong cdc-system → false confidence → claim Done.

### Sai lầm (Anti-pattern)
- ❌ Chọn tree dựa trên `ls /Users/trainguyen/Documents/work/` (cdc-system xuất hiện trước data-hub alphabetical) → blind pick.
- ❌ Không inspect running process trước khi edit (lsof /proc cwd, k8s pod image, deployment manifest).
- ❌ Test pass trong tree X = false signal. Pass ≠ runtime reload.
- ❌ Skip §3 "Verify Before Done" — không hỏi "Staff Engineer có duyệt PR này khi runtime tree chưa được xác nhận không?".

### Pattern phát hiện (Smell Detection)
- Repo có ≥2 thư mục cùng tên service (vd: `cdc-system/centralized-data-service/` và `data-hub/centralized-data-service/`) → 99% có drift.
- Một trong các tree có git log mới hơn / có file mới (vd: `KafkaPostConsumeEvent` hook chỉ tồn tại ở 1 tree) → tree đó nhiều khả năng là active.
- K8s deployment / Dockerfile / Makefile build context trỏ về tree Y nhưng agent edit tree X.

### Fix kỹ thuật áp dụng
1. **Pre-edit Resolution Step (BẮT BUỘC)**: trước khi edit file source ở repo đa tree:
   ```bash
   # 1. Tìm process đang chạy
   lsof -p $(pgrep -f <binary-name>) | grep cwd
   # 2. Hoặc k8s
   kubectl -n <ns> get deploy <svc> -o yaml | grep -A2 'image:\|build-context'
   # 3. Đối chiếu với tree X dự định edit
   ```
2. **Diff guard**: nếu 2 tree cùng tồn tại file giống tên, `diff -r treeX treeY <relative-path>` → nếu khác (vd hook ở 1 tree không có ở tree kia) → tree nào có hook = tree active.
3. **Verification After Edit**: rebuild + verify binary từ ĐÚNG tree, sau đó k8s rollout / process restart, sau đó observe activity_log SQL count = 0 (KHÔNG dừng ở "go test pass").

### Áp dụng được cho 3 dự án khác?
- ✅ **Monorepo migration đang giữa chừng**: `legacy-services/auth/` (stale) vs `services/auth-v2/` (active) — agent fix bug ở legacy nhưng deploy chạy v2.
- ✅ **Fork/upstream sync drift**: team có `vendored/` snapshot và `main/` upstream → agent patch vendored, runtime build từ main.
- ✅ **Multi-env worktree**: dev tạo `worktree-feature-A/` để thử nghiệm nhưng quên cleanup → agent edit worktree, prod binary build từ main worktree.

### Hệ luỵ governance
- Vi phạm CLAUDE.md §3 (Plan & Verify — Verification Before Done).
- Vi phạm spirit của §9 (Quản trị Quy mô lớn — Workspace-First) vì agent không xác định runtime context trước khi nạp file vào edit.
- Cần thêm pre-flight: "Đây có phải tree mà runtime đang load không? Evidence?".

### Tags
#dual-tree-drift #stale-tree-edit #runtime-vs-source #monorepo-migration #verify-before-done #lsof-cwd #k8s-deployment-context #global-pattern #worktree-confusion #governance-§3-violation

### File chứng cứ
- WRONG tree (edited first, NO RUNTIME EFFECT):
  - `/Users/trainguyen/Documents/work/cdc-system/centralized-data-service/internal/handler/kafka_consumer.go`
  - `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-web/src/pages/ActivityLog.tsx`
- RIGHT tree (re-applied after user pushback "má mày giỡn mặt"):
  - `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/kafka_consumer.go:882` (kc.db.Create removed, hook retained)
  - `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityLog.tsx:103` (option removed from FE filter)
- Progress audit: `agent/memory/workspaces/bug-snapshot-v2-host-uri-2026-05-21/05_progress.md` Followup #5

---

## [2026-05-25] Global Pattern: LWW Guard for Dual-Stream Consistency (Snapshot + CDC)

- **Trigger**: Pipeline CDC đồng thời chạy luồng Snapshot V2 (lấy data tĩnh, chậm) và luồng CDC Debezium (realtime event), ghi chung vào một shadow table trên PostgreSQL. Gặp hiện tượng Snapshot ghi đè dữ liệu mới hơn của luồng Realtime (race condition) vì Timestamp wall-clock của Worker không đồng nhất và luồng Snapshot luôn ghi đè nếu Timestamp lớn hơn.
- **Root Cause**: 
  1. Sử dụng wall-clock `time.Now()` của Worker làm thời gian source cho luồng Snapshot, dẫn đến clock skew với Debezium (`source_ts`).
  2. Mệnh đề Optimistic Concurrency Control (OCC) xử lý trùng Timestamp (`<=`) cho phép Snapshot đến sau được phép ghi đè.
  3. Thiếu `_source_ts` ở schema V1.
- **Correct Pattern [Global Pattern: LWW Guard cho luồng kép]**:
  1. **Logical Clock**: Luôn lấy Logical Clock từ nguồn làm mốc thời gian tuyệt đối. Với MongoDB, đó là `clusterTime` (lấy qua lệnh `hello` hoặc `replSetGetStatus`). Tuyệt đối không dùng wall-clock của Worker để so sánh thứ tự ghi DB.
  2. **Tiebreaker Discriminator**: Khi trùng lặp Timestamp (`_source_ts = EXCLUDED._source_ts`), cần có một định danh ưu tiên (`_source`). Cấu hình `WHERE` clause để nguồn có độ trễ lớn hơn (Snapshot) luôn thua nguồn Realtime.
  3. **Universal Schema Guard**: Bổ sung `_source_ts` cho TẤT CẢ các bảng shadow (backport V2->V1) để bật LWW guard thay vì dùng hash-dedup ở các phiên bản cũ.
- **Tags**: #cdc #lww #strong-eventual-consistency #cluster-time #logical-clock #occ-guard #tiebreaker #global-pattern

## [2026-05-25] Global Pattern: Incomplete Execution Scope (Agent Mid-Session Correction)

- **Trigger**: Agent được cấp một workspace với các file `00_context.md`, `02_plan.md`, `09_solution.md` và user ra lệnh thực thi. Agent chỉ thực hiện file `09_solution.md` (chỉ chứa backend logic) mà bỏ quên toàn bộ các yêu cầu frontend/API từ ngữ cảnh ban đầu (được nêu trong tin nhắn trước đó hoặc ngầm định trong workspace context). Khi báo cáo hoàn tất, User phản hồi gay gắt: "sao làm cái này, phải là làm theo toàn bộ cái này chứ em."
- **Root Cause**:
  1. Agent có tính cục bộ (tunnel vision): Chỉ tập trung vào file design cuối cùng hoặc task cụ thể được chỉ định trong một file nhỏ (ở đây là `09_solution.md` cho Muscle) mà bỏ qua bối cảnh tổng thể (Control Plane đòi hỏi cả UI và API để End-to-End).
  2. Bỏ qua pre-flight check của Workspace-First Rule: Không đối chiếu lại End Goal (Definition of Done) trong `00_context.md` và các tin nhắn của User với những phần đã thực thi.
- **Correct Pattern [Global Pattern: Full-Scope Execution Validation]**:
  1. **Holistic Definition of Done**: Trước khi kết luận một tính năng hoàn tất, bắt buộc phải kiểm tra chéo (Cross-check) với mọi yêu cầu trong `00_context.md` và lịch sử chat (VD: yêu cầu "thêm 1 tab monitor...").
  2. **End-to-End Traceability**: Tính năng backend không thể gọi là "xong" nếu không có trigger từ Frontend hoặc API tương ứng nếu thiết kế là tính năng tương tác (Control Plane).
  3. **Mid-Session Recovery**: Khi bị User nhắc nhở "làm toàn bộ", agent phải dừng lại, ghi nhận Lesson, lên kế hoạch (Implementation Plan) cho các phần còn thiếu, và tiếp tục hoàn tất ngay thay vì cự cãi hoặc chỉ dừng lại ở backend.
- **Tags**: #execution-scope #tunnel-vision #end-to-end #workspace-context #global-pattern #mid-session-correction

---

## [2026-05-26] Global Pattern: Execution without Planning and Verification (Governance Violation)

- **Trigger**: User phàn nàn về lỗi SLOW SQL và sai lệch tiến trình ("100% nhưng vẫn đang chạy ẩn"). Agent ngay lập tức nhảy vào xem code, sửa code `BatchBuffer` và `SchemaAdapter`, tự compile rồi restart lại service mà không hề đưa ra Implementation Plan, không xin phép thay đổi, và không tạo báo cáo (report). Hành động này phá vỡ toàn bộ quy trình kiểm soát (Governance framework) do User đặt ra.
- **Root Cause**:
  1. **Bỏ qua Planning Phase**: Phớt lờ quy tắc bắt buộc lập plan cho mọi task > 3 bước. Mất đi bước "Review Plan" khiến User mất quyền kiểm soát code và hệ thống. Tự ý thay đổi logic cốt lõi (batching, synchronizing) mà không có sự đồng thuận.
  2. **Thiếu tính Minh bạch và Audit (Verification Phase)**: Agent tự ý phán đoán thành công qua log, không tổng hợp các file thay đổi, không chứng minh bằng kết quả thực tế ("vì sao ko đc report láo"). 
  3. **Không tuân thủ Report Artifacts**: Bỏ qua việc tạo file `report_*.md` để ghi nhận các thay đổi, khiến User không có cơ sở track lại lịch sử cấu trúc khi cần rollback.
- **Correct Pattern [Global Pattern: Strict Governance & Plan-Execute-Verify Loop]**:
  1. **Plan First**: Trước khi viết 1 dòng code, BẮT BUỘC phải sinh file `implementation_plan.md` (hoặc trình bày rõ ràng) với giải pháp cụ thể (kèm code demo) và DỪNG LẠI CHỜ User duyệt. Tuyệt đối không "hỏi cái là làm".
  2. **Verify & Report Before Done**: Không bao giờ được báo "done" bằng miệng. Bắt buộc tạo file `report_[TaskName]_[Date].md` ghi lại danh sách file đã thay đổi, mục đích, và cách verification. 
  3. **Respect Core Systems**: Không dùng trick cheat hệ thống. Mọi luồng fix bug phải tuân theo chuẩn kiến trúc và phải được test lại service work mới báo done.
- **Tags**: #governance-violation #no-planning #report-integrity #discipline #process

---

## [2026-05-25] Global Pattern: Over-complicating Scope vs Simple Request (Spec-Drift)

- **Trigger**: User yêu cầu một sửa đổi đơn giản (thêm cờ `overwrite` dạng true/false khi bấm nút snapshot trên UI để truyền xuống API snapshot-v2). Agent tự ý suy diễn thêm scope lớn (thêm ddl_status dropdown vào Edit Modal, chia tách routing của shadow-binding và source-object, thay đổi logic check database ở Worker) tạo ra kế hoạch phức tạp khiến User phản ứng: "mày rất vớ vẩn. check lại xem cái TableRegistry.tsx của tao vẫn chạy đc. giờ chỉ cần truyền overwrite vào là đc cho true hay false. đi check tùm lum".
- **Root Cause**:
  1. **Spec-Drift (Lệch đặc tả do suy diễn)**: Agent không bám sát yêu cầu tối giản của User, tự động mở rộng scope sang những tính năng không được yêu cầu dưới danh nghĩa "thiết kế thanh lịch" (Demand Elegance) hoặc "chống lỗi hệ thống" (Root Cause Analysis).
  2. **Tư duy Over-engineering**: Thay vì giải quyết trực tiếp pain-point hiện tại của User (giao diện snapshot-v2 đang thiếu cờ overwrite), Agent lại đi thiết kế lại cách vận hành metadata của cả Table Registry.
- **Correct Pattern [Global Pattern: Minimum Viable Change (MVC)]**:
  1. **Bám sát yêu cầu tối giản**: Khi User yêu cầu một chức năng A, chỉ implement đúng A. Tuyệt đối không tự suy diễn các mối quan hệ B, C, D để làm thay đổi kiến trúc hiện có, trừ khi đó là lỗi runtime crash trực tiếp.
  2. **Xác nhận Scope trước khi thiết kế**: Nếu thấy một giải pháp có nguy cơ làm phình to scope hoặc đụng chạm đến nhiều layer (FE + BE API + Worker + DB schema), dừng lại và hỏi/chỉ rõ mức độ tác động tối thiểu để User lựa chọn, thay vì tự đưa vào plan như một yêu cầu bắt buộc.
- **Tags**: #spec-drift #over-engineering #minimum-viable-change #simple-request #discipline #governance-violation


---

## L-CDC-source-identifier-2026-05-22 — Envelope.source: short identifier, not transport path

**Trigger**: CDC realtime path đẩy `record.Source = "/kafka/cdc.goopay.X.Y"`
(41 ký tự) → SQLSTATE 22001 trên cột `_source VARCHAR(20)`, 0 row ghi vào
shadow table. Đồng thời path-form phá literal-match trong LWW guard SQL
(`_source = 'snapshot:v2'`).

**Global Pattern [A propagates raw transport-path B into persisted identity field X]**
→ Result Y: column overflow + break downstream literal-match semantics.

**Đúng**: Transport layer dựng envelope với SHORT STABLE identifier
(e.g., `debezium`, `snapshot:v2`). Transport metadata (topic, subject, partition)
đẩy qua kênh riêng (subject parameter, headers) — không nhét vào identity field
có constraint/literal-compare.

**Khi nào áp dụng**:
- Bất kỳ pipeline nào có column-level constraint (VARCHAR(n), ENUM) trên field
  identity được multi-source populate.
- Bất kỳ guard/UPSERT SQL nào dùng literal compare (`= 'X'`) trên identity field.
- Khi mix nhiều source (snapshot, realtime, bridge, retry) cùng ghi vào 1 store.

**Audit checklist khi tạo envelope mới**:
1. Identity field có constraint không? (`VARCHAR(n)`, enum, regex check)
2. Có downstream nào literal-match trên field này không?
3. Source identifier có dùng transport-specific format
   (`/kafka/...`, `nats://...`, URL) không? Nếu có → refactor thành short ID.

**Cross-project applicability**: Pattern này gặp ở:
- Event-driven systems (Kafka, NATS, RabbitMQ): source field
- Audit log: actor field
- Multi-tenant: tenant_id mixed with tenant_path
- HTTP middleware: user_agent vs canonical client_name

---

## L-CDC-circuit-breaker-2026-05-22 — DLQ-mode pipelines need a circuit breaker

**Trigger**: snapshot.v2 non-strict mode khi `HandleRaw` fail trên every doc
(deterministic VARCHAR overflow) → `continue` loop qua 6M rows → DLQ flood,
"chạy điên", ẩn root cause khỏi operator log.

**Global Pattern [A pipeline using DLQ-on-error pattern over a stream of N items
encounters a deterministic failure F that affects every item]**
→ Result Y: N items processed, N DLQ rows written, log spam, no operator signal,
no halt — wasted hours of compute + storage before someone notices.

**Đúng**: DLQ-mode pipelines PHẢI ship kèm circuit breaker với ≥ 2 trip conditions:
1. **Consecutive failure threshold** (e.g., 100 in a row) — bắt deterministic F.
2. **Window/batch error-ratio threshold** (e.g., ≥50% errors with ≥10 absolute)
   — bắt systemic F mà success-rate ngắt quãng làm reset consecutive counter.

Khi trip: PHẢI flush partial DLQ trước (forensics), persist halt state
(`status='error'`), log với ALL counters + last error, return từ work loop để
defer activity ghi nhận. Resume sau fix là khả thi vì checkpoint còn nguyên.

**Khi áp dụng**:
- Batch ETL với per-item fault-tolerance (skip bad row, continue).
- CDC snapshot / replay pipelines.
- Bất kỳ "consumer X items, drop failures into DLQ" pattern nào.
- Mass-mail / mass-notification senders.

**Anti-pattern**: "DLQ là đủ rồi, operator sẽ thấy". KHÔNG — DLQ là forensics
storage, KHÔNG phải alerting mechanism. Without CB, by the time someone notices
DLQ count, the pipeline đã waste compute/IO trên triệu items.

**Defaults gợi ý** (tune theo workload):
- Consecutive: 100 (deterministic F sẽ trip trong batch đầu).
- Ratio: 50% với min 10 absolute (tránh false trip trên batch nhỏ).

**Cross-project applicability**:
- Kafka consumers với skip-on-error
- Airflow tasks với `trigger_rule='all_failed'` patterns
- Webhook fan-out với retry+DLQ
- Mass migration scripts (DB schema, file conversion)

**Implementation note** (Go): centralise error bookkeeping vào closure
(`recordError`) để strict-vs-non-strict mode + counter increments + trip check
+ DLQ append tập trung 1 chỗ — tránh logic drift khi maintainer thêm error site mới.

---

## [2026-05-26] L-integration-test-drops-dev-db-table — Integration tests dropping shared tables in local developer environment

- **Trigger**: Chạy unit/integration tests (`go test`) trong `centralized-data-service` gây mất hoàn toàn bảng `cdc_system.failed_sync_logs` và các phân vùng của nó trên database local `cdc_dw`, làm crash API `cdc-cms-service` khi query system health với lỗi `relation "cdc_system.failed_sync_logs" does not exist`.
- **Root Cause**:
  1. **Tác động trực tiếp vào database chia sẻ**: Tệp kiểm thử `partition_dropper_test.go` kết nối trực tiếp vào local database phát triển và thực hiện `DROP TABLE IF EXISTS cdc_system.failed_sync_logs CASCADE` ở đầu và cuối test để dọn dẹp môi trường.
  2. **Không khôi phục/cách ly dữ liệu**: Việc sử dụng bảng trùng tên với bảng thật trong test suite mà không có cơ chế cô lập (ví dụ sử dụng database test riêng hoặc đặt tên bảng test khác) dẫn tới xoá sạch bảng thật của môi trường dev local sau khi chạy test.
- **Correct Pattern [Global Pattern: Isolated Test Tables & DB Sandbox]**:
  1. **Sử dụng bảng có hậu tố test**: Thay vì tác động trực tiếp vào thực thể bảng thật (ví dụ `failed_sync_logs`), hãy đổi tên bảng trong test suite thành `failed_sync_logs_test` và ghi đè (override) các quy tắc (rules) hoạt động của component trong test để chỉ truy vấn bảng test này.
  2. **Tránh xoá sạch bảng dùng chung**: Tuyệt đối không dùng lệnh `DROP TABLE CASCADE` trên các bảng cấu trúc cốt lõi của database dùng chung trong quá trình chạy test, trừ khi đó là database sandbox hoàn toàn độc lập (ví dụ database kết thúc bằng `_test` và được khởi tạo mới trên mỗi lượt chạy).
- **Tags**: #integration-test #database-cleanup #shared-database #relation-missing #isolated-testing

---

## L-CDC-route-empty-silent-skip-2026-05-26 — Snapshot.v2 silent-skip khi route cache stale

**Trigger**: Lần snapshot.v2 ĐẦU TIÊN cho một source vừa register → activity_log báo `status=success, rows_affected=N` (số doc Mongo Find quét được) nhưng shadow table 0 row. Lần snapshot kế tiếp work bình thường.

**Failure chain (4 lớp chồng nhau)**:
1. **L4 Cache stale**: Worker `MetadataRegistryService.ReloadAll` chỉ chạy (a) startup, (b) khi nhận NATS subject `schema.config.reload`. Signal là fire-and-forget. Khi CMS commit → publish reload → trả 202, user lập tức click "Snapshot Now" → race với ReloadAll chưa xong.
2. **L3 Silent skip**: `event_handler.processEvent` trả `(0, nil)` + log Debug khi `ResolveSourceRoutes` rỗng. Debug log = tắt mặc định ở mọi env → operator không bao giờ thấy.
3. **L2 Caller discard**: `snapshot_runner` vứt return value `written` (`_, err := HandleRaw(...)`). Mất tín hiệu route-miss → CB không trip.
4. **L1 Misleading metric**: `rowsTotal += int64(len(batch))` đếm doc Find, không phải doc routed. activity_log báo láo.

**Global Pattern [A pipeline with multi-layer cache + silent-skip-on-cache-miss + fire-and-forget cache-reload signal]** → Result Y: bất kỳ caller nào fire một mutation rồi dùng resource phụ thuộc cache reload sẽ gặp first-call-fail mà KHÔNG có tín hiệu lỗi, vì:
- Cache miss = silent-skip (cố ý cho streaming use case, ví dụ Kafka consumer skip topic chưa cấu hình).
- Caller discard return → không phân biệt được "success: routed" vs "success: skipped".
- Metric đếm số input thay vì số output → log success với rows > 0.

**Đúng (4-layer defense in depth)**:
1. **L4 Pre-flight sync trigger**: caller force in-process `ReloadAll(ctx)` ngay trước hot loop. KHÔNG phụ thuộc signal async.
2. **L4 Hard-assert**: sau reload, gọi resolve một lần — empty → fail fast với message chỉ đích danh root cause (cờ `is_active`, missing entry).
3. **L3 Log level cho operator**: silent-skip log = Warn (không Debug) + đầy đủ context (subject, db, table) để greppable.
4. **L2 Use return**: caller dùng return value `written`. `written == 0 && err == nil` = silent-skip → treat như doc error, đếm vào circuit breaker.
5. **L1 Metric trung thực**: `rowsTotal += writtenSum`, KHÔNG `+= len(batch)`.

**Cross-project applicability**:
- Bất kỳ pipeline ETL nào có per-table/per-source routing + cache + fire-and-forget reload (Kafka consumer, NATS subscriber, ETL worker, CDC connector).
- Mass-mail worker với template cache + async reload signal.
- API gateway với route cache + config reload.
- Search indexer với schema cache + reindex trigger.

**Audit checklist khi thiết kế silent-skip + cache reload**:
1. Skip log level: Warn hay Debug? Operator có greppable từ stdout không?
2. Caller có inspect return count không? Hay vứt `_, err`?
3. Metric đếm input (msg.consumed) hay output (rows.routed)?
4. Có pre-flight sync option để caller force reload không?
5. Có hard-assert trước hot loop khi caller biết phải có route (snapshot/replay) không?

**Anti-pattern**: "Tất cả silent-skip thì caller tự lo" → caller code thường chỉ care error, không nhận biết được "success but skipped" — đặc biệt khi return shape là `(int, error)` mà int dễ bị discard.

**Implementation note** (Go): khi sửa silent-skip + caller, ưu tiên:
- Giữ silent-skip ở producer side (an toàn cho stream/multi-tenant).
- Thêm Warn log có greppable keys.
- Caller (batch/snapshot/replay) thêm pre-flight reload + assert + return-value check ở hot path.

**File reference**:
- centralized-data-service/internal/handler/event_handler.go:84-99 (Warn log fix)
- centralized-data-service/internal/handler/snapshot_runner_handler.go:279-307 (pre-flight + assert)
- centralized-data-service/internal/handler/snapshot_runner_handler.go:461-495 (written return check)
- centralized-data-service/internal/handler/snapshot_runner_handler.go:518-521 (rowsTotal accurate)

**Tags**: #cdc #snapshot-v2 #cache-reload #silent-skip #fire-and-forget #metric-accuracy #defense-in-depth #global-pattern

---

## L-2026-05-26 Per-severity log sampling silently drops Info logs without warning (telemetry)

**Bối cảnh**:
Worker `centralized-data-service` log đầy đủ trên stdout nhưng SigNoz chỉ nhận được subset (~10%). Hiện tượng: log `*.start` đôi khi xuất hiện nhưng matching `*.ok` thì biến mất. Periodic log (metadata reload, kafka discover, batch upsert) gần như không bao giờ thấy. KHÔNG có log warning "degraded" trên stdout — nghĩa là OTLP exporter healthy, không phải fallback mute.

**Root cause**:
`pkgs/observability/otel.go` có wrapper `severityAwareCore` áp dụng per-severity dice-roll sampling TRƯỚC khi forward sang OTel exporter. Config default `info: 0.1` → 90% Info log silently bị drop. Console branch KHÔNG được wrap (theo design) → stdout đầy đủ. Đây là divergence trông như "log missing" với operator không đọc kỹ config.

**Global Pattern [A applies probabilistic sampling to B at telemetry layer X → effect on observer Y]**:
- A = sampling wrapper (severityAwareCore)
- B = log entries (or traces/metrics)
- X = bridge between application code và backend exporter
- Y = observer thấy KHÔNG đủ data → false alarm "exporter broken" / "service stalled" / "missed event"

**Sai lầm phổ biến**:
1. Đặt sampling ratio < 1 cho Info BẰNG default mà không phân loại "milestone Info" (rare, business-critical) vs "noisy Info" (every-tick, periodic).
2. Wrapper sample tại `Check(ent, ce)` time → KHÔNG có field context → không thể discriminate based on field tag.
3. Operator không có dashboard hiển thị "drop rate" → silent loss.

**Đúng**:
1. **Layered sampling**: Production giữ ratio thấp (cost) NHƯNG bypass cho entry có tag `audit=true` (or equivalent priority field).
2. **Defer sampling tới Write** không phải Check — Write có fields trong scope. Pattern:
   ```go
   func (s *core) Check(ent, ce) *CheckedEntry {
     if muted { return ce }
     if ratioFor(level) <= 0 { return ce }
     return ce.AddCore(ent, s)   // register self, not inner
   }
   func (s *core) Write(ent, fields) error {
     if muted { return nil }
     if hasAuditField(fields) || s.audited {
       return s.Core.Write(ent, fields)  // bypass sampling
     }
     // ... dice roll ...
   }
   ```
3. **Propagate audit flag qua With**: derived cores (logger.With(...)) phải mang theo `audited bool` → contextual audit logger.
4. **Mute KHÔNG bypass**: degraded sink = unhealthy, đừng force-push.
5. **Document trade-off**: comment in code + config nói rõ "Info=0.1 = 90% drop" để operator biết.

**Anti-pattern**:
- Set ratio < 1 cho Warn/Error: alert sẽ miss. Luôn ratio = 1.0 cho Warn+.
- Sample Trace headers theo cùng ratio: gãy distributed context. Sample cha-con consistent.

**File reference**:
- centralized-data-service/pkgs/observability/otel.go:96-180 (severityAwareCore Check/Write/hasAuditField)
- centralized-data-service/config/config-local.yml:60 (info: 1.0 cho local)
- centralized-data-service/config/config-production.yml:84 (info: 0.05 + rely on audit bypass)
- centralized-data-service/internal/handler/command_handler.go:1195,1209,1266,1321 (audit field tag)

**Tags**: #observability #otel #signoz #log-sampling #silent-drop #zap #severity-aware #audit-bypass #global-pattern

---

## L-2026-05-26-trace — Child span + log correlation pattern (Global)

**Context**: hot-path function trong Go service dùng OTel có nhiều error branch. Muốn span luôn được record error + log carrier trace_id/span_id, không phụ thuộc nhớ thủ công ở mỗi branch.

**Global Pattern**: khi A (Go function) cần child span tại B (operation entry) trong pipeline X (CDC/HTTP/RPC), dùng deferred-pointer error pattern:

```go
func op(ctx context.Context, ...) (result T, err error) {
    ctx, span := observability.ChildSpan(ctx, "B",
        attribute.String("X.key", ...),
    )
    defer observability.EndSpan(span, &err)
    // body — return ở bất kỳ branch nào
}
```

Trong đó `EndSpan(span, &err)` đảm bảo:
- `span.RecordError(*err) + span.SetStatus(codes.Error, msg)` khi `*err != nil`.
- `span.End()` luôn được gọi.
- Capture cả panic-translated error (vì named return).

**Pattern phụ — log carrier**:
```go
observability.Ctx(ctx, logger).Error("msg",
    observability.ErrorField(err),
    observability.Attrs(zap.String("k", v), ...),
)
```
→ inject `trace_id`/`span_id` từ ctx, error encode đúng template OTel (`{error: {kind, message, stack}}`), attributes namespace dưới `attributes`.

**Sai (anti-pattern)**:
- Manual `span.RecordError(err)` rải rác mỗi error branch — dễ miss, status không nhất quán.
- `attribute.String("error", err.Error())` thay vì `RecordError` — SigNoz Exception tab không nhận, status không Error.
- Log trực tiếp `logger.Error(...)` mà không qua `observability.Ctx(ctx, ...)` — SigNoz Linked Logs/Spans N/A → không correlate được.

**Kết quả Y**: SigNoz hiển thị (1) trace tree đầy đủ parent→child, (2) Exception tab có dữ liệu 100 0.000000e+00rror path, (3) Logs → Trace navigation hoạt động qua trace_id, (4) span attributes query-able trong query builder.

**Áp dụng được**: bất kỳ Go service dùng OTel + zap + hot-path function có ≥2 error branch. Đã validate trên 4 module (kafka_consumer/event_handler/batch_buffer/schema_inspector), 4 unit test PASS.

**Helper cần có** (tạo 1 lần, reuse):
- `ChildSpan(ctx, name, attrs...)` — wrap `tracer.Start(ctx, name, WithAttributes(...))`.
- `EndSpan(span, *err)` — defer-safe, nil-safe.
- `Ctx(ctx, base) *zap.Logger` — inject trace fields.
- `ErrorField(err)`, `Attrs(fields...)` — template-compliant encoding.

**Files reference**:
- centralized-data-service/pkgs/observability/trace_helpers.go (helpers)
- centralized-data-service/pkgs/observability/log_template.go (Phase B)
- centralized-data-service/internal/handler/kafka_consumer.go:382-470 (đầy đủ pattern)

**Tags**: #observability #otel #signoz #trace #span #child-span #log-correlation #deferred-pointer #global-pattern


---

## L-2026-05-26-legacy-config-gate-kills-feature — Legacy single-config gate disables entire feature (Global)

**Context**: Service S có feature F phụ thuộc nhiều phụ thuộc {C1, C2, C3, ...}. Phụ thuộc C1 có legacy config field `cfg.C1.URL` (single instance). Sau khi service migrate sang V2 multi-source/multi-tenant architecture, C1 được resolve **per-source** từ registry (vd `connection_registry`). Tuy nhiên init code cũ vẫn dùng `if cfg.C1.URL != "" { initF(...) }` để gate toàn bộ init feature F.

**Symptom**: Trong deployment V2-only (không set cfg.C1.URL vì đã có per-source registry), feature F **chết âm thầm** — log scheduler/handler ghi `"skipped (C1 not configured)"` mỗi tick. Operator tưởng config missing, nhưng thực ra V2 registry đã đủ thông tin → mismatch giữa nhận thức và sự thật runtime.

**Global Pattern (sai)**:
```
A (init code) gates feature-F-construction by legacy single C1 config
  ↓
V2 deployment (no C1 single config, but registry R has per-source C1 URIs)
  ↓
F never constructed → scheduler/handler tick logs "skipped (C1 missing)"
  ↓
Operator confusion: registry đầy đủ nhưng feature dead
  ↓
Result Y: silent feature death + misleading log message
```

**Đúng (correct flow)**:
1. **Tách init**: feature F init **luôn**. Chỉ subsystem cần single-default C1 mới gate bởi cfg.
   ```go
   var defaultC1 *C1Client = nil
   if cfg.C1.URL != "" {
       defaultC1 = newC1Client(cfg.C1.URL)
   }
   // F init UNCONDITIONALLY
   f := NewF(perSourceResolver, defaultC1)  // defaultC1 may be nil
   ```
2. **Populate per-source identity từ V2 registry**: ở registry reload path, build `URIByCode` map từ rows đã fetch (dùng helper resolver chung) → mỗi `entry.SourceURI` non-empty cho V2 sources.
3. **Lazy resolve per-source**: F's worker dùng `entry.SourceURI` để lazy-create per-source client. Fallback `defaultC1` khi URI empty.
4. **Hard-assert defense in depth**: khi cả `entry.SourceURI=="" && defaultC1==nil` → return error rõ "verify registry row OR set legacy cfg.C1.URL", KHÔNG silent panic.
5. **Subscriber asymmetry**: handler luôn register subjects (vd NATS), tự return structured error khi service-instance nil. Né lesson L3100 (conditional subscriber gating → publisher orphan).
6. **Defensive log upgrade**: nếu init regression xảy ra (F vẫn nil sau fix), log **Error** "wiring regression: F should be init unconditionally since <date>", không phải Warn "missing config".

**Layered fix tương ứng failure mode độc lập**:
- L1 (architectural): registry populate per-source identity → fix gốc rễ.
- L2 (legacy gate): bỏ gate quanh feature-F-init → fix triệu chứng tức thì.
- L3 (defense in depth): hard-assert + error message rõ → debuggable nếu L1/L2 regress.

**Sai (anti-pattern)**:
- Set `cfg.C1.URL` fake để bypass gate → workaround che bug, không fix gốc.
- Refactor TẤT CẢ subsystems (C2, C3, ...) sang per-source cùng lúc → scope blow-up cho 1 bug fix. Migrate dần, giữ legacy fallback cho subsystem chưa refactor.
- Gate cả subscriber registration bằng `if defaultC1 != nil { register(subject, handler) }` → vi phạm L3100 (publisher gửi vào subject chết).

**Kết quả Y**: V2 deployment không cần legacy cfg field → feature F chạy đúng per-source. Legacy deployment (cfg.C1.URL set) backward compat. Operator log message phân biệt rõ "config missing" vs "wiring regression".

**Áp dụng được**: bất kỳ service migrate single-config → multi-source registry mà còn legacy gate. Đã validate trên CDC reconcile feature (centralized-data-service, ReconCore × MongoDB × connection_registry V2).

**Files reference**:
- centralized-data-service/internal/server/worker_server.go:170-213 (tách defaultClient init khỏi reconCore init)
- centralized-data-service/internal/server/worker_server.go:467-525 (reconHandler luôn register, healer/backfill gate riêng)
- centralized-data-service/internal/server/worker_server.go:860-880 (Warn → Error wiring regression)
- centralized-data-service/internal/service/metadata_registry_service.go:178-212,378-410,564-580 (resolveSourceURIFromConn helper + populate SourceURL)
- centralized-data-service/internal/service/recon_source_agent.go:189-203 (hard-assert getClient)

**Tags**: #legacy-config-gate #v2-migration #multi-source #silent-feature-death #defense-in-depth #connection-registry #global-pattern

---

## L-2026-05-26-governance-violation — Workspace-First enforcement bypassing and root cause analysis (Global)

**Context**: Agent starts performing queries/exploratory actions (like grep_search, view_file) on a new feature/bug request before creating the workspace folder (`agent/memory/workspaces/[FeatureName]`), violating Rule #9 (Workspace-First Rule).

**Global Pattern (Wrong)**:
Agent receives a request -> jumps straight to searching the code or reading files -> locates the issue -> starts fixing -> realizes the workspace is not initialized -> has to pause, initialize workspace, and record root cause post-facto.
This pollutes the initial session context and violates governance gates.

**Correct Flow (Workspace-First)**:
1. **Gate #0 Check**: As soon as a new user request or issue is received, immediately check if there is an active workspace directory under `agent/memory/workspaces/` for this topic.
2. **Immediate Initialization**: If none exists, stop. Do NOT run grep, find, or view_file on the codebase. Immediately create the workspace directory, write the mandatory files (`00_context.md`, `02_plan.md`, `05_progress.md`), and document the scope first.
3. **Trace and Document**: Only then proceed to execute the task.

**Tags**: #governance #workspace-first #protocol-discipline #gate-zero #kaizen #global-pattern


---

## L-2026-05-26-metric-defined-but-never-set — Metric/Alert dead vì define-only, không có call-site (Global)

**Context**: Service S import library metric (vd Prometheus client). Developer khai báo metric M (counter/gauge/histogram) tại file `metrics.go` với name + labels. Alert rule R trong `prometheus.yml` hoặc Grafana đã threshold trên M. Nhưng trong runtime code path, KHÔNG có bất kỳ `.Set()` / `.Inc()` / `.Observe()` call site nào để update M.

**Symptom**: Dashboard panel hiển thị M = 0 hoặc "no data" hằng định. Alert R không bao giờ kích. Khi sự cố thật xảy ra (vd Kafka consumer lag tăng), operator không nhận thông báo. Hệ quả là **false confidence** — dashboard "xanh đẹp" không phản ánh trạng thái thực.

**Global Pattern (sai)**:
```
A (developer) defines metric M in metrics package
  ↓
A defines alert rule R on M with threshold T
  ↓
A KHÔNG implement .Set()/.Inc()/.Observe() call ở runtime path P
  ↓
Operator sees dashboard panel for M = always 0
  ↓
Alert R DEAD — never fires even when underlying problem (e.g. consumer lag) is real
  ↓
Result Y: silent monitor failure + false confidence
```

**Đúng (correct flow)**:
1. **Definition + call-site coupling**: mỗi metric M phải có ≥1 call-site update trong runtime code path. Lint hoặc CI check phát hiện "defined but never used".
2. **Smoke test metric**: integration test bật service → đợi N giây → curl `/metrics` → assert metric M có value ≠ 0 (hoặc có sample) trước khi merge.
3. **Synthetic alert test**: chạy alert rule với artificial data (vd metric forcibly set qua test fixture) → verify alert firing pipeline (notification reaches Slack/PagerDuty).
4. **Definition near use**: khai báo metric trong cùng package với call-site khi có thể. Tránh `pkgs/metrics/` "metric warehouse" — dễ orphan.
5. **Cross-ref grep**: trước khi merge metric mới, grep call-site. Nếu count = 0 → block PR.

**Sai (anti-pattern)**:
- "Metric warehouse pattern": tất cả metric khai báo 1 file `metrics.go` global → khó audit call-site.
- Khai báo metric M nhưng dùng alternative path (vd CMS scrape kafka-exporter trực tiếp thay vì M từ worker) mà KHÔNG xóa M → 2 nguồn truth, M dead nhưng vẫn render dashboard.
- Test chỉ check metric exists trong `/metrics` output (string match), không check value ≠ 0.
- Khai báo alert R trên M trước khi implement call-site → "alert-driven development" backward.

**Kết quả Y khi áp dụng đúng**: Dashboard phản ánh trạng thái thật. Alert R kích đúng khi vấn đề xảy ra. Operator confidence ↔ runtime state.

**Áp dụng được**: bất kỳ service production có observability stack (Prometheus / StatsD / DataDog / OpenTelemetry metric). Đã validate qua audit CDC pipeline (`cdc_kafka_consumer_lag` gauge định nghĩa nhưng không có `.Set()` call → alert `HighConsumerLag` từ worker side dead, dù CMS scrape kafka-exporter trực tiếp cũng work).

**Liên quan lesson**:
- L985 silent-skip pattern (định nghĩa code path bỏ qua mà không log/alert).
- L3100 conditional subscriber (publisher publish vào subject không có subscriber).

**Files reference**:
- centralized-data-service/pkgs/metrics/prometheus.go:73-79 (gauge ConsumerLag định nghĩa)
- centralized-data-service/internal/handler/kafka_consumer.go (KHÔNG có ConsumerLag.Set() — grep confirm)
- cdc-cms-service/internal/infra/observability/probes/kafka_lag.go:34-125 (alternative path via kafka-exporter scrape)

**Tags**: #observability #metric-dead #prometheus #false-confidence #alert-silent #audit-pattern #global-pattern

## L-2026-05-27-audit-driven-gap-fix-workflow
- **Date**: 2026-05-27
- **Workspace**: `plan-cdc-qa-gap-fix-2026-05-27`
- **Global Pattern**: Khi audit `A` cho ra rating matrix `R(L0..L4)` với composite score `S/Smax`, để vá gap đúng quy trình phải: (1) phân loại gap theo `priority(P0/P1/P2)` dựa trên blocker/release/backlog; (2) mỗi gap có `file:line` evidence + code demo trong markdown (không touch source ở Brain phase); (3) mỗi phase có `verify command` định lượng PASS/FAIL; (4) tạo UI dashboard `D` đọc state `S(gap)` từ DB (không YAML) để operator follow real-time; (5) workflow Brain plan → User verb (`execute p_n`) → Muscle execute → re-audit score → APPEND progress.
- **Đúng**: Score audit `S0` → plan `n` phase với delta `ΔS_i` rõ ràng → mỗi phase có DoD + verify command + composite recompute → UI dashboard cho visibility (không tự ý đổi score) → Verb-driven approval (User chốt phase nào execute) → Audit log APPEND-only.
- **Sai (anti-pattern)**: (a) Plan không có evidence file:line → bịa fix; (b) Brain tự sửa code thay vì document → vi phạm §12; (c) UI state lưu YAML → mất real-time + redeploy mỗi update; (d) Phase parallel không khai báo dependency → execute sai thứ tự; (e) Không có verify command → "đã xong" không chứng minh được.
- **Áp dụng được cho**: bất kỳ audit-driven improvement nào (security audit, performance audit, accessibility audit, compliance audit) với rating matrix + gap → fix → score delta pattern.
- **Liên quan**: L-2026-05-26-metric-defined-but-never-set (audit gốc), §1 Brain/Muscle, §7 Full Doc Set, §11 APPEND-only, §12 Brain Code Prohibition, §14 Pre-flight.

## L-2026-05-27-hardcoded-mask-violates-data-accuracy-law
- **Date**: 2026-05-27
- **Workspace**: `plan-sensitive-masking-fix-2026-05-27`
- **Global Pattern**: Khi hệ thống `S` đồng bộ dữ liệu cá nhân từ source `A` sang sink `B` (CDC, ETL, replica), nếu masking layer `M` thay thế giá trị nhạy cảm bằng **chuỗi cứng** `L` (vd `"***"`, `"masked"`, `"XXX"`) tại write-path, thì:
  - (1) phá hủy tính chính xác `Accuracy(A→B)` → vi phạm luật quyền chỉnh sửa/audit của chủ thể dữ liệu (vd Luật BVDLCN VN 91/2025 Điều 13, GDPR Art.16);
  - (2) làm sink mất khả năng đối soát + đếm distinct cho field định danh (CCCD, card_number);
  - (3) không qua được kiểm toán "biện pháp kỹ thuật phù hợp" của cơ quan thanh tra vì không có audit trail strategy + actor;
  - (4) Anti-pattern hợp lệ duy nhất là `L` chỉ xuất hiện trong **log/error path** (rotation ngắn, không persistence), không bao giờ trong DB sink.
- **Đúng (Compliance-correct flow)**: Per-field `MaskStrategy` enum {NONE, DROP, HASH_HMAC(salt), PARTIAL(prefix,suffix), TOKENIZE(vault)} configurable qua mapping_rule → dispatch theo Strategy pattern → `DROP` set NULL (không literal), `HASH_HMAC` dùng HMAC-SHA256 với secret key versioned + rotation, `PARTIAL` format-preserving cho display → audit log `mask_audit_log(event_id, table, field, strategy, key_version)` sample rate + `mask_config_audit(actor, old, new, changed_at)` cho thanh tra → admin UI self-service config + preview.
- **Sai (anti-pattern)**: (a) hardcode `"***"` ở write-path; (b) 1 strategy duy nhất cho mọi loại field (over-mask non-PII hoặc under-mask PII); (c) test assert `"***"` lock anti-pattern; (d) SHA256 thuần không salt → rainbow table; (e) lưu key trong code/git; (f) masking phía sink BI tool trong khi BD đã lưu plaintext.
- **Áp dụng được cho**: bất kỳ hệ thống nào persist PII xuyên zone (CDC, ETL, data warehouse, ML feature store) tại thị trường có luật BVDLCN (VN 91/2025, EU GDPR, Singapore PDPA, Brazil LGPD).
- **Liên quan**: §1+§12 Brain plan-only, ADR pattern (chọn HMAC vs SHA256 dựa rainbow table threat), L-2026-05-23-cross-cutting-concern-single-source-of-truth (masking là cross-cutting, phải centralized qua MaskingService không inline).

## L-2026-05-28-mark-done-without-completeness-guard
- **Date**: 2026-05-28
- **Workspace**: `bug-snapshot-progress-mismatch-2026-05-28`
- **Global Pattern**: Khi tiến trình `P` chuyển trạng thái `state(P) = terminal_success` (vd `status=done`), nếu chỉ dựa trên metric **intermediate-layer** `M_i` (counter tại layer `i` của pipeline) mà không có **invariant guard ở edge** (terminal transition point) so sánh `M_persisted >= M_expected * threshold(τ)`, thì:
  - (1) bug fix ở layer `i` (vd counter từ enqueue → PG RowsAffected) sẽ **trồi sang layer `j ≠ i`** (cursor exhaustion, pause fall-through, partial flush, ...) — **whack-a-mole pattern**;
  - (2) report success layer "lừa" status quan sát ngoài (UI / activity log), vì terminal transition không kiểm tra ground truth `expected_total`;
  - (3) caller path tương lai (refactor, feature mới) bypass intermediate check → invariant tiếp tục vỡ;
  - (4) lesson `Define DoD at destination` (counter từ persistence layer) CẦN nhưng CHƯA ĐỦ — phải bổ sung `DoD completeness` (`actual >= expected * τ`) tại transition edge.
- **Đúng (Defense-in-depth flow)**: (a) Counter `rows_processed` từ destination ground truth (vd PG `RowsAffected`); (b) Capture `expected_total` từ source (vd Mongo `EstimatedDocumentCount`) lưu local var, không chỉ DB write-and-forget; (c) Terminal transition `markDone(actual, expected)` guard: `if expected > 0 && actual < expected * τ → markError(reason="incomplete: actual<expected*τ")` với `τ` configurable (default `0.99` cho phép estimate skew + concurrent write); (d) Pause/cancel paths phải `return` ngay, không fall-through xuống terminal transition; (e) Cursor / stream exhaustion check phải dùng `empty-result` (`len==0`) thay vì `partial-result` (`len < batch_size`) — partial có thể do replication lag / pagination quirk; (f) Prometheus counter `partial_done_total{reason}` cho mỗi guard trip → alert on-call.
- **Sai (anti-pattern)**: (a) `if len(batch) < batchSize { break }` làm điều kiện exhaustion → sai khi source partial-mid-stream; (b) `break` trong loop khi pause/cancel nhưng để code tiếp tục xuống final flush + markDone → ghi đè status; (c) `markDone(actual)` không nhận `expected` → không thể guard; (d) Guard ở mỗi caller thay vì ở terminal function (DRY violation + miss caller mới); (e) Threshold `τ = 1.00` strict → false trip do estimate skew; `τ = 0.95` lỏng → không catch bug 23%; default `0.99` cân bằng.
- **Áp dụng được cho**: snapshot/replica/migration runners (CDC, ETL backfill, Kafka mirror), long-running job với resume capability, batch processors có pause/cancel, bất kỳ workflow nào terminal transition mark-success cần invariant `actual_output ≈ expected_input`.
- **Liên quan**: L-2026-05-26-metric-defined-but-never-set (DoD destination — tiền đề), L-2026-05-27-flush-chain-pass-through (fix layer Flush — chưa đủ), §1+§12 Brain plan-only, §3 Verify Before Done, §6 Simplicity First (1 guard ở edge thay vì N guard ở caller).

## L-2026-05-28-cleanup-is-not-remove
- **Date**: 2026-05-28
- **Workspace**: `cleanup-gpay-cols-2026-05-28`
- **Global Pattern**: Khi user nói "field `B` rác kỹ thuật, đã có field `X` rồi" về 2 field cùng semantic (cùng concept, khác naming convention) trong schema `S`, thì intent thực là **RENAME/MERGE `B → X`** chứ KHÔNG phải **DELETE `B`** + giữ logic. Nếu agent chọn nhánh DELETE (drop column + xóa logic), agent sẽ: (1) phá vỡ ràng buộc/invariant gắn với `B` (vd partial UNIQUE INDEX, ON CONFLICT key, OCC tiebreaker); (2) khiến user phải sửa lại direction giữa session (chi phí cao nhất); (3) đẩy work scope từ "mechanical rename" thành "remove + reconstruct" lớn gấp 3–5 lần. Đây là một biến thể của lesson `anti over-correct` nhưng ngược chiều: thay vì over-engineer, agent **over-defer** thành "3 option remove" trong khi intent là 1 patch rename đơn giản.
- **Đúng (Intent-verify flow)**: (a) Trước khi pick scope, **semantic mapping** từng cặp field nghi ngờ trùng: cùng `(data_type, nullability, default, role)` ⇒ cùng concept ⇒ rename target. (b) Pick nhánh tối thiểu: `ALTER TABLE RENAME COLUMN B TO X` idempotent (skip nếu `X` đã tồn tại → DROP duplicate). (c) Code edit theo cùng nguyên tắc: replace identifier, KHÔNG xóa logic, KHÔNG xóa index/constraint trừ tên ràng buộc cũ refer `B`. (d) Migration SQL idempotent + reversible (kèm `reverse.sql`). (e) Verify destination: `\d <table>` + grep zero-residue cả 2 tên cũ.
- **Sai (anti-pattern)**: (a) Hiểu "rác" = "remove" mặc định, không verify với user; (b) Present 3 option (conservative/mid/full) khi user intent là 1 patch rename — bắt user lựa chọn thay vì agent chốt; (c) Drop column + xóa logic invariant đi kèm (UNIQUE INDEX, ON CONFLICT key) → break OCC older-wins guard; (d) Edit nguồn trước khi viết migration SQL → DB drift với code; (e) Quên ops script (`smoke_*.sh`) khi grep — residue lọt qua zero-residue check.
- **Áp dụng được cho**: cleanup PK/identifier duplicate ở DB schema cross-service, refactor API field naming (V1 `legacy_id` ↔ V2 `id`), config key consolidation, log field naming standardization, ML feature store column dedup — bất kỳ tình huống nào 2 identifier cùng concept khác naming convention cần unify.
- **Liên quan**: L-2026-05-20-anti-over-correct (cùng họ, ngược chiều over-defer), L-2026-05-20-verify-at-destination (verify zero-residue ở mọi consumer), L-2026-05-26-metric-defined-but-never-set (DoD destination — grep + `\d` ở DB), §3 Plan Verify Before Done, §6 Simplicity First (rename < remove + reconstruct), §11 APPEND-only progress, §12 Brain Code Prohibition (Muscle execute rename, Brain document).

## L-2026-05-28-rename-blind-creates-duplicate
- **Date**: 2026-05-28
- **Workspace**: `cleanup-gpay-cols-2026-05-28`
- **Mid-session correction (§5 GEMINI)**: User dừng giữa session: "rename thì mày cũng phải xem logic của flow đó đã có chưa, có thì ko cần rename mà BỎ".
- **Global Pattern**: Khi cleanup field `B` trùng concept với `X` ở schema `S` đa-path (multiple CREATE/SELECT/INSERT sites), agent KHÔNG được rename blanket `B → X`. Phải audit từng site `s_i`: nếu site đã có `X` riêng (independent reference) → **DROP `B`** ở site đó (vì `X` đã đảm nhiệm concept); nếu site chỉ có `B`, chưa có `X` → **RENAME `B → X`**. Blind rename ở site đã có cả 2 sẽ tạo: (1) Go struct slice duplicate entry (no compile error nhưng smell + runtime no-op repeated ALTER); (2) SQL CREATE TABLE column lặp (runtime error 42701); (3) record map double-set cùng concept (last-wins overrides intent); (4) ON CONFLICT key ambiguity nếu cả 2 cùng UNIQUE.
- **Đúng (Per-site audit flow)**: (a) Build inventory pre-rename: cho mỗi file/site, list cột/field hiện có. (b) Cho mỗi site `s_i`, phân loại 2 case `BOTH_PRESENT` (DROP B) hoặc `ONLY_LEGACY` (RENAME B→X). (c) Audit dual-presence dấu hiệu: cùng `cols slice`, cùng `record map`, cùng SQL `INSERT INTO ... (...)`. (d) Apply edit theo case-specific verb (DROP vs RENAME). (e) Post-edit grep + count: zero residue `B` + zero duplicate `X` trong cùng site. (f) Test cả SQL execution (không chỉ build) vì duplicate column lỗi runtime, không compile-time.
- **Sai (anti-pattern)**: (a) `sed -i 's/B/X/g'` blanket toàn codebase; (b) `Edit(replace_all=true)` không phân loại site; (c) chỉ verify "build pass + test pass" mà không grep duplicate column trong cùng CREATE/INSERT statement; (d) chỉ đếm `B` residue, quên đếm `X` lặp 2 lần; (e) áp dụng L-2026-05-28-cleanup-is-not-remove ("cleanup = rename không phải remove") đến mức nuốt luôn trường hợp legitimate REMOVE — cleanup là **mixture của RENAME (target chưa có) + REMOVE (target đã có)**, không phải pure RENAME.
- **Phụ thuộc lesson**: L-2026-05-28-cleanup-is-not-remove dạy "cleanup ≠ pure remove"; lesson này thêm "cleanup ≠ pure rename" — đúng action là per-site case analysis.
- **Áp dụng được cho**: DB column dedup multi-path (CDC shadow ↔ master ↔ DW), API field versioning multi-endpoint, config key consolidation multi-service, log field schema harmonization, ML feature store column unification — bất kỳ refactor naming nào trên codebase có nhiều CREATE/SELECT/INSERT site.
- **Liên quan**: L-2026-05-28-cleanup-is-not-remove (sister lesson — phải đọc cặp), L-2026-05-20-anti-over-correct (ban đầu over-defer, sau over-correct → rename blanket), L-2026-05-20-verify-at-destination (verify zero-residue ở DB schema runtime, không chỉ code-level), §3 Verify Before Done (test runtime SQL, không chỉ build), §6 Simplicity First (per-site audit < blanket replace_all).

---

## L-2026-05-28-enumerate-all-inferrers-before-fix
**Date**: 2026-05-28
**Trigger**: User báo bug "shadow column = int8 nhưng UI mapping = TEXT approved" tại `/shadow/19/mappings` field `id` (UUID Mongo).

**Global Pattern**: Khi system [A] có **nhiều entry-point [B1, B2, …Bn]** cùng quyết định một property [X] cho entity [Y] (vd: nhiều inferrer cùng decide PG column type cho field source), một bug ở property [X] sai value KHÔNG bao giờ chỉ nằm ở **một** entry-point. Phải **enumerate toàn bộ variant [B*]** trước khi fix, không thì sẽ patch một path rồi xây compensating mechanism (repair endpoint, migration) cho hậu quả của các path khác — vừa rườm rà vừa không trị tận gốc.

**Đúng (enumerate-before-fix)**:
1. Grep literal/constant đặc trưng của decision sai (vd `"BIGINT"` literal, `default BIGINT`) trên toàn bộ codebase BE.
2. Liệt kê callers/sites — phân loại theo input domain (vd JSON sample vs BSON sample vs MySQL information_schema).
3. Tìm policy chung [Z] resolve mọi variant cùng kiểu (vd "ưu tiên TEXT khi có string xuất hiện trong sample" cho schemaless source).
4. Apply policy [Z] đến mọi variant. Helper chung dùng lại.
5. Verify build + grep lại đảm bảo no remaining literal mặc định sai.

**Sai (anti-pattern em vừa mắc)**:
1. Fix 1 path đầu tiên gặp (CREATE TABLE pkType default).
2. Báo "đã xong" + propose user thao tác per-table thủ công (Sync Fields từng bảng).
3. Khi user chỉ "hàng ngàn table" → vội xây endpoint bulk repair + worker handler sweep tất cả binding + FE button (~150 LOC dư thừa).
4. User yêu cầu "tìm tới root cause đừng rườm rà" — lúc đó mới enumerate đủ và phát hiện 2 entry-point còn lại (`inferMongoCols` sample 1 doc, `processDiscoveryRows` set theo row đầu).

**Áp dụng được cho**:
- Type inference multi-source (BSON/JSON/SQL information_schema/Avro/Protobuf).
- Default-value policy multi-handler (vd nhiều migration generator cùng decide nullable/default).
- Validation rule multi-API (input validate ở gateway vs service vs DB constraint).
- Cache invalidation multi-trigger (write path A, write path B, batch job C — phải nhất quán policy).
- Feature flag evaluation multi-runtime (BE Go vs FE TS vs mobile SDK — cùng feature, 3 inferrer).

**Heuristic phát hiện sớm**: Nếu fix của em dạng "patch 1 chỗ + xây repair tool cho data cũ" → dừng lại, grep literal mặc định sai trên toàn codebase trước. Repair tool chỉ build khi root cause đã được trị TẤT CẢ entry-point.

**Liên quan**:
- §3 Plan Node Default — re-plan khi user feedback chỉ ra over-scoping.
- §6 Simplicity First — repair endpoint cho hàng ngàn table = mất công lớn vì chưa enumerate đủ.
- §6 Demand Elegance — policy chung [Z] (resolveMongoSampledType) elegant hơn 3 fix riêng biệt.
- §8 Escalation — user feedback "đi làm tùm lum" = signal phải enumerate trước.

**File evidence**:
- `centralized-data-service/internal/handler/provisioning_step_handlers.go:639` `inferMongoCols` (path 1).
- `centralized-data-service/internal/handler/command_handler.go:485` `processDiscoveryRows` (path 2).
- `centralized-data-service/internal/handler/command_handler.go:608-614` CREATE TABLE pkType default (path 3).
- Policy chung: `resolveMongoSampledType` (provisioning_step_handlers.go).

## Lesson 2026-05-28 — Log Spam Without Operator Value IS a Log Bug (Don't Argue "Design Intent")

**Global Pattern**:
- **Sai**: Khi operator [B] báo log [X] bắn nhiều/không value, agent [A] biện hộ "log level INFO không phải ERROR nên không vi phạm startup-clean (#820), đây là expected design intent" → dismiss valid pain → mất uy tín.
- **Đúng**: Log không actionable ở volume/frequency hiện tại = **bug của log**, độc lập với level. Tiêu chí "bug" của observability KHÔNG phải severity, mà là signal-to-noise + actionability cho operator.

**Trigger thực tế (2026-05-28)**:
- User báo: 33 dòng INFO "dlq state machine replayed message" trong 103ms khi `cdc-worker` start.
- Em audit kết luận "Không phải bug, expected catch-up behavior".
- User phản hồi: "log bắn tùm lum mà ko mang lại giá trị nó là bug của log. cãi cãi cái gì".

**Quy tắc kế thừa (bổ sung cho #820)**:
1. **Per-message INFO trong loop batch** (poll N rows → 1 INFO/row) → mặc định nên là **Debug**. INFO chỉ dành cho aggregate/cycle summary.
2. **Mỗi cycle batch job phát 1 INFO** với counters: `polled`, `<success>`, `<failure_subtypes>`, `skipped`. Operator chỉ cần 1 dòng để biết cycle khoẻ hay không.
3. Nếu cycle có `polled=0` → KHÔNG log INFO (silent OK).
4. WARN/ERROR vẫn 1/event vì đó là exception path cần truy vết riêng.

**Lesson kèm — SigNoz/OTel-zap bridge body=msg pattern (cùng phiên)**:
- User báo SigNoz UI chỉ render "title" (= log body = zap `msg` string), fields phải click detail mới thấy → phiền.
- Nguyên nhân: `otelzap.NewCore` map zap `msg` → OTel log body; zap fields → OTel attributes. SigNoz default chỉ hiện cột body.
- **Quy tắc**: Khi backend log là OTel/SigNoz, msg string phải **tự mô tả**: nhét key context inline (`fmt.Sprintf("dlq replayed id=%d subject=%s retry=%d", ...)`) **kèm theo** zap.Field (để vẫn query attribute được).
- **Anti-pattern**: `logger.Info("processing done", zap.String("resource", x))` → SigNoz hiện "processing done" trống rỗng.
- **Đúng**: `logger.Info(fmt.Sprintf("processing done resource=%s", x), zap.String("resource", x))` → SigNoz hiện đủ context.

**Áp dụng được cho**:
- Mọi batch/cron job có log per-item (consumer poll, scheduler tick, replay worker, archive sweep).
- Mọi service dùng OTel log bridge (Datadog, NewRelic, Honeycomb, SigNoz, Tempo) với UI default chỉ render body.
- Code review checklist: thấy `for ... { logger.Info(...) }` → ép xuống Debug + thêm aggregate.
- Audit observability: thấy log level đúng mà operator vẫn complain → check msg self-descriptive chưa.

**Heuristic**:
- Đếm ratio `INFO_per_minute / actionable_signal_per_minute`. Nếu > 10 → log hygiene bug.
- Mỗi log msg đứng riêng đọc ra có hiểu chuyện gì không? Nếu phải xem field mới hiểu → msg cần inline context.

**Liên quan**:
- Lesson #820 (startup log clean): chỉ nói về ERROR/WARN, BỔ SUNG: INFO spam = bug.
- Lesson #866 (symptom vs upstream): "design intent" không phải lá chắn miễn trừ — operator pain là 1 upstream hợp lệ.
- §6 Simplicity First & Demand Elegance: 1 INFO/cycle elegant hơn 100 INFO/cycle.
- §0 "Khi trả lời 1 vấn đề, luôn làm planning trước": planning audit phải hỏi "log này operator có dùng được không" chứ không chỉ "log có đúng level không".

**File evidence**:
- `centralized-data-service/internal/handler/dlq_state_machine.go:150-154` (trước): per-msg INFO trong loop.
- `centralized-data-service/internal/handler/dlq_state_machine.go` (sau fix): per-msg Debug + 1 aggregate INFO/cycle với fmt.Sprintf inline counters.
- `centralized-data-service/cmd/worker/main.go:42-91`: zap.NewProduction + otelzap bridge — context vì sao msg=body cần self-descriptive.
- Report: `data-hub/report_dlq_startup_log_spam.md`.

## L-2026-05-29-enumerate-includes-upstream-config-payload
- **Date**: 2026-05-29
- **Trigger**: Sau khi fix L-2026-05-28 (3 type-inferrer paths), user test lại vẫn thấy bug — `id` UI hiển thị TEXT nhưng tạo shadow column vẫn ra `BIGINT` (int8). User chửi giữa session (§5 mid-session correction).
- **Global Pattern (mở rộng L-2026-05-28)**: Khi enumerate entry-points của một decision [X], **PHẢI bao gồm cả config/registry/payload upstream** truyền giá trị đó xuống. Pattern "enumerate inline inferrers" chỉ trị các path tự quyết định tại chỗ; còn path **nhận từ upstream** (CMS config → worker payload, DB seed → runtime) cũng là entry-point hợp lệ và thường bị bỏ qua vì nó không phải `if/else` rõ rệt — nó là một field gán giá trị trực tiếp.
- **Trường hợp cụ thể**:
  - `cdc_system.source_object_registry.primary_key_type` (seed BIGINT mặc định legacy) → `CreateDefaultColumnsCommand.PrimaryKeyType` → `payload.PKType` → worker `pkType := payload.PKType` (line 706 trước fix).
  - Logic Mongo TEXT-first đặt ở fallback `if pkType == ""` → không bao giờ chạy khi registry seed BIGINT.
- **Đúng (fix)**: Worker enforce TEXT khi `isMongoPK` bất kể payload.PKType:
  ```go
  pkType := strings.TrimSpace(payload.PKType)
  if isMongoPK { pkType = "TEXT" }       // override upstream
  else if pkType == "" { pkType = "BIGINT" }
  ```
  Mongo PK luôn là string (ObjectId hex / UUID hex) → physically incompatible với BIGINT → enforcement an toàn.
- **Sai (anti-pattern mở rộng)**: Khi enumerate inferrers, chỉ tìm `default BIGINT` literal trong các nhánh `if/else`/`switch`. Bỏ qua sites mà giá trị đến từ **field assignment** từ upstream (struct field, JSON payload, config) — vì nó không "trông giống" inferrer.
- **Heuristic phát hiện**: Khi grep entry-points cho decision [X], hỏi thêm: "[X] có thể được set từ struct/field/payload upstream nào? Trace ngược từ assignment đó."
- **Áp dụng được cho**:
  - DDL type decision multi-stage (registry → command → worker).
  - Feature flag override (config → env → request payload).
  - Tenant routing key (URL → header → JWT claim → registry).
  - Quota policy (default → org-level → user-level override).
  - Timeout/retry settings (yaml → env → runtime override).
- **Quy tắc kèm — Anti-feature-distraction**: Khi user báo bug critical, KHÔNG triển khai feature mới (kể cả feature đã được approve trong scope cũ) cho tới khi bug critical được verify fix. Pattern: bug-critical-takes-absolute-priority. Vi phạm phổ biến: agent vừa fix bug vừa song song implement feature khác trong cùng phiên → user mất kiên nhẫn vì agent "vừa làm vừa phá".
- **Liên quan**:
  - L-2026-05-28-enumerate-all-inferrers-before-fix (sister — lesson này MỞ RỘNG nó để cover upstream payload path).
  - §3 Verify Before Done — verify thực sự khi user test, không chỉ build pass.
  - §6 Simplicity First — 1 dòng override tại worker < migration registry mass-update.
  - §8 Escalation — user chửi = signal escalation, phải dừng feature work ngay.
- **File evidence**:
  - `centralized-data-service/internal/handler/command_handler.go:706-712` (trước fix): `pkType := payload.PKType; if pkType == "" { ... }` — Mongo default chỉ kick in khi empty.
  - `centralized-data-service/internal/handler/command_handler.go:706-718` (sau fix): `if isMongoPK { pkType = "TEXT" }` — override upstream payload.
  - `cdc-cms-service/internal/infra/persistence/source_object_v2_sync.go:139` (sibling root): registry INSERT lưu primary_key_type → upstream gốc.

## L-2026-05-29-system-default-fields-truth-source-sinkworker
- **Date**: 2026-05-29
- **Trigger**: User mid-session correction: "từ lúc nào field ID nó vào System Default Fields". Em đã render PK source (id) như system default — sai schema spec.
- **Global Pattern**: Khi system [A] có nhiều code path tạo schema [S] cho entity [Y] (vd nhiều worker/handler cùng CREATE TABLE shadow), **truth source của "system default" KHÔNG phải path đầu tiên em đọc**, mà là path **runtime thực sự apply liên tục** (sinkworker / sink-side / write-through). Path one-shot bootstrap (vd `HandleCreateDefaultColumns` legacy) có thể divergence — đừng coi nó là spec.
- **Cụ thể**: Shadow table có 2 đường tạo:
  - Path A (legacy `HandleCreateDefaultColumns`, user-triggered "Sync Fields to Shadow"): PK = source PK (id BIGINT/TEXT). 10 CDC system cols.
  - Path B (runtime `sinkworker schema_manager.createShadowTable`): PK = `_gpay_id` BIGINT (sonyflake internal). 11 system cols (10 CDC + `_gpay_id`). Đây là **runtime truth** — mọi message Debezium đi qua sinkworker.
  - Path B là spec, Path A là legacy bootstrap. Khi hiển thị "System Default Fields" cho operator, phải dùng Path B.
- **Đúng (truth-source identification flow)**:
  1. Liệt kê mọi path tạo schema cho entity Y.
  2. Phân loại: bootstrap (one-shot, user-initiated) vs runtime (continuous, message-driven). Runtime path = source of truth.
  3. Validate runtime path schema = UI display schema.
  4. Bootstrap path nếu divergence → mark legacy, không dùng làm display reference.
- **Sai (anti-pattern em vừa mắc)**: Đọc `ensureCDCColumnsInSchema` (10 cols) trong `HandleCreateDefaultColumns`, đếm thiếu 1, đoán PK source là cột thứ 11 → render `id` vào System Default Fields card.
- **Heuristic phát hiện**: Nếu count system cols giữa 2 path khác nhau → KHÔNG ráng "balance" bằng cách thêm field từ path khác. Trace runtime truth path trước.
- **Áp dụng được cho**:
  - Schema metadata UI (mọi system nào có bootstrap + runtime DDL parallel).
  - Default config display (config seeded vs config runtime-set).
  - Permission display (default role vs runtime-grant).
  - Cron schedule UI (declared vs actual runtime schedule).
  - Resource quota UI (admin-set vs runtime-enforced).
- **Liên quan**:
  - L-2026-05-28-enumerate-all-inferrers-before-fix (cùng họ — multi-path divergence).
  - L-2026-05-29-enumerate-includes-upstream-config-payload (cùng phiên — divergence từ upstream config).
  - §6 Simplicity First — static 11-entry array < dynamic builder(pkField) gây confusion.
  - §3 Verify Before Done — runtime DDL spec phải verify bằng `\d <shadow_table>` thật, không bằng đọc bootstrap handler.
- **File evidence**:
  - `centralized-data-service/internal/sinkworker/schema_manager.go:225-237` — runtime truth (11 cols incl `_gpay_id`).
  - `centralized-data-service/internal/sinkworker/sinkworker.go:253-271` — `shouldSkipBusinessKey` confirm system fields set.
  - `centralized-data-service/internal/handler/command_handler.go:168-179, 738-756` — legacy bootstrap (10 CDC + source PK) — KHÔNG dùng làm display reference.
  - Fix: `cdc-cms-web/src/pages/MappingFieldsPage.tsx` — static `SYSTEM_DEFAULT_FIELDS` 11 entries với `_gpay_id` thay vì PK source dynamic.

## L-2026-05-29-log-tech-depth-component-op-duration-errtype
- **Title**: Log message body PHẢI có technical anchors (component/op/phase/duration_ms/err_type) — "notification-style" log không debuggable
- **Date**: 2026-05-29
- **Trigger**: User mid-session correction: "log nó phải có hướng tech chứ. để còn biết mà debug. kiểu thông báo thôi vậy". Em đã apply inline-msg pattern (id/subject/retry trong body) nhưng thiếu technical direction: không component tag, không operation name, không timing, không error taxonomy. Log đọc như "notification" → không filter/sort/correlate được.
- **Global Pattern**: Khi system [A] log event [E] qua bridge `msg → body, fields → attributes` (vd OTel zap → SigNoz / Loki / Elastic), body PHẢI chứa **5 technical anchors** để operator debug **không cần mở detail panel**:
  1. `component=<service_module>` — nguồn log (worker_server, kafka_consumer, dlq_state_machine). Filter: "tất cả log từ kafka consumer".
  2. `op=<operation_name>` — verb cụ thể (pg_init, fetch_message, retry, send_to_dlq, reconcile_cycle). Filter: "mọi op=retry".
  3. `phase=<lifecycle>` — vị trí trong flow (started/completed/skipped/transient/fatal/paused). Phân biệt cùng op khác phase.
  4. `<X>_duration_ms=<int>` — timing cho mọi op timed (init_duration_ms, fetch_duration_ms, cycle_duration_ms, db_duration_ms, publish_duration_ms). Sort by latency, alert on p99.
  5. `err_type=<taxonomy>` — classify error qua helper `classifyXErr(err)` map về tập đóng (ctx_deadline_exceeded, ctx_canceled, nats_timeout, pg_sqlstate, net_conn_refused, kafka_not_leader, schema_not_found, timeout, unknown). Filter `err_type=nats_timeout` thay vì grep free text.
  - Thêm resource counters tuỳ op: payload_bytes, partition, offset, reader_lag, topic_count, broker_count, run_count, throughput_msg_per_sec, batch_size, retry, attempt.
  - Anchors phải có trong `fmt.Sprintf(msg)` **và** zap.Field (giữ attribute query). Body = msg-with-anchors, attributes = same key/values type-safe.
- **Đúng** (tech-depth log line — operator scan body & biết ngay):
  ```
  kafka fetch transient error retrying component=kafka_consumer op=fetch_message phase=transient fetch_duration_ms=187 reader_lag=4231 err_type=kafka_request_timeout err=...
  ```
- **Sai (notification-style — em vừa mắc)**:
  ```
  kafka fetch transient error retrying err=context deadline exceeded
  ```
  → không biết component, không op, không timing, không err classification. SigNoz column "Body" chỉ thấy free text.
- **Implementation template** (Go + zap):
  ```go
  start := time.Now()
  err := doX(ctx, resource)
  if err != nil {
      errType := classifyXErr(err)
      logger.Warn(fmt.Sprintf("X failed component=%s op=%s phase=%s duration_ms=%d resource_id=%s err_type=%s err=%s",
          component, op, phase, time.Since(start).Milliseconds(), resourceID, errType, err.Error()),
          zap.String("component", component),
          zap.String("op", op),
          zap.String("phase", phase),
          zap.Int64("duration_ms", time.Since(start).Milliseconds()),
          zap.String("resource_id", resourceID),
          zap.String("err_type", errType),
          zap.Error(err))
  }
  ```
- **Error taxonomy helper** (per-domain, share keys khi cross-domain):
  - Add `classifyXErr(err error) string` next to existing transient check.
  - Map về **tập đóng** (closed set), include `unknown` cho fallback.
  - Cross-domain (Kafka + DLQ + DB) share keys: `ctx_deadline_exceeded`, `ctx_canceled`, `net_conn_refused`, `net_conn_reset`, `timeout`, `unknown`.
  - Domain-specific keys: `nats_timeout`/`nats_conn_closed`, `kafka_not_leader`/`kafka_request_timeout`/`kafka_rebalance`, `pg_sqlstate_XXXXX`/`pg_duplicate`, `schema_not_found`.
- **Áp dụng được cho**:
  - Mọi service xài OTel zap bridge → SigNoz/Datadog/Honeycomb (body=msg).
  - JSON log aggregator (Loki/Elastic/CloudWatch) khi UI default render top-level msg field.
  - Microservice startup logs: thêm `init_duration_ms` cho mỗi resource (pg, redis, kafka, nats, mongo).
  - Cron/scheduler logs: thêm `phase=started|completed|skipped` + `cycle_duration_ms`.
  - HTTP/gRPC handler logs: thêm `endpoint`, `method`, `latency_ms`, `status_code`, `err_type`.
  - Background batch worker logs: thêm `batch_size`, `processed`, `success`, `failed`, `throughput_per_sec`.
- **Heuristic phát hiện log thiếu tech depth**:
  - Hỏi: "Tôi grep log này với `err_type=X` được không?" — không → thiếu taxonomy.
  - Hỏi: "Tôi sort log này theo latency được không?" — không → thiếu duration_ms.
  - Hỏi: "Tôi filter chỉ log từ kafka consumer được không?" — không → thiếu component tag.
  - Hỏi: "Operator đọc 1 dòng body biết error class và resource bị ảnh hưởng?" — không → notification-style, chưa đủ.
- **Liên quan**:
  - Sub-lesson của Log Spam IS Log Bug (2026-05-28) + SigNoz body=msg pattern.
  - §6 Demand Elegance — body inlined 5 anchors thay vì 50 attribute fields rời rạc.
  - §3 Verify Before Done — verify bằng cách read body line + tự hỏi "có debug được không?".
- **File evidence**:
  - `centralized-data-service/internal/handler/kafka_consumer.go:1280-1320` — `classifyKafkaErr` helper, closed-set taxonomy.
  - `centralized-data-service/internal/handler/kafka_consumer.go:468-527` — fetch loop với fetch_duration_ms + reader_lag + err_type + phase=transient|fatal|reader_closed.
  - `centralized-data-service/internal/handler/dlq_state_machine.go` — classifyDLQErr + retry với db_duration_ms/publish_duration_ms/payload_bytes/subject_source/backoff.
  - `centralized-data-service/internal/server/worker_server.go:79-88` — PostgreSQL connect với init_duration_ms.
  - `centralized-data-service/internal/server/worker_server.go:1003-1052` — reconcile_cycle với phase + cycle_duration_ms + drift_detected counters.


---

## L-2026-05-29-three-shadow-bootstrap-paths-must-align — Mọi DDL bootstrap path PHẢI cùng trỏ về 1 truth source

- **Ngày**: 2026-05-29
- **Workspace**: bug-mapping-rescan-status-reset-2026-05-29
- **Trigger**: User test tạo shadow table mới qua CMS → cột vẫn là `id` + thiếu `_gpay_id` PK (ban đầu render `◯ pending` ở FE). Patch FE không giải quyết; patch chỉ sinkworker không giải quyết. Có 3 path khác nhau đang CREATE/ALTER shadow table với spec lệch nhau.
- **Root cause**:
  - Codebase có ≥ 3 entrypoint tạo/normalize shadow table, mỗi nơi giữ một bản DDL riêng:
    1. `centralized-data-service/internal/sinkworker/schema_manager.go` (runtime, chạy per-event — TRUTH SOURCE).
    2. `centralized-data-service/internal/handler/command_handler.go` (NATS bootstrap qua HandleCreateDefaultColumns + ensureCDCColumnsInSchema).
    3. `cdc-cms-service/internal/infra/persistence/shadow_automator.go` (CMS-triggered sync bootstrap).
  - Khi sinkworker đã chuẩn hóa column thành `_gpay_id BIGINT PRIMARY KEY` + `_source_id TEXT NOT NULL` + partial UNIQUE INDEX, hai path còn lại vẫn tạo `id` / `source_id`. Hệ quả: lần đầu admin bind 1 shadow mới → DDL sinh từ path #2 hoặc #3 (lệch) → khi event tới sinkworker thì cột `_gpay_id` không tồn tại → upsert vỡ.
  - Đồng thời `source_id` (column shadow/master) trùng namespace với business field source — không an toàn dài hạn.
- **Global Pattern [A creates resource X through multiple paths P1..Pn] → Nếu spec X không centralize, mỗi Pi sẽ drift → bug "X ở chỗ này khác X ở chỗ kia"**. Đúng: chọn 1 path Pi là truth source, các Pi còn lại PHẢI hoặc (a) delegate sang Pi truth, hoặc (b) tự kiểm trùng spec qua hằng số chia sẻ / test snapshot. Không bao giờ duplicate DDL literal ở nhiều entrypoint.
- **Quy tắc rút ra (general)**:
  - Khi bug "tạo X ra sai shape", grep TẤT CẢ entrypoint tạo X (CREATE TABLE / DDL / schema builder) trước khi patch — partial fix sẽ tái xuất qua entrypoint còn lại.
  - Khi rename column hệ thống (system-default), prefix `_` để cô lập khỏi namespace business field — tránh va chạm khi source data có cùng tên (ví dụ `source_id` của event vs `_source_id` của shadow).
  - Khi user nói "sắp release, anh sẽ xoá DB" → KHÔNG đề xuất dual-read/backward-compat shim, chỉ làm sạch — hỏi backward-compat là lãng phí prompt cycle.
- **Test hồi quy**:
  - Có thể thêm 1 contract test so sánh 3 DDL path: tạo table qua mỗi path → introspect `information_schema.columns` → assert giống nhau. (TODO sau release.)
- **Kiểm tra trên 3 dự án khác**:
  - Áp dụng cho mọi service có pattern "bootstrap qua nhiều entrypoint" (user-onboarding init, migration runner, runtime auto-create, admin tool create). Tất cả PHẢI delegate về 1 schema source.
- **File evidence**:
  - Truth source: `centralized-data-service/internal/sinkworker/schema_manager.go:225-269` — 11 cột CDC + partial UNIQUE INDEX trên `_source_id WHERE NOT _deleted`.
  - Path #2 đã align: `centralized-data-service/internal/handler/command_handler.go:149-213,689-710`.
  - Path #3 đã align: `cdc-cms-service/internal/infra/persistence/shadow_automator.go:75-104` (thêm `_source_ts BIGINT` thiếu trước đó + partial UNIQUE INDEX + sonyflake trigger trên `_gpay_id`).
  - Master cùng pattern: `centralized-data-service/internal/service/master_ddl_generator.go:89,100,136,148` + `transmuter.go:87,328,335,362,449,456`.
  - FE alignment: `cdc-cms-web/src/pages/MappingFieldsPage.tsx:53` (SYSTEM_DEFAULT_FIELDS).
  - Soft-delete path: `centralized-data-service/internal/handler/event_handler.go:233-249` — single V2 statement `ON CONFLICT (_source_id)`.
- **Build verify**: `go build ./...` xanh cho cả centralized-data-service và cdc-cms-service; `tsc -b` xanh cho cdc-cms-web.
- **Liên quan**:
  - §6 Simplicity First — không workaround từng path, đi tìm root cause "có bao nhiêu path tạo X".
  - §3 Verify Before Done — verify cả 3 path bằng grep `CREATE TABLE`/`ALTER TABLE` shadow trước khi báo xong.
  - §0 Quy tắc chính — không đề xuất phương án backward-compat khi user đã chốt wipe DB.

## L-2026-05-29 — scope-param-must-end-to-end-dispatch-not-stop-at-handler
- **Trigger**: snapshot-v2 multi-binding bug — `POST /source-objects/:id/snapshot-v2?binding_id=B2` 202 OK nhưng worker silent default sang binding B1 (DDL pending) thay vì B2 (DDL ready).
- **Root cause**: `binding_id` được dùng để resolve dispatch scope ở handler **rồi bị drop**. Command struct + worker payload + registry route fan-out không có khái niệm binding → worker `ResolveSourceRoutes(srcDB,srcColl)` trả master+clone, ghi vào cả 2 bảng. Khi B1 fail (DDL pending) → circuit-breaker trip → cả 2 binding stuck.
- **Global Pattern [HTTP scope param X dispatch xuống worker Y]**: Nếu X chỉ đi đến edge handler để resolve "scope của lần gọi này" (e.g. 1 trong N candidate row khớp source key) thì PHẢI plumb tiếp X sang [Command field → Wire payload → Worker filter → Storage key]. Bất kỳ chặng nào drop X → worker fan-out theo source key mặc định → silent route vào candidate khác.
  - **Sai**: parse X ở handler, dispatch command chỉ chứa source key (parent id). Worker không biết X → resolveByParentID → race/silent fallback.
  - **Đúng**: handler parse X → set vào Command struct → publish bus serialize → worker payload có X → worker filter route theo X (ctx scope, route filter, hoặc cả hai) → dedup row in storage (snapshot_progress) include X trong unique key. Khi worker reload route mà thiếu X → fail-loud (markProgressError) thay vì fallback.
- **Apply (snapshot_v2_multi_binding)**:
  - `cdc-cms-service/internal/app/commands/recon_async.go` — `SnapshotV2Command.ShadowBindingID int64`.
  - `cdc-cms-service/internal/api/source_object_actions_handler.go:552-615` — `SnapshotV2`: `parseBindingIDQuery` → `resolveDispatchScope` → validate `scope.SourceObjectID == id` (400 mismatch) → 409 nếu ambiguous → set `cmd.ShadowBindingID`.
  - `centralized-data-service/internal/handler/snapshot_runner_handler.go` — `snapshotV2Payload.ShadowBindingID`; `runSnapshot` load `shadow_binding`, override `targetTable`, filter `ResolveSourceRoutes` keeping route với `ShadowBinding.ID == p.ShadowBindingID` (fail-loud nếu rỗng), `ctx = WithBindingScope(ctx, sb.ID)` để `eventHandler.HandleRaw` áp filter.
  - `centralized-data-service/internal/handler/event_handler.go` — `WithBindingScope` / `bindingScopeFromCtx` context-key, `processEvent` filter routes khi scope > 0; CDC consumer KHÔNG set scope → fan-out master+clone giữ nguyên.
  - `cdc-cms-service/migrations/schema/core/066_add_shadow_binding_id_to_snapshot_progress.sql` — `snapshot_progress.shadow_binding_id BIGINT`, index `(source_object_id, shadow_binding_id, status, started_at DESC)`, `claimProgress` dùng `IS NOT DISTINCT FROM` để NULL vs id là 2 dedup group.
- **Áp dụng được cho 3 dự án khác**: ✅
  1. Multi-tenant query API → `?tenant_id=T` phải đi Handler→Command→Worker→QueryFilter→DB index, KHÔNG dừng ở middleware.
  2. Multi-region webhook dispatch → `region_id` phải đi từ HTTP query xuống worker payload + retry log dedup key.
  3. Multi-version pipeline (model version) → `version_id` phải đi từ HTTP→Job→Worker→Cache key, không suy ra từ source_id ở worker.
- **Liên quan**:
  - §3 Plan & Verify — verify "param X có rò ở chặng nào không" trước khi báo done.
  - §6 Demand Elegance — context-key + helper (WithBindingScope) elegant hơn pass `bindingID int64` qua mọi public method của EventHandler.
  - L-2026-05-29-log-tech-depth-component-op-duration-errtype — log scope-resolved phase phải có `component=snapshot_runner op=run_snapshot phase=scope_resolved shadow_binding_id=N target_table=X` để debug SigNoz.

---

## L-2026-05-29-multi-binding-cache-key-scalar-overwrite-silent-corruption

- **Global Pattern**: `[A keyed B by scalar key K when entity X has multiple bindings to K]` → `[Result Y: every loop iteration overwrites prior, last-write-wins; downstream lookups by K return only the final binding, silently dropping the rest]`.
- **Triệu chứng (B = registry cache, K = source_object_id, X = shadow_binding)**:
  - `metadata_registry_service.go` có 2 cache song song:
    - `routeCache map[string][]*ResolvedSourceRoute` — slice per `sourceKey` (đúng).
    - `routeBySourceID map[int64]*ResolvedSourceRoute` — **SCALAR per source_id** (sai khi 1 source có ≥2 binding).
  - Loop fan-out `for binding := range allBindings { routeBySourceID[src.ID] = route }` → overwrite, chỉ giữ binding cuối.
  - Hậu quả downstream: `mapping_cache` attach chỉ vào target_table của binding cuối → snapshot v2 cho binding đầu chạy "thành công" về schema nhưng tất cả field mapping NULL → silent corruption không log error.
- **Bug surface song song (A = FE route, K = source_object_id)**:
  - Route `/shadow/:id/mappings` của FE (TableRegistry → MappingFieldsPage) chỉ truyền `source_object_id`. Click 2 row khác binding của cùng source → cùng URL → cùng resolve về binding default (active first) → FE render cùng 1 set rule cho mọi binding.
- **Đúng (correct flow)**:
  - **Cache layer**: khi `entity X` có quan hệ N-1 với key K → cache phải dùng `map[K][]*X` (slice) hoặc đổi sang composite key `(K, X.ID)`. Tất cả loop downstream phải iterate slice.
  - **Route layer**: route URL phải mang ĐỦ defining key của instance (source_id + binding_id), không default ở backend.
  - **Repo SQL layer**: lookup pin bằng `sb.id = ?` thay vì `sb.shadow_table = tr.target_table` (vì shadow_table có thể giống nhau giữa các binding).
  - **Error taxonomy**: khi pre-flight reload xong mà cache vẫn miss → DB cross-check để phân biệt `binding_inactive` (DB nói false, reload đã filter đúng) vs `registry_reload_silent_drop` (DB nói active, cache miss = race/SQL drift) → log `err_type=<X>` thay vì chung chung.
- **Files**:
  - `centralized-data-service/internal/service/metadata_registry_service.go:163,223,245,262` — scalar→slice cho `routeBySourceID`, B3 clone loop, mapping_cache attach loop.
  - `centralized-data-service/internal/handler/snapshot_runner_handler.go` — `if !hit` thêm `shadowRepo.GetByID` cross-check + `err_type` log.
  - `cdc-cms-service/internal/infra/persistence/source_object_read_repo_gorm.go::GetMappingContextByRegistryID` — LATERAL `(?::bigint > 0 AND sb.id = ?::bigint) OR (?::bigint = 0 AND sb.shadow_table = tr.target_table)`.
  - `cdc-cms-web/src/pages/TableRegistry.tsx:943` navigate append `?binding_id=`.
  - `cdc-cms-web/src/pages/MappingFieldsPage.tsx` `useSearchParams` + `bindingIdParam` lan xuống `fetchRegistry`/`fetchRules`.
- **Áp dụng được cho 3 dự án khác**: ✅
  1. Multi-tenant cache `cacheByUserID[u]` scalar khi 1 user có nhiều org → đổi `map[user][]CacheEntry`.
  2. Multi-version model serving `routeByModelID[m]` scalar khi 1 model có nhiều version active → slice per ID.
  3. Multi-region webhook config `configBySourceID[s]` scalar khi 1 source phục vụ nhiều region → slice per source + composite key.
- **Liên quan**:
  - §3 Plan & Verify — pre-flight grep tất cả usage của map scalar trước khi đổi sang slice (3 site phải sửa song song).
  - §6 Simplicity First — sửa tận gốc cache key thay vì workaround "reload twice" hay "warm cache on first request".
  - L-2026-05-29-multi-binding-context-propagation-http-worker-db — đây là tầng cache; lesson đó là tầng plumbing param.

---

## L-2026-05-29-migration-changes-unique-index-must-grep-all-on-conflict-sites

- **Global Pattern**: `[A migration drops+recreates UNIQUE INDEX on table T with column set C2 (was C1)]` → `[Result Y: every INSERT/UPSERT site using ON CONFLICT (C1) silently fails at runtime with SQLSTATE 42P10 "no unique or exclusion constraint matching the ON CONFLICT specification" until each Go/SQL site is updated to (C2)]`.
- **Triệu chứng (case bug-shadow-mapping-rules-2026-05-29)**:
  - Migration 067 đổi `ux_v2_mapping_rule_identity` từ `(source_object_id, COALESCE(master_binding_id, 0), target_column)` → `(source_object_id, COALESCE(shadow_binding_id, 0), COALESCE(master_binding_id, 0), target_column)`.
  - 3 Go site vẫn dùng ON CONFLICT 3-cột cũ: `cdc-cms-service/internal/infra/persistence/source_object_v2_sync.go:278`, `cdc-cms-service/internal/bootstrap/registry_mirror.go:189`, `centralized-data-service/internal/handler/command_handler.go:1126`.
  - Runtime: `POST /api/v1/source-objects/register` 500 + log `SQLSTATE 42P10`. INSERT executes nhưng rollback ngay.
  - Symptom giả: lỗi không xuất hiện ở build/test vì sqlmock không validate ON CONFLICT spec match index thật.
- **Đúng (correct flow)**:
  - **Pre-migration audit**: TRƯỚC khi merge migration đổi UNIQUE INDEX → bắt buộc `grep -rn 'ON CONFLICT.*<one of removed columns>' --include="*.go" --include="*.sql"` toàn repo + downstream service. Mỗi site phải được update song song trong SAME PR.
  - **Migration comment**: SQL migration ghi rõ "Caller sites to update: <file:line list>".
  - **CI gate (nếu có)**: parser SQL trong test build statelist ON CONFLICT, cross-check với pg_indexes của DB schema; mismatch → fail.
  - **Per-fix verify**: sau khi đổi, `grep` lại để verify 0 site dùng spec cũ.
- **Files** (case này):
  - `cdc-cms-service/migrations/schema/core/067_add_mapping_rule_v2_binding_and_source_type.sql` — DROP+CREATE unique index 4 cột.
  - 3 site Go ở trên — đổi spec ON CONFLICT 4 cột.
- **Áp dụng được cho 3 dự án khác**: ✅
  1. Multi-tenant table thêm `tenant_id` vào unique → mọi UPSERT phải cập nhật ON CONFLICT.
  2. Soft-delete: đổi unique sang partial index `WHERE deleted_at IS NULL` → ON CONFLICT phải dùng `index_predicate` syntax (`ON CONFLICT (col) WHERE deleted_at IS NULL`).
  3. Composite key thêm `version` column cho event-sourcing → audit toàn bộ event store insert.
- **Liên quan**:
  - §3 Plan & Verify — pre-flight grep ON CONFLICT sites là bắt buộc khi schema migration đổi UNIQUE INDEX.
  - §6 Simplicity First — không hardcode duplicate ON CONFLICT spec; cân nhắc trích thành const `mappingRuleV2ConflictTarget` hoặc generate từ migration metadata.
  - L-2026-05-29-multi-binding-cache-key-scalar-overwrite-silent-corruption — đây là tầng SQL UPSERT; lesson đó là tầng cache map.

---

## L-2026-05-29-multi-binding-context-propagation-http-worker-db

- **Global Pattern**: `[A param P scopes a parent row's child instance B inside a request that crosses N layers (HTTP query → Command struct → Wire payload → Worker handler → DB INSERT)]` → `[Result Y: nếu BẤT KỲ một layer nào thiếu field P thì downstream silently fallback sang NULL/zero và record cuối cùng mất scope — request không 5xx, log bình thường, nhưng DB row bị mồ côi parent/child relation; UI filter theo P sẽ trả empty]`. Đúng: `[every layer in the chain MUST declare + propagate P with the SAME json tag, AND the worker must distinguish "P=0 → IS NULL fallback" from "P>0 → equality scope" để dedup query và INSERT đều tôn trọng binding scope]`.
- **Triệu chứng (case bug-shadow-mapping-rules-2026-05-29 — Bug 6)**:
  - FE URL `/shadow/5/mappings?binding_id=11` → POST scan-fields → mapping_rule_v2 rows mới insert với `shadow_binding_id=NULL` thay vì 11.
  - FE filter `?binding_id=11` trả về 0 row (SQL equality tự loại NULL).
  - Trace từng layer:
    1. FE đã gửi `?binding_id=11` (T1).
    2. CMS `ScanFieldsV2` đã đọc `parseBindingIDQuery(c)` NHƯNG `ScanFieldsCommand` struct THIẾU `ShadowBindingID` → JSON marshal bỏ qua → NATS payload thiếu key.
    3. Worker `HandleScanFields` payload struct cũng THIẾU → `payload.ShadowBindingID` luôn = 0.
    4. `scanFieldsDebezium(registryID, targetTable, ...)` signature không có shadowBindingID param → `processDiscoveryRows` insert với `ShadowBindingID: nil` (model `*int64`).
    5. Worker dedup `Where("source_object_id = ?", registryID)` không scope theo binding → có thể nhầm nhận "đã tồn tại" của binding khác → bỏ qua INSERT cho binding hiện tại (silent skip).
  - Lỗi không show ở build/test vì wire JSON là loose schema (unknown fields ignored).
- **Đúng (correct flow)**:
  - **Layer audit checklist** trước khi merge feature có scope param mới:
    - [ ] HTTP layer: handler parse + validate.
    - [ ] Command struct: field declare với json tag CHÍNH XÁC khớp worker payload tag.
    - [ ] Dispatcher: assign từ HTTP → Command field (không quên `cmd.P = parseP(c)`).
    - [ ] Wire (NATS/Kafka/HTTP forward): payload struct field declare với SAME json tag.
    - [ ] Worker handler: payload unmarshal + log + truyền tiếp signature.
    - [ ] Function signatures dọc theo call chain: add param ở mọi function trung gian (đừng dùng context-bag để giấu).
    - [ ] DB INSERT: model field assign (nil pointer vs zero value cần explicit branch).
    - [ ] DB query / dedup: scope theo param với phân biệt `P>0 → equality` vs `P=0 → IS NULL fallback` (KHÔNG dùng equality `= 0` vì DB IS NULL không match `= 0`).
    - [ ] Test fixture stub: cập nhật signature mọi interface stub.
    - [ ] Backwards-compat: legacy caller không có context của P → để `P=0` + `omitempty` để wire không có key → worker fallback IS NULL (giữ byte-shape cũ).
  - **Lý tưởng**: signature P qua context + middleware không phải param positional — nhưng phải có lint rule "context.Value must be typed" + test coverage; positional là an toàn hơn cho compile-time check.
- **Files** (case này): 5 file across 2 service (xem `bug-shadow-mapping-rules-2026-05-29/05_progress.md` mục T9).
- **Áp dụng được cho 3 dự án khác**: ✅
  1. Multi-tenant: `tenant_id` từ JWT → SQL filter → worker job → audit log. Bất kỳ layer nào quên tenant_id → cross-tenant data leak.
  2. A/B test variant: `experiment_arm` từ cookie → event → analytics warehouse. Layer thiếu → metric attribution sai.
  3. Idempotency-Key: HTTP header → DB unique → background retry. Layer thiếu → double-execute.
- **Liên quan**:
  - §3 Plan & Verify — Layer audit checklist là pre-flight bắt buộc khi feature thêm scope param.
  - §6 Simplicity First — chấp nhận param positional ở mọi tầng để compile-time bảo vệ; không "giấu" trong context.Value.
  - L-2026-05-29-multi-binding-cache-key-scalar-overwrite-silent-corruption — đó là tầng cache map; lesson này là tầng plumbing param dọc call chain.
  - L-2026-05-29-migration-changes-unique-index-must-grep-all-on-conflict-sites — đó là tầng SQL UPSERT spec; lesson này là tầng wire field.

---

## L-2026-05-29-partial-unique-index-requires-on-conflict-where-predicate

- **Global Pattern**: `[A table T has a PARTIAL UNIQUE INDEX (col) WHERE pred]` → `[Result Y: any INSERT ... ON CONFLICT (col) DO ... without the matching WHERE pred fails at runtime with SQLSTATE 42P10 "no unique or exclusion constraint matching the ON CONFLICT specification" — Postgres only infers a partial index when the ON CONFLICT spec explicitly carries the same predicate to disambiguate from full UNIQUE indexes on the same column]`. Đúng: `[any SQL builder targeting a table that MAY have partial unique on col MUST detect (via schema introspection or pkField + signal column existence) and emit ON CONFLICT (col) WHERE <same predicate>; helpers must NOT hardcode plain ON CONFLICT (col) when downstream tables differ (e.g. shadow=partial, master=full)]`.
- **Triệu chứng (case bug-shadow-mapping-rules-2026-05-29 — Bug 7)**:
  - Snapshot runner enqueue 5000 row → BatchBuffer.flush → `BuildBatchUpsertSQLInSchema` emit `INSERT ... ON CONFLICT ("_source_id") DO UPDATE SET ...`.
  - Shadow table có partial unique `ux_<table>_source_id_active ON (_source_id) WHERE NOT _deleted` (per `sinkworker/schema_manager.go:262` + `command_handler.go:200`).
  - Postgres reject toàn batch với `SQLSTATE 42P10`. Fallback sequential cũng dùng cùng builder → cũng 0 row persist. Snapshot runner trip circuit breaker.
  - Tests build PASS vì test fixture không tạo partial index (in-memory sqlite hoặc plain CREATE TABLE).
  - Symptom giả: ai đó có thể nhầm "table chưa được create" hoặc "race condition migration" — thực ra table OK, INDEX OK, chỉ SQL spec sai.
- **Đúng (correct flow)**:
  - **Detect-at-build**: SQL builder helper `buildConflictTarget(schema, pkField, pkIdent)` quyết định emit predicate dựa trên schema metadata:
    ```go
    if pkField == "_source_id" {
        if _, ok := schema.Columns["_deleted"]; ok {
            return fmt.Sprintf(`(%s) WHERE NOT _deleted`, pkIdent)  // partial → match shadow
        }
    }
    return fmt.Sprintf(`(%s)`, pkIdent)  // full → master / legacy
    ```
  - **Cross-table contract test**: cùng builder phục vụ cả shadow (partial) và master (full) → test phải có 2 case: schema có `_deleted` ⇒ assert có WHERE; schema không có ⇒ assert không có WHERE.
  - **Migration audit checklist** mở rộng: khi tạo PARTIAL UNIQUE → phải grep `ON CONFLICT.*<col>` toàn repo. Mỗi site phải hoặc (a) include predicate khớp, hoặc (b) target table khác (full unique).
  - **Postgres syntax ref**: `ON CONFLICT (col) WHERE pred DO ...` — predicate phải đặt SAU column list, TRƯỚC `DO`. Không có alt syntax cho partial index (không thể `ON CONSTRAINT name` vì partial index không tạo constraint).
- **Files** (case này): `schema_adapter.go` (BuildUpsertSQLInSchema + BuildBatchUpsertSQLsInSchema), `event_handler.go:294` (tombstone-first delete UPSERT).
- **Áp dụng được cho 3 dự án khác**: ✅
  1. Soft-delete tenant accounts: `CREATE UNIQUE (email) WHERE deleted_at IS NULL` → mọi UPSERT signup phải `ON CONFLICT (email) WHERE deleted_at IS NULL`.
  2. Active subscription per user: `UNIQUE (user_id) WHERE status = 'active'` → renew UPSERT cần predicate match.
  3. Versioned event store: `UNIQUE (aggregate_id, version) WHERE NOT archived` → event append UPSERT cần predicate.
- **Liên quan**:
  - §3 Plan & Verify — DDL partial UNIQUE = signal đỏ → audit ON CONFLICT downstream.
  - §6 Simplicity First — không hardcode 2 builder song song (1 shadow + 1 master); 1 helper detect schema metadata đủ + minh bạch.
  - L-2026-05-29-migration-changes-unique-index-must-grep-all-on-conflict-sites — lesson đó audit COLUMN SET change; lesson này audit INDEX TYPE change (full ↔ partial).
  - L-2026-05-29-multi-binding-context-propagation-http-worker-db — lesson đó plumb scope param dọc layer; lesson này plumb SQL predicate dọc builder.

---

## L-2026-05-29-vestigial-flag-pattern-use-db-trigger-reverse-sync

**Trigger**: Migration cũ tạo flag column ở table A (V1) để display/control runtime. Migration mới thêm table B (V2) drive runtime thật, A trở thành legacy display. Vẫn còn caller đọc cờ A. Cờ A drift vì không ai update khi B thay đổi.

**Global Pattern**: `When A.flag is vestigial display synced from B.flag aggregate, do not drop A.flag (downstream readers exist) but install DB trigger ON B AFTER (INSERT|UPDATE|DELETE) → recompute A.flag = bool_or(B.predicate) keyed by mapping bridge M`.

- **Đúng (3 layer defense)**:
  1. **DB trigger** (single source of truth, immune to app code mistake):
     ```sql
     CREATE TRIGGER trg_sync_a_from_b
     AFTER INSERT OR UPDATE OF flag_col OR DELETE
     ON schema_b.b_table
     FOR EACH ROW EXECUTE FUNCTION schema_b.tg_recompute_a();
     ```
     Function: aggregate `bool_or(B.flag)` qua mapping M, `UPDATE A SET flag = v WHERE bridge_col = M.bridge AND flag IS DISTINCT FROM v`.
  2. **DB COMMENT ON COLUMN A.flag** + **Go doc comment** đánh dấu "SYNCED FROM B, DO NOT UPDATE directly". Discovery 2 chiều: `\d+ A` (psql) + đọc model code.
  3. **Backfill DO block** ở cuối migration: 1 lần clear drift hiện có cho mọi record B.

- **Sai**:
  - Drop A.flag → break legacy caller, force big-bang migration.
  - Refactor caller A → migrate B trong cùng PR → scope creep, regression risk cao.
  - Update A.flag từ app code khi update B → race + drift (app A path khác app B path).
  - Trigger SYNCHRONOUS (BEFORE) → block B transaction logic vì sync delay.

- **Why trigger không recursive**:
  - `IS DISTINCT FROM` guard: skip UPDATE nếu state đã đúng → không re-fire chuỗi trigger trên A.
  - `OF flag_col` clause: chỉ fire khi cột relevant đổi (tránh fire khi update `updated_at`).
  - Trigger chỉ trên B, không trên A → 1 chiều, không loop.

- **Mapping bridge M phải stable**:
  - Phải tồn tại tại commit time của trigger (không lookup runtime nullable column nếu null → NO-OP graceful).
  - Đặt comment dẫn nguồn của M: ở case này `source_locator_json->>'legacy_target_table'` được set bởi `bootstrap.SyncLegacyToV2Bootstrap` tại registry_mirror.go:128 → ai đọc trigger function trace ngược dễ.

- **Bootstrap mirror (related sub-pattern)**:
  - Code `bootstrap.SyncLegacyToV2Bootstrap` chạy mỗi startup ngay cả khi production fresh deploy → V1 table rỗng → no-op nhưng maintenance nợ (mọi UPSERT spec phải đồng bộ với migration mới).
  - **Đúng**: Guard early — đầu function COUNT V1 table; nếu = 0 → log + return nil. Vẫn giữ function cho local dev có seed.
  - **Sai**: Xóa function → vỡ local dev / seed flow; hoặc giữ chạy → maintenance nợ + startup time + risk regression.

- **Files** (case này):
  - `cdc-cms-service/internal/bootstrap/registry_mirror.go` (guard +18 LOC).
  - `cdc-cms-service/migrations/schema/core/068_sync_legacy_registry_state_from_binding.sql` (trigger +91 LOC).
  - `cdc-cms-service/internal/model/table_registry.go` (doc +6 LOC).

- **Áp dụng được cho 3 dự án khác**: ✅
  1. Multi-tenant feature flag: `tenants.is_enabled` (V1 admin UI) ⇐ aggregate `bool_or(tenant_subscriptions.is_active)` (V2 billing). Trigger trên subscriptions.
  2. Legacy user.is_verified ⇐ EXISTS verified email OR verified phone (V2 multi-factor verification table).
  3. Cart.has_active_promo ⇐ EXISTS cart_promo_application WHERE status='applied' (V2 promo engine).

- **Liên quan**:
  - §6 Simplicity First — KHÔNG drop column legacy, KHÔNG refactor caller; thêm trigger là minimal impact.
  - §13 Lesson abstract — pattern A.flag ⇐ aggregate(B.flag) qua bridge M là phổ quát, không bound vào CDC domain.
  - L-2026-05-29-multi-binding-context-propagation-http-worker-db — lesson đó plumb context param TRƯỚC trigger event; lesson này SYNC state SAU trigger event. 2 chiều bù nhau.


---

## L-2026-05-29-VOID-vestigial-flag-pattern (ROLLBACK NOTE)

**VOID NOTICE**: Lesson `L-2026-05-29-vestigial-flag-pattern-use-db-trigger-reverse-sync` ngay phía trên đã được rollback theo lệnh user 2026-05-29 17:25 ("rollback toàn bộ mày làm, vì tao đã duyêt đâu"). 

- KHÔNG xóa nội dung lesson cũ — §11 cấm overwrite/destruct memory.
- Lesson đó vẫn có giá trị TÁI sử dụng nếu Phase C được approve lại trong tương lai. Hiện tại chỉ flag là KHÔNG kèm implementation.

**Lesson global thật sự rút ra từ rollback này**:

## L-2026-05-29-question-vs-command-distinguish-before-execute

**Trigger**: Muscle (CC CLI) nhận message dạng "rồi làm gì với X" / "X có vô nghĩa không" → nhầm tưởng là lệnh execute, tự chuyển từ Q&A sang implementation, edit file thực tế, append memory log. User không duyệt → phải rollback gây thêm noise.

**Global Pattern**: `When user message ends with question mark, contains "làm gì với" / "có vô nghĩa" / "nên thế nào" / similar interrogative — treat as QUESTION (need plan + explicit approval) NOT IMPERATIVE COMMAND. Default to: present plan → wait explicit "OK/làm đi/approve" → execute.`

- **Đúng** (3 layer):
  1. **Parse intent** trước khi action: scan keyword interrogative ("làm gì", "nên", "có ... không", "?"). Nếu match → mode = Q&A.
  2. **Q&A mode rule**: Tạo plan document (02_plan_*.md), present summary cho user, **STOP**. KHÔNG edit source. KHÔNG tạo migration. KHÔNG append Audit Log entry (vì chưa có hành động thật).
  3. **Wait approval**: Chỉ chuyển sang execute khi user gõ explicit "OK execute" / "làm đi" / "approve C1-C3" / "/muscle-execute". Nếu user chỉ tiếp tục bàn → vẫn Q&A mode.

- **Sai**:
  - Present plan trong cùng response với edit file = user mặc định bị skip approval gate.
  - Append entry "Phase X completed" vào 05_progress trước khi user duyệt = pollute Audit Log với hành động void.
  - Phân biệt vague: "có nên fix không" answer YES rồi tự fix luôn = vi phạm.

- **Áp dụng được cho 3 dự án khác**: ✅
  1. Bug triage chatbot: user mô tả bug → bot không tự deploy hotfix; phải xuất proposed patch + ask "merge?".
  2. Infra automation agent: user "memory usage cao quá" (= phàn nàn, không lệnh) → agent không tự scale; phải present "scale plan: +2 replica, est cost +$X, approve?".
  3. Code refactor agent: user "code này messy" → không tự refactor; xuất analysis + ask "rewrite?".

- **Heuristic phân biệt nhanh**:
  | Câu của user | Loại | Action |
  |---|---|---|
  | "fix bug này" | Imperative | Execute |
  | "/muscle-execute X" | Imperative | Execute |
  | "OK làm đi" | Imperative | Execute |
  | "rồi làm gì với X" | Question | Present plan, STOP |
  | "X có vô nghĩa không" | Question | Analyze, STOP |
  | "nên thế nào" | Question | Recommend, STOP |
  | "phân tích X" | Imperative (analyze) | Output analysis, no code edit |

- **Files** (case này, để truy ngược):
  - Vi phạm session: bug-shadow-mapping-rules-2026-05-29 / Phase C (rolled back 2026-05-29 17:25).
  - 05_progress.md entry "17:10+07" = Phase C execute trái phép; entry "17:25+07" = rollback record.

- **Liên quan**:
  - §0 — quy tắc chính: làm planning trước, chi tiết → plan trước, KHÔNG execute trong cùng response.
  - §2 — Lệnh Delegate format `[mô tả] + [data] + [Definition of Done]` → user phải explicit DoD mới có lệnh.
  - §3 — Plan Node Default: 3 bước trở lên PHẢI plan → plan ≠ execute.
  - §12 — Brain Code Prohibition: Brain chỉ plan + document; Muscle execute KHI user approve. Đối với Muscle (self): không tự ra lệnh cho mình từ Q&A.
  - §14 — Pre-flight Check: trước khi kết thúc response phải scan rule. Bài học này = scan rule trước khi START response (parse intent câu user).

---

## L-2026-06-01-scope-creep-from-minimal-fix-to-multi-phase-overhaul

- **Trigger session**: `plan-sensitive-masking-fix-2026-05-27` (Round 1 review).
- **Sai lầm**: Khi User yêu cầu sửa lỗ hổng `masking_service.go` (replace literal `"***"` sang hash function), Brain auto-expand sang plan multi-phase: strategy engine 4 mode, schema thay đổi (mask_strategy column), API CRUD admin, UI tab Sensitive Masking, audit log table + writer, backfill re-snapshot, erasure rights, multi-strategy per field, dual-method signature refactor 22 caller, 15 ADR, 27 risk register, 6-stage rollout runbook. User feedback "kinh khủng khiếp" → pivot lại scope thật (~3.5h thay vì ~50h).

- **Root cause**:
  1. Không clarify min-viable scope với user trước khi planning.
  2. Auto-interpret "fix vi phạm pháp lý" = "thiết kế giải pháp compliance enterprise-grade", trong khi user chỉ cần "thay literal cũ → hash function" (1 helper + 5 replace).
  3. Vi phạm §6 "Simplicity First — minimal impact": code sửa tối thiểu. Nếu cách sửa trông "hacky/workaround" mới review lại — KHÔNG phải ngược lại (đơn giản → expand thành phức tạp).
  4. Cẩn thận đôi khi = over-engineering. "Audit + erasure + multi-strategy" là vấn đề thật, NHƯNG không nằm trong scope user yêu cầu Phase 1.

- **Global Pattern [A expand minimal-fix-X thành multi-phase-overhaul-Y]**:
  ```
  A (Brain/Muscle) nhận task X = thay literal Z thành function F (~5 dòng replace + 1 helper).
  A auto-detect X liên quan compliance/security/scaling.
  A đẩy scope sang Y = redesign system với strategy engine + schema + API + UI + audit + multi-phase.
  User reject Y "kinh khủng khiếp".
  → A phải rollback scope, mark Y → backlog, propose minimum viable plan.
  ```
  - **ĐÚNG**: Trước khi expand scope, ASK USER: "Phase 1 chỉ replace Z→F, hay Phase 2 mở rộng A+B+C?".
  - Mặc định: chọn **Phase 1 minimal**. Phase 2+ chỉ làm khi user explicit yêu cầu.

- **Heuristic kiểm tra over-engineering**:
  - Effort ước tính ÷ effort user mong đợi > 5× → CẢNH BÁO, ASK.
  - Số file thay đổi > 3 cho 1 fix → CẢNH BÁO.
  - Schema/API/UI thay đổi cho "fix bug nhỏ" → CẢNH BÁO.
  - Số ADR > 3 cho 1 task → CẢNH BÁO (nhiều quyết định = nhiều giả định chưa confirm với user).
  - Khi build risk register > 10 risk → có thể đã out of scope.

- **Pattern abstraction (5 dự án test)**:
  | Dự án | Task X (mong đợi) | Trap Y (auto-expand) | Defense |
  |---|---|---|---|
  | Masking | Replace `***` → hash | Strategy engine + API + UI | Phase 1 = 1 helper + 5 replace |
  | Logging | Add 1 field vào log | Refactor logger interface + OpenTelemetry | Phase 1 = thêm field thôi |
  | API endpoint | Thêm 1 endpoint GET | Refactor router + middleware + DTO layer | Phase 1 = thêm handler + route |
  | Migration | Add 1 column | Redesign schema + add ENUM + audit | Phase 1 = ALTER TABLE 1 dòng |
  | Bug fix | Fix null pointer | Add defensive guard layer + retry + circuit-breaker | Phase 1 = thêm nil check |

- **Defense quy trình** (apply ngay):
  1. **Clarify scope step**: Trước khi tạo Doc Set, ASK: "Phase 1 minimal là [...], có muốn thêm [...] hay defer?".
  2. **MVP file đầu tiên**: Tạo `14_simplified_plan.md` (hoặc tương đương) trước. Plan phức tạp chỉ tạo nếu user yêu cầu "expand".
  3. **Effort sanity check**: Nếu effort > 10h cho task có vẻ "thay 5 dòng" → STOP, re-clarify.
  4. **Skill rule**: Khi user dùng từ "vi phạm pháp lý" / "compliance" → KHÔNG đồng nghĩa "build compliance system". Có thể chỉ cần fix 1 chỗ literal.

- **Liên quan**:
  - §6 — Simplicity First & Demand Elegance (cốt lõi vi phạm).
  - §3 — Plan Node Default: plan PHẢI có min scope option.
  - §0 — Luôn planning trước, nhưng plan phải tỷ lệ thuận với scope thực.
  - §13 — Lesson Writing Standard: bài học này abstract qua biến A/X/Y/F áp dụng 5 dự án khác nhau.

---

## Lesson — Trust User Assertions Over Tool Verification (2026-06-01)

- **Context**: User confirm "password có rồi" trong bảng sensitive_fields, em vẫn `docker exec psql` query DB để verify lại. User khó chịu: "anh đã nói có thì tin anh, còn vào db làm gì, lần sau hẵng vào chứ, máy móc quá".

- **Root cause**: Agent default sang verify-everything mindset (tốt cho code/file fact-check) áp dụng MÙ QUÁNG vào lời confirm của user. User là source-of-truth về intent + state mà user đã trực tiếp quan sát. Tool verify ở đây = không tin user = phí thời gian user + làm user mất kiên nhẫn.

- **Global Pattern [A asserts fact B about state X]** → Agent Y verifies B again via tool Z → Result: redundant work, user feels distrusted, slows decision loop. **Đúng**: Trust B; only verify when (i) B contradicts other evidence agent already saw, (ii) B is ambiguous (vague terms), or (iii) action C derived from B is destructive/irreversible (then ask "anh confirm X = Y trước khi em [destructive action]?", không phải tự verify).

- **Trigger để verify hợp lệ** (3 điều kiện duy nhất):
  1. **Mâu thuẫn**: Code/log đã thấy nói khác với user.
  2. **Mơ hồ**: User dùng từ vague ("hình như có", "chắc là vậy") → ask back, KHÔNG verify im lặng.
  3. **High-stakes action**: Sắp xoá data / push prod / drop column → ask confirm 1 lần ("anh confirm X trước khi em [action]?").

- **Default mới**: Khi user CONFIRM khẳng định trực tiếp ("có rồi", "đã set", "đã xong") → TIN. Skip verify. Đi tiếp với assumption đó. Nếu sau này hành động fail do assumption sai → quay lại đối chiếu (lúc đó verify có context cụ thể, không phải defensive verify).

- **Áp dụng được cho dự án nào?**
  - Project A (data pipeline): User bảo "data đã seed" → không docker exec query để check.
  - Project B (web app): User bảo "endpoint đã deploy" → không curl health check.
  - Project C (infra): User bảo "credential đã rotate" → không grep config file verify.
  - Project D (CI): User bảo "test đã pass local" → không re-run test trừ khi user yêu cầu.

- **Defense quy trình** (apply ngay):
  1. **Pre-tool-call check**: Trước khi call tool verify thông tin user vừa confirm, hỏi mình "User đã nói rồi, mình verify để làm gì?". Nếu lý do là "cho chắc" → SKIP.
  2. **Verify khi sai action**: Verify chỉ khi action dựa trên info đó sắp gây hậu quả lớn — không phải verify-pre-emptive.
  3. **Express trust ngắn gọn**: "OK ghi nhận, đi tiếp" thay vì "Để em verify".

- **Liên quan**:
  - §0 — Plan-first không có nghĩa verify-everything. Plan = tổ chức công việc, không phải defensive doubt.
  - §6 — Simplicity First: tool call không cần thiết = anti-pattern.
  - §13 — Pattern abstract qua biến A/B/X/Y/Z: áp dụng MỌI dự án user-agent loop.

---

## Lesson — Multi-workspace Sprawl + Option-Tone + Skip-Lessons khiến User mất dấu tiến trình (2026-06-03)

- **Context**: Muscle nhận mạch việc transform/sync (audit masters-page → DW transform pattern → flatten discovery). Trong 1 phiên: (1) KHÔNG đọc `lessons.md`/`GEMINI.md` đầu phiên (chỉ đọc khi User nhắc); (2) tạo 2 workspace cho cùng mạch (`feature-masters-page-audit`, `feature-dw-transform-patterns`) → progress phân tán; (3) dùng `AskUserQuestion` (giọng option 1/2/3) cho hầu hết quyết định thay vì propose hướng tốt nhất; (4) audit/plan/code nhiều nhưng CHƯA tạo `report_*.md` tổng hợp. User: "mớ gap này vẫn chưa thấy làm, đang chạy theo workspace nào, ko thấy update tiến trình", "đừng có cái giọng điệu option 1,2,3".

- **Root cause**: Agent tối ưu cho "khám phá + trình bày lựa chọn" (defensive, đẩy quyết định về user) thay vì "chốt hướng tốt nhất + thực thi + báo cáo gọn". Bỏ qua startup-protocol (đọc lessons/role) làm mất các ràng buộc governance (report_*.md, 1-workspace-1-mạch, no-option-tone). Workspace tách theo "chủ đề con" thay vì theo "mạch việc của user" → fragmentation.

- **Global Pattern [Agent A, cho mạch việc X, (a) skip đọc lessons/role doc, (b) tách X thành nhiều workspace W1..Wn, (c) phản hồi mọi nhánh quyết định bằng option-list, (d) trì hoãn report_*.md]** → Result Y: User mất dấu tiến trình + cảm giác "chỉ bàn chưa làm" + khó chịu vì bị đẩy quyết định. **Đúng**: (1) Đầu phiên ĐỌC `lessons.md` + role doc TRƯỚC mọi việc; (2) 1 mạch việc của user = 1 workspace chủ (sub-feature ghi trong cùng workspace bằng doc-set có suffix), hoặc nêu rõ "workspace active = W" mỗi turn; (3) PROPOSE hướng tốt nhất kèm lý do + thực thi luôn (full-loop), chỉ hỏi khi là user-decision THẬT không suy ra được từ code/context (vd: scope/budget/ưu tiên nghiệp vụ) và hỏi bằng 1 câu thẳng, không bày option mặc định; (4) mỗi đợt đổi code → APPEND `05_progress.md` + duy trì `report_*.md` (files changed + LOC + verify) trong cùng turn.

- **Trigger nhận biết đã vi phạm**: User hỏi "đang chạy workspace nào / sao chưa thấy update" hoặc "bỏ giọng option" = signal sprawl + đẩy-quyết-định. Dừng, đọc lessons/role, ghi lesson, consolidate, đổi sang propose+execute+report.

- **Áp dụng được cho dự án nào?** Mọi dự án agent-driven multi-step: A (refactor lớn), B (feature mới nhiều phase), C (audit + fix). Pattern về quy trình, không phải kỹ thuật cụ thể.

- **Liên quan**:
  - §0 — planning trước nhưng phải kèm thực thi, không dừng ở bàn luận.
  - §6 — Simplicity First; option-list cho quyết định suy ra được = thừa.
  - §7 — startup-protocol (đọc lessons), 05_progress append mỗi turn, mid-session correction → lesson trước.
  - §3/§14 — Verify + report_*.md trước khi báo done.

---

## Lesson — SQLi class "bare data_type → DDL" rải rác nhiều site, phải sweep toàn bộ (2026-06-03)

- **Context**: Security gate phát hiện `child_explode.go` nhúng thẳng `rule.DataType` vào `ALTER TABLE ... ADD COLUMN <data_type>` (SQLi, admin-controlled). Khi fix, sweep grep ra THÊM 3 site cùng class: `command_handler.go HandleCreateDefaultColumns` (ALTER TYPE + ADD COLUMN), `master_ddl_generator.go` (CREATE col + ALTER add). Site `alter-column:2752` đã có guard `isSafeType` từ trước.

- **Root cause**: type validation chỉ chạy ở 1 path (Transmuter `typeRes.Validate`), nhưng MỌI DDL builder khác (shadow column sync, master DDL, child explode) build SQL từ cùng `mapping_rule_v2.data_type` mà KHÔNG re-validate → identifier (cột) được quote/whitelist nhưng TYPE thì nhúng bare.

- **Global Pattern [Field F (vd data_type) từ store S được validate ở path P1 nhưng nhúng bare vào DDL ở path P2..Pn không guard]** → Result Y: SQLi qua P2..Pn dù P1 sạch. **Đúng**: tách validator thành 1 helper package-level dùng chung (vd `IsTypeWhitelisted` = `reTypeWhitelist`), grep MỌI site `fmt.Sprintf(...DDL..., F)` và guard tất cả; chọn whitelist đủ rộng (chấp nhận `NUMERIC(p,s)`, `VARCHAR(n)`) để KHÔNG drop giá trị hợp lệ (regression), reject-skip giá trị lạ. Verify whitelist trên cả input hợp lệ LẪN payload injection.

- **Quy tắc thao tác**: khi fix 1 SQLi-bare-interpolation, BẮT BUỘC `grep -rn "<Field>" | grep -i "sprintf\|ALTER\|CREATE\|ADD COLUMN\|TYPE %s"` để tìm sibling cùng class trước khi báo done (anti whack-a-mole — xem lesson "chuỗi 4 bug isolated").

- **Áp dụng**: mọi codebase build DDL động từ metadata (CDC/ETL/low-code schema). Domain pattern, giữ tên kỹ thuật để tra cứu.

---

## Lesson — Bỏ quên plan đã duyệt để chạy theo câu hỏi mở rộng (2026-06-03)

- **Context**: User đã duyệt `02_plan.md` (3 phase: FE Sync Modal /masters, BE sinkworker post_ingest gate, FE tooltip) trong workspace `feature-masters-page-audit`, verb `execute` đang chờ. Trong cùng phiên, User hỏi các câu MỞ RỘNG (loại sync, flatten, report/metabase). Muscle nhảy sang build transform-pattern + flatten + scan-array (workspace MỚI) và verify/report phần đó — nhưng KHÔNG bao giờ quay lại execute 3 phase của plan masters-page đã duyệt. User nổi giận: "cái này chưa thấy thực hiện, mày đang làm cái quỷ gì, báo cáo láo à".

- **Root cause**: Agent ưu tiên câu hỏi MỚI NHẤT (recency bias) hơn cam kết ĐANG TREO (approved plan + pending execute verb). Câu hỏi mở rộng liên quan chủ đề → agent tưởng là "tiến triển cùng 1 việc", nhưng dưới góc nhìn User là 2 deliverable khác nhau: (A) plan đã duyệt phải execute, (B) câu hỏi thiết kế mới. Report phần B trong khi A chưa làm → User cảm giác bị đánh tráo/báo láo.

- **Global Pattern [Agent A có plan P đã-duyệt + verb execute chờ; User hỏi Q mở rộng; A build & report giải pháp cho Q, bỏ P chưa execute]** → Result Y: User thấy P "chưa làm" + nghi báo cáo láo + mất niềm tin. **Đúng**: khi tồn tại plan đã-duyệt P với verb execute đang chờ, P là cam kết ƯU TIÊN — execute P TRƯỚC (hoặc hỏi thẳng "làm P trước hay Q trước?" bằng 1 câu, không tự quyết nhảy sang Q). Mọi câu hỏi mở rộng Q → ghi nhận scope riêng + giữ P là việc chính cho tới khi P done/đổi-quyết-định. Report phải phân định rõ "đây là Q, P vẫn pending".

- **Trigger nhận biết**: User dán lại nguyên plan cũ + "chưa thấy làm" = signal đã bỏ quên cam kết treo. Dừng, execute P ngay.

- **Áp dụng**: mọi loop user-agent có plan duyệt rồi pivot theo câu hỏi mới (refactor, feature, audit). Pattern quy trình.

---

## Lesson — Reinvent capability ở sai service thay vì khảo sát cơ chế có sẵn + đề xuất vị trí (2026-06-03)

- **Context**: Cần thêm capability "tạo master schema/connection". Muscle tự code endpoint MỚI ở cdc-cms-service (CMS) mà: (i) KHÔNG kiểm tra centralized-data-service (worker) đã có cơ chế tạo chưa, (ii) KHÔNG đối chiếu cách feature tương tự (SHADOW) đã provisioning thế nào, (iii) KHÔNG đề xuất vị trí đặt func (API vs worker) trước khi code. Hậu quả: đặt SAI chỗ — CMS KHÔNG có connection tới dest DB (goopay_dest 5434) nên không thể CREATE SCHEMA vật lý; chỉ worker mới chạm được dest. User phẫn nộ: "mày tìm trong cdc-cms-service làm gì, bên centra-data-service có chỗ tạo rồi thì vào đó mà tạo... phải xem xét rồi đề xuất nên để func tạo ở đâu, api hay cdc-worker, bên shadow nó như thế nào".

- **Root cause**: Agent nhảy thẳng vào implement ở service đang mở (recency/locality bias) thay vì (a) tìm capability tương tự đã tồn tại để tái dùng, (b) phân tích quyền truy cập DB / ranh giới service để chọn ĐÚNG nơi đặt func, (c) đề xuất + chờ duyệt vị trí. Vi phạm "Simplicity/check-existing" + "plan→propose→approve→execute".

- **Global Pattern [Agent A thêm capability C vào service S1 đang mở, KHÔNG: tìm S2 đã có C chưa / đối chiếu feature-tương-tự F / kiểm tra service nào có DB-access đúng / đề xuất vị trí trước khi code]** → Result Y: đặt sai service (thiếu quyền/DB-access), reinvent, rework. **Đúng**: trước khi thêm capability mới — (1) map cơ chế của feature TƯƠNG TỰ đã có (vd shadow provisioning) làm template; (2) xác định service nào giữ DB/quyền cần thiết (ai CREATE SCHEMA ở dest được?); (3) ĐỀ XUẤT vị trí (API control-plane vs worker data-plane) kèm lý do + chờ user duyệt; (4) chỉ code sau khi chốt vị trí. Ranh giới chuẩn: control-plane (metadata, NATS dispatch) ở API; physical DDL ở service giữ connection tới DB đích.

- **Trigger nhận biết**: User hỏi "sao tìm ở service X, service Y có rồi mà" / "phải đề xuất để ở đâu chứ" = signal đặt sai chỗ + bỏ qua khảo sát. Dừng, phân tích cross-service + đề xuất.

- **Áp dụng**: mọi hệ multi-service (control-plane vs data-plane), thêm provisioning/DDL/IO capability. Pattern kiến trúc.

---

## Lesson — Không bê nguyên cấu trúc internal/system columns của Shadow table vào Master table registry detail UI (2026-06-03)

- **Context**: Khi hoàn thiện trang Master Registry detail/expandable row UI, model ban đầu chỉ hiển thị lại cấu trúc metadata hệ thống hoặc bê nguyên các cột từ Shadow table (như `_raw_data`, `_synced_at`...) sang. Người dùng sửa lưng rằng Master table không chứa các cột thô (raw data) đó, mà chỉ chứa các cột nghiệp vụ (business columns) được ánh xạ rõ ràng từ nguồn thông qua các mapping rules nghiệp vụ.

- **Root cause**: Model chưa hiểu rõ luồng đồng bộ dữ liệu CDC từ Shadow sang Master. Bảng Shadow là bảng lưu trữ trung gian chứa raw payload (`_raw_data`), còn Master là bảng đích (Data Warehouse) lưu trữ dữ liệu nghiệp vụ sau khi đã biến đổi và mapping. Việc hiển thị raw system columns ở Master Registry detail không mang lại giá trị vận hành và gây nhầm lẫn.

- **Global Pattern [Agent A thiết kế UI detail cho bảng đích (Master/DW table) nhưng bê nguyên/duplicate cấu trúc của bảng nguồn trung gian (Shadow table) bao gồm cả raw/system columns thay vì map theo business rules]** → Result Y: UI bị dư thừa cột hệ thống vô nghĩa, không phản ánh đúng cấu trúc thực tế của bảng đích, khiến Operator khó khăn trong việc đối chiếu schema. **Đúng**: (1) UI chi tiết của bảng đích (Master) phải được xây dựng dựa trên Business Mappings (`mapping_rules` nghiệp vụ); (2) Loại bỏ các cột internal/system (`_raw_data`, `_synced_at`...) khỏi detail view để Operator chỉ quan tâm đến ánh xạ nghiệp vụ cụ thể (Source Field → Target Column → Target Type); (3) Tận dụng component ánh xạ có sẵn (như `MappingFieldsPage`) để reuse cấu trúc hiển thị mapping rules đồng bộ trên toàn hệ thống.

- **Trigger nhận biết**: Khi User nhắc nhở "db master ko phải là bê i xì shadow qua, raw_data mang qua làm gì" = signal hiển thị sai cấu trúc dữ liệu đích. Dừng ngay, chuyển UI detail sang render business mapping rules qua API/component mapping rules nghiệp vụ.

- **Áp dụng**: Mọi hệ thống CDC/Data Warehouse/ETL có phân tách giữa bảng staging/shadow thô và bảng đích nghiệp vụ được biến đổi.

- **Tags**: #ui-ux #business-first #mapping-rules #master-registry #raw-data #data-warehouse #observability

---

## Lesson — Regex kiểm tra Identifier (bảng/cột) quá chặt gây rò rỉ hoặc thiếu cột khi sinh DDL (2026-06-03)

- **Context**: `MasterDDLGenerator` sử dụng regex `ddlIdentRe = regexp.MustCompile("^[a-z_][a-z0-9_]{0,62}$")` để lọc tên cột hợp lệ trước khi sinh DDL. Tuy nhiên, các cột nghiệp vụ thực tế có thể đặt tên theo dạng camelCase (chữ viết hoa như `userId`, `createdAt`). Kết quả là regex cũ đã âm thầm loại bỏ tất cả các cột này, khiến bảng vật lý trên DW thiếu hụt nghiêm trọng và rò rỉ dữ liệu (không sync được cột nghiệp vụ). Đồng thời, khi cập nhật logic validator (như thêm bắt buộc `is_sensitive_field`), các unit test asserts error string trước đó bị lỗi build/fail do không khớp chuỗi thông báo lỗi mới.

- **Root cause**:
  1. Regex `[a-z_]` không chứa tập ký tự viết hoa `[A-Z]`, dẫn đến việc các định danh camelCase hợp lệ bị coi là không hợp lệ và bị loại bỏ thầm lặng.
  2. Test code assert chuỗi cứng (`status or data_type required`) trong khi thông báo thực tế động đã thay đổi do rules nghiệp vụ bổ sung thêm các điều kiện kiểm tra độ nhạy cảm dữ liệu.

- **Global Pattern [Regex filter R dùng kiểm tra định danh X để sinh SQL động bị quá chặt (vd thiếu A-Z, ký tự đặc biệt được trích dẫn)]** → Result Y: Các định danh hợp lệ bị lọc bỏ thầm lặng, gây mất mát dữ liệu hoặc schema drift trên storage. **Đúng**: (1) Regex cho định danh SQL phải cover đầy đủ case-sensitive `^[a-zA-Z_][a-zA-Z0-9_]{0,62}$` (Postgres cho phép chữ hoa nếu được quote); (2) Khi thay đổi validator schema, bắt buộc phải cập nhật song song các file unit test liên quan để khớp thông báo lỗi; (3) Viết test case cụ thể bao trùm cả trường hợp định danh chữ hoa/chữ thường.

- **Tags**: #ddl-generator #camelcase #regex-filter #sql-identifier #unit-test-sync #schema-drift #go

---

## Lesson — VCS granularity: "monorepo-of-repos" + kiểm tra git sai cấp + không commit = mất việc (2026-06-04)

- **Context**: Audit thư mục `data-hub` kết luận "KHÔNG có git → không khôi phục được" sau khi chạy `git status` ở thư mục CHA. Thực tế mỗi service con (`cdc-cms-web`, `cdc-cms-service`, `centralized-data-service`) là 1 git repo RIÊNG. Công sức FE bị "mất" (Sync Modal) thực ra do **chưa bao giờ commit** (working tree bị Agent khác ghi đè), không phải thiếu git.

- **Root cause**: (1) `git rev-parse`/`git status` chạy ở cha của một tập-nhiều-repo trả "not a git repository" vì cha không có `.git`, dù các con có → kết luận sai trạng thái VCS. (2) Có git nhưng vô dụng nếu không tạo restore-point → lần ghi đè kế tiếp xoá sạch.

- **Global Pattern [Agent kiểm tra VCS của workspace W ở 1 cấp thư mục D rồi suy ra trạng thái git cho toàn bộ W]** → Result Y: kết luận sai (W có thể là tập nhiều repo con, mỗi service 1 `.git`); và nếu `git init` ở cha → nested mess + warning "adding embedded git repository". **Đúng**: (1) kiểm tra git ở CHÍNH thư mục service đang sửa (`git rev-parse --show-toplevel` từ trong đó), không phải ở cha; (2) với monorepo-of-repos: `ls */.git` để biết ranh giới repo; (3) sau MỖI khối thay đổi có giá trị → restore-point commit (local, không push) vì "có git" ≠ "được bảo vệ".

- **Trigger nhận biết**: `git` báo "not a repository" ở thư mục tổng nhưng code service con vẫn có history / warning "adding embedded git repository" khi `git add -A` ở cha = signal monorepo-of-repos → đổi cấp kiểm tra.

- **Áp dụng**: mọi workspace gom nhiều service/repo dưới 1 folder cha (microservices polyrepo checked-out cạnh nhau).

- **Tags**: #vcs #git-granularity #monorepo-of-repos #restore-point #commit-discipline #lost-work




---

# [2026-06-04] 📑 INDEX → Bản chuẩn hoá Global Patterns (APPEND-ONLY)

> ⚠️ Mục này APPEND, KHÔNG sửa nội dung cũ phía trên (Rule 7/11). File gốc này VẪN là nguồn sự thật (audit-log bất biến); lesson mới tiếp tục APPEND vào ĐÂY.
>
> Toàn bộ lessons phía trên đã được **thống kê + tổng hợp + sắp xếp + chuẩn hoá** theo taxonomy 8 nhóm (format Rule 13: `Global Pattern [A does B to X] → Y. Đúng: ...`) tại:
> **`agent/memory/global/lessons_global_normalized.md`** — **229 Global Patterns** (gồm 3 lesson user APPEND giữa chừng 2026-06-03/06-04 đã được chuẩn hoá bổ sung).
>
> Phân nhóm: 01 Process&Governance (63) · 02 Architecture&Design (41) · 03 Schema&Migration (28) · 04 CDC/Data-Pipeline (34) · 05 Config&Environment (16) · 06 Serialization&Type (12) · 07 Testing&Verification (21) · 08 Memory&Knowledge (14).
>
> File normalized là *view phái sinh* (read-optimized), re-generate định kỳ — KHÔNG thay thế audit-log này.
