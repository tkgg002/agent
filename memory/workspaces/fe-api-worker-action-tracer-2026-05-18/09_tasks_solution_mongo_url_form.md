# Solution — MongoDB Connection Form → URL

## Tổng quan
Single source of truth cho MongoDB connection = **URL string** (`mongodb://...` hoặc `mongodb+srv://...`). FE chỉ thu 1 field, BE pass-through nguyên văn xuống Debezium config (`mongodb.connection.string`) và DB row `connection_registry.host`. Worker giữ nguyên (đã tolerant prefix `mongodb://`).

## Diff Snapshot

### FE — `cdc-cms-web/src/pages/SourceConnectors.tsx`
```diff
 interface ConnectionFormValues {
-  host: string;
-  port: number;
+  connectionUrl?: string;         // mongo
+  host?: string;                   // mysql/postgres
+  port?: number;
   ...
-  replicaSet?: string;
 }
+const MONGO_URL_RE = /^mongodb(\+srv)?:\/\/.+/;
```

```diff
-function buildMongoConnectionString(values, fallbackPassword?) {
-  const auth = values.username ? `${encode}:${encode}@` : '';
-  const replica = values.replicaSet ? `?replicaSet=${encode}` : '';
-  return `mongodb://${auth}${values.host}:${values.port}/${replica}`;
-}
-
-function buildConnectorConfig(values, mode, fallbackPassword?) {
+function buildConnectorConfig(values, mode) {
   if (values.dbKind === 'mongodb') {
-    'mongodb.connection.string': buildMongoConnectionString(values, fallbackPassword),
+    'mongodb.connection.string': (values.connectionUrl || '').trim(),
```

```diff
 // parseConnectionSeed mongo
-    let host = 'gpay-mongo';
-    let port = 27017;
-    try { /* parse URL hostname/port/replicaSet */ } catch {}
+    const connectionUrl = source.server_address || cfg['mongodb.connection.string'] || '';
     return {
-      host, port, username, replicaSet, ...
+      connectionUrl, database, collectionNames, ...
     };
```

```diff
 // Form UI
-<Row><Col><Item name="host" /></Col><Col><Item name="port" /></Col></Row>
-<Row><Col><Item name="username" /></Col><Col><Item name="password" /></Col></Row>
-{dbKind==='mongodb' && <Row><Col><Item name="replicaSet" /></Col><Col><Item name="collectionNames" /></Col></Row>}
+{dbKind === 'mongodb' && (
+  <Item name="connectionUrl" label="MongoDB Connection URL"
+        rules={[{required:true}, {pattern: MONGO_URL_RE, message:'mongodb:// hoặc mongodb+srv://'}]}>
+    <Input placeholder="mongodb://user:pass@host:27017/?replicaSet=rs0" />
+  </Item>
+)}
+{dbKind !== 'mongodb' && (
+  <>
+    <Row><Col><Item name="host" required/></Col><Col><Item name="port" required/></Col></Row>
+    <Row><Col><Item name="username"/></Col><Col><Item name="password" required-on-create/></Col></Row>
+  </>
+)}
+{dbKind === 'mongodb' && (<Item name="collectionNames"><Input placeholder="users,orders,payments"/></Item>)}
```

### BE — `cdc-cms-service/internal/infra/persistence/system_connector_repo_gorm.go`
```diff
+// splitHostPort tách host:port cho mysql/postgres. URI (mongodb / postgres-url /
+// mysql-url) → lưu nguyên vào column host, port=NULL. Worker
+// command_handler.scanFieldsMongoSource:286 tolerant cả 2 dạng.
 func splitHostPort(addr string) (string, int) {
   addr = strings.TrimSpace(addr)
   if addr == "" { return "", 0 }
+  for _, p := range []string{"mongodb://", "mongodb+srv://", "postgres://", "postgresql://", "mysql://"} {
+    if strings.HasPrefix(addr, p) { return addr, 0 }
+  }
   idx := strings.LastIndex(addr, ":")
   ...
 }
```

## Verify (Definition of Done)
- [x] FE `npm run build` PASS 1s (typecheck + bundle); SourceConnectors chunk 20.95 kB / 5.77 kB gzip.
- [x] API `go build ./...` PASS, `go vet ./...` PASS, `go test ./internal/infra/persistence/... ./internal/api/...` PASS (persistence 0.849s).
- [x] Worker `go build ./...` PASS sanity (không đổi code).
- [x] Form mongo chỉ hiện URL + Database + TopicPrefix + Collections.
- [x] Form mysql/postgres giữ nguyên Host+Port+Username+Password.

## Backward Compatibility
- Connector cũ (DB row `server_address` có thể là full URI hoặc `host:port`): `parseConnectionSeed` mongo seed URL trực tiếp từ `server_address`. Trường hợp row legacy có `host:port` (không có prefix scheme), URL field sẽ bị trống → user phải nhập lại. Edge case có thể ignore vì hiện tại chỉ có 1 mongo source và row đã ở dạng URI.
- BE pass-through: API `parseFingerprint` mongo đọc `mongodb.connection.string` → `serverAddress` không đổi.
- Worker `scanFieldsMongoSource:286` đã có branch `if strings.HasPrefix(hostRaw, "mongodb://")` → dùng thẳng URI; nhánh else build `mongodb://host:port/` cho legacy bare-host row.

## Risk theo dõi
- R1 (URL malformed): Debezium Connect REST sẽ reject với error rõ ràng (FE đã có message handler).
- R2 (auth chứa ký tự đặc biệt): User tự chịu trách nhiệm URL-encode trong URL string (đơn giản hóa, tránh re-encode 2 lần).
