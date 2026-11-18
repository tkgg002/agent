---
tier: T2
suffix: mapping_and_trigger_cms
workspace: master-mapping-bulk-and-trigger-cms-2026-11-16
---
# 08 Tasks — Điều Tra & Thiết Kế Giải Pháp Master Mapping Bulk Config (Task 1) + Downstream CDC Trigger Governance (Task 2)

## Task List

### T1. [S2 Investigate] Khảo sát hiện trạng Master Mapping UI & Backend APIs (Task 1)
- **Context (`file:line`):**
  - Frontend: `cdc-cms-web/src/pages/MasterMappingFieldsPage.tsx`, `cdc-cms-web/src/components/master/*`, `cdc-cms-web/src/services/api.ts`
  - Backend CMS: `cdc-cms-service/internal/api/master/master_mapping_rule_handler.go`, `cdc-cms-service/internal/router/router.go`
  - Worker DDL: `centralized-data-service/internal/handler/master/master_ddl_handler.go`, `internal/service/master/master_ddl_generator.go`
- **Mục tiêu kiểm tra:**
  - Kiểm kê toàn bộ thao tác hàng loạt (Bulk Actions) đã có sẵn trên `/masters/:name/mappings` (như `SyncMasterRulesFromShadow`, `BatchUpdateMasterMappingRules`, `ApproveAllColumnsDDL`, `DropRejectedColumnsDDL`, auto snake_case...).
  - Xác định chính xác vì sao User vẫn phải làm thủ công từng dòng một (thiếu Bulk Snake-Case / Bulk Edit / Import từ DDL hoặc JSON Spec / Auto-Match với Target Table Schema có sẵn trên DB đích).
  - Kiểm chứng đối chiếu 101 Mapping Rules của `binding_id=95` mà User gửi với đặc tả 48 cột `payment_bill` + 13 cột `payment_bill_payload`.
- **Definition of Done:** Danh sách bằng chứng `file:line` đầy đủ về những gì đã có vs những gì đang thiếu, kèm bảng đối chiếu sai lệch cụ thể trên `binding_id=95`.

### T2. [S2 Investigate & S2.5 Pre-Plan Critique] Phân tích Kiến trúc & Phản biện SQL Trigger `cdc_payment_bill -> payment_bill / payment_bill_payload` (Task 2)
- **Context (`file:line`):**
  - Kiểm tra trong `cdc-cms-service`, `centralized-data-service`, `cdc-cms-web` xem đã có khái niệm Post-DDL SQL / Trigger / Downstream Sync Hook nào chưa.
  - Phản biện trực diện (DG0) đoạn SQL `fn_sync_bill_derived_fields()`, `trg_sync_cdc_to_payment_bill()` và `trg_sync_cdc_to_payment_bill_payload()` trong prompt của User đối chiếu với danh sách cột `cdc_payment_bill` ở Task 1 và thiết kế `Payment-v2`.
- **Definition of Done:**
  - Chỉ rõ mọi lỗi thực tế trong SQL của User (lệch tên cột `camelCase` vs `snake_case`, cột chưa tạo/approve ở `cdc_payment_bill`, cột bị thiếu như `refunded_operation_ids`, `source_field_types = '{}'::jsonb` làm mất lossless BSON metadata, hàm thừa `fn_sync_bill_derived_fields`, thiếu `CREATE TRIGGER` cho payload, lỗi soft-delete `isDelete` vs `_deleted`...).
  - Đánh giá khách quan **"Có nên đưa lên CMS không?"** (tách rõ phần NÊN và phần KHÔNG NÊN) dựa trên kiến trúc Control Plane / Data Plane của `data-hub`.

### T3. [S3 Solution + DG0/DG1/DG2] Thiết kế Giải pháp Tối ưu Nhất & Code Demo
- **Mục tiêu:**
  - Thiết kế **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** cho Task 1 (Schema-Driven Bulk Mapping / Import DDL-JSON Spec + Bulk Transform trên `MasterMappingFieldsPage`) và Task 2 (Quản trị Controlled Post-Sync Trigger / SQL Hook có Versioning, Dry-Run Check & Audit trên CMS — hoặc giải pháp thay thế chuẩn kiến trúc).
  - Chạy `$T/laya_gate.py run --stage dg0` và `$T/laya_gate.py run --stage dg1` đạt `PASS`.
  - Lưu vào `09_tasks_solution_mapping_and_trigger_cms.md`, hiển thị Code Demo trên chat và dừng chờ User `APPROVE`.
