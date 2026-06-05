# Requirements — MongoDB Connection Form → URL

## Origin
User: "chuyển đổi caí form connect mongo => url cho chuẩn"

Bối cảnh: dispatch-status endpoint trả error `failed to introspect mongo source ... lookup gpay-mongo: no such host` — DB row `connection_registry.host` hiện chứa full URI `mongodb://gpay-mongo:27017/?replicaSet=rs0` (vô tình đúng vì `splitHostPort` parse fail). Worker `command_handler.go:286` đã tolerant cả 2 dạng. Cần làm FE → API → DB nhất quán: MongoDB lấy URL làm "single source of truth".

## Functional Requirements
- FR-1: Form **New Connect** và **Edit Config** cho `dbKind=mongodb` chỉ hiện 1 field **MongoDB Connection URL** (full width).
- FR-2: Các field Host / Port / Username / Password / Replica Set bị ẨN cho mongodb (đã embed trong URL).
- FR-3: Field **Database** + **Topic Prefix** + **Collections** vẫn giữ (Debezium cần riêng).
- FR-4: Khi Edit, URL được seed từ `source.server_address` (BE trả full URI).
- FR-5: mysql/postgresql giữ nguyên form Host+Port+Username+Password (không nằm trong scope user).
- FR-6: Validation FE: URL phải bắt đầu `mongodb://` hoặc `mongodb+srv://`.

## Non-functional / Persistence
- NFR-1: `splitHostPort` (repo) phải EXPLICIT detect URI prefix (mongodb / postgres / mysql) → store full URI vào `connection_registry.host`, `port=NULL`. Hiện tại chỉ "vô tình đúng" do Atoi fail; làm rõ intent.
- NFR-2: KHÔNG đụng tới worker — `scanFieldsMongoSource` đã tolerant.
- NFR-3: KHÔNG migration DB — row hiện tại đã đúng format.

## Out of Scope
- mysql/postgres URL form (user chỉ nói mongo).
- Parse-on-blur populate field (đơn giản hóa: URL = single field).
- DB schema thêm column `connection_url` riêng.

## Definition of Done
- [ ] FE build PASS, form mongo chỉ show URL + Database + TopicPrefix + Collections.
- [ ] Create + Edit mongo connector → POST `mongodb.connection.string` = URL user nhập nguyên văn.
- [ ] API `parseFingerprint` (mongo) đã đọc `mongodb.connection.string` → `fp.serverAddress` (không đổi).
- [ ] Repo `splitHostPort` có URI-prefix branch, kèm comment.
- [ ] API build + test PASS, repo unit không hồi quy.
- [ ] Workspace docs đầy đủ (01/02/03/08/09 + append 05).
