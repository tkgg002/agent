# 13 Analysis: TASK-11-M02 (`cdc-cms-service`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `cdc-cms-service/internal/api/dto/audit_dto.go:3 (type QASummaryResponse struct {)` | `VERIFIED` | `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/dto/audit_dto.go` (missing on disk and not in diff.patch: cdc-cms-service:internal/api/dto/audit_dto.go) |
| Hợp đồng Đầu vào (`Wave 0`) | `VERIFIED` | `UpstreamReady = true` (DependsOn: ``) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-11/task-11-m02` (`.orchestrator/worktrees/TASK-11/cdc-cms-service`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/dto/audit_dto.go`)
```go
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/dto/audit_dto.go (lines 3-27)
   3 | type QASummaryResponse struct {
   4 |     CompositeScore   float64             `json:"composite_score"`     // 0-100
   5 |     TotalCriteria    int                 `json:"total_criteria"`       // 16
   6 |     RatingBreakdown  map[string]int      `json:"rating_breakdown"`     // {"L4":1,"L3":6,...}
   7 |     GapBreakdown     map[string]GapCount `json:"gap_breakdown"`        // {"P0":{open:4,closed:0},...}
   8 |     LastAudited      string              `json:"last_audited"`         // RFC3339
   9 |     Criteria         []CriterionRow      `json:"criteria"`
  10 | }
  11 | 
  12 | type GapCount struct {
  13 |     Open       int `json:"open"`
  14 |     InProgress int `json:"in_progress"`
  15 |     Closed     int `json:"closed"`
  16 |     Wontfix    int `json:"wontfix"`
  17 | }
  18 | 
  19 | type CriterionRow struct {
  20 |     ID       string `json:"id"`
  21 |     Group    string `json:"group"`
  22 |     Title    string `json:"title"`
  23 |     Rating   string `json:"rating"`
  24 |     Evidence string `json:"evidence"`
  25 | }
  26 | 
  27 | type GapRow struct {
```
