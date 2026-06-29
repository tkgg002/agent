# Workspace Context: Integration of Recon Smoke Test (Phases 5, 6, 7)

## Bối cảnh & Mục tiêu
Tích hợp và hoàn thiện tính năng Reconciliation Smoke Test trên toàn bộ hệ thống Centralized Data Service (CDS) bao gồm:
- **Phase 5 (Backend Service)**: Đọc kết quả từ bảng `cdc_recon_smoke_result` thông qua repo GORM, map sang API DTO tương thích để giao tiếp với Web UI.
- **Phase 6 (Frontend Web)**: Cải tiến giao diện Data Integrity của `cdc-cms-web` thành 3 cột hiển thị rõ ràng và hỗ trợ hiển thị Lag theo segment.
- **Phase 7 (Worker & Metrics)**: Định nghĩa các Prometheus metrics an toàn dạng O(1) và emit chúng từ Worker trong quá trình chạy smoke test.

## Cấu trúc Dự án liên quan
- **Backend Service**: `cdc-cms-service` (Golang)
- **Frontend UI**: `cdc-cms-web` (React/TypeScript)
- **Worker & Metrics**: `cdc-worker` (Golang), `pkgs/metrics`
