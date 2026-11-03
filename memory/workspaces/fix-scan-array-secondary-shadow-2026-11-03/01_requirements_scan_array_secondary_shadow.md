# 01 Requirements: Kiểm tra & Thiết kế Giải pháp Lỗi Scan Array (Flatten) Trên Bảng Secondary Shadow (`transaction_history_payload`, `binding_id=77`)

tier: T2
suffix: scan_array_secondary_shadow
date: 2026-11-03

## 1. Yêu cầu của User
- Kiểm tra lỗi tại màn hình `http://localhost:5173/masters/transaction_history/mappings?binding_id=77`:
  - Khi bấm **Scan Array (Flatten) → Tạo rules PENDING để duyệt**, hệ thống không thực hiện được và báo lỗi **không có DB Shadow** (`ko có db shadow` / shadow table hoặc shadow connection không tìm thấy).
  - Bối cảnh thực tế do User cung cấp:
    - Bảng chính (Primary Shadow) của Master Binding `binding_id=77` là `transaction_history`.
    - Trường (field) cần flatten (`Scan Array (Flatten)`) lại nằm ở bảng phụ (Secondary Shadow Table) là `transaction_history_payload`.
    - Do kiến trúc `multi_shadow` (`transaction_history` + `transaction_history_payload` gộp chung vào 1 Master Binding `binding_id=77`), việc bấm `Scan Array (Flatten)` trên field thuộc bảng phụ `transaction_history_payload` bị lỗi báo không có DB Shadow / bảng Shadow.
- Lên giải pháp kỹ thuật triệt để cho User (Propose-Only kèm Code Demo, chờ lệnh `APPROVE`).

## 2. Ràng buộc Thực thi từ Bài học (Technical Constraints — S0 Boot)
- **`L013` (`Propose-Only Discipline & Explicit Approval Gate`):** Giữ chế độ **100% Read-Only (Propose-Only)** trên toàn bộ source code `data-hub` khi đang ở bước điều tra và thiết kế giải pháp (`S1–S3`). Hiển thị trực tiếp **Code Demo** (trước/sau kèm `file:line`) ngay trên khung chat và dừng chờ lệnh `APPROVE` tường minh từ User trước khi sửa code.
- **`L030` (`Direct-Answer Summary First & 1-to-1 Finding Mapping`):** Mở đầu phản hồi bằng khối **Direct Summary** (2–4 câu) trả lời trực diện nguyên nhân gốc rễ tại `file:line`, và trình bày giải pháp ánh xạ **1-1 theo từng phát hiện (`F1..Fn`)**.
- **`L012` & `L028` (`Explicit Runtime Call-Site & End-to-End Callchain Tracing`):** Truy vết toàn trình chuỗi gọi hàm của tính năng `Scan Array (Flatten)`: từ Frontend (`MappingEditor.tsx` / `api.ts`) → API Route & Handler tại `cdc-cms-service` → NATS Command / Service / Repository tại `cdc-cms-service` và `centralized-data-service` → Database (`cdc_system.master_binding`, `cdc_system.shadow_binding`, `cdc_system.field_mappings`).
- **`L042` & `L047` (`Full Multi-Tenant Identity Tuple & Pure CDC Unified Multi-Shadow Binding`):** Trong mô hình 1 Master Binding đa Shadow (`multi_shadow` chứa `primary` + `secondary_shadows`), mỗi field hoặc thao tác Scan Array trên Master Mapping phải định tuyến chính xác về đúng `shadow_binding` (hoặc bảng Shadow vật lý `shadow_schema.shadow_table_name` và `shadow_connection_id`) chứa field đó (ví dụ `transaction_history_payload`), thay vì mặc định lấy cứng bảng chính `transaction_history` hoặc tra cứu sai khóa.
- **`L016` (`Single-Responsibility Module Split & Line-Budget Guard`):** Kiểm tra `wc -l` của các file dự kiến sửa; nếu file đã lớn, tách logic phân giải Secondary Shadow cho Scan Array thành hàm/file đơn trách nhiệm đúng layer, đảm bảo `NOW <= HEAD` trên các God-files.
- **`L019` (`Pre-Plan Adversarial Critique Wave DG0`):** Kiểm chứng toàn bộ giả định về `binding_id=77`, cấu hình `transform_spec` (`multi_shadow`), `field_mappings`, và luồng xử lý `scan-array` thực tế trong code trước khi lập `09_tasks_solution`, chạy `laya_gate.py run --stage dg0`.
- **`L039` & `L040` (`Zero DB CLI Or Container Exec When DB Access Forbidden / No Cheat DB`):** Không can thiệp/sửa dữ liệu DB trực tiếp; phân tích chính xác từ mã nguồn tĩnh và API/log sẵn có.
