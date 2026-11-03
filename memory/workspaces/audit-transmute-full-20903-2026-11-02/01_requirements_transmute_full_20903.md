---
tier: T2
suffix: transmute_full_20903
workspace: audit-transmute-full-20903-2026-11-02
created_at: 2026-11-02T11:48:00+07:00
---

# 01. Requirements — Điều tra lỗi "Chạy ngay toàn bộ" chỉ đồng bộ `20,903 / 20,903` bản ghi (`binding_id=76`, `transaction_history_2`)

## 1. Yêu cầu của User
- Kiểm tra màn hình Mapping của bảng Master `transaction_history_2` (`http://localhost:5173/masters/transaction_history_2/mappings?binding_id=76`).
- Giải thích chính xác tại sao khi bấm **"Chạy ngay toàn bộ" (Full Transmute)** thì kết quả chỉ báo `Synced (20,903 / 20,903)` thay vì toàn bộ tổng số bản ghi của bảng Shadow nguồn.
- Tìm nguyên nhân gốc rễ (Root Cause) có bằng chứng `file:line` và runtime/API thật, lên phương án giải pháp duy nhất tại `09_tasks_solution_transmute_full_20903.md` kèm Code Demo trên chat và chờ `APPROVE`.

## 2. Ràng buộc Thực thi (Technical Constraints trích xuất từ Lessons `INDEX.md`)
- **`L008` & `L025` (Propose-Only Discipline & Direct-Answer Summary First):** Mở đầu câu trả lời bằng khối **Direct Summary** 2–4 câu trả lời trực diện câu hỏi chính; giữ chế độ Read-Only tuyệt đối với source code cho tới khi có `APPROVE` của User.
- **`L007` & `L023` (Explicit Runtime Call-Site & End-to-End Callchain Tracing):** Truy vết đầy đủ luồng từ nút bấm trên `cdc-cms-web` (`MasterMappingPage` / `MasterRegistry`) -> API `cdc-cms-service` -> NATS / Worker `centralized-data-service` (`TransmuterModule` / `TransmuteTable` / `countShadowRows` / `fetchShadowBatch`).
- **`L034` & `L035` (Zero DB CLI Or Container Exec & No Write-Shaped SQL):** Tuyệt đối không chạy lệnh can thiệp DB hay `docker exec ... psql`; chỉ điều tra qua mã nguồn tĩnh, log/API đọc (HTTP GET) của hệ thống đang chạy.
- **`L036` & `L039` (No Full-Table Scans & Dynamic Chunk Loop / Query Clauses):** Kiểm tra kỹ điều kiện `WHERE`, giới hạn `LIMIT`, cursor/window hoặc điều kiện thoát vòng lặp chunk trong luồng Transmute và hàm đếm tiến độ (`Synced (X / Y)`).
- **`L037` & `L042` & `L048` (Multi-Tenant Identity Tuple `binding_id=76`, Multi-Shadow & Spec-Driven PK/Timestamp):** Kiểm tra cách phân giải bảng Shadow nguồn, `binding_id=76`, `shadow_table`, `transform_spec`, hoặc điều kiện lọc thời gian/trạng thái của `transaction_history_2`.
