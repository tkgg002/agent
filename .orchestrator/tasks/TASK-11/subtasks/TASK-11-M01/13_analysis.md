# 13 Analysis: TASK-11-M01 (`cdc-cms-service`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 (type ListLatestReportsQuery struct{})` | `VERIFIED` | `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/recon/list_latest_reports.go` (missing on disk and not in diff.patch: cdc-cms-service:internal/app/queries/recon/list_latest_reports.go) |
| Hợp đồng Đầu vào (`Wave 0`) | `VERIFIED` | `UpstreamReady = true` (DependsOn: ``) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-11/task-11-m01` (`.orchestrator/worktrees/TASK-11/cdc-cms-service`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/recon/list_latest_reports.go`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/queries/recon/list_latest_reports.go (lines 9-33)
   9 | type ListLatestReportsQuery struct{}
  10 | 
  11 | func (q ListLatestReportsQuery) Type() string { return "recon.list_latest" }
  12 | 
  13 | // ListLatestReportsResult returns the raw rows. The HTTP handler
  14 | // fills the four `gorm:"-"` enrichment fields (DriftPct,
  15 | // ComputedStatus, ErrorMessageVI, SourceQueryMethod) from
  16 | // ComputeDriftStatus + ErrorMessagesVI + deriveSourceQueryMethod —
  17 | // those helpers stay in `internal/api/` because they own a 16-case
  18 | // unit test suite (ComputeDriftStatus_test.go) we don't want to
  19 | // move yet.
  20 | type ListLatestReportsResult struct {
  21 | 	Data  []LatestReportRow
  22 | 	Count int
  23 | }
  24 | 
  25 | // ListLatestReportsHandler resolves the query against an injected
  26 | // ReconReader.
  27 | type ListLatestReportsHandler struct {
  28 | 	reader ReconReader
  29 | }
  30 | 
  31 | func NewListLatestReportsHandler(r ReconReader) *ListLatestReportsHandler {
  32 | 	return &ListLatestReportsHandler{reader: r}
  33 | }
```
