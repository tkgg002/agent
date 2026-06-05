# 09_solution_proposed — Consolidate DSN resolver về 1 nguồn truth duy nhất

> **Pattern (user verb 2026-05-21)**: "ko viêt vào 1 nơi để khi nào cần thì dùng lại, pattern phải vậy mới clean chứ" → DRY resolver, single source of truth.

## Phạm vi sửa

### Edit #1 — Extend `GetSourceDSN` với `Host`-as-URI layer

File: `centralized-data-service/internal/service/metadata_registry_service.go`

Trong `GetSourceDSN` (line 341), CHÈN 2 layer mới SAU `ApplyConnectionOverride` (line 352) và TRƯỚC `tryPlainDSN(SecretRef)`:

```go
// Some connections (cdc-cms "create source" UI flow) write the full
// connection URI into the `host` column and leave `port` NULL. Reuse
// the same scheme detectors as for secret_ref so the resolver path is
// symmetric across columns.
if conn.Host != nil {
    if dsn := tryPlainDSN(*conn.Host); dsn != "" {
        return dsn, nil
    }
    if dsn := tryEnvPointer(*conn.Host); dsn != "" {
        return dsn, nil
    }
}
```

→ Layer ordering mới: override → host-as-URI → host-as-env-ptr → secret-as-URI → secret-as-env-ptr → build-from-fields → AES.

### Edit #2 — `scanFieldsMongoSource` dùng chung resolver (xoá duplicate)

File: `centralized-data-service/internal/handler/command_handler.go`

Block `if uri, ok := service.ApplyConnectionOverride(...); ok { dsn = uri } else { ... fmt.Sprintf("mongodb://%s:%d/", ...) }` (line 310-330) → thay bằng:

```go
dsn, err := h.metadata.GetSourceDSN(ctx, conn.ConnectionCode)
if err != nil || strings.TrimSpace(dsn) == "" {
    return 0, 0, fmt.Errorf("resolve DSN for connection_code=%s (id=%d): %w",
        conn.ConnectionCode, registry.SourceConnectionID, err)
}
```

Block `dispatchPath := "fallback"; if _, hit := service.ApplyConnectionOverride(...); hit { dispatchPath = "override" }` GIỮ nguyên — vẫn cần cho log diagnostics.

### Edit #3 — Unit test mới cho host-as-URI path

File: `centralized-data-service/internal/service/metadata_registry_dsn_test.go` (đã tồn tại)

Append test `TestGetSourceDSN_HostFullURI` (mock connectionRepo) — cover:
- `Host = "mongodb://h:27017/"`, `Port = nil` → return as-is.
- `Host = "postgres://x:5432/db"`, `Port = nil` → return as-is.
- `Host = "env:MONGO_URI"`, env set → return env value.

## Verify gates (BẮT BUỘC chạy trước khi đóng task)

1. `go build ./...` (worker repo) → EXIT 0.
2. `go vet ./...` → EXIT 0.
3. `go test ./internal/service/... ./internal/handler/...` → EXIT 0.
4. **Runtime gate**: User Ctrl-C worker hiện tại + `go run cmd/worker/main.go` lại (vì `go run` cache binary).
5. **Smoke gate**: User trigger snapshot.v2 cho `source_object_id=18` (goopay-pbs) qua FE → log expect:
   - `snapshot.v2 started ... connection_code=goopay-pbs ...`
   - NO `snapshot.v2 run failed`.
   - DB row `cdc_system.snapshot_progress` cuối có `status = 'done'`.

## Non-goals

- ❌ KHÔNG đụng `provisioning_step_handlers.go:pickSourceDSN` — nó đã có shape tốt hơn (4-param fallback chain với env safety net), không trùng pattern này.
- ❌ KHÔNG remove `dispatchPath` log — vẫn hữu ích để diagnose override vs fallback.
- ❌ KHÔNG migrate seed DB — chỉ fix resolver, data shape giữ nguyên.
- ❌ KHÔNG đổi interface `MetadataRegistry`.

## Risk & rollback

- **Risk**: thấp. 2 layer mới được chèn SAU override + TRƯỚC tryPlainDSN(SecretRef) → connection nào có valid SecretRef vẫn behave như cũ. Connection có Host=URI (đang silent-fail) → fixed.
- **Rollback**: `git revert` hoặc xoá 2 layer + revert command_handler block.

## Lesson (sẽ APPEND `lessons.md` sau khi verify)

**Global Pattern [A duplicates resolver R inline trong caller C để bypass limitation của shared resolver S]** → Result: khi convention input mở rộng, S không cover được nhưng C thì có → cùng input gây 2 outcome khác nhau, debug khó. Đúng: [Mọi caller C của resolver S phải gọi S; nếu C cần convention chưa cover, mở rộng S thay vì duplicate logic inline. "Single source of truth" cho mọi cross-cutting concern (DSN resolve, auth, mask, etc.)].
