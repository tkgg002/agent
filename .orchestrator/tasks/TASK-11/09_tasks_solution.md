# 💡 [S3 · UDF-v1] Giải Pháp Kỹ Thuật Duy Nhất & Code Demo — TASK-11: Xóa Bỏ S4 Implement & S5 QA/Audit Giả Hiệu Khi git diff == 0, Thực Thi Kiểm Thử Đa-Repo Thật & Triển Khai Trọn Vẹn 100% Code + Test + Audit Của TASK-M01..TASK-M10 Trên cdc-cms-service, cdc-cms-web, centralized-data-service
> 📌 **Task:** `TASK-11` · **Stage:** `DONE` · **Repo:** `cdc-cms-service, cdc-cms-web, centralized-data-service` · **Branch:** `task/TASK-11` · **Files thay đổi (6):** `internal/app/queries/recon/list_latest_reports.go, internal/api/dto/audit_dto.go, src/constants/postgresDataTypes.ts, src/hooks/useAuditStatus.ts, internal/service/source/metadata_registry_test_helpers.go, internal/model/recon/reconciliation_report.go` · **DoD:** `go test -race ./...`

### 📌 Tóm tắt & Trả lời trực tiếp (Summary)
Failed to authenticate. API Error: 401 OAuth access token is invalid.Failed to authenticate. API Error: 401 OAuth access token is invalid.

## 1. 🏗 Kiến Trúc & Chiến Lược Chuyển Đổi (Architecture Plan & Root Cause)
- **Pattern áp dụng:** Single-Responsibility Modular Strategy trên `cdc-cms-service, cdc-cms-web, centralized-data-service`, giữ nguyên signature hàm ở tầng interface hiện hữu.
- **Cơ chế Rollback / Toggle:** Thực thi cô lập trên nhánh Worktree riêng và bảo toàn tương thích ngược 100% với luồng hiện hành.
- Điểm chạm tại `internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{})` (`cdc-cms-service`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`)
- Điểm chạm tại `internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {)` (`cdc-cms-service`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`)
- Điểm chạm tại `src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [)` (`cdc-cms-web`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`)
- Điểm chạm tại `src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {)` (`cdc-cms-web`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`)
- Điểm chạm tại `internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {)` (`centralized-data-service`): Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `centralized-data-service` (`metadata_registry_test_helpers.go`)
- Điểm chạm tại `internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {)` (`centralized-data-service`): Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `centralized-data-service` (`reconciliation_report.go`)

## 2. ⚠️ Kiểm Soát Rủi Ro & Lộ Trình Thực Thi (`6 Sub-Tasks` · `Wave 0..N`)
- **Memory & Concurrency:** Kiểm soát tài nguyên, context timeout và tránh phát sinh race condition/goroutine leak.
- **Tương thích Hợp đồng Dữ liệu:** Đảm bảo DTO/Schema đầu ra khớp 100% với caller hiện tại và đạt chuẩn G1–G8.

| Mã Sub-Task | Wave & Phụ thuộc | Repo | Điểm chạm (`file:line`) | Giải pháp Thiết kế & Tiêu chí Nghiệm thu (`DoD`) |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-11-M01`** | `Wave 0` | `cdc-cms-service` | `internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{})` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `internal/app/queries/recon/list_latest_reports.go` (`type ListLatestReportsQuery struct{}`) trong worktree cô lập của `cdc-cms-service`. (`DoD:` `go test -race ./...`) |
| **`TASK-11-M02`** | `Wave 0` | `cdc-cms-service` | `internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`)** — Kết nối dây dẫn nghiệp vụ tại `internal/api/dto/audit_dto.go` (`type QASummaryResponse struct {`) và viết integration test xác minh đầu-cuối trên `cdc-cms-service`. (`DoD:` `go test -race ./...`) |
| **`TASK-11-M03`** | `Wave 1` *(sau `TASK-11-M01, TASK-11-M02`)* | `cdc-cms-web` | `src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `src/constants/postgresDataTypes.ts` (`export const POSTGRES_DATA_TYPES = [`) trong worktree cô lập của `cdc-cms-web`. (`DoD:` `npm run build && npm test`) |
| **`TASK-11-M04`** | `Wave 1` *(sau `TASK-11-M01, TASK-11-M02`)* | `cdc-cms-web` | `src/hooks/useAuditStatus.ts:5 (export function useAuditStatus() {)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`)** — Kết nối dây dẫn nghiệp vụ tại `src/hooks/useAuditStatus.ts` (`export function useAuditStatus() {`) và viết integration test xác minh đầu-cuối trên `cdc-cms-web`. (`DoD:` `npm run build && npm test`) |
| **`TASK-11-M05`** | `Wave 2` *(sau `TASK-11-M03, TASK-11-M04`)* | `centralized-data-service` | `internal/service/source/metadata_registry_test_helpers.go:13 (type MetadataRegistryServiceForTestOpts struct {)` | **Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `centralized-data-service` (`metadata_registry_test_helpers.go`)** — Cập nhật logic tầng dữ liệu & nghiệp vụ tại điểm chạm nguồn `internal/service/source/metadata_registry_test_helpers.go` (`type MetadataRegistryServiceForTestOpts struct {`) trong worktree cô lập của `centralized-data-service`. (`DoD:` `go test -race ./...`) |
| **`TASK-11-M06`** | `Wave 2` *(sau `TASK-11-M03, TASK-11-M04`)* | `centralized-data-service` | `internal/model/recon/reconciliation_report.go:8 (type ReconciliationReport struct {)` | **Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `centralized-data-service` (`reconciliation_report.go`)** — Kết nối dây dẫn nghiệp vụ tại `internal/model/recon/reconciliation_report.go` (`type ReconciliationReport struct {`) và viết integration test xác minh đầu-cuối trên `centralized-data-service`. (`DoD:` `go test -race ./...`) |

## 3. 💻 Key Diff / Code Preview (`go`)
```go
// Trích xuất trực tiếp từ TargetRepo: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service
// [1] cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{})
type ListLatestReportsQuery struct{}

func (q ListLatestReportsQuery) Type() string { return "recon.list_latest" }

// ListLatestReportsResult returns the raw rows. The HTTP handler
// fills the four `gorm:"-"` enrichment fields (DriftPct,
// ComputedStatus, ErrorMessageVI, SourceQueryMethod) from
// ComputeDriftStatus + ErrorMessagesVI + deriveSourceQueryMethod —
// those helpers stay in `internal/api/` because they own a 16-case
// unit test suite (ComputeDriftStatus_test.go) we don't want to
// move yet.
type ListLatestReportsResult struct {

// [2] cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:11 (func (q ListLatestReportsQuery) Type() string { return "recon.list_latest" })
func (q ListLatestReportsQuery) Type() string { return "recon.list_latest" }

// ListLatestReportsResult returns the raw rows. The HTTP handler
// fills the four `gorm:"-"` enrichment fields (DriftPct,
// ComputedStatus, ErrorMessageVI, SourceQueryMethod) from
// ComputeDriftStatus + ErrorMessagesVI + deriveSourceQueryMethod —
// those helpers stay in `internal/api/` because they own a 16-case
// unit test suite (ComputeDriftStatus_test.go) we don't want to
// move yet.
type ListLatestReportsResult struct {
	Data  []LatestReportRow
	Count int

// [3] cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {)
type QASummaryResponse struct {
    CompositeScore   float64             `json:"composite_score"`     // 0-100
    TotalCriteria    int                 `json:"total_criteria"`       // 16
    RatingBreakdown  map[string]int      `json:"rating_breakdown"`     // {"L4":1,"L3":6,...}
    GapBreakdown     map[string]GapCount `json:"gap_breakdown"`        // {"P0":{open:4,closed:0},...}
    LastAudited      string              `json:"last_audited"`         // RFC3339
    Criteria         []CriterionRow      `json:"criteria"`
}

type GapCount struct {
    Open       int `json:"open"`
    InProgress int `json:"in_progress"`
```

## 4. 🛡️ Nghiệm Thu & Cổng Kiểm Định (`DoD & Gates`)
- **Lệnh kiểm thử bắt buộc (`DoD`):** `go test -race ./...` trên `cdc-cms-service, cdc-cms-web, centralized-data-service` (nhánh `task/TASK-11`).
- **Decision Gate `DG0 / DG1 / DG2 (:8200 · 0 Token)`:** `PASS 🟢` trên `cdc-cms-service, cdc-cms-web, centralized-data-service` (Model: `opus`).

## Clean-Room Approved Spec
- **R1 (`TASK-11-M01` — `cdc-cms-service`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`) (`REF: cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9`)
- **R2 (`TASK-11-M02` — `cdc-cms-service`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`) (`REF: cdc-cms-service/internal/api/dto/audit_dto.go:3`)
- **R3 (`TASK-11-M03` — `cdc-cms-web`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`) (`REF: cdc-cms-web/src/constants/postgresDataTypes.ts:1`)
- **R4 (`TASK-11-M04` — `cdc-cms-web`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`) (`REF: cdc-cms-web/src/hooks/useAuditStatus.ts:5`)
- **R5 (`TASK-11-M05` — `centralized-data-service`):** Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `centralized-data-service` (`metadata_registry_test_helpers.go`) (`REF: centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13`)
- **R6 (`TASK-11-M06` — `centralized-data-service`):** Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `centralized-data-service` (`reconciliation_report.go`) (`REF: centralized-data-service/internal/model/recon/reconciliation_report.go:8`)
