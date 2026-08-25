# project_context.md — Context của Dự án & Workspace

## Dự án
- **Repository:** `data-hub` (`cdc-cms-service`, `centralized-data-service`, `cdc-auth-service`).
- **Domain:** Hệ thống CDC Data Hub truyền dẫn dữ liệu từ Source -> Shadow -> Master.

## Workspace Hiện Tại
- **Tên:** `fix-getmasterdb-connection-key`
- **Mục tiêu:** Xử lý sự cố ghi nhầm dữ liệu vào sai schema/bảng Master, chuẩn hóa định danh bảng Master FQN `<schema>.<table>`, bảo đảm an toàn với `NULL` trong PostgreSQL và đồng bộ tầng API CMS.
