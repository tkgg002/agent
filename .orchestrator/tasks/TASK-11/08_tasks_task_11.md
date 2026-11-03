# 08 Tasks: Danh mục Sub-Task — TASK-11 (Xóa Bỏ S4 Implement & S5 QA/Audit Giả Hiệu Khi git diff == 0, Thực Thi Kiểm Thử Đa-Repo Thật & Triển Khai Trọn Vẹn 100% Code + Test + Audit Của TASK-M01..TASK-M10 Trên cdc-cms-service, cdc-cms-web, centralized-data-service)

> **Affected Repos**: `cdc-cms-service, cdc-cms-web, centralized-data-service`
> **Tier**: T2
> **Suffix**: `task_11`

### Nhóm 1: Thực thi & Kiểm thử trên `cdc-cms-service` (389 files)
- [ ] **TASK-11-M01 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`)** (`Wave 0` · Repo: `cdc-cms-service` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{})`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `internal/app/queries/recon/list_latest_reports.go` (`type ListLatestReportsQuery struct{}`) trong worktree cô lập của `cdc-cms-service`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-11-M02 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`)** (`Wave 0` · Repo: `cdc-cms-service` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)`)
  - **Context (`file:line`)**: `cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `internal/api/dto/audit_dto.go` (`type QASummaryResponse struct {`) và viết integration test xác minh đầu-cuối trên `cdc-cms-service`.
  - **Test / DoD**: go test -race ./...

### Nhóm 2: Thực thi & Kiểm thử trên `cdc-cms-web` (53 files)
- [ ] **TASK-11-M03 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`)** (`Wave 1` · Repo: `cdc-cms-web` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-11-M01, TASK-11-M02`)
  - **Context (`file:line`)**: `cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `src/constants/postgresDataTypes.ts` (`export const POSTGRES_DATA_TYPES = [`) trong worktree cô lập của `cdc-cms-web`.
  - **Test / DoD**: npm run build && npm test
- [ ] **TASK-11-M04 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`)** (`Wave 1` · Repo: `cdc-cms-web` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-11-M01, TASK-11-M02`)
  - **Context (`file:line`)**: `cdc-cms-web/src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `src/hooks/useAuditStatus.ts` (`export function useAuditStatus() {`) và viết integration test xác minh đầu-cuối trên `cdc-cms-web`.
  - **Test / DoD**: npm run build && npm test

### Nhóm 3: Thực thi & Kiểm thử trên `centralized-data-service` (245 files)
- [ ] **TASK-11-M05 — Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `centralized-data-service` (`metadata_registry_test_helpers.go`)** (`Wave 2` · Repo: `centralized-data-service` · Assignee: `Coder` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-11-M03, TASK-11-M04`)
  - **Context (`file:line`)**: `centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {)`
  - **Mô tả**: Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `internal/service/source/metadata_registry_test_helpers.go` (`type MetadataRegistryServiceForTestOpts struct {`) trong worktree cô lập của `centralized-data-service`.
  - **Test / DoD**: go test -race ./...
- [ ] **TASK-11-M06 — Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `centralized-data-service` (`reconciliation_report.go`)** (`Wave 2` · Repo: `centralized-data-service` · Assignee: `Tester` · Model: `claude-opus-4-6 (antigravity)` · DependsOn: `TASK-11-M03, TASK-11-M04`)
  - **Context (`file:line`)**: `centralized-data-service/internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {)`
  - **Mô tả**: Kết nối dây dẫn nghiệp vụ tại `internal/model/recon/reconciliation_report.go` (`type ReconciliationReport struct {`) và viết integration test xác minh đầu-cuối trên `centralized-data-service`.
  - **Test / DoD**: go test -race ./...
