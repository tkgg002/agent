# 10 — Gap Analysis (Brain findings, sự thật trên disk)

**Date**: 2026-05-04 17:05+07
**Method**: Scan thực tế disk + docker ps + lsof + git diff + go build + tsc.
**Note**: Đây KHÔNG phải plan. Plan sẽ ở `02_plan.md` SAU khi user chốt scope.

---

## A — Code drift / hygiene

| # | Gap | Evidence | Severity |
|---|-----|----------|----------|
| A1 | 5 file Go uncommitted (admin-api hardening + helpers.go fix Phase F1/F3) chưa commit | `git status --short` | LOW (sạch về mặt logic, chỉ cần commit) |
| A2 | Chỉ có 6 commit total trên `main`. 4 dịch vụ chia sẻ 1 git history → workflow theo monorepo, nhưng commit message yếu (`update`, `fixing fe`, `init`) | `git log --oneline` | MED (audit trail nghèo, blame khó) |
| A3 | `architecture.md` mô tả Airbyte như hybrid path nhưng commit `8ef7d71 remove airbyte` đã gỡ — doc drift | `architecture.md` line 64 vs git log | MED |
| A4 | Memory `active_plans.md` ghi `feature-cdc-integration` còn "Hybrid Debezium + Airbyte" — out-of-date | `active_plans.md:17` | LOW |
| A5 | `feature-cdc-system-refactor/07_status.md` báo "Implemented and validated locally" nhưng `10_gap_analysis.md` flag DLQ wiring pending → status không khớp gap | 2 file workspace cũ | MED |
| A6 | Nhiều workspace song song chưa close: `feature-cdc-integration` (12+ phase), `feature-multi-pg-isolation-e2e` (P2/P3/P4 plan chưa execute) | `agent/memory/workspaces/` | MED (context fragmentation) |

## B — Test coverage gap

| # | Gap | Evidence | Severity |
|---|-----|----------|----------|
| B1 | `cdc-auth-service`: 0 test trên 9 .go file | `find ./cdc-auth-service -name "*_test.go" \| wc -l = 0` | HIGH (auth = security boundary) |
| B2 | `cdc-cms-service`: 10 test / 76 .go = ~13% coverage by file count | đếm file | MED |
| B3 | `centralized-data-service`: 39 test / 144 .go = ~27% by file count, NHƯNG Phase F1 đã thêm 4 test mới (auth/rate-limit/sanitize/body-limit) | đếm file | MED |
| B4 | `cdc-cms-web`: chỉ có `tsc --noEmit`, KHÔNG có unit / e2e test | `package.json` không có script `test` | MED |

## C — Service runtime gap

| # | Gap | Evidence | Severity |
|---|-----|----------|----------|
| C1 | `cdc-auth-service` không thấy chạy local; user mandate "service work các chức năng" → phải start | `lsof / ps` không match | HIGH nếu CMS-Web cần auth |
| C2 | `cdc-cms-web` không chạy dev server | không có vite process | HIGH nếu user expect operator UI work |
| C3 | `cdc-cms-service` chạy local 5 ngày — log chưa rotate, có thể leak memory | `ps -p 13653 -o etime` | LOW |
| C4 | `gpay-cdc-worker` Docker chỉ 2h uptime — vẫn còn nóng từ phiên debug | `docker ps` | LOW |
| C5 | Phase F1 admin-api dev mode unsafe nếu deploy real (cần `ADMIN_API_TOKEN`) — đã có boot fail-fast nhưng config production chưa wired | `cmd/admin-api/main.go:64-74` | MED |

## D — Architectural debts (đã ghi nhận trong workspace cũ, chưa fix)

| # | Gap | Evidence | Severity |
|---|-----|----------|----------|
| D1 | `SchemaAdapter.PrepareForCDCInsertInSchema` chỉ ALTER, fail nếu shadow table chưa tồn tại — operator phải `CREATE TABLE` thủ công | plan curried-waddling-spindle P2 | MED |
| D2 | 10 hàng V1 legacy seed trong `source_object_registry` (`object_code LIKE 'legacy_%'`) còn tồn tại với `is_active=false` nhưng vẫn nhiễu UNIQUE constraint khi V2 source_object_name trùng | plan curried-waddling-spindle P3 | MED |
| D3 | `TransmuteScheduler` set `last_status='running'` rồi fire NATS, từng treo `running` → ĐÃ CÓ JobMonitor close-loop ăn `cdc.evt.transmute.completed`, log live xác nhận hoạt động (xem `verify V2 bridge` mới đây). P4 thực ra đã làm. | log `/tmp/cdc-worker.log` | RESOLVED |
| D4 | Issue 6/7/8 từ Phase E5 report (SourceLocator typed struct, CORS, audit trail actor/IP/UA) chưa làm — defer Phase F2 | `report_phase_e5...` | LOW |
| D5 | Master cascade auto-trigger từ admin-api: admin-api hiện chỉ tạo source_object_registry + shadow_binding, không cascade master_binding | workspace feature-cdc-integration note | MED |
| D6 | OTel collector spam `failed to upload metrics: lookup otel-collector` trong worker log do DNS local resolve sai | `/tmp/cdc-worker.log` | LOW (cosmetic) |
| D7 | `helpers.go` đã có 3 vị trí cùng pattern bug "Mongo collection fallback" đã fix — cần lesson ghi nhận `L-input-fallback-pattern` để abstract Global Pattern (CLAUDE.md §13). Hiện CHƯA append vào lessons.md | session vừa rồi | MED |

## E — Documentation / governance gaps

| # | Gap | Evidence | Severity |
|---|-----|----------|----------|
| E1 | `agent/memory/global/project_context.md` còn template, chưa fill placeholders | đọc file | MED |
| E2 | `agent/memory/global/tech_stack.md` còn template, chưa fill | đọc file | MED |
| E3 | Không có `CHANGELOG.md` cho cdc-system | ls root | LOW |
| E4 | `cdc-system/README.md` chỉ có 1 dòng `# cdc-system` | đọc | MED (onboarding kém) |
| E5 | `cdc-system/security-audit-report.md` + `security-regression-matrix.md` chưa rà mới sau Phase F1 | ls root | LOW |

---

## Tóm tắt severity buckets

- **HIGH**: B1 (auth 0 test), C1 (auth không chạy), C2 (FE không chạy)
- **MED**: A2/A3/A5/A6, B2/B3/B4, C5, D1/D2/D5/D7, E1/E2/E4
- **LOW / RESOLVED**: A1/A4, C3/C4, D3/D4/D6, E3/E5

## Đề xuất bucket cho user chọn (ở `02_plan.md` sau)

- **B1 — Quick Wins** (1 buổi): commit 5 file pending, fix doc drift architecture.md, fill template global memory, append lesson L-input-fallback.
- **B2 — Tooling & Hygiene** (1-2 ngày): start cdc-auth-service + cdc-cms-web local cho operator path, smoke test E2E, viết script start-all.
- **B3 — Architectural Debts** (3-5 ngày): D1 (auto-create shadow), D2 (prune legacy), D5 (master cascade), append lesson D7.
- **B4 — Testing Reinforcement** (3-5 ngày): B1 (auth tests), B4 (FE smoke), property tests cho mapping.
- **B5 — Big Refactor** (open-ended, RỦI RO): chỉ làm khi user chốt rõ — review module boundary, tách `centralized-data-service` thành nhiều binary nhỏ, thay viper config bằng đơn giản hơn, etc.

User cần chọn **bucket nào trước** để Brain plan chi tiết.
