# 03_implementation_phase_p1 — Chi tiết kỹ thuật P1 (Worker Integration)

> **REVIEW ROUND 1 NOTES (2026-06-01)**:
> - Path prefix: `data-hub/centralized-data-service/...` (C2).
> - Bổ sung M-7b cho 3 caller missing: `dlq_handler.go`, `dlq_worker.go`, `schema_inspector.go` (C3).
> - Caller refactor theo `13_caller_inventory.md` Phase B order (priority hot-path trước).
> - Dùng `*Ctx` variant + dual-method (C4).

## M-6 — DynamicMapper integration

### File sửa: `centralized-data-service/internal/service/dynamic_mapper.go`

**Vị trí hiện tại (lines 67, 114)**:
```go
rawJSON, _ := json.Marshal(dm.maskRawData(targetTable, rawData))
```

**Sau refactor**:
```go
// Pass eventID + sourceCode để MaskingService biết audit context.
masked, err := dm.maskingSvc.MaskTableData(ctx, evt.ID, evt.SourceCode, targetTable, rawData)
if err != nil {
    return fmt.Errorf("mapper: mask: %w", err)
}
rawJSON, _ := json.Marshal(masked)
```

### Loại bỏ helper `maskRawData()` cũ (lines 123-127)
Vì MaskingService giờ tự dispatch theo strategy, không cần wrapper.

### Verify
- `go test ./internal/service -run TestDynamicMapper -v` PASS sau khi cập nhật mock.
- Integration test inject mapping_rule `mask_strategy='DROP'` cho field `password` → assert `_raw_data->>'password' IS NULL`.

---

## M-7 — BatchBuffer + ReconHeal + KafkaConsumer + ReconHandler align

### File sửa: `internal/handler/batch_buffer.go`
**Vị trí hiện tại (lines 374-383)**: `sanitizeRawData()` gọi `MaskJSONPayload()` trả về `"***"`.

**Sau**:
```go
// DLQ path vẫn cần mask để không leak khi store failed_sync_log.
// Dùng SAME registry, nhưng default strategy cho DLQ là DROP (vì failed event
// thường có thể có giá trị invalid, không đáng tin để hash).
func (b *BatchBuffer) sanitizeRawData(ctx context.Context, table string, raw []byte) ([]byte, error) {
    var data map[string]any
    if err := json.Unmarshal(raw, &data); err != nil {
        // Invalid JSON → DROP toàn bộ thay vì "***".
        return []byte("null"), nil
    }
    masked, err := b.maskingSvc.MaskTableData(ctx, "dlq", "internal", table, data)
    if err != nil {
        return nil, err
    }
    return json.Marshal(masked)
}
```

### File sửa: `internal/service/recon_heal.go`
**Vị trí (lines 771-776)**: `buildMaskedRawJSON()` đã gọi `MaskTableData()` → giờ tự dispatch strategy đúng.
Chỉ cần update test expectation.

### File sửa: `internal/handler/kafka_consumer.go`
**Vị trí (lines 1124-1125)**: tương tự → đã chạy qua MaskingService refactored. Không thay đổi logic, chỉ update assert.

### File sửa: `internal/handler/recon_handler.go:701`
**Vị trí (line 701)**: `MaskJSONPayload(table, raw)` → `MaskJSONPayloadCtx(ctx, MaskMeta{EventID:reconID}, table, raw)`. Convert string casting cẩn thận (`string(out)` thay vì cast `json.RawMessage`).

### Verify M-7
- `go test ./internal/handler -run TestBatchBuffer_Sanitize -v` PASS.
- `go test ./internal/handler -run TestReconHandler -v` PASS.
- E2E (M-9) verify failed_sync_log không chứa `"***"`.

---

## M-7b — Missing caller refactor (C3 — phát hiện round 1)

### File sửa: `internal/handler/dlq_handler.go:335`
**Hiện tại**: `return d.masking.MaskJSONPayload(table, data)` — trả `json.RawMessage` legacy.
**Sau**: `return d.masking.MaskJSONPayloadCtx(c.Context(), MaskMeta{EventID: dlqID, SourceCode: source}, table, data)`. Caller handle error.

### File sửa: `internal/service/dlq_worker.go:359`
**Hiện tại**: `masked := w.masking.MaskTableData(table, payload)`.
**Sau**:
```go
masked, err := w.masking.MaskTableDataCtx(ctx, MaskMeta{EventID: dlqMsg.ID, SourceCode: dlqMsg.Source}, table, payload)
if err != nil {
    w.logger.Error("dlq mask failed", zap.Error(err))
    return // hoặc retry policy
}
```

### File sửa: `internal/service/schema_inspector.go:211`
Theo P0/M-4b — đổi `MaskFieldSample()` thành `PreviewField()` trả `FieldSample` metadata.
Caller cũ:
```go
// before
sample := si.masking.MaskFieldSample(tableName, fieldName, value)
return sample // type any, có thể là "***"

// after — caller phải nhận FieldSample struct.
sample := si.PreviewField(ctx, tableName, fieldName, value)
// UI consumer chuyển sang render IsMasked badge thay vì hiển thị literal.
```

### Verify M-7b
- `grep -rn "MaskTableData(" data-hub/centralized-data-service/internal/ | grep -v _test.go | grep -v "MaskTableDataCtx"` → có thể vẫn còn caller legacy (giữ dual). Mục tiêu phase B (migrate dần) không phải sunset ngay.
- `grep -rn '"\*\*\*"' data-hub/centralized-data-service/internal/handler/ data-hub/centralized-data-service/internal/service/` → KHÔNG còn (trừ `text_sanitizer.go` theo ADR-008).

---

## M-8 — Audit log writer (sample rate + batch)

### File NEW: `internal/service/masking/audit_writer.go`

```go
package masking

import (
    "context"
    "math/rand"
    "time"

    "gorm.io/gorm"
    "go.uber.org/zap"
)

type AuditRecord struct {
    EventID     string
    Source      string
    Table       string
    Field       string
    Strategy    string
    KeyVersion  int16
}

type AuditWriter struct {
    db         *gorm.DB
    ch         <-chan AuditRecord
    sampleRate float64
    batchSize  int
    flushEvery time.Duration
    logger     *zap.Logger
}

func NewAuditWriter(db *gorm.DB, ch <-chan AuditRecord, rate float64, logger *zap.Logger) *AuditWriter {
    return &AuditWriter{
        db: db, ch: ch, sampleRate: rate,
        batchSize: 500, flushEvery: 5 * time.Second,
        logger: logger,
    }
}

func (w *AuditWriter) Run(ctx context.Context) {
    buf := make([]AuditRecord, 0, w.batchSize)
    tick := time.NewTicker(w.flushEvery)
    defer tick.Stop()

    flush := func() {
        if len(buf) == 0 { return }
        if err := w.db.WithContext(ctx).
            Table("cdc_system.mask_audit_log").
            Create(&buf).Error; err != nil {
            w.logger.Error("audit_writer: flush", zap.Error(err))
        }
        buf = buf[:0]
    }

    for {
        select {
        case <-ctx.Done():
            flush()
            return
        case rec := <-w.ch:
            if rand.Float64() > w.sampleRate { continue } // sample
            buf = append(buf, rec)
            if len(buf) >= w.batchSize { flush() }
        case <-tick.C:
            flush()
        }
    }
}
```

### Config knob
```yaml
masking:
  enabled: true
  defaultKeyVersion: 1
  auditSampleRate: 0.01  # 1% default, có thể bump 1.0 khi điều tra
```

### Verify
- Set `auditSampleRate: 1.0` chạy 100 event → `SELECT COUNT(*) FROM cdc_system.mask_audit_log` ≈ 100 record/field.

---

## M-9 — E2E integration test với testcontainers

### File NEW: `internal/service/masking_e2e_test.go`

```go
//go:build integration

package service_test

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"

    // ... import service + masking ...
)

func TestMaskingE2E_FullPipeline(t *testing.T) {
    ctx := context.Background()
    pgC, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("cdc_test"),
        postgres.WithUsername("test"), postgres.WithPassword("test"),
    )
    require.NoError(t, err)
    defer pgC.Terminate(ctx)

    dsn, _ := pgC.ConnectionString(ctx, "sslmode=disable")
    db := openGorm(t, dsn)
    applyMigrations(t, db, "../migrations/")

    // Seed mapping rule
    db.Exec(`INSERT INTO cdc_system.cdc_mapping_rules
        (target_table, target_column, mask_strategy, mask_options, mask_key_version)
        VALUES
        ('users', 'password', 'DROP', '{}', 1),
        ('users', 'cccd', 'HASH_HMAC', '{"key_ref":"v1"}', 1),
        ('users', 'phone', 'PARTIAL', '{"prefix":0,"suffix":3,"placeholder":"*"}', 1)
    `)

    os.Setenv("MASKING_HMAC_KEY_V1", "this_is_a_test_key_32_bytes_long!!")
    svc := buildMaskingService(t, db)

    out, err := svc.MaskTableData(ctx, "evt-1", "mongo-src", "users", map[string]any{
        "password": "secret123",
        "cccd":     "001234567890",
        "phone":    "0901234567",
        "name":     "Alice",
    })
    require.NoError(t, err)

    // Assertion compliance
    require.Nil(t, out["password"], "DROP password phải nil")
    require.NotEqual(t, "***", out["password"])
    require.Len(t, out["cccd"], 64, "HMAC 64 chars")
    require.Equal(t, "*******567", out["phone"])
    require.Equal(t, "Alice", out["name"], "Non-sensitive giữ nguyên")

    // Verify audit log
    time.Sleep(6 * time.Second) // chờ flush
    var count int64
    db.Table("cdc_system.mask_audit_log").Count(&count)
    require.GreaterOrEqual(t, count, int64(3), "audit log có ≥ 3 record")
}
```

### Verify
- `go test -tags=integration ./internal/service/ -run TestMaskingE2E -v` PASS.
- Check Postgres `_raw_data` column sau pipeline: không có chuỗi `***` literal.

---

## Composite impact P1
- Worker pipeline end-to-end dùng Strategy engine, không còn `"***"` literal.
- Audit log generate evidence compliance cho thanh tra.
- E2E test chứng minh full flow PASS với 4 strategy.
