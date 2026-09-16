# 00_context — Scope & Context

## Workspace
`FixBatchTransformSensitiveMasking20260907`

## Vấn đề
`POST /api/v1/source-objects/201/transform?binding_id=201`
payload `{"force":true,"force_fields":["status1"]}`

`status1` có `Sensitive = true`, `Mask Strategy = hmac` → **KHÔNG mask**, ghi plaintext vào shadow table `shadow_traitestces.export_jobs`.

## Root Cause (đã xác định từ source code)
`BatchTransformHandler` thiếu `*governance.MaskingService`.
- Handler chỉ có `hmacKey string` (raw string), không có MaskingService.
- Khi gặp sensitive field, nó gọi `metadata.BuildCastExprWithRule(rule, h.hmacKey)` → sinh SQL `encode(hmac(...)::bytea, 'sha256')` (pgcrypto).
- Đúng pattern phải là: dùng Go native `governance.MaskingService.MaskByStrategy(value, strategy)` giống `DynamicMapper.maybeMaskColumn()`.

## Affected Services
- `centralized-data-service` (BE Worker)

## Files liên quan
- `internal/handler/shadow/batch_transform_handler.go`
- `internal/service/metadata/mapping_utils.go`
- `internal/server/server_setup.go`
