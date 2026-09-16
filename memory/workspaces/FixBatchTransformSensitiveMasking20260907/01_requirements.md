# 01_requirements — Fix Batch Transform Sensitive Masking

## Task ID
`FixBatchTransformSensitiveMasking20260907`

## Loại task
**Hotfix** (Micro-task)

## Vấn đề cụ thể
`POST /api/v1/source-objects/201/transform?binding_id=201`
payload: `{"force":true,"force_fields":["status1"]}`
- `status1`: `Sensitive = true`, `Mask Strategy = hmac`
- **Kết quả thực tế**: field `status1` được ghi plaintext vào `shadow_traitestces.export_jobs` — không mask.
- **Kết quả kỳ vọng**: field `status1` phải được hash thành HMAC-SHA256 hex (64 chars).

## Root Cause (đã xác nhận từ source code)
`BatchTransformHandler` thiếu `*governance.MaskingService`:
- Struct chỉ có `hmacKey string` (line 47), không có `MaskingService`.
- Gọi `BuildCastExprWithRule(rule, h.hmacKey)` → sinh SQL `encode(hmac(...)::bytea, 'sha256')` (pgcrypto, không phải Go native).
- `DynamicMapper` (real-time) dùng `masking.MaskByStrategy(value, strategy)` → Go `crypto/hmac` + `crypto/aes` — đây là pattern đúng.
- Với `aes_gcm` strategy: `BuildCastExprWithRule` `return expr` → ghi plaintext hoàn toàn.

## Requirements

### R1 — Wiring MaskingService vào BatchTransformHandler
`BatchTransformHandler` phải được inject `*governance.MaskingService` để gọi Go native masking.

### R2 — Tách sensitive fields khỏi bulk SQL loop
Sensitive fields (IsSensitiveField=true, strategy ≠ none) phải được xử lý riêng, KHÔNG đưa vào bulk SQL UPDATE.

### R3 — Per-row Go masking cho sensitive fields
Sau bulk SQL phase: fetch `_raw_data` per-row → `MaskByStrategy(value, strategy)` → UPDATE từng row.
Bao gồm cả `hmac` lẫn `aes_gcm` strategy.

### R4 — Backward Compatibility
- Không thay đổi signature hàm `BuildCastExprWithRule` (giữ 2 params).
- Không ảnh hưởng real-time CDC path (`DynamicMapper`, `MaskingService`).
- Không thay đổi NATS payload schema.
- Khi `maskingSvc == nil` (tests cũ): skip sensitive masking phase, không crash.

### R5 — Server wiring
`server_setup.go` phải wire `maskingSvc` vào `batchTransformHandler.SetMaskingService(maskingSvc)`.

## Definition of Done
- [ ] `go build ./cmd/...` PASS
- [ ] `go test ./internal/handler/shadow/... ./internal/service/metadata/...` PASS
- [ ] Gọi API force transform với `status1` → shadow DB ghi HMAC hex 64 chars, không phải plaintext
- [ ] `aes_gcm` strategy cũng hoạt động đúng (encrypt, không plaintext)
