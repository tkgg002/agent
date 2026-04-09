# CDC Schema Detection Tasks

## Task: Schema Detection via Airbyte Discover API
- **Phase**: Phase 1.6 (Airbyte Orchestration)
- **Service Group**: Business (CDC Integration)
- **Service(s)**: `centralized-data-service` (CDC Worker), `cdc-cms-service` (CMS API)
- **Mô tả**: Thay đổi logic `HandleIntrospect` trong CDC Worker. Gọi Airbyte Discover Schema API để lấy catalog fields của MongoDB source collection, so sánh với DW table columns (`information_schema.columns`), rồi ghi fields mới vào `pending_fields` table cho CMS user duyệt. Chú ý table `export_jobs` đang thiếu `_raw_data`.
- **Trạng thái**: [ ] TODO

### [Context]
- Current state: Worker config sai port (18006), CMS API đang dùng Basic Auth sai cho việc fetch từ Airbyte OSS.
- Table `export_jobs` có Airbyte auto-schema evolution nhưng CDC API lại quét `_raw_data`. Do table của Airbyte manage ko ghi `_raw_data` nên count keys = 0 hoặc bỏ sót cột.
- Option B (approved): Move `Discover` sang Airbyte Catalog scan thay vì `_raw_data` scan.

### [Definition of Done]
- [x] Sửa config của Worker để nạp vào Airbyte OAuth (thay port 18000).
- [x] Implement `DiscoverSourceSchema` với OAuth token trên Worker Airbyte Client.
- [x] Đổi `HandleIntrospect` để gọi API thay vì quét `_raw_data` db (Do `export_jobs` chưa có `_raw_data` luôn, schema evolution do Airbyte làm).
- [x] Viết chức năng xử lý `add_field_alter` vào `pending_fields`.
- [x] **[QA Gate]**: workflow `/qa-agent` verify Unit Tests.
- [x] **[Security Gate]**: workflow `/security-agent` verify auth keys ko hardcode (Đã kiểm chứng env loading).
- [x] Model Tracking: Ghi nhận task vào `05_progress.md` với tag model.

---
**Status**: [✅] Hoàn thành
