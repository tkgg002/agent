# 08_tasks — Fix Batch Transform Sensitive Masking

## Workspace
`FixBatchTransformSensitiveMasking20260907`

## Task Checklist (Muscle sẽ execute sau APPROVE)

### T1 — `batch_transform_handler.go`
- [ ] T1.1: Thêm `import "centralized-data-service/internal/service/governance"` (nếu chưa có)
- [ ] T1.2: Thêm field `maskingSvc *governance.MaskingService` vào struct `BatchTransformHandler`
- [ ] T1.3: Thêm method `SetMaskingService(svc *governance.MaskingService)`
- [ ] T1.4: Trong `runTransformJob`: khai báo `var sensitiveRules []mastermodel.MappingRuleV2`
- [ ] T1.5: Trong vòng for xây dựng `setClauses`: thêm nhánh tách sensitive rules ra khỏi bulk SQL
- [ ] T1.6: Viết hàm `runSensitiveMasking(ctx, execDB, schemaName, pureTable, pkCol, sensitiveRules, whereExpr) (int64, error)`
- [ ] T1.7: Sau chunked bulk loop: gọi `runSensitiveMasking` nếu `len(sensitiveRules) > 0 && h.maskingSvc != nil`

### T2 — `mapping_utils.go`
- [ ] T2.1: Đơn giản hóa `BuildCastExprWithRule`: đổi param `hmacKey` thành `_`, delegate về `BuildCastExpr`
- [ ] T2.2: Xoá toàn bộ logic if/else strategy (hmac/aes/none) khỏi hàm này

### T3 — `server_setup.go`
- [ ] T3.1: Thêm dòng `batchTransformHandler.SetMaskingService(maskingSvc)` sau `SetHMACKey`

### T4 — Verification
- [ ] T4.1: `go build ./cmd/...` → Exit code 0
- [ ] T4.2: `go test -v ./internal/handler/shadow/... ./internal/service/metadata/...` → PASS
- [ ] T4.3: Gọi API force transform, query shadow DB → `status1` = HMAC hex 64 chars
- [ ] T4.4: Update `05_progress.md` với timestamp thực tế

## Files sẽ thay đổi
| File | Lines ± |
|---|---|
| `internal/handler/shadow/batch_transform_handler.go` | +~80 |
| `internal/service/metadata/mapping_utils.go` | -9, +2 |
| `internal/server/server_setup.go` | +1 |
