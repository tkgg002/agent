# Implementation — Phase 14 Registry Surface Pruning

## Audit kết luận

`/api/registry` chưa thể xóa nguyên cụm, vì FE/operator-flow vẫn đang dùng thật các route:

- `GET /api/registry`
- `POST /api/registry`
- `PATCH /api/registry/{id}`
- `POST /api/registry/batch`
- `POST /api/registry/{id}/standardize`
- `POST /api/registry/{id}/scan-fields`
- `POST /api/registry/{id}/transform`
- `POST /api/registry/{id}/create-default-columns`
- `POST /api/registry/{id}/detect-timestamp-field`
- `GET /api/registry/{id}/dispatch-status`
- `GET /api/registry/{id}/transform-status`
- `GET /api/registry/stats`
- `GET /api/sync/health`

Các route dead/retired không còn caller:

- `GET /api/sync/reconciliation`
- `GET /api/registry/{id}/status`
- `POST /api/registry/scan-source`
- `POST /api/registry/{id}/sync`
- `GET /api/registry/{id}/jobs`
- `POST /api/registry/{id}/discover`
- `POST /api/registry/{id}/drop-gin-index`

## Thay đổi đã áp dụng

- Gỡ các route dead khỏi `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/router/router.go`
- Xóa các handler dead trong `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/api/registry_handler.go`
- Đổi swagger/comment của nhóm route còn sống từ `Table Registry` sang `Source Objects`

## Kết luận kiến trúc

`/api/registry` hiện là **compatibility surface có chọn lọc**, chưa phải API dư hoàn toàn.
Nó vẫn đang gánh:

- source-object registration transitional
- shadow maintenance commands
- async dispatch status cho operator-flow

Nên phase này chỉ prune phần dead, chưa ép xóa nguyên cụm.
