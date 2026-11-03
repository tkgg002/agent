---
tier: T2
suffix: mongo_createindex_dupkey
workspace: audit-mongo-createindex-dupkey-2026-11-02
mode: standard
created_at: 2026-11-02T13:10:30+07:00
---

# 01. Requirements — Điều tra & Xử lý lỗi `mongo createIndex ux_transaction_history_2_idempotencyKey on transaction_history_2 failed: (DuplicateKey)`

## 1. Yêu cầu của User
- Giải thích nguyên nhân gốc rễ vì sao **mọi cột** (`_id`, `transId`, `idempotencyKey`, ...) trên collection `devtest.transaction_history_2` đều thất bại khi tạo Unique Index (`cái nào cũng bị hết`), tuyệt đối không dùng "cheat code".
- Giải thích trực diện kèm bằng chứng `file:line` & runtime (`VERIFIED`) cho câu hỏi:
  1. **Vì sao khi tạo Master Binding lại tự sinh thêm 1 rule `_id` (`target_column = "_id"`) bên cạnh rule `id` (`target_column = "id"`)?**
  2. **Vì sao khi Transmute sang MongoDB Master (`devtest.transaction_history_2`), `_id` lại hiển thị dạng chuỗi UUID `"01a19b05-b237-713f-8651-ef672d810f2d"` thay vì `_id: ObjectId('66c7557fb4b3fa830aeb1598')`, cùng toàn bộ các bug liên quan trong luồng tạo Master & Transmute?**
- Đề xuất phương án xử lý triệt để gốc rễ trong codebase `data-hub` kèm **Code Demo** trực tiếp trên khung chat (chế độ **Propose-Only** chờ `APPROVE`).

## 2. Ràng buộc Thực thi (Technical Constraints trích xuất từ `GEMINI.md` & `lessons/INDEX.md`)
- **`L056` (Exhaustive Multi-Case Log Verification & No UI Workaround or Hardcoded Filter Hacks):**
  - Bắt buộc đối chiếu toàn bộ nhật ký thực thi (`/tmp/cds-worker.log:45654, 49977, 50020, 51208, 51924`) để giải thích cả 3 nhóm lỗi: cột `_id` (`InvalidIndexSpecificationOption`), cột bắt buộc `transId` (`DuplicateKey` do `loadRules` đánh rớt rule `_id` có `data_type=""`, làm fallback về `_gpay_id` và nhân bản document), và cột tùy chọn `idempotencyKey` (`DuplicateKey` do cả nhân bản document lẫn thiếu `SetSparse(true)`).
  - Tuyệt đối cấm hướng dẫn mẹo lách trên UI hoặc gán cứng `"$gt": ""` trong code.
- **`L008` & `L025` (Propose-Only Discipline, Direct-Answer Summary First & 1-to-1 Finding Mapping):**
  - Mở đầu phản hồi bằng khối **Direct Summary** 2–4 câu trả lời trực diện câu hỏi.
  - Ánh xạ giải pháp 1-1 theo từng phát hiện `F1..F3`, hiển thị trực tiếp **Code Demo** trên khung chat và dừng chờ `APPROVE` trước khi chạm vào source code.
- **`L034`, `L035`, `L003` (Zero DB CLI & Clean CLI Args):**
  - Tuyệt đối không chạy bất kỳ lệnh CLI nào can thiệp vào DB của User; không nhúng từ khóa SQL/DDL nhạy cảm hoặc nối chuỗi lệnh trong tham số CLI của `laya_gate.py`.
- **`L036` (Zero Runtime DDL & No Full-Table Scans on Large Databases):**
  - Không chạy `UpdateMany` quét toàn bộ collection trong hàm `CreateIndex`; xử lý ở cấp định danh khóa chính Transmuter (`_id`) và `IndexOptions` chuẩn của MongoDB (`SetSparse(true)`).
- **`L011` (Single-Responsibility & Line-Budget Guard):**
  - Giữ kích thước file `index_driver_mongo.go`, `transmuter_identity.go`, `base_handler.go` gọn nhẹ (`< 500` dòng).
