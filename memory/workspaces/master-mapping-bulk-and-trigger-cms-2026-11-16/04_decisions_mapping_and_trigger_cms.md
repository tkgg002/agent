---
tier: T2
suffix: mapping_and_trigger_cms
workspace: master-mapping-bulk-and-trigger-cms-2026-11-16
---
# 04 Decisions — Master Mapping Bulk Spec (Task 1) & Controlled Downstream Trigger Governance (Task 2)

- **ADR-MMT-01 (Tách bạch Bulk Spec Apply & Staging DDL trên Master Mapping):** Giữ nguyên cơ chế staging an toàn (`pending_rename_from`, `pending_type_change`) của `UpdateColumnMetadataHandler` cho các cột đã `in_master = true`, và bổ sung endpoint tổng quát `POST /api/v1/master-mapping-rules/bulk-apply-spec` (kèm `dry_run` preview diff) để đồng bộ tên cột `snake_case`, kiểu dữ liệu, tạo nested/subset rules và reject các trường dư thừa chỉ trong 1 transaction.
- **ADR-MMT-02 (Không mở Arbitrary SQL Console — Chỉ hỗ trợ Controlled Master Trigger Governance):** Tuyệt đối không cho phép chạy raw SQL tự do từ CMS lên DB đích. Thay vào đó, quản trị Downstream Trigger gắn chặt vào `master_binding_id` (`cdc_system.master_downstream_triggers`) với 6 chốt chặn: Whitelist chỉ nhận Function/Trigger gắn đúng `master_schema.master_table`, kiểm tra tĩnh `NEW.<col>`/`OLD.<col>` khớp `mapping_rule_master`, Dry-Run compile trong transaction `ROLLBACK` trên Worker, ép `SET LOCAL lock_timeout = '5s'` + ưu tiên `CREATE OR REPLACE FUNCTION` không lấy `AccessExclusiveLock` khi không đổi chữ ký trigger, phát hiện lệch (`DB Drift`) qua `pg_trigger`/`pg_proc`, và bảo toàn trigger khi `SwapMasterTable`.
- [2026-11-16T14:02:09+07:00] [Agent:laya_gate] DG0 C-REQ-1 FLAG resolved: Verified directly in MasterMappingFieldsPage.tsx: handleBatchUpdate only sends ids and status to PUT /api/v1/master-mapping-rules/batch. REF: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx:351-360
- [2026-11-16T14:02:39+07:00] [Agent:laya_gate] DG0 C-CONTRA-1 FLAG resolved: Verified directly in master_ddl_generator.go: PostgreSQL master table initializes only 4 system columns (_gpay_id, _source_ts, _deleted, _updated_at) and never creates _id unless mapped. REF: /Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/master_ddl_generator.go:148-163
