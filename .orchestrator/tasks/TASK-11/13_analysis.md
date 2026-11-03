# 🔍 [S2 · UDF-v1] Điều Tra Mã Nguồn Đa Repo & Bằng Chứng `file:line` — TASK-11: Xóa Bỏ S4 Implement & S5 QA/Audit Giả Hiệu Khi git diff == 0, Thực Thi Kiểm Thử Đa-Repo Thật & Triển Khai Trọn Vẹn 100% Code + Test + Audit Của TASK-M01..TASK-M10 Trên cdc-cms-service, cdc-cms-web, centralized-data-service
> 📌 **Task:** `TASK-11` · **Stage:** `S2 INVESTIGATE` · **Repos (3):** `cdc-cms-service, cdc-cms-web, centralized-data-service` · **Files Scanned:** `687` · **Symbol Hits:** `24` · **Gate G2:** `VERIFIED 🟢`

## 1. 🎯 Tóm Tắt & Nguyên Nhân Gốc Rễ (Root Cause & Context)
- **Phạm vi điều tra (`Read-Only S2`):** Đã quét trực tiếp `687` file mã nguồn trên `3` repositories (`cdc-cms-service, cdc-cms-web, centralized-data-service`) và phát hiện `24` điểm chạm thực tế (`file:line`).
- **Phân loại bằng chứng (`VERIFIED` vs `HYPOTHESIS`):** Toàn bộ điểm chạm của các Sub-Tasks dưới đây đều đã được đối chiếu trên đĩa (`VERIFIED`), loại bỏ mọi giả định suy diễn (`HYPOTHESIS = 0`).
- **Kịch bản Tái hiện `Gate G2 (Red -> Green)`:** Kiểm tra trạng thái trước khi sửa (`RED`) bằng `VerifySubTaskCodeOnDisk` và nghiệm thu sau khi sửa (`GREEN`) kèm biên lai `EV-...` (`exit=0`).

## 2. 💡 Kiến Trúc Giải Pháp & Bảng Điểm Chạm (`VERIFIED` vs `HYPOTHESIS`)
| Mã / ID | Phân loại | Repo | Tham chiếu (`REF: file:line`) | Chẩn đoán Điểm chạm & Mục tiêu Khắc phục |
| :--- | :---: | :--- | :--- | :--- |
| **`TASK-11-M01`** | `VERIFIED` | `cdc-cms-service` | `REF: cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9` | Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-service` (`list_latest_reports.go`) (`DoD:` go test -race ./...) |
| **`TASK-11-M02`** | `VERIFIED` | `cdc-cms-service` | `REF: cdc-cms-service/internal/api/dto/audit_dto.go:3` | Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-service` (`audit_dto.go`) (`DoD:` go test -race ./...) |
| **`TASK-11-M03`** | `VERIFIED` | `cdc-cms-web` | `REF: cdc-cms-web/src/constants/postgresDataTypes.ts:1` | Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `cdc-cms-web` (`postgresDataTypes.ts`) (`DoD:` npm run build && npm test) |
| **`TASK-11-M04`** | `VERIFIED` | `cdc-cms-web` | `REF: cdc-cms-web/src/hooks/useAuditStatus.ts:5` | Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `cdc-cms-web` (`useAuditStatus.ts`) (`DoD:` npm run build && npm test) |
| **`TASK-11-M05`** | `VERIFIED` | `centralized-data-service` | `REF: centralized-data-service/internal/service/source/metadata_registry_test_helpers.go:13` | Thiết kế & Triển khai tầng Model / Repository cốt lõi tại `centralized-data-service` (`metadata_registry_test_helpers.go`) (`DoD:` go test -race ./...) |
| **`TASK-11-M06`** | `VERIFIED` | `centralized-data-service` | `REF: centralized-data-service/internal/model/recon/reconciliation_report.go:8` | Tích hợp tầng Service / Handler & Kiểm thử nghiệm thu DoD tại `centralized-data-service` (`reconciliation_report.go`) (`DoD:` go test -race ./...) |
| `SYM-01` | `VERIFIED` | `cdc-cms-service/internal/app/queries/recon` | `REF: cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9` | Symbol thực tế: `type ListLatestReportsQuery struct{}` |
| `SYM-02` | `VERIFIED` | `cdc-cms-service/internal/app/queries/recon` | `REF: cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:11` | Symbol thực tế: `func (q ListLatestReportsQuery) Type() string { return "recon.list_latest" }` |
| `SYM-03` | `VERIFIED` | `cdc-cms-service/internal/api/dto` | `REF: cdc-cms-service/internal/api/dto/audit_dto.go:3` | Symbol thực tế: `type QASummaryResponse struct {` |
| `SYM-04` | `VERIFIED` | `cdc-cms-service/internal/api/dto` | `REF: cdc-cms-service/internal/api/dto/audit_dto.go:12` | Symbol thực tế: `type GapCount struct {` |
| `SYM-05` | `VERIFIED` | `cdc-cms-service/internal/api/governance` | `REF: cdc-cms-service/internal/api/governance/audit_handler.go:15` | Symbol thực tế: `type QASummaryHandler interface {` |
| `SYM-06` | `VERIFIED` | `cdc-cms-service/internal/api/governance` | `REF: cdc-cms-service/internal/api/governance/audit_handler.go:19` | Symbol thực tế: `type GapsHandler interface {` |

## 3. 💻 Code Demo / Trích Đoạn Mã Nguồn Thực Tế Trên Đĩa (`VERIFIED Snippets`)
```go
// [1] cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:9 — type ListLatestReportsQuery struct{}
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

// [2] cdc-cms-service/internal/app/queries/recon/list_latest_reports.go:11 — func (q ListLatestReportsQuery) Type() string { return "recon.list_latest" }
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

// [3] cdc-cms-service/internal/api/dto/audit_dto.go:3 — type QASummaryResponse struct {
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

// [4] cdc-cms-service/internal/api/dto/audit_dto.go:12 — type GapCount struct {
type GapCount struct {
    Open       int `json:"open"`
    InProgress int `json:"in_progress"`
    Closed     int `json:"closed"`
    Wontfix    int `json:"wontfix"`
}

type CriterionRow struct {
    ID       string `json:"id"`
    Group    string `json:"group"`
    Title    string `json:"title"`
    Rating   string `json:"rating"`
```

## 4. 🛡️ Kiểm Định Cổng `/agent` & Bước Tiếp Theo (`Gates & Action`)
- **Xác nhận Lesson Tripwires (`H4t`):**
- Không có tripwire rủi ro cao bị kích hoạt (`tripwires.json` = CLEAN).
- **Đối chiếu Bài học Lịch sử (`350` indexed lessons):**
- **`L024` [2026-10-27 · `T2:L166`] Giả lập bước `S4 Implement` và `S5 Local QA & Audit` kiểu "làm trong suy nghĩ" (`server.go:1046-1180`): bấm chạy `APPROVE & Chạy S4–S5` nhưng không thực thi bất kỳ Sub-Task (`TASK-M01..M10`) hay ghi dòng code nào vào các repo đích (`cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`), tự fallback `workDir = s.Root` để chạy `go vet` trên `ai-orchestrator`, ghi 9 dòng `audit_report.md` giả và báo `DONE` khi `git diff` bằng 0 (Hollow S4 Implement & S5 QA/Audit Theatre Reporting Pass/Done With Zero Code Changes, Empty Worktree Fallback & Unexecuted Multi-Repo Sub-Tasks):** (1) Dừng lại ngay lập tức (Rule #5), ghi bài học vào `lessons.md` và chạy `build_lessons_index.py`; (2) Thực hiện `S1–S3 Re-Plan (Task 11)` trong workspace `feature-modular-agent-o...
- **`L028` [2026-10-26 · `T2:L194`] Gener…:** Khi tiếp nối phiên làm việc sau khi ngữ cảnh bị nén (`Context Compaction`) để thực hiện bước đóng phiên và Audit phản biện (`S5 -> S6`) cho tác vụ sửa lỗi đồng bộ dữ liệu liên tầng...
- **`L007` [2026-10-27 · `T2:L46`] -> Hậu quả [X]: Vi phạm kỷ luật Single-Responsibility & `NOW <= HEAD`, tạo rác trùng lặp khó bảo trì, để lọt nội dung mâu thuẫn (`Knex/PG Pool` & đè V1) ngay trong file vật lý `.orchestrator/tasks/TASK-13/09_tasks_solution.md`, và báo cáo hoàn thành khi tiến trình `:8090` vẫn chạy binary cũ [Y]. **Đúng:** (1) Mọi thay đổi bắt buộc đặt đúng layer và tách thành các module đơn trách nhiệm gọn nhẹ (`repo_tech_profile.go` chuyên trách normalize path & nhận diện Tech Stack/DoD/Fence dùng chung toàn package `webui`, `task_revise_workflow.go` chuyên trách luồng `/revise` & Clean-Room Spec, `subtask_git_apply.go <= HEAD 312 dòng`); (2) Xóa bỏ 100% câu chữ mẫu nhắc tới `Knex` hoặc sửa đè `trans-his.model.ts` trong `BuildUnifiedS3SolutionDoc`, normalize relative path ở mọi cột bảng Sub-Tasks, và chuẩn hóa `TestDoD` chính xác tuyệt đối (`npm run build && npm test` cho `package.json`, `go test -race ./...` cho `go.mod`) kèm test assertion nghiêm ngặt; (3) Kiểm chứng trực tiếp trên repo thật (`/Users/trainguyen/Documents/work/data-hub/centralized-export-service`), hồ sơ `.orchestrator/tasks/TASK-13` thật trên đĩa và trace giá trị JSON runtime thật trên tiến trình `:8090` đã nạp binary mới.:** (1) Dừng ngay theo Rule #5, ghi bài học vào `lessons.md`, kiểm tra `wc -l` (`2536 -> 2543`) và chạy `build_lessons_index.py`; (2) Thực hiện vòng lặp tự chữa lành `S5 -> S3 -> S4 ->...
- **`L051` [2026-10-23 · `T1:L355`] -> Probe -> Compose` [A]) tích hợp vào hệ thống quản trị `/agent` [B], Agent mắc 5 lỗi logic tinh vi bị che khuất bởi bộ unit test chỉ thêm file mới trong RAM [X]: (1) **Hợp nhất 2 cây cấu hình (`t_inv` và `t_dom`) bằng `merged.update(t_inv); merged.update(t_dom)` (`harness2_engine.py:642-644, 675-677`):** Vì cả `t_inv` lẫn `t_dom` đều là bản sao đầy đủ của `base_tree` cộng thêm phần sửa riêng của mỗi nhánh, việc gọi `merged.update(t_dom)` sau `merged.update(t_inv)` khiến toàn bộ các file có sẵn trong `base_tree` mà `t_inv` vừa sửa bị giá trị gốc chưa sửa trong `t_dom` ghi đè mất sạch 100%, đồng thời các file bị xóa ở `t_inv`/`t_dom` lại bị khôi phục từ `base_tree`; (2) **Thiếu `task_id` và nội dung `base_tree` trong khóa băm `_run_cached_session` (`harness2_engine.py:559-563`):** Gọi `call_fingerprint_sha256(label=label, mission=f"propose:{dim}:{label}", system=self.surface.name)` mà không băm kèm `task_id` và nội dung `base_tree`/`context_dir`, dẫn tới 2 task khác nhau dùng chung `cache_dir` sinh ra cùng mã băm cho `inv1`/`dom1` (Cross-Task Cache Poisoning); (3) **Lệch khóa `rollout_outputs["base"]` và dùng điểm số `score` trực tiếp trong hàm tuyển chọn ẩn danh (`harness2_engine.py:614, 653, 700-706`):** Lưu rollout cơ sở dưới khóa `"base1..basek"` / `"b1..bk"` nhưng khi chọn ứng viên lại tra cứu `rollout_outputs.get("base")` (luôn trả về `None` / `0.0`), khiến ứng viên `"base"` (`KEEP_BASE`) bị loại oan và làm vô hiệu hóa ý nghĩa chống thiên kiến vị trí của `id_map`; (4) **Đếm độ dài `always_on` bằng số ký tự `len(body)` thay vì số byte UTF-8 `len(body.encode("utf-8"))` (`harness2_engine.py:116`) và tự sinh mảng `s3_docs = [f"{pfx}_ok.md" ...]` truyền vào `can_commit_adopted_harness_to_domain_library` (`harness2_engine.py:711-716`):** Văn bản tiếng Việt dài `18,091` ký tự nhưng thực tế chiếm `20,483` bytes UTF-8, và việc tự tạo mảng `s3_docs` giả làm điều kiện `has_all_s3` luôn đúng một cách hình thức (`L044`); (5) **Bỏ sót việc tách file `tooling/orchestrator/proposer_axes.py` và gọi đồng bộ `build_lessons_index.py` đã ghi trong `08_tasks` mà không cập nhật `04_decisions`** -> Hậu quả [Y]: Code pass unit test và 3-Mutant Gate (do test chỉ kiểm tra trường hợp thêm file mới trên 1 task đơn lẻ) nhưng khi chạy thực tế với nhiều task hoặc khi nhánh `INV` sửa file có sẵn sẽ bị mất thay đổi và nhiễm bẩn cache. **Đúng:** (1) Khi hợp nhất 2 nhánh cấu hình trực giao `t_inv` và `t_dom` từ `base_tree`, BẮT BUỘC thực hiện **3-Way Diff Merge (`three_way_merge_trees(base, t_inv, t_dom)`)**: chỉ áp dụng các key có `t_branch[k] != base[k]` và xóa các key bị xóa ở `t_branch`; (2) Khóa băm SHA-256 của session cache BẮT BUỘC phải băm toàn bộ `(task_id, label, dim, system, base_tree_digest, context_dir)`; (3) Gắn kết quả rollout cơ sở vào khóa `"base"` (hoặc truyền hàm `selector_fn(anonymous_pool)` không phụ thuộc điểm số `score` ẩn); (4) Đo hạn ngạch `always_on_max` theo byte UTF-8 (`len(body.encode("utf-8"))`), tách YAML frontmatter theo dấu `---`, và kiểm tra `s3_docs_present` từ thư mục `workspace_dir` thật trên đĩa; (5) Giữ nghiêm kỷ luật `S5 -> S3 Re-Plan` (Propose-Only): ghi toàn bộ phát hiện và giải pháp vào `audit_report_*.md` + `09_tasks_solution_*.md`, chạy `/laya_router` và chờ User `APPROVE` trước khi sửa code.:** (1) Ghi bài học vào `lessons.md` và chạy `build_lessons_index.py`; (2) Chạy `/laya_router` (`DG3`) kiểm định toàn bộ 7 phát hiện phản biện (`AF-01` $\rightarrow$ `AF-07`); (3) Lưu ...
- **`L012` [2026-10-27 · `T1:L81`] WebUI…:** Khi thực thi vòng lặp tự chữa lành (`Self-Healing Loop S5 -> S3 -> S4 -> S5` theo `GEMINI.md` Rule #9.8d) trên `ai-orchestrator` [A], nếu hàm `runTaskLifecycleActionWithDelay` (`in...