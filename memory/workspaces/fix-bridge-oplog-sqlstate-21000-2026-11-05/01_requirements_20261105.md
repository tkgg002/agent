---
tier: T2
suffix: 20261105
workspace: fix-bridge-oplog-sqlstate-21000-2026-11-05
project: centralized-data-service
---

# 01. Requirements — Khắc phục lỗi `SQLSTATE 21000` và lỗi không kích hoạt `transmute` trong `bridge-oplog`

## 1. Yêu cầu của User
- User báo:
  1. `"bridge đang bị lỗi ERROR: ON CONFLICT DO UPDATE command cannot affect row a second time (SQLSTATE 21000)"`.
  2. `"và khi bridge thì sao ko có transmute. chán chưa, làm đi làm lại vẫn bug là sao"`.
- Điều tra nguyên nhân gốc rễ tại sao luồng `bridge-oplog` (`centralized-data-service/internal/handler/source/bridge_handler.go`) vừa bị nổ lỗi PostgreSQL `SQLSTATE 21000` khi thực thi `batchUpsert`, vừa không kích hoạt/thực thi `transmute` dữ liệu sang bảng Master sau khi bridge, và thiết kế giải pháp khắc phục triệt để 1 lần duy nhất.

## 2. Ràng buộc Thực thi (Technical Constraints từ Global Lessons)
- **L001 (`Deployed-vs-Local Diff Grounding Over Speculative Failure Guessing`):** Đối chiếu trực tiếp mã nguồn tại HEAD và lịch sử thay đổi, không suy đoán chung chung.
- **L009 (`Verbatim Current-File Code Demo & Cross-Layer Default Parameter Check`):** Mọi khối trích dẫn và Code Demo bắt buộc tạo trực tiếp từ các dòng văn bản nguyên văn của tệp tại HEAD hiện hành.
- **L018 (`Propose-Only Discipline & Explicit Approval Gate`):** Giữ chế độ 100% Read-Only đối với source code, trình bày phân tích + giải pháp + Code Demo trực tiếp trên chat và dừng chờ lệnh `APPROVE` tường minh.
- **L035 (`Direct-Answer Summary First, Macro Topology & 1-to-1 Finding Mapping`):** Mở đầu phản hồi bằng khối `Direct Summary` 2–4 câu trả lời trực diện câu hỏi chính; ánh xạ giải pháp 1-1 theo từng phát hiện `F1..Fn`.
- **L049 (`Transaction Aborted Isolation & GORM SQL Quirks`):** Nắm vững cơ chế Batch Upsert và tránh để duplicate conflict key trong cùng một câu lệnh `INSERT ... ON CONFLICT DO UPDATE`.
- **L051 (`MongoDB OCC Pipeline Update & SQL Lateral Join`):** Khử trùng lặp (deduplicate) các sự kiện cùng khóa trong cùng một lô trước khi ghi xuống DB.
- **L058 (`Spec-Driven PK & Timestamp Field Resolution`):** Phân giải khóa chính (`pkField` / `ConflictTarget`) từ cấu hình binding/spec thay vì đoán mò cứng.
