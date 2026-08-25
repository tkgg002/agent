# 00_context.md — Scope & Context (Fix Schema-Qualified MasterTable & GetMasterDB)

## 1. Bối cảnh hệ thống (Background)
Hệ thống CDC Data Hub bao gồm:
- **`cdc-cms-service` (CMS):** Giao diện quản trị, API lập lịch (`/api/v1/schedules`), Dispatcher trigger RunNow, REST endpoint quản lý metadata.
- **`centralized-data-service` (CDS Worker):** Core engine thực thi CDC: Shadow Ingest -> Transmute Worker -> Master DB.
- **NATS Message Bus:** Kênh truyền tin bất đồng bộ (`cdc.cmd.transmute`, `cdc.evt.transmute.completed`).

## 2. Vấn đề phát sinh (Incident Trigger)
1. **Sự cố RunNow:** Khi kích hoạt RunNow cho bảng `bank_requests` thuộc schema `master_bidv_connector_service`, telemetry log ghi nhận `rows_updated=2000` nhưng bảng đích ở Master DB rỗng.
2. **Nguyên nhân gốc (Root Cause):**
   - Tên bảng được truyền đi ở dạng không tường minh (`"bank_requests"` thay vì `"master_bidv_connector_service.bank_requests"`).
   - Hàm `loadMaster()` query DB với `WHERE mb.master_table = ? LIMIT 1` (không lọc `master_schema`) dẫn đến việc bốc nhầm `master_binding` của schema khác.
   - Hàm `GetMasterDB(ctx, key)` trong `ConnectionManager` discard hoàn toàn tham số `key`, hardcode vào `RoleDestination`.
3. **Các lỗ hổng phát hiện qua Adversarial Review:**
   - Postgres string concat `mb.master_schema || '.' || mb.master_table` trả về `NULL` nếu `master_schema` là `NULL`.
   - `ScheduleCreateRequest` và HTTP handler ở CMS thiếu field `master_schema`.
   - `Save()` query so sánh `NULL` không an toàn.

## 3. Phạm vi giải pháp (Scope)
- Chuẩn hóa 100% việc định danh bảng Master thành **Schema-Qualified FQN** (`<master_schema>.<master_table>`) qua toàn bộ các tầng: CMS API, Domain Repository, Cron Scheduler, NATS Payload, và Worker Consumer.
- Đảm bảo tính an toàn với giá trị `NULL` trong PostgreSQL (`COALESCE(NULLIF(..., ''), 'public')`).
- Đồng bộ DTO Request, Controller, và Command Handler ở CMS API layer.
