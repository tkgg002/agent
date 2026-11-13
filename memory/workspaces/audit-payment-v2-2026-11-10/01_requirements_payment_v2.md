---
tier: T2
mode: standard
suffix: payment_v2
workspace: audit-payment-v2-2026-11-10
created_at: 2026-11-10T10:50:55+07:00
---
# 01 Requirements — Audit Tài Liệu Confluence `Payment-v2` & Đánh Giá Mức Độ Đáp Ứng Hệ Thống `data-hub`

## 1. Yêu cầu của User
- Đọc kỹ tài liệu Confluence tại: `https://goopay.atlassian.net/wiki/spaces/~712020a3dcebcbe93646fea464f119dd38b08c/pages/512557312/Payment-v2` (bao gồm cả nội dung trang, bảng biểu, sơ đồ và các trang con liên quan nếu có).
- Nhận định xem hệ thống hiện tại (`data-hub`: `centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) đã đáp ứng được yêu cầu trong tài liệu `Payment-v2` chưa, các điểm nào đã đáp ứng (kèm bằng chứng `file:line`), và **cần làm gì thêm không** (Gap Analysis + Giải pháp đề xuất nếu có).
- **Câu hỏi tiếp theo (về `F1` — Kiểu mảng PostgreSQL `TEXT[]`, `BIGINT[]`, `INTEGER[]`, `UUID[]`):**
  1. Trong PostgreSQL, cột mảng (`TEXT[]`, `BIGINT[]`, `INTEGER[]`, `UUID[]`) thực chất là `data_type` / `udt_name` gì?
  2. Hệ thống CMS (`cdc-cms-web`, `cdc-cms-service`) và tầng DDL / TypeResolver (`centralized-data-service`) đã hỗ trợ kiểu mảng PostgreSQL này chưa? Nếu chưa đầy đủ ở khâu nào thì chỉ rõ `file:line` và đề xuất phương án bổ sung.
  3. Tại sao trên trang Shadow Mapping (`/shadow/:id/mappings?binding_id=...`) và Master Mapping (`/masters/:id/mappings`), ô chọn Data Type Target không có các kiểu mảng có gạch dưới `_` (như `_TEXT`, `_INT8`...) hoặc `TEXT[]`? Hệ thống đang thiếu chỗ này như thế nào?
  4. Đối với PostgreSQL kiểu `_text`: Dữ liệu MongoDB là mảng sub-documents (`item: [...]`) hoặc mảng chuỗi (`refundedOperationIds: [...]`), khi nạp vào PostgreSQL có phải bắt buộc chuyển đổi sang định dạng array literal `{...}` không? Hệ thống đã có bước chuyển đổi này chưa?

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **L001 (Rule-Flag Driven Metadata Column Detection Over Hardcoded Column Names):** Tuyệt đối CẤM so sánh cứng chuỗi tên cột metadata (`source_field_types`, `source_field_values`) trong bộ đối soát (`recon_multi_shadow_b.go`), bộ khôi phục BSON (`mongo_bson_restorer.go`, `mongo_extjson_unwrapper.go`) hay bộ dọn trường thừa (`mongo_stale_field_pruner.go`). Bắt buộc nhận diện vai trò cột qua cờ **"Chế độ tạo: (JSON) Subset — Loại Subset: Type/Presence | Value"** của quy tắc ánh xạ (`@type_presence:{...}` và `{...}`).
- **L002 (Zero Table-Specific Hardcoding In Generic Sync Engine & Strict Service Scope Discipline):** Tuyệt đối CẤM gán cứng tên cột đặc thù của một bảng (`transaction_history`, `payment_bill`, `payment`) hoặc quy tắc nghiệp vụ riêng lẻ vào lõi động cơ CDC (`centralized-data-service`). Phải dùng cơ chế 100% động dựa trên `mapping_rule`, `transform_spec` và cờ Subset của rule. Bám sát 100% phạm vi CDC (`data-hub`) và tôn trọng kiến trúc có bước Trigger từ `Master CDC -> Master Business`.
- **L073 (Schema Verification & ANSI SQL Type Normalization):** Khi đọc `information_schema.columns` của PostgreSQL, cột mảng có `data_type = 'ARRAY'` và `udt_name = '_text' | '_int8' | '_int4' | '_uuid' | '_varchar'`, cần chuẩn hóa đúng sang `TEXT[]`, `BIGINT[]`, `INTEGER[]`, `UUID[]`, `VARCHAR[]` thay vì chỉ `strings.ToUpper(dataType)`.
- **L015 (Inventory Complete Domain Action Catalog):** Kiểm kê đầy đủ các hành động nghiệp vụ từ UI -> Control Plane -> Data Plane kèm `file:line`.
- **L016 (Define Strategy & Driver Registry Patterns For Polymorphic Axes):** Đối chiếu các trục mở rộng đa nguồn, đa đích, đa loại biến đổi (`copy_1_to_1`, `flatten`, `multi_shadow`, `child_explode`, v.v.) theo chuẩn Design Pattern của hệ thống.
- **L017 (Respect Live Production Baseline):** Bám sát trạng thái vận hành thực tế của hệ thống (`system_overview_and_design_pattern.md`), bảo toàn 100% tương thích ngược (Open-Closed) và cấu trúc phân tầng chuẩn.
- **L031 (Verbatim Current-File Code Demo):** Mọi trích dẫn mã nguồn và Code Demo đề xuất phải đối chiếu trực tiếp từ file thật kèm `file:line`.
- **L033 (Executive Issues Summary Table):** Bắt buộc có Bảng Tổng Hợp Đối Chiếu Mức Độ Đáp Ứng & Các Việc Cần Làm Thêm ngay đầu báo cáo.
- **L040 (Propose-Only Discipline):** Giữ chế độ 100% Read-Only đối với source code, lập `09_tasks_solution_payment_v2.md` qua `DG0` và `DG1/DG2`, hiển thị Code Demo trên chat và dừng chờ `APPROVE`.
- **L074/L080/L081:** Đối chiếu chuẩn CDC Multi-Shadow / Flatten / Spec-Driven PK & Timestamp / Recon Window Alignment với yêu cầu nghiệp vụ Payment-v2.
