# Phân Tích Kỹ Thuật (AI Analysis): Cơ Chế Append Báo Cáo Daily Confluence & Nguyên Nhân 403

## 1. Phân tích cơ chế Append vào 1 trang Confluence duy nhất
- Trước đây, `confluence_runner.go` thay thế `{{.ReportDate}}` vào `TitleTemplate` khiến mỗi ngày tạo ra 1 trang con riêng biệt, và `CreateOrUpdateChildPage` ghi đè toàn bộ `body.storage.value`.
- Với cơ chế mới:
  1. `resolveSingleReportPageTitle` loại bỏ `{{.ReportDate}}` và khoảng trắng/dấu gạch ngang dư thừa để mọi ngày đều trỏ về **1 trang báo cáo cố định duy nhất** (ví dụ `[Daily Data Check] Core Trans Proxy History Service` hoặc `[Daily Data Check] Core Trans Proxy History Service - envTessting`).
  2. `findPageByTitle` lấy cả `version.number` lẫn `body.storage.value` (`expand=version,body.storage`).
  3. Do Confluence Server tự động chuyển đổi ký tự tiếng Việt có dấu thành HTML entities (`<h2>B&aacute;o C&aacute;o... (2026-10-15)</h2>`) nhưng giữ nguyên chuỗi ASCII `(2026-10-15)</h2>` và `</table>`, hàm `MergeDailySectionXHTML` định vị theo `(YYYY-MM-DD)</h2>` để:
     - **Append** xuống dưới cùng (`\n<hr/>` + section ngày mới) khi sang ngày tiếp theo.
     - **Update tại chỗ (in-place)** đúng section của ngày đó nếu bấm chạy lại trong ngày, tuyệt đối không làm mất dữ liệu các ngày trước và không bị nhân đôi bảng.

## 2. Phân tích lỗi 403 trên Confluence (`search content returned status 403`)
- Khi kiểm tra trực tiếp với Atlassian API (`goopay.atlassian.net`), response thực tế trả về từ Atlassian là:
  `403 FORBIDDEN: {"message":"Request rejected because caller cannot access Confluence"}`
- Kiểm tra `/rest/api/3/myself?expand=groups,applicationRoles` cho thấy tài khoản `trai.nguyen@goopay.vn` tại thời điểm chạy chỉ còn thuộc nhóm `jira-users-goopay`, bị mất nhóm quyền truy cập sản phẩm Confluence (`cf_user_gp` / `cf_def_GP`).
- Việc bổ sung đọc `resp.Body` trong `findPageByTitle` giúp hiển thị rõ nguyên nhân từ Atlassian trên `cdc_activity_log`, kết hợp với `doWithRetry` để tự động vượt qua các lỗi 403/429/5xx thoáng qua từ Atlassian Cloud Edge.
