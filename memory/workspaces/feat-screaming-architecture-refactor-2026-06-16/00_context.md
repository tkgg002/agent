# Workspace Context: Screaming Architecture - Functional Groups

## Objective
Tái cấu trúc thư mục `internal/app/commands` và `internal/app/queries` trong project `cdc-cms-service` từ cấu trúc phẳng (flat) sang cấu trúc phân nhóm theo chức năng (Screaming Architecture) nhằm tăng tính modular, dễ maintain và đúng định hướng thiết kế trong `NOTE.ini`:
- Nhóm `source`: Quản lý Registry, Sources, Table Discovery.
- Nhóm `shadow`: Quản lý Shadow Bindings và Schema Transform.
- Nhóm `master`: Quản lý Master Registry, Master Mapping Rules.
- Nhóm `governance`: Quản trị schema, các đề xuất (Proposals), phê duyệt và thay đổi cấu trúc DB vật lý (Apply DDL, Drop Column).
- Nhóm `recon`: Đối soát dữ liệu (Reconciliation) và xử lý log lỗi.
- Nhóm `scheduler`: Quản lý các tiến trình nền (Schedules, Worker jobs, Wizard session).
- Nhóm `system`: Quản lý các chức năng hệ thống (Health Check, Metrics, Audit, Alerts).

## Governance Compliance
- Trạng thái vi phạm: Không vi phạm. Workspace được tạo ngay khi nhận chỉ thị tái cấu trúc mới.
- Gốc rễ lỗi vi phạm quy trình Governance trước đó: Không có (N/A).
