# Kế hoạch Triển khai: Tích hợp Cảnh báo Đối soát (Reconciliation Alerts Integration)

## 1. Yêu cầu & Mục tiêu
Tích hợp cảnh báo đối soát thời gian thực vào khung giám sát sức khỏe hệ thống (System Health Monitoring Framework), liên kết các luồng đối soát và healing với bảng lưu trữ trạng thái `cdc_alerts`, đồng thời hiển thị và cho phép Acknowledge/Silence trực tiếp từ frontend.

## User Review Required
> [!IMPORTANT]
> - Để đồng bộ cảnh báo giữa Backend Worker (Centralized Data Service) và CMS Health Collector, chúng tôi sẽ chuẩn hóa định dạng labels cho alert `ReconDrift` và `ReconError` là: `{"segment": "<segment>", "table": "<table_name>"}`.
> - Bổ sung `ReconError` vào danh sách `ownedAlertNames` của collector trong CMS để hệ thống có thể tự động quét dọn (sweep) và resolve khi lỗi được khắc phục.

## Open Questions
Hiện tại không có câu hỏi mở nào. Mọi cấu trúc và thiết kế API đều khớp với hạ tầng hiện có.

## Proposed Changes

---

### Tải lên / Cập nhật Backend Worker (centralized-data-service)

#### [MODIFY] [recon_alert.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_alert.go)
- Thêm hàm `ResolveAlert(ctx context.Context, name string, labels map[string]string)` để cập nhật trạng thái alert sang `resolved` trong bảng `cdc_alerts`.
- Cập nhật hàm `alertOnReport` để tự động gọi `ResolveAlert` cho cả `ReconDrift` và `ReconError` khi status của run là thành công (`ok` hoặc `ok_empty`).

#### [MODIFY] [recon_engine.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine.go)
- Trong hàm `stampA`, thêm lời gọi `rc.alertOnReport(context.Background(), report.Segment, tableName, report.Status, report.MissingCount, report.Diff)` tương tự như ở `stampB` của Segment B.

#### [MODIFY] [recon_engine_segment_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_engine_segment_b.go)
- Trong hàm `stampB`, thêm lời gọi `rc.alertOnReport(context.Background(), report.Segment, tableName, report.Status, report.MissingCount, report.Diff)`.

#### [MODIFY] [recon_tier_b.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_b.go)
- Xóa bỏ các lời gọi `rc.alertOnReport` thủ công tại dòng 267 và 458 để tránh bị lặp cảnh báo (vì `stampB` đã tự động xử lý khi lưu report).

#### [MODIFY] [recon_execute_heal_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go)
- Trong hàm `finalizeReport`, khi trạng thái đạt `isFullyHealed`, gọi `h.reconCore.ResolveAlert(ctx, "ReconDrift", labels)` và `h.reconCore.ResolveAlert(ctx, "ReconError", labels)` để tắt cảnh báo ngay lập tức trên UI.

---

### Tải lên / Cập nhật CMS Service (cdc-cms-service)

#### [MODIFY] [system_health_queries.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_queries.go)
- Cập nhật query trong `queryReconciliation` trả về thêm trường `segment` cho snapshot.

#### [MODIFY] [system_health_alerts.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/observability/system_health_alerts.go)
- Bổ sung `ReconError` vào `ownedAlertNames`.
- Cập nhật `detectConditions` để tạo label alert `ReconDrift` dạng `{"segment": segment, "table": table}`.
- Bổ sung logic phát hiện `ReconError` khi status của report là `"error"`, với label tương tự.

---

### Tải lên / Cập nhật Frontend (cdc-cms-web)

#### [NEW] [useActiveAlerts.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useActiveAlerts.ts)
- Tạo hook React Query để gọi `GET /api/alerts/active`, `POST /api/alerts/:fingerprint/ack`, và `POST /api/alerts/:fingerprint/silence`.

#### [MODIFY] [SystemHealth.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/SystemHealth.tsx)
- Hiển thị danh sách cảnh báo từ `/api/alerts/active` một cách trực quan trong phần "Cảnh báo" thay thế cho banner tĩnh hiện tại.
- Cho phép người dùng bấm **Xác nhận (Acknowledge)** và **Tạm ẩn (Silence)** cảnh báo trực tiếp từ UI.

## Verification Plan

### Automated Tests
- Chạy toàn bộ unit test của `centralized-data-service` và `cdc-cms-service` để đảm bảo không lỗi regression.
- Lệnh kiểm tra quy trình governance: `python3 agent/tooling/verify_governance.py`

### Manual Verification
- Sử dụng subagent browser để verify hiển thị danh sách cảnh báo, các nút bấm Acknowledge, Silence trên UI trang SystemHealth.
