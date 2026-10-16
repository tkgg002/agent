# Kế Hoạch Triển Khai Chi Tiết (AI Implementation Plan): Confluence Daily Append & 403 Fix

1. **Bước 1 — Nâng cấp Confluence HTTP Client (`pkgs/confluence/client.go`):**
   - Đọc `resp.Body` khi `findPageByTitle` trả mã khác `200 OK` để giữ nguyên thông điệp lỗi từ Atlassian.
   - Bổ sung `doWithRetry` cho các mã `403`, `429`, `500`, `502`, `503`, `504`.
   - Mở rộng `findPageByTitle` với `expand=version,body.storage` để lấy nội dung XHTML hiện tại của trang.
   - Cài đặt `MergeDailySectionXHTML(existingXHTML, sectionDate, newSectionXHTML)` dùng mốc `(YYYY-MM-DD)</h2>` để nối tiếp ngày mới xuống cuối trang hoặc cập nhật tại chỗ nếu trùng ngày.
2. **Bước 2 — Chuẩn hóa Single-Page Title trong `confluence_runner.go`:**
   - Thêm `resolveSingleReportPageTitle` để tự động loại bỏ `{{.ReportDate}}` khỏi `title_template` (kể cả lịch trình cũ trong DB) và truyền `SectionDate: reportDateStr`.
3. **Bước 3 — Cập nhật UI `ActivityManager.tsx`:**
   - Đặt mặc định `confluence_title_template` là `'[Daily Data Check] Core Trans Proxy History Service'`.
4. **Bước 4 — Kiểm thử (`go test`):**
   - Viết unit test kiểm chứng toàn bộ luồng tạo mới, append ngày kế tiếp, re-run cùng ngày (idempotent update), và retry khi gặp 403 tạm thời.
