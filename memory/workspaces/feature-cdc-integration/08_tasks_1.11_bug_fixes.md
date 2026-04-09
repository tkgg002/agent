# Phase 1.11 Task: Fix CDC Tồn Đọng

## Status: COMPLETED

## Context
Quét toàn bộ codebase ngày 2026-04-07, xác minh 5 tồn đọng thực tế (đã loại bỏ false positives từ doc cũ đã fix).
Giải pháp chi tiết: `09_tasks_solution_1.11.md`

## Task Checklist

### Bug 1 — P0 CRITICAL: SourceTable vs TargetTable Mismatch
- [x] **Fix `registry_service.go`**: Thêm `sourceCache` reverse index, method `GetTableConfigBySource()`
- [x] **Fix `event_handler.go`**: Dùng `GetTableConfigBySource()`, `tableConfig.TargetTable` cho SQL/mapping/batch
- [x] **Build verify**: `go build ./...` — OK

### Bug 2 — P1: Missing PATCH /api/mapping-rules/:id
- [x] **Add `UpdateStatus` handler**: `mapping_rule_handler.go`
- [x] **Register route**: `router.go` — `admin.Patch("/mapping-rules/:id", ...)`
- [x] **Build verify**: `go build ./...` — OK

### Bug 3 — P1: List API bỏ qua filter status
- [x] **Fix `List()` method**: Đọc `status`, `rule_type` query params, gọi `GetAllFiltered()`
- [x] **Build verify**: `go build ./...` — OK

### Bug 4 — P1: NATS reload payload không nhất quán
- [x] **Add `PublishReload()` helper**: `nats_client.go`
- [x] **Update 6 call sites**: `registry_handler.go` (3), `mapping_rule_handler.go` (2), `airbyte_handler.go` (1), `approval_service.go` (1)
- [x] **Build verify**: `go build ./...` — OK

### Bug 5 — P2: Rename SchemaChanges → MappingApproval
- [x] **Update `App.tsx`**: Đổi menu label
- [x] **Build verify**: `tsc --noEmit` — OK

## Definition of Done
- [x] Tất cả 3 service build OK
- [x] Mọi thay đổi được ghi vào `05_progress.md`


---
1.
http://localhost:8083/api/mapping-rules?status=approved&page=1&page_size=15
Request Method
GET
Status Code
500 Internal Server Error

2.
Request URL
http://localhost:8090/api/airbyte/sources
Referrer Policy
strict-origin-when-cross-origin
=> 500

3.
http://localhost:5173/queue => vẫn bị crash

4. 
http://localhost:5173/registry 
click Source Database => vẫn không hiện full các schema đang có ở Source Database. 

5.
http://localhost:5173/registry
khi chọn vào 1 record thì phải nhảy ra trang hiển thị danh sách các field đang đc maping. không nhảy qua /schema-changes (cho dù có filter). 

6.
Trang hiển thị danh sách các field đang đc maping => khôi phục custom mapping. để có thể thêm thủ công các mapping mới. kết hợp vừa maping tự động từ airtype, vừa thêm đc maping thủ công.

7.
Trang hiển thị danh sách các field đang đc maping => chưa có hiển thị các field defalut của hệ thống ( _raw_data, _source, _synced_at , _version, _hash, _deleted, _created_at, _updated_at)

8.
Trang hiển thị danh sách các field đang đc maping => phải có active/inactive để chọn mapping hay ko => đồng bộ trực tiếp vào airtype

9. 
hiện tại AirbyteHandler đang nằm ở cdc api, xem xet system design nó có hợp lý không. cdc worker dùng để làm gì. 

10.
http://localhost:8083/api/mapping-rules?status=pending&page=1&page_size=15
Request Method
GET
Status Code
500 Internal Server Error

11.
http://localhost:8083/api/mapping-rules?status=pending&page=1&page_size=15
=> không có tìm đc field đang có trong db source mà chưa có trong db destinations

12.
http://localhost:5173/registry -> FE
action đang là : scan fields, std, disc đang không trực quan. 

13. 
Queue phải đc hiện thị đúng với thực tế. 

----
fix lần 2

1.
http://localhost:8083/api/introspection/scan/export_jobs
Request Method
GET
Status Code
500 Internal Server Error

2.
cdc-cms-web
sao http://localhost:18000/ , http://localhost:8083 lại ko xài ở .env, check lại toàn bộ

3. 
http://localhost:5173/queue
=> cảm giác nó chỉ là thông số ảo. mới test update active mà nó re

4. 
http://localhost:5173/registry 
khi reg 1 table mới vào. nó tạo đc record nhung active đang true, để nó false.


5. cơ chế sync
http://localhost:5173/registry 
khi trên airtype chọn active 1 schema, dưới cms chưa có table mới này. 

6.
http://localhost:8083/api/registry/11/scan-fields
Request Method
POST
Status Code
400 Bad Request

7. 
http://localhost:8083/api/introspection/scan/xxx
Request Method
GET
Status Code
500 Internal Server Error
{
    "error": "failed to discover airbyte schema: decode response: invalid character '\u003c' looking for beginning of value"
}

8.
vẫn không thấy bất cứ field nào đang có trong db source (airtype) mà chưa có trong db destinations (cdc worker). cái này là core chính của phase này. 

----