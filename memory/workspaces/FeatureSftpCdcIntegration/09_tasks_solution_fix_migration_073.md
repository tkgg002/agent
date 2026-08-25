# 09_tasks_solution_fix_migration_073.md — Hồ sơ Giải pháp Kỹ thuật Khắc phục Lỗi Migration 073

> **Ngày tạo**: 2026-08-07  
> **Người thực hiện**: Brain (lập giải pháp) & Muscle (thực thi sau khi User duyệt)  
> **Mục tiêu**: Khắc phục lỗi SQLSTATE 42P01 `relation "cdc_table_registry" does not exist` khi boot `cdc-cms-service`.

---

## 1. Phân tích Nguyên nhân Gốc rễ (Root Cause)

- **Lỗi báo từ CMS**: `apply migrations: migrate: apply 073_add_sftp_source_type: ERROR: relation "cdc_table_registry" does not exist (SQLSTATE 42P01)`
- **Nguyên nhân**: File migration [`073_add_sftp_source_type.sql`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/migrations/schema/core/073_add_sftp_source_type.sql) khai báo `ALTER TABLE cdc_table_registry` nhưng **thiếu schema qualifier `cdc_system.`**.
- **Lịch sử Kiến trúc**: Từ migration `037_move_system_tables_to_cdc_system.sql`, toàn bộ các bảng hệ thống (trong đó có `cdc_table_registry`) đã được di chuyển từ schema `public` sang schema `cdc_system`. Do đó, mọi câu lệnh DDL thao tác trên bảng này trong cụm `schema/core/` BẮT BUỘC phải chỉ định rõ `cdc_system.cdc_table_registry`.

---

## 2. Phương án Giải pháp Duy nhất (The Single Best Approach)

Bổ sung schema qualifier `cdc_system.` vào file migration `073_add_sftp_source_type.sql`.

### Chi tiết thay đổi file `cdc-cms-service/migrations/schema/core/073_add_sftp_source_type.sql`:

```diff
- ALTER TABLE cdc_table_registry DROP CONSTRAINT IF EXISTS ctr_check_source_type;
- ALTER TABLE cdc_table_registry ADD CONSTRAINT ctr_check_source_type
+ ALTER TABLE cdc_system.cdc_table_registry DROP CONSTRAINT IF EXISTS ctr_check_source_type;
+ ALTER TABLE cdc_system.cdc_table_registry ADD CONSTRAINT ctr_check_source_type
      CHECK (source_type IN ('mongodb', 'mysql', 'postgresql', 'sftp'));
```

---

## 3. Kế hoạch Phân công Thực thi (Brain / Muscle Protocol)

1. **Brain**: Trình bày Plan này và chờ lệnh **APPROVE** của User. (TUYỆT ĐỐI KHÔNG tự sửa source code).
2. **Muscle**: Sau khi có lệnh Approve, Muscle sẽ áp dụng diff vào file `073_add_sftp_source_type.sql`.
3. **Verification**: Chạy `go build -o /dev/null ./cmd/server/main.go` để verify build integrity, và kiểm tra lại cú pháp SQL.
