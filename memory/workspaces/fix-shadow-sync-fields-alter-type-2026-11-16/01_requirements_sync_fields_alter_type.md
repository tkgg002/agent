---
tier: T2
mode: standard
suffix: sync_fields_alter_type
workspace: fix-shadow-sync-fields-alter-type-2026-11-16
created_at: 2026-11-16T15:01:30+07:00
---
# 01 Requirements — Điều Tra & Xử Lý Lỗi "Sync Fields" Không Thực Hiện `ALTER COLUMN TYPE` Khi Đổi `TEXT` → `JSONB` Trên Shadow Table

## 1. Yêu cầu của User
- **Hiện tượng lỗi thực tế:** Trên trang Shadow Mapping Rules (`cdc-cms-web/src/pages/MappingFieldsPage.tsx`), cột `Data Type Current` hiển thị cảnh báo:
  > `Target: JSONB · Current: TEXT → Cần "Sync Fields" để ALTER COLUMN TYPE`
  Tuy nhiên khi User bấm nút **"Sync Fields to Shadow"** (`handleSyncFields` gọi `POST /api/v1/source-objects/:id/create-default-columns`), cột trên bảng Shadow vẫn giữ nguyên `TEXT` — lệnh `ALTER COLUMN TYPE` sang `JSONB` không chạy hoặc bị chặn/lỗi/bỏ qua.
- **Yêu cầu:** Kiểm tra nguyên nhân gốc rễ (`file:line` + luồng End-to-End từ FE -> `cdc-cms-service` -> NATS/Command -> `centralized-data-service` Worker `schema_manager` + logs/runtime state) và thiết kế giải pháp khắc phục triệt để (Propose-Only chờ `APPROVE`).

## 2. Ràng buộc Thực thi (Technical Constraints từ `GEMINI.md` & `lessons/INDEX.md`)
- **L005 (Zero Table-Specific Hardcoding In Generic Sync Engine):** Không hardcode tên bảng/cột cụ thể vào generic sync/schema manager; giải pháp phải tổng quát theo metadata rule và kiểu dữ liệu chuẩn PostgreSQL.
- **L072 & L073 (Zero DB CLI / No Write-Shaped SQL Cheat):** Tuyệt đối không chạy DDL/DML trực tiếp vào DB để sửa hộ cột của User; chỉ đọc metadata/logs Read-Only để xác minh nguyên nhân.
- **L074 (Zero Runtime DDL in DML Hot-Path & PostgreSQL Lock Safety):** Không nhét DDL tự phát vào luồng DML upsert mỗi bản ghi của worker; luồng `Sync Fields` (`create-default-columns`) là control-plane command chủ động của operator, cần xử lý an toàn khóa và ép kiểu dữ liệu (`USING`) hoặc cảnh báo rõ ràng khi dữ liệu `TEXT` không phải JSON hợp lệ.
- **L076 (Schema Verification & ANSI SQL Type Normalization):** Kiểm tra chuẩn hóa tên kiểu dữ liệu (`TEXT` vs `JSONB`, `information_schema.columns` `data_type` / `udt_name`) và cú pháp PostgreSQL `ALTER TABLE ... ALTER COLUMN ... TYPE JSONB USING ...` (vì PostgreSQL **không có implicit/assignment cast từ `TEXT` sang `JSONB` nếu thiếu mệnh đề `USING`**, hoặc kiểm tra xem code worker đang chặn/whitelist chuyển đổi kiểu nào).
- **L077 (Transaction Aborted Isolation `SQLSTATE 25P02`):** Nếu nhiều lệnh `ALTER TABLE` chạy chung transaction hoặc chạy tuần tự, một cột lỗi không được làm hỏng toàn bộ hoặc nuốt lỗi im lặng mà FE vẫn báo `success` giả tạo.
- **L097 (No Legacy Data Salvage Band-Aids & Pure Forward Core Fix — Chỉ thị trực tiếp của User `2026-11-17`):** **TUYỆT ĐỐI KHÔNG** đụng vào dữ liệu cũ, không viết logic khôi phục từ `_raw_data` hay regex lọc chuỗi rác `map[...]` trong câu lệnh DDL `ALTER COLUMN`. Các bảng cũ hỏng dữ liệu User có thể xóa làm lại; giải pháp chỉ tập trung chuẩn hóa tinh gọn luồng Core hiện tại và về sau (marshal JSON chuẩn ngay lúc ghi, `AlterColumn` chuẩn mực fail-fast, xóa cache `BatchBuffer` và phản hồi kết quả/lỗi đồng bộ lên UI).
- **Propose-Only (Rule #1, #9, #13):** Giữ 100% Read-Only đối với source code trong pha Brain (`S1 -> S2 -> DG0 -> S3 -> DG1/DG2`), trình bày **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** kèm **Code Demo** trực tiếp trên chat và dừng chờ User `APPROVE`.
