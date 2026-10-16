# Hồ sơ Giải pháp Kỹ thuật: Gom Báo Cáo Confluence Daily vào 1 Trang Duy Nhất & Append Ngày Tiếp Theo Phía Dưới

## 1. Mục tiêu Yêu cầu
- Thay vì mỗi ngày tạo 1 trang con (1 file) Confluence mới hoặc ghi đè mất dữ liệu ngày cũ:
  1. **Ghi chung vào 1 trang (1 file) Confluence cố định** cho mỗi lịch trình/service.
  2. **Append phía dưới dữ liệu cũ khi sang ngày tiếp theo:** Ngày mới (`2026-10-15`, `2026-10-16`...) sẽ được nối (append) xuống ngay bên dưới báo cáo của ngày trước đó (`2026-10-14`).
  3. **Idempotent khi chạy lại trong cùng 1 ngày:** Nếu bấm "Chạy ngay" nhiều lần trong cùng 1 ngày, hệ thống chỉ cập nhật lại đúng bảng của ngày đó chứ không nhân đôi 2 bảng trùng ngày.

---

## 2. Phân tích Kỹ thuật & Xử lý Đặc thù Confluence Storage XHTML

### 2.1. Chuẩn hóa Tiêu đề Trang Cố định (`resolveSingleReportPageTitle`)
- Hiện tại các lịch trình đang lưu `title_template` có biến `{{.ReportDate}}`, ví dụ:
  - `"[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service"`
  - `"[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service - envTessting"`
- Nếu thay `{{.ReportDate}}` bằng ngày cụ thể thì mỗi ngày tiêu đề trang sẽ khác nhau -> Confluence sẽ tạo trang mới mỗi ngày.
- **Giải pháp:**
  - Hàm `resolveSingleReportPageTitle(titleTemplate string)` sẽ tự động loại bỏ placeholder `{{.ReportDate}}` (và dọn khoảng trắng/dấu gạch ngang thừa liền kề) để tạo ra tên file cố định xuyên suốt các ngày:
    - `"[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service"` ➔ **`"[Daily Data Check] Core Trans Proxy History Service"`**
    - `"[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service - envTessting"` ➔ **`"[Daily Data Check] Core Trans Proxy History Service - envTessting"`**
  - Nhờ đó, không cần sửa lại từng schedule trong DB mà tất cả các lần chạy tự động tiếp theo đều gom vào đúng 1 trang duy nhất.

### 2.2. Đọc `body.storage` hiện tại và Append / Update theo Ngày (`pkgs/confluence/client.go`)
1. **Lấy nội dung hiện tại của trang khi `findPageByTitle`:**
   - Mở rộng query GET:
     `%s/wiki/rest/api/content?spaceKey=%s&title=%s&expand=version,body.storage`
   - Parse `body.storage.value` vào `PageContentItem`.
2. **Lưu ý kỹ thuật quan trọng về HTML Entities của Confluence:**
   - Thực tế kiểm tra từ Page `501088257` trên `goopay.atlassian.net` cho thấy: khi gửi chuỗi tiếng Việt có dấu `<h2>Báo Cáo Đối Soát Dữ Liệu Hằng Ngày (2026-10-15)</h2>`, Confluence tự động chuyển các ký tự có dấu thành HTML entity:
     `<h2>B&aacute;o C&aacute;o Đối So&aacute;t Dữ Liệu Hằng Ng&agrave;y (2026-10-15)</h2>`
   - Tuy nhiên, phần **`(2026-10-15)</h2>`** và các thẻ `<h2>...</h2>...<table...>...</table>` là **100% ASCII**, giữ nguyên tuyệt đối trong `body.storage.value`.
3. **Thuật toán `mergeDailySectionXHTML(existingXHTML, sectionKey, newSectionXHTML)`:**
   - Với `sectionKey = "(" + reportDate + ")</h2>"` (ví dụ `"(2026-10-15)</h2>"`):
   - **Trường hợp 1 — Ngày mới (chưa có `sectionKey` trong `existingXHTML`):**
     - Nếu `existingXHTML` rỗng: trả về `newSectionXHTML`.
     - Nếu `existingXHTML` đã có dữ liệu của các ngày trước: nối thêm `<hr/>` + `newSectionXHTML` vào **cuối trang (phía dưới ngày cũ)**:
       ```go
       return strings.TrimSpace(existingXHTML) + "\n<hr/>" + newSectionXHTML
       ```
   - **Trường hợp 2 — Chạy lại cùng ngày (đã có `sectionKey` trong `existingXHTML`):**
     - Tìm vị trí thẻ `<h2>` mở đầu chứa `sectionKey` và vị trí kết thúc `</table>` của khối ngày đó, thay thế đúng đoạn `<h2>...</table>` đó bằng `newSectionXHTML` mới nhất (giữ nguyên toàn bộ các ngày trước và sau).

---

## 3. Các File Dự Kiến Thay Đổi (Sau khi User `APPROVE`)

### File 1: [`centralized-data-service/pkgs/confluence/client.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/pkgs/confluence/client.go)
- Bổ sung trường `SectionDate string` (hoặc `AppendMode bool`) trong `CreateChildPageRequest`.
- Trong `findPageByTitle`: lấy `expand=version,body.storage` và đọc đầy đủ `resp.Body` khi `StatusCode != 200` (kèm retry backoff).
- Khi trang đã tồn tại (`existingPage != nil`):
  - Gộp `existingPage.Body.Storage.Value` với `req.BodyXHTML` bằng `mergeDailySectionXHTML(existingPage.Body.Storage.Value, req.SectionDate, req.BodyXHTML)` rồi gọi `updatePage`.

### File 2: [`centralized-data-service/internal/service/confluence/confluence_runner.go`](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/confluence/confluence_runner.go)
- Thêm hàm `resolveSingleReportPageTitle(titleTemplate string) string` để cố định tên trang báo cáo (bỏ `{{.ReportDate}}` khỏi tiêu đề trang để các ngày tiếp theo cùng ghi vào 1 file).
- Truyền `SectionDate: reportDateStr` vào `CreateOrUpdateChildPage`.

### File 3: [`cdc-cms-web/src/pages/ActivityManager.tsx`](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/ActivityManager.tsx)
- Cập nhật giá trị mặc định của `confluence_title_template` khi tạo mới lịch trình thành `"[Daily Data Check] Core Trans Proxy History Service"` (không gắn `{{.ReportDate}}` vào tên file).
