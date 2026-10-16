# Kế Hoạch & Kết Quả Kiểm Thử (Validation): Confluence Daily Append & 403 Diagnostics

## 1. Danh mục kịch bản kiểm thử (Test Cases)
1. **TC-01 (`TestMergeDailySectionXHTML_AppendAndIdempotentUpdate`):**
   - Khi trang Confluence chưa có nội dung -> trả về nguyên vẹn section ngày đầu tiên (`2026-10-14`).
   - Sang ngày kế tiếp (`2026-10-15`) -> nối (`append`) section `2026-10-15` xuống phía dưới section `2026-10-14` ngăn cách bởi `<hr/>` (kể cả khi Confluence đã encode tiếng Việt thành HTML entities như `B&aacute;o C&aacute;o...`).
   - Khi chạy lại (`re-run`) cùng ngày `2026-10-15` -> cập nhật tại chỗ (`in-place`) đúng khối `2026-10-15`, giữ nguyên `2026-10-14` và `2026-10-16`, không bị nhân đôi bảng.
2. **TC-02 (`TestCreateOrUpdateChildPage_AppendFlowAndRetry403`):**
   - Giả lập lỗi `403 Forbidden` tạm thời ở lần gọi `GET /wiki/rest/api/content` đầu tiên -> `doWithRetry` tự động retry và thành công ở lần thử thứ 2.
   - Ngày 1 (`2026-10-14`): Tạo mới trang (`POST /wiki/rest/api/content`, `version = 1`).
   - Ngày 2 (`2026-10-15`): Tìm thấy trang hiện có với `expand=version,body.storage`, gộp nối tiếp nội dung ngày `2026-10-15` xuống dưới ngày `2026-10-14` và cập nhật (`PUT /wiki/rest/api/content/{id}`, `version = 2`).
3. **TC-03 (`TestResolveSingleReportPageTitle`):**
   - Chuẩn hóa mọi cấu hình `title_template` cũ có chứa `{{.ReportDate}}` (như `"[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service"` hoặc `"[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service - envTessting"`) về đúng 1 tên trang cố định duy nhất (`"[Daily Data Check] Core Trans Proxy History Service"` / `"[Daily Data Check] Core Trans Proxy History Service - envTessting"`).

## 2. Bằng chứng thực thi (Execution Output)
Lệnh chạy:
```bash
go test -v ./pkgs/confluence/... ./internal/service/confluence/...
```
Kết quả:
```text
=== RUN   TestXHTMLEscaping
--- PASS: TestXHTMLEscaping (0.00s)
=== RUN   TestMergeDailySectionXHTML_AppendAndIdempotentUpdate
--- PASS: TestMergeDailySectionXHTML_AppendAndIdempotentUpdate (0.00s)
=== RUN   TestCreateOrUpdateChildPage_AppendFlowAndRetry403
--- PASS: TestCreateOrUpdateChildPage_AppendFlowAndRetry403 (2.93s)
PASS
ok  	centralized-data-service/pkgs/confluence	3.477s
=== RUN   TestCandidateObjectNames
--- PASS: TestCandidateObjectNames (0.00s)
=== RUN   TestExtractCleanObjectName
--- PASS: TestExtractCleanObjectName (0.00s)
=== RUN   TestBuildConfluenceXHTML
--- PASS: TestBuildConfluenceXHTML (0.00s)
=== RUN   TestLiveBranchLineageAndReportRow
--- PASS: TestLiveBranchLineageAndReportRow (0.04s)
=== RUN   TestResolveSingleReportPageTitle
--- PASS: TestResolveSingleReportPageTitle (0.00s)
PASS
ok  	centralized-data-service/internal/service/confluence	0.577s
```
