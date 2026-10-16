# 01 Requirements — JSON Subset type/presence

## Mục tiêu
Thêm chế độ **"JSON Subset type/presence"** vào modal "Tạo Master Mapping Rule" tại
`/masters/payment_bills/mappings?binding_id=<id>`.

## Yêu cầu
1. **type + presence = 1 output thống nhất** (KHÔNG tách 2 mode).
   - `present`: `{ "type": "<bsonType>", "presence": "present" }`
   - `null`: `{ "type": "null", "presence": "null" }`
   - `missing`: `{ "presence": "missing" }` (không có type)
2. **Không hardcode CORE_FIELDS** — user đã chọn field từ Shadow UI.
3. **Không cheat `_id`/`transId` validation** vào hàm subset — tầng ingest lo.
4. **Minimal Impact** — theo pattern hiện tại (Switch toggle + Select tags + POST cùng endpoint).

## Ví dụ output
Document: `{_id: ObjectId("..."), transId: "TRX", value: 50000, serviceCode: null}`
SourceFields: `["transId", "value", "serviceCode", "channelId"]`

```json
{
  "transId":     { "type": "string", "presence": "present" },
  "value":       { "type": "int32",  "presence": "present" },
  "serviceCode": { "type": "null",   "presence": "null" },
  "channelId":   { "presence": "missing" }
}
```
