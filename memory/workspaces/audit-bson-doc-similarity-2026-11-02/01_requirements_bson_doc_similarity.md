---
tier: T2
suffix: bson_doc_similarity
workspace: audit-bson-doc-similarity-2026-11-02
created_at: 2026-11-02T11:58:10+07:00
---

# 01. Requirements — Đánh giá Mức độ Tương đồng giữa Document MongoDB Gốc và Bản Backup Mới Tạo (`transId: 240822ELRTWZ`)

## 1. Yêu cầu của User
- Đối chiếu chi tiết giữa bản ghi MongoDB gốc (`_id: ObjectId('66c7557fb4b3fa830aeb1598')`, `transId: '240822ELRTWZ'`) và bản ghi backup mới tạo.
- Đánh giá mức độ tương đồng về: (1) Cấu trúc trường (Top-level & Nested fields), (2) Giá trị nghiệp vụ, (3) Kiểu dữ liệu BSON (`ObjectId`, `ISODate`, `NumberInt` vs `NumberDecimal`, `null` vs `missing`).
- Truy vết nguyên nhân gốc rễ trong codebase `data-hub` (`mongo_type_converter.go`, `mongo_bson_restorer.go`, `mongo_extjson_unwrapper.go`) giải thích vì sao xuất hiện các điểm khác biệt và đề xuất phương án chuẩn hóa (Propose-Only chờ `APPROVE`).

## 2. Ràng buộc Thực thi (Technical Constraints trích xuất từ Lessons `INDEX.md`)
- **`L008` & `L025` (Propose-Only Discipline & Direct-Answer Summary First):** Mở đầu phản hồi bằng khối **Direct Summary** 2–4 câu trả lời trực diện mức độ tương đồng; giữ chế độ Read-Only tuyệt đối với source code cho tới khi có `APPROVE`.
- **`L032` & Rule #20 (End-to-End Secret & PII Masking):** Che giấu (mask) thông tin cá nhân (`email`, `phone`) khi lưu vào các file hồ sơ trong `agent/memory/workspaces/`.
- **`L040` & `L042` (BSON Coercion, UTC Timezone & Multi-Shadow BSON Type-Presence Restorer):** Đối chiếu chính xác quy trình ép kiểu BSON (`CoerceToBSONValue` -> `ApplySourceFieldTypesToBSONDoc` -> `coerceDefaultInt32IfInRange`) và cơ chế bảo toàn trường lồng nhau (`sender`, `receiver`, `info`, `error`).
