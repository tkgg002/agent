# 00_context — Bug: snapshot.v2 fail vì `Host` chứa full URI nhưng GetSourceDSN không support

> **Workspace**: `bug-snapshot-v2-host-uri-2026-05-21`
> **Repo**: `centralized-data-service` (worker plane)
> **Trigger**: User báo "check lại vụ snapshot v2" + log mới `1779347177` (2026-05-21 14:06:17 ICT) tiếp tục fail trên binary worker hiện tại (PID 37577, started 14:03).

## Evidence Log

```
{"level":"error","ts":1779347177.19167,"msg":"snapshot.v2 run failed",
 "source_object_id":18,"trace_id":"fe-snapshot-4cb066fd-...",
 "error":"resolve mongo URI for \"goopay-pbs\": cannot resolve DSN for connection
          \"goopay-pbs\" (engine=mongodb): no usable DSN in secret_ref nor host/port fields"}
```

Nhưng 6 phút trước đó, `scan-fields` cho cùng `goopay-pbs` (registry_id=18) **resolve thành công** qua path `dispatch_path: "fallback"`:

```
{"level":"info","msg":"scan-fields mongo introspect",
 "connection_code":"goopay-pbs","dispatch_path":"fallback",
 "sanitized_dsn":"mongodb://***@10.200.187.11:27017,10.200.187.12:27017,10.200.187.13:27017/?replicaSet=goopay&authSource=admin"}
```

Cùng connection, cùng worker, 2 result trái ngược → **2 code path resolve DSN khác nhau**.

## Code Paths

| Caller | File | Method gọi | Behavior khi `Host` chứa full URI |
|--------|------|-----------|-------------------------------------|
| `scan-fields` | `internal/handler/command_handler.go:321-322` | Logic inline | `if strings.HasPrefix(hostRaw, "mongodb://") → dsn = hostRaw` → ✅ PASS |
| `snapshot.v2` | `internal/handler/snapshot_runner_handler.go:168` | `registrySvc.GetSourceDSN(ctx, conn.ConnectionCode)` | `GetSourceDSN` không check `Host` chứa URI → ❌ FAIL |

## Root Cause

`MetadataRegistryService.GetSourceDSN` (`internal/service/metadata_registry_service.go:341-370`) chỉ check `SecretRef` qua `tryPlainDSN` / `tryEnvPointer`, không bao giờ thử `tryPlainDSN(*conn.Host)`. Layer 3 (`buildDSNFromFields`) yêu cầu `conn.Port != nil` — nhưng row mà cdc-cms UI ghi vào với `Host = full URI` thường để `Port = NULL` (vì URI đã chứa port). → Layer 3 cũng skip → error.

Trong khi đó, caller `scanFieldsMongoSource` đã có logic riêng (line 314-330) handle exactly trường hợp này — **logic bị duplicate** ở 2 chỗ với độ phủ khác nhau.
