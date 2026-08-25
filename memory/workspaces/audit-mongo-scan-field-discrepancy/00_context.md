# 00_context.md - Context & Bối cảnh: Lỗi Lệch Schema & Sai Type khi Scan Field MongoDB

## 1. Bối cảnh
Người dùng thực hiện tính năng **Scan Fields** trên giao diện CDC CMS Web cho một đối tượng Source MongoDB (hoặc bảng Shadow tương ứng). Kết quả trả về trên bảng review mapping hiển thị:
- `createdAt` -> Data Type Target: `JSONB`
- `extraData` -> Data Type Target: `JSONB`
- `_id` -> Data Type Target: `JSONB`
- `id` -> Data Type Target: `TEXT`
- `requestData` -> Data Type Target: `JSONB`
- `requestId` -> Data Type Target: `TEXT`
- `requestType` -> Data Type Target: `TEXT`
- `responseData` -> Data Type Target: `JSONB`
- `status` -> Data Type Target: `TEXT`
- `updatedAt` -> Data Type Target: `JSONB`

Trong khi đó, tài liệu mẫu (sample document) thực tế trên cơ sở dữ liệu MongoDB mà người dùng đang đối chiếu lại có cấu trúc:
```json
{
  "_id": { "$oid": "69fc0e8d9697ea33e58afa7b" },
  "id": "f8b8b295-12e0-4e0d-b2ab-9df4b27a6be1",
  "bankTransactionId": "GOOP2605071001",
  "createdAt": { "$date": "2026-05-07T04:01:17.830Z" },
  "extraData": {},
  "logs": [
    {
      "step": "BANK_TRANSFER",
      "time": { "$date": "2026-05-07T04:01:17.822Z" },
      "success": true,
      "error": null,
      "request": { "payload": { ... } },
      "response": { "payload": { ... } },
      "requestId": "GOOP2605071001"
    }
  ],
  "requestId": "BANK_TRANSFER-386859015032",
  "requestType": "BANK_TRANSFER",
  "status": "SUCCESS",
  "updatedAt": { "$date": "2026-05-07T04:01:17.830Z" }
}
```

## 2. Các điểm bất thường cần giải trình (Discrepancies)
1. **Lệch trường (Missing & Unexpected Fields)**:
   - Trong DB sample có `bankTransactionId` và `logs` nhưng bảng Scan Fields **hoàn toàn không có**.
   - Trong bảng Scan Fields lại xuất hiện `requestData` và `responseData` trong khi DB sample **không hề có** 2 trường này.
2. **Sai lệch kiểu dữ liệu (Type Inference Distortion)**:
   - Các trường thời gian (`createdAt`, `updatedAt`) và khóa chính Mongo (`_id`) bị suy luận thành kiểu **`JSONB`** thay vì `TIMESTAMPTZ` và `TEXT` / `VARCHAR(24)`.

## 3. Phạm vi Audit & Thành phần liên quan
- Frontend: `cdc-cms-web/src/pages/MappingFieldsPage.tsx`, `cdc-cms-web/src/components/TableRegistry.tsx`
- Backend API: `cdc-cms-service/internal/api/system/introspection_handler.go`, `cdc-cms-service/internal/api/source/source_object_actions_handler.go`
- Worker Core:
  - `centralized-data-service/internal/handler/source/discover_handler.go`
  - `centralized-data-service/internal/handler/source/discover_handler_mongo.go`
  - `centralized-data-service/internal/handler/source/discover_handler_utils.go`
  - `centralized-data-service/internal/service/source/source_router.go` (`InferTypeFromRawData`)
  - `centralized-data-service/internal/service/source/mongo_introspection.go` (`IntrospectCollection`)
  - `centralized-data-service/internal/service/source/scan_service.go` (`ScanRawData`)
