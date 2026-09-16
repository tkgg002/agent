# 01 Requirements — Feature: Add "Data Type Current" Column to Mapping Fields Page

## 1. Context & Business Needs
- **URL:** `http://localhost:5173/shadow/72/mappings?binding_id=190`
- **Mục tiêu:** Cho phép Operator quan sát kiểu dữ liệu thực tế (Physical Data Type) đang tồn tại trên shadow table trong PostgreSQL, đối chiếu với kiểu dữ liệu cấu hình trong mapping rule (Target Data Type) để phát hiện drift schema cần đồng bộ DDL (Sync Fields).

## 2. Functional Requirements
- **FR-1:** Backend cung cấp endpoint `GET /api/introspection/shadow-columns-with-types/:table?schema=...` trả về map `{ [column_name: string]: string }` chứa kiểu dữ liệu vật lý đã được chuẩn hóa.
- **FR-2:** Backend thực hiện chuẩn hóa ANSI standard data types từ `information_schema.columns` (`character varying` -> `VARCHAR`, `timestamp with time zone` -> `TIMESTAMPTZ`, `timestamp without time zone` -> `TIMESTAMP`, `character` -> `CHAR`, `udt_name` cho `USER-DEFINED`/`ARRAY`).
- **FR-3:** Frontend `MappingFieldsPage` hiển thị cột "Data Type Current" ngay sau cột "Data Type Target".
- **FR-4:** Drift Detection trực quan:
  - Nếu cột chưa tồn tại trên shadow DB: hiển thị `—` (dash).
  - Nếu kiểu dữ liệu trùng khớp: hiển thị `<Tag color="green">{currentType}</Tag>`.
  - Nếu có drift (lệch kiểu dữ liệu): hiển thị `<Tag color="orange">{currentType} ⚠️</Tag>` kèm Tooltip cảnh báo cần Sync Fields để ALTER COLUMN TYPE.
- **FR-5:** So sánh kiểu dữ liệu loại trừ tham số precision/length (ví dụ `VARCHAR(255)` vs `VARCHAR`) để không phát sinh false positive drift.

## 3. Non-Functional Requirements
- **NFR-1 (Backward Compatibility):** Không phá vỡ endpoint legacy `/api/introspection/shadow-columns/:table`.
- **NFR-2 (Fail-Safe):** Khi endpoint introspection lỗi, Frontend fallback hiển thị an toàn `—` mà không làm crash page.
- **NFR-3 (Core Systems Safety):** Chỉ thực hiện SELECT metadata từ `information_schema.columns`, tuyệt đối không can thiệp DDL runtime.
