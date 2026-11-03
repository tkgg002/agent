# 01. REQUIREMENTS — `master_add_field_live_10m_20261030`

- **Workspace:** `audit-master-add-field-live-10m-2026-10-30`
- **tier:** T2
- **mode:** standard
- **Suffix:** `master_add_field_live_10m_20261030`
- **Ngày tạo:** `2026-10-30T15:16:45+07:00`

## 1. Yêu cầu của User
User đặt bài toán vận hành thực tế:
- Luồng dữ liệu: `source -> shadow -> master`.
- Trạng thái hiện tại: `master` đang chạy **live luồng đọc (live read traffic)**, `source` đang chạy **live luồng ghi (live write traffic / CDC)**.
- Yêu cầu nghiệp vụ: Muốn **`master` có thêm 1 field lấy từ `shadow`**.
- Quy mô dữ liệu: Khoảng **10 triệu (`10,000,000`) bản ghi**.
- Câu hỏi trọng tâm: **"Hệ thống hiện tại có đáp ứng được để thêm 1 field từ `shadow` sang `master` (cả dữ liệu mới lẫn backfill 10 triệu bản ghi lịch sử) mà luồng live (đọc trên `master`, ghi CDC từ `source`) không bị ảnh hưởng không?"**

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **L005 (Propose-Only Discipline):** 100% Read-Only (Propose-Only), phân tích trên source code thực tế kèm `file:line`, tuyệt đối không sửa source code khi chưa có `APPROVE`.
- **L020 (End-to-End Callchain Tracing):** Truy vết toàn trình từ lúc thêm Field Mapping trên CMS (`cdc-cms-web` -> `cdc-cms-service`) -> DDL mở cột mới trên `master` (PostgreSQL `ALTER TABLE ADD COLUMN` / MongoDB dynamic field) -> Reload mapping cho luồng CDC live (`centralized-data-service`) -> Luồng Backfill / Re-Transmute 10 triệu bản ghi từ `shadow` sang `master` (kiểm tra khóa bảng, batch size, xung đột OCC `_source_ts` giữa luồng backfill dữ liệu cũ với luồng CDC live đang ghi dữ liệu mới).
- **L022 (Direct-Answer Summary First):** Mở đầu phản hồi bằng khối **Direct Summary** trả lời trực diện câu hỏi của User, sau đó trình bày phân tích chi tiết và ánh xạ 1-1 theo từng phát hiện `F1..Fn`.
- **L031 & L032 (Zero DB CLI):** Chỉ kiểm tra qua source code tĩnh, không chạy lệnh can thiệp DB.
- **L033 (Zero Runtime DDL & No Full-Table Lock/Scan):** Kiểm tra kỹ cơ chế DDL khi thêm cột vào bảng `master` 10 triệu dòng và cơ chế đọc/ghi lô khi backfill 10 triệu dòng có gây lock/I/O storm hay đè dữ liệu live không.
- **L038 (MongoDB OCC Pipeline Update & PostgreSQL OCC):** Kiểm tra điều kiện OCC (`_source_ts`) trên cả PostgreSQL Master và MongoDB Master khi backfill lại dữ liệu cũ trong lúc luồng live CDC vừa cập nhật bản ghi mới hơn.
