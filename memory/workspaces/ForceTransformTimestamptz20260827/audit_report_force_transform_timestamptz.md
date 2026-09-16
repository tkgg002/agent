# Audit Report — Deep Code Audit: Sensitive Masking & Timestamptz

**Workspace:** `ForceTransformTimestamptz20260827`  
**Thời gian:** 2026-08-27 13:53  
**Người thực hiện:** Lead QC & Systems Architect

---

## 1. Audit Vấn đề 1: Sensitive field `utm = "openapi_gateway"` không được che giấu (Mask)

### Truy vết Gốc rễ từ Source Code (Root Cause Analysis):
1. **Trong Database:**
   - Rule `utm` trong `cdc_system.mapping_rule_v2` có `is_sensitive_field = true`, `mask_strategy = 'hmac'`.
2. **Trong Sink Worker (Realtime CDC):**
   - `DynamicMapper.maybeMaskColumn` thực hiện hash HMAC-SHA256 trong bộ nhớ trước khi INSERT.
3. **Trong Batch Transform (`batch_transform_handler.go`):**
   - Hàm `runTransformJob` trước đây chỉ gọi:
     ```go
     castExpr := metadata.BuildCastExpr(rule.SourceField, rule.DataType)
     ```
   - `BuildCastExpr` chỉ sinh ra biểu thức SQL thuần `(_raw_data->>'utm')`.
   - `BatchTransformHandler` **hoàn toàn bỏ qua `rule.IsSensitiveField` và `rule.MaskStrategy`**.
   - Do đó, khi lệnh `UPDATE ... SET "utm" = (_raw_data->>'utm')` chạy trên PostgreSQL, nó đã bóc thẳng chuỗi plaintext `"openapi_gateway"` từ `_raw_data` và ghi đè vào cột `utm`.

### Giải pháp kỹ thuật (Implementation Fix):
1. Thêm hàm `BuildCastExprWithRule(rule mastermodel.MappingRuleV2, hmacKey string)` trong `mapping_utils.go`:
   - Nếu `rule.IsSensitiveField = true` và `rule.MaskStrategy = "hmac"` (hoặc default):
     ```sql
     CASE WHEN (castExpr) IS NULL THEN NULL ELSE encode(hmac((castExpr)::text::bytea, '<key>'::bytea, 'sha256'), 'hex') END
     ```
   - Nếu `hmacKey` rỗng, fallback sang hàm băm chuẩn `encode(sha256((castExpr)::text::bytea), 'hex')`.
2. Tiêm `cfg.MaskingHMACKey` từ file cấu hình vào `BatchTransformHandler` qua `SetHMACKey(key)`.
3. Trong `batch_transform_handler.go`, gọi `BuildCastExprWithRule(rule, h.hmacKey)`.

---

## 2. Audit Vấn đề 2: Lệch múi giờ `Timestamptz` (`2026-08-27 13:40:30.354`)

### Truy vết Gốc rễ từ Source Code (Root Cause Analysis):
1. `BuildCastExpr` trước đây với `timestamptz`:
   ```sql
   WHEN jsonb_typeof(_raw_data->'col') = 'number'
   THEN to_timestamp((NULLIF(_raw_data->>'col', ''))::NUMERIC::BIGINT / 1000.0) AT TIME ZONE 'UTC'
   ```
2. **Cơ chế nội tại của PostgreSQL:**
   - `to_timestamp(epoch_seconds)` trả về kiểu **`TIMESTAMP WITH TIME ZONE`** (timestamptz).
   - Biểu thức `timestamptz AT TIME ZONE 'UTC'` sẽ chuyển đổi `timestamptz` thành **`TIMESTAMP WITHOUT TIME ZONE`** (naive timestamp hiển thị theo giờ UTC).
   - Khi gán `TIMESTAMP WITHOUT TIME ZONE` vào cột đích có kiểu `TIMESTAMPTZ`, PostgreSQL sẽ re-cast giá trị naive đó theo session timezone hiện tại (`Asia/Ho_Chi_Minh` = `+07`).
   - Hậu quả: Giờ UTC (ví dụ 06:40:30) bị gán nhãn thành 06:40:30+07, lệch mất 7 tiếng so với thực tế 13:40:30+07.

### Giải pháp kỹ thuật (Implementation Fix):
- Xóa bỏ `AT TIME ZONE 'UTC'` trong toàn bộ các nhánh ép kiểu sang `timestamptz`.
- Giữ nguyên `to_timestamp((NULLIF(...))::NUMERIC::BIGINT / 1000.0)` để PostgreSQL trực tiếp lưu trữ điểm thời gian chính xác dạng `TIMESTAMPTZ`.

---

## 3. Bằng chứng kiểm thử tự động (Automated Verification Proofs)

| Module | Lệnh thực thi | Kết quả |
|---|---|---|
| `internal/service/metadata` | `go test -v ./internal/service/metadata` | 3/3 tests PASS (`TestBuildCastExpr_Timestamptz`, `TestBuildCastExpr_TimestampWithoutTZ`, `TestBuildCastExprWithRule_SensitiveField`) |
| `internal/handler/shadow` | `go test -v ./internal/handler/shadow` | 4/4 tests PASS (Bao gồm force mode và chunked CTE) |
| Worker Binary | `go build ./cmd/...` | Exit code 0 (PASS) |
| CMS API Binary | `go build ./cmd/...` | Exit code 0 (PASS) |
| Frontend Web | `npm run build` | Exit code 0 (PASS) |
