# Báo Cáo Thay Đổi (Change Report): Confluence Single-Page Daily Append & 403 Retry/Logging

## 1. Danh sách các file đã thay đổi
| File | Số dòng thay đổi (ước tính) | Mô tả thay đổi (Overview) |
|---|---|---|
| `centralized-data-service/pkgs/confluence/client.go` | +160 / -35 dòng | - Bổ sung `SectionDate` vào `CreateChildPageRequest`.<br>- Nâng cấp `findPageByTitle` lấy thêm `expand=version,body.storage` và đọc trọn vẹn `resp.Body` khi gặp lỗi HTTP (`403`, `401`, `5xx`...) thay vì nuốt lỗi.<br>- Bổ sung `doWithRetry` (retry 3 lần với exponential backoff + jitter và headers `User-Agent`, `X-Atlassian-Token: no-check`).<br>- Bổ sung `MergeDailySectionXHTML` để nối (`append`) báo cáo ngày mới xuống dưới trang hiện có (`<hr/>` + section ngày mới), hoặc cập nhật tại chỗ (`in-place`) nếu chạy lại cùng một ngày. |
| `centralized-data-service/internal/service/confluence/confluence_runner.go` | +30 / -5 dòng | - Thêm hàm `resolveSingleReportPageTitle(titleTemplate)` để chuẩn hóa tiêu đề về 1 trang cố định duy nhất (tự động loại bỏ `{{.ReportDate}}` và dấu gạch ngang thừa ở cả các lịch trình cũ đang lưu trong DB).<br>- Truyền `SectionDate: reportDateStr` vào `CreateOrUpdateChildPage`. |
| `cdc-cms-web/src/pages/ActivityManager.tsx` | +4 / -4 dòng | - Đổi giá trị mặc định `confluence_title_template` từ `'[Daily Data Check] {{.ReportDate}} - Core Trans Proxy History Service'` sang `'[Daily Data Check] Core Trans Proxy History Service'`. |
| `centralized-data-service/pkgs/confluence/client_test.go` | +165 dòng | - Thêm unit test `TestMergeDailySectionXHTML_AppendAndIdempotentUpdate` và `TestCreateOrUpdateChildPage_AppendFlowAndRetry403`. |
| `centralized-data-service/internal/service/confluence/confluence_runner_test.go` | +48 / -4 dòng | - Thêm unit test `TestResolveSingleReportPageTitle` và đồng bộ chuỗi trạng thái trong test. |
