# Yêu cầu: Loại bỏ Smoke Check khỏi Lịch sử đối soát

## 1. Bối cảnh
Khi truy vấn lịch sử đối soát của một bảng, các bản ghi Smoke Check (được sinh ra mỗi phút) gây trôi các bản ghi Chữa lành (Heal) hoặc các bản ghi đối soát chính ra khỏi trang đầu tiên. Việc này khiến người dùng khó theo dõi các hoạt động quan trọng.

## 2. Mục tiêu
Bổ sung tham số `exclude_smoke` vào API truy vấn Lịch sử đối soát và tích hợp trên Frontend (đặc biệt là trong modal Chữa lành `ExecuteHealModal`) để ẩn toàn bộ các bản ghi Smoke Check từ database khi cần thiết.

## 3. Các file cần chỉnh sửa
1. `cdc-cms-service/internal/app/queries/recon/recon_reader.go` (Interface)
2. `cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go` (Implement SQL GORM)
3. `cdc-cms-service/internal/app/queries/recon/get_table_history.go` (Query Struct & Handler)
4. `cdc-cms-service/internal/api/recon/reconciliation_handler_reports.go` (HTTP API Handler)
5. `cdc-cms-service/test/internal/app/queries/queries_test.go` (Test Mock Stub)
6. `cdc-cms-web/src/hooks/useReconStatus.ts` (Frontend Hooks)
7. `cdc-cms-web/src/components/ExecuteHealModal.tsx` (Frontend Components)
