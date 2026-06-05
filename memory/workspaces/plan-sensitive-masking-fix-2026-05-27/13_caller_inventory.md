# 13_caller_inventory — Masking Function Call Sites

> Created: 2026-06-01 (review round 1).
> Mục tiêu: liệt kê 22 call-site của masking function — checklist refactor đảm bảo không bỏ sót (R-03, R-04).

## Source: `data-hub/centralized-data-service/internal/service/masking_service.go`

| Function | Signature hiện tại | Trạng thái sau refactor |
|---|---|---|
| `MaskTableData(table, data)` | `(string, map[string]any) map[string]any` | **DUAL**: giữ legacy + thêm `MaskTableDataCtx(ctx, meta, table, data) (map[string]any, error)` |
| `MaskJSONPayload(table, data)` | `(string, []byte) json.RawMessage` | **DUAL**: thêm `MaskJSONPayloadCtx(ctx, meta, table, data) (json.RawMessage, error)` |
| `MaskFieldSample(table, field, value)` | `(string, string, any) any` | **REFACTOR**: cho preview path — return type+length info, không value (M-4b mới) |
| `maskMapRecursive` (internal) | recursive map walker | **GIỮ**: dispatch leaf qua Strategy registry |
| `maskAnyRecursive` (internal) | recursive any walker | **GIỮ**: chỉ thay branch `return "***"` → dispatch strategy |

## Tất cả call-site (22 — đã `grep -rn` xác nhận)

### Production code (15 call-site, 8 file)

| # | File | Line | Function call | Phase | Status |
|---|---|---|---|---|---|
| 1 | `internal/handler/dlq_handler.go` | 335 | `MaskJSONPayload(table, data)` | **P1/M-7b** | ⬜ TODO (PLAN MISSING — phát hiện round 1) |
| 2 | `internal/handler/batch_buffer.go` | 412 | `MaskJSONPayload(tableName, raw)` | P1/M-7 | ⬜ |
| 3 | `internal/handler/recon_handler.go` | 701 | `MaskJSONPayload(table, raw)` | P1/M-7 | ⬜ |
| 4 | `internal/handler/kafka_consumer.go` | 1390 | `MaskJSONPayload(table, raw)` | P1/M-7 | ⬜ |
| 5 | `internal/service/schema_inspector.go` | 211 | `MaskFieldSample(table, field, value)` | **P0/M-4b** | ⬜ TODO (PLAN MISSING) |
| 6 | `internal/service/recon_heal.go` | 807 | `MaskTableData(targetTable, data)` | P1/M-7 (đã có) | ⬜ |
| 7 | `internal/service/dlq_worker.go` | 359 | `MaskTableData(table, payload)` | **P1/M-7b** | ⬜ TODO (PLAN MISSING) |
| 8 | `internal/service/dynamic_mapper.go` | 67 | `dm.maskRawData(targetTable, rawData)` | P1/M-6 | ⬜ |
| 9 | `internal/service/dynamic_mapper.go` | 114 | `dm.maskRawData(targetTable, rawData)` | P1/M-6 | ⬜ |
| 10 | `internal/service/dynamic_mapper.go` | 123 | `func maskRawData(...)` (helper định nghĩa) | P1/M-6 (xoá wrapper) | ⬜ |
| 11 | `internal/service/dynamic_mapper.go` | 127 | `dm.masking.MaskTableData(targetTable, rawData)` | P1/M-6 | ⬜ |
| 12 | `internal/service/masking_service.go` | 55 | `MaskTableData(...)` (định nghĩa) | P0/M-4 | ⬜ |
| 13 | `internal/service/masking_service.go` | 66 | `MaskJSONPayload(...)` (định nghĩa) | P0/M-4 | ⬜ |
| 14 | `internal/service/masking_service.go` | 89 | `MaskFieldSample(...)` (định nghĩa) | **P0/M-4b** | ⬜ |
| 15 | `internal/service/masking_service.go` | 129, 141 | `maskMapRecursive`, `maskAnyRecursive` (định nghĩa) | P0/M-4 | ⬜ |

### Test code (7 call-site)

| # | File | Line | Function call | Phase | Status |
|---|---|---|---|---|---|
| 16 | `internal/handler/batch_buffer_test.go` | ? | assert `"***"` | P0/M-5 (cập nhật assertion) | ⬜ |
| 17 | `internal/handler/recon_handler_test.go` | ? | assert `"***"` | P0/M-5 | ⬜ |
| 18 | `internal/service/text_sanitizer_test.go` | ? | assert `"***"` | (giữ — log path) | ✓ N/A |
| 19 | `internal/service/masking_service_test.go` (NEW) | - | strategy test | P0/M-5 | ⬜ |
| 20 | `internal/service/masking_e2e_test.go` (NEW) | - | E2E full pipeline | P1/M-9 | ⬜ |
| 21–22 | (rserved cho test caller chưa khám phá) | - | - | - | ⬜ |

## Refactor strategy chi tiết

### Phase A — Bổ sung dual-method (không break caller)

```go
// File: internal/service/masking_service.go

// LEGACY — giữ nguyên chữ ký, route qua strategy với empty meta.
func (ms *MaskingService) MaskTableData(table string, data map[string]any) map[string]any {
    out, _ := ms.MaskTableDataCtx(context.Background(), MaskMeta{}, table, data)
    return out
}

// MỚI — đầy đủ ctx + meta + error.
func (ms *MaskingService) MaskTableDataCtx(
    ctx context.Context, meta MaskMeta, table string, data map[string]any,
) (map[string]any, error) {
    return ms.applyStrategy(ctx, meta, table, data)
}

// applyStrategy walker — recursive, giải quyết R-05 (nested object).
func (ms *MaskingService) applyStrategy(
    ctx context.Context, meta MaskMeta, table string, value any,
) (any, error) {
    switch v := value.(type) {
    case map[string]any:
        out := make(map[string]any, len(v))
        for k, child := range v {
            rule := ms.ruleFor(table, k)
            if rule.Strategy == "NONE" {
                // Đệ quy vào nested map/array, chứ không chỉ giữ nguyên.
                processed, err := ms.applyStrategy(ctx, meta, table, child)
                if err != nil { return nil, err }
                out[k] = processed
                continue
            }
            res, err := ms.dispatch(ctx, meta, rule, k, child)
            if err != nil { return nil, err }
            if res.ShouldDrop {
                out[k] = nil
            } else {
                out[k] = res.Value
            }
        }
        return out, nil
    case []any:
        out := make([]any, len(v))
        for i, item := range v {
            processed, err := ms.applyStrategy(ctx, meta, table, item)
            if err != nil { return nil, err }
            out[i] = processed
        }
        return out, nil
    default:
        return v, nil
    }
}
```

### Phase B — Gradually migrate caller

Mỗi caller chuyển sang `*Ctx` variant một file/PR, không big-bang.

| Order | Caller | Lý do ưu tiên |
|---|---|---|
| 1 | `dynamic_mapper.go` | Hot-path chính, throughput cao nhất |
| 2 | `kafka_consumer.go` | Hot-path event ingestion |
| 3 | `batch_buffer.go` | Buffer/DLQ path |
| 4 | `recon_heal.go` | Recon background job |
| 5 | `recon_handler.go` | API handler manual recon |
| 6 | `dlq_worker.go` | DLQ replay |
| 7 | `dlq_handler.go` | DLQ API |
| 8 | `schema_inspector.go` | Schema preview (M-4b refactor) |

### Phase C — Sunset legacy (sau khi 0 caller dùng)

Sau 1 release cycle ổn định, xoá `MaskTableData()` & `MaskJSONPayload()` legacy. Để lại deprecation comment 1 release trước khi remove.

## `MaskFieldSample` xử lý đặc biệt (M-4b)

`schema_inspector.go:211` dùng để preview field value trong UI schema explorer. Hiện tại trả `"***"` cho sensitive field.

**Refactor**: thay vì trả masked value, trả metadata:

```go
type FieldSample struct {
    Type      string  `json:"type"`
    Length    int     `json:"length"`
    IsMasked  bool    `json:"is_masked"`
    Strategy  string  `json:"strategy,omitempty"`
    Sample    *string `json:"sample,omitempty"` // nil nếu IsMasked = true
}
```

→ UI hiển thị: "string, 12 chars, MASKED via HASH_HMAC" thay vì literal `"***"`.

## Verify checklist sau khi refactor toàn bộ

```bash
# 1. Không còn caller nào dùng method legacy (sau phase C).
grep -rn "MaskTableData(" data-hub/centralized-data-service/internal/ \
  | grep -v "_test.go" | grep -v "MaskTableDataCtx"
# Expected: 0 result.

# 2. Không còn "***" literal trong production code path.
grep -rn '"\*\*\*"' data-hub/centralized-data-service/internal/service/masking_service.go
# Expected: 0 result.

# 3. Nested object test PASS (regression guard cho R-05).
go test ./internal/service -run TestMaskingService_NestedObject -v

# 4. All 22 call-site đã được refactor.
# Đếm checkbox ✓ trong file này = 22.
```
