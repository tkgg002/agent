---
tier: T2
mode: standard
suffix: sync_shadow_unique
workspace: fix-master-mapping-sync-shadow-unique-2026-11-16
created_at: 2026-11-16T14:38:00+07:00
---
# 01 Requirements — Kiểm Tra & Xử Lý Lỗi Vi Phạm Ràng Buộc Duy Nhất Khi "Sync Từ Shadow" Trên Trang Master Mapping (`/masters/cdc_payment_bill/mappings?binding_id=61`)

## 1. Yêu cầu của User
- **Hiện tượng lỗi thực tế:** Khi thực hiện chức năng **"Sync từ Shadow"** ở trang `/masters/cdc_payment_bill/mappings?binding_id=61`, hệ thống báo lỗi:
  > `"Bản ghi đã tồn tại (vi phạm ràng buộc duy nhất)"`
- **Yêu cầu:** Kiểm tra nguyên nhân gốc rễ (`file:line` + schema/migration/runtime state liên quan) và đề xuất giải pháp khắc phục triệt để.

## 2. Ràng buộc Thực thi (Technical Constraints từ `GEMINI.md`, `INDEX.md` & `system_overview_and_design_pattern.md`)
- **L075 (Full Multi-Tenant Identity Tuple & 1-N-M Lineage SQL Resolution):** Trong mô hình `1 Source -> N Shadows -> M Masters` (nơi 1 bảng tên `cdc_payment_bill` có thể gắn với nhiều `master_binding_id`, hoặc nhiều `shadow_binding` cùng nguồn, hoặc ràng buộc `UNIQUE` cũ theo `master_table` phẳng thay vì `master_binding_id` / xung đột `target_column`), phải kiểm tra toàn bộ Unique Constraints / Indexes của bảng `cdc_system.mapping_rule_master` và mệnh đề `ON CONFLICT` / logic kiểm tra tồn tại trong `SyncMasterRulesFromShadow`.
- **L076 (Schema Verification & ANSI SQL Type Normalization):** Bắt buộc đọc toàn bộ chuỗi file migration DDL của `mapping_rule_master` (và kiểm tra read-only `pg_index` / `pg_constraint` nếu cần mà không dùng lệnh dạng ghi) trước khi kết luận constraint nào bị nổ `SQLSTATE 23505` và câu lệnh `INSERT`/`ON CONFLICT` nào gây ra.
- **L077 (Transaction Aborted Isolation & GORM SQL Quirks):** Nếu `SyncFromShadow` thực hiện batch insert/upsert hoặc lặp trong một transaction PostgreSQL, phải kiểm tra xem có bị trùng khóa trong cùng lô (duplicate source/target column), lệch Partial Unique Index (`WHERE ...`), hoặc lệch khóa định danh giữa `master_binding_id` và `master_table` hay không.
- **L072 & L073 (No DB Cheat / Zero Runtime DDL — L074):** Tuyệt đối cấm sửa/xóa dữ liệu trực tiếp trong DB hoặc nhét DDL `ALTER TABLE ... DROP CONSTRAINT` vào luồng runtime. Chỉ dùng truy vấn tĩnh / read-only thuần nếu cần xác minh metadata, và sửa triệt để ở đúng tầng nghiệp vụ (`cdc-cms-service` / migration chuẩn nếu schema thiếu khóa đa kết nối).
- **Hexagonal + CQRS Pattern (`system_overview_and_design_pattern.md` Mục 0, 2.2, 5):** Tuân thủ cấu trúc `internal/api/master` -> `internal/app/commands/master` -> `internal/domain/master` <- `internal/infra/persistence/master`, bảo toàn 100% Backward Compatibility cho mọi caller khác.
- **Propose-Only (Rule #1, #9, #13):** Giữ 100% Read-Only đối với source code trong pha Brain (`S1 -> S2 -> DG0 -> S3 -> DG1/DG2`), trình bày **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** kèm **Code Demo** trực tiếp trên chat và dừng chờ User `APPROVE`.
