# Report — Initial Scan System Refactor 2026-05

**Date**: 2026-05-04 17:05+07
**Author**: Brain (Antigravity)
**Workspace**: `agent/memory/workspaces/feature-system-refactor-2026-05/`
**Verdict**: ⏳ **Discovery xong, đợi User chốt scope trước khi Brain plan chi tiết và delegate Muscle.**

> User mandate: "hệ thống thành nồi cám heo, refactor toàn hệ thống, giúp service chạy được các chức năng. Đọc lessons trước, làm theo `agent/`, dùng skill từ `CLAUDE.md`, report dựa kết quả thực tế, kiểm tra service work mới báo done. Luôn có file `report_*.md`."

---

## 1. Skills + Governance đã làm theo CLAUDE.md

- §0 Plan trước, chi tiết — đang ở giai đoạn Discovery, chưa code.
- §7 Full Doc Set — workspace mới có `00_context.md`, `05_progress.md`, `10_gap_analysis.md`, file report này.
- §11 APPEND-only — `05_progress.md` chỉ APPEND.
- §12 Brain code prohibition — Brain CHƯA edit `.go/.ts/.sql` nào.
- §14 Pre-flight — đang tự quét rule trước khi kết thúc câu trả lời.
- Đã đọc lessons (TOC + recent), active_plans, project_context, tech_stack.

## 2. Skills (tools) thực tế đã dùng phiên này

- `Read` (lessons, active_plans, project_context, tech_stack, architecture.md, workspace cũ).
- `Bash`: `git status / log / diff`, `ls`, `find`, `wc -l`, `lsof -nPi`, `docker ps`, `ps -ef`, `go build ./...`, `npx tsc --noEmit`.
- `TaskCreate / TaskUpdate / TaskList`.
- `Write` (00_context, 05_progress, 10_gap_analysis, report này).
- `ScheduleWakeup` (đã set 17:19 cho /loop verify V2 bridge — không liên quan refactor task).

## 3. Sự thật trên disk (không bịa)

### 3.1 Layout monorepo (1 git, branch `main`, 6 commit)

| Service | LOC / file | Test | Live runtime | Build |
|---|---|---|---|---|
| `cdc-auth-service` | 9 .go | 0 | ❌ KHÔNG chạy | ✅ exit 0 |
| `cdc-cms-service` | 76 .go | 10 | ✅ `/tmp/cms-server` PID 13653 (5d) | ✅ exit 0 |
| `centralized-data-service` | 144 .go (4 binary) | 39 | ✅ docker `gpay-cdc-worker` (2h) + `/tmp/cdc-admin-api-f3v2` | ✅ exit 0 |
| `cdc-cms-web` | 22 .tsx, 7634 LOC | tsc clean | ❌ KHÔNG chạy dev server | ✅ tsc --noEmit exit 0 |

### 3.2 Live infra (docker ps)

13 container healthy: 4 PG (5432/5433/5434/5435), Mongo (17017), MariaDB (13307), Redis (16379), Kafka (19092/19093), Kafka-Connect (18083), Schema-Registry (18081), NATS (14222), OTel (14317/14318), cdc-worker (8080).

### 3.3 Uncommitted

5 file Go = 286 line (Phase F1 admin-api hardening + F3 helpers.go fix). Đã verify build PASS, 21 test PASS từ phiên trước.

## 4. Gaps tìm thấy (chi tiết ở `10_gap_analysis.md`)

### HIGH (3 gap)

- **B1**: `cdc-auth-service` 0 test — auth là biên bảo mật, đáng lo.
- **C1**: `cdc-auth-service` không chạy local — operator path qua web sẽ đứt ở bước login nếu user expect "FE work".
- **C2**: `cdc-cms-web` không chạy dev server — UI vận hành không truy cập được local.

### MED (10 gap)

A2 commit message yếu, A3 doc drift Airbyte, A5 status workspace cũ không khớp gap, A6 nhiều workspace song song chưa close, B2/B3/B4 test thưa, C5 admin-api production token chưa wired, D1 shadow auto-create, D2 V1 legacy prune, D5 master cascade từ admin-api, D7 chưa append lesson `L-input-fallback`, E1/E2/E4 tài liệu global template chưa fill / README 1 dòng.

### LOW / RESOLVED (8 gap)

A1 commit pending (lành), A4 active_plans drift Airbyte (cosmetic), C3/C4 service uptime, **D3 transmute close-loop ĐÃ HOẠT ĐỘNG** (verify log cron tick 09:51:13 UTC `success`, JobMonitor closed schedule 1/2/3/13/14/15), D4 issue 6/7/8 LOW E5, D6 OTel DNS spam, E3 không CHANGELOG, E5 security audit chưa rà sau F1.

## 5. Drift "nồi cám heo" — Brain hiểu user kêu gì

- Code thực ra đang **build OK + service đang chạy**. Pipeline V2 (Mongo→Debezium→Kafka→Worker→Shadow→Master) verify lúc 09:51:13 UTC vừa rồi: 6/6 schedule `success`, JobMonitor close-loop healthy.
- "Cám heo" Brain đoán nghiêng về:
  1. **Tài liệu drift** (architecture nói Airbyte còn, code đã gỡ; status nói "Done" nhưng gap còn flag pending).
  2. **Workspace fragmentation** — 26 workspace, nhiều cái Active không close.
  3. **2 service KHÔNG chạy** (auth + web) → khi user thử click vào CMS UI là chết → cảm giác "không work".
  4. **Test mỏng cho auth** + **0 unit test FE**.
  5. **Phase F1/F3/Track-D-P2/P3** chưa commit, code lăn lóc trong working tree.

→ Brain CẦN user xác nhận hiểu đúng (xem §7 câu hỏi).

## 6. Đề xuất 5 bucket (chi tiết ở `10_gap_analysis.md`)

| Bucket | Effort | Risk | Outcome |
|---|---|---|---|
| **B1 Quick Wins** | nửa ngày | thấp | Commit pending, fix doc drift, fill template, append lesson L-input-fallback |
| **B2 Tooling & Hygiene** | 1-2 ngày | thấp | Start auth + FE local, smoke E2E operator path, script start-all |
| **B3 Architectural Debts** | 3-5 ngày | trung | Auto-create shadow, prune V1 legacy, master cascade từ admin-api |
| **B4 Testing Reinforcement** | 3-5 ngày | thấp | Auth tests, FE smoke, property tests mapping |
| **B5 Big Refactor** | open-ended | CAO | Chỉ làm khi user chốt RÕ — module boundary, tách binary, replace viper, etc. |

## 7. 5 câu hỏi cần User trả lời để Brain plan chi tiết (TUYỆT ĐỐI KHÔNG đoán)

1. **Scope**: refactor 4 service đồng thời, hay 1 service trước? (Brain RECOMMEND ưu tiên `cdc-cms-web` + `cdc-auth-service` trước vì 2 service này không chạy → user không thấy operator UI).
2. **"Chức năng work"** = chức năng nào cụ thể? Có flow operator nào user mới test thấy đứt? (Login? Tạo source registry? Approve schema proposal? Master create? Schedule?)
3. **Goal 1 dòng**: clean drift / merge frag / rebuild / fix UX / consolidate config? (Brain RECOMMEND: B1 + B2 trước → minimal effort, max OPS reality, sau đó mở B3.)
4. **Risk tolerance**: cho phép kill service đang chạy để rebuild + restart không?
5. **Time budget**: 1 ngày spike hay open-ended?

## 8. Đề xuất next-action (chỉ thực thi sau user trả lời)

- Brain plan chi tiết `02_plan.md` + `08_tasks.md` + `09_tasks_solution.md` cho bucket user chọn.
- Brain đề xuất Muscle execute (CLAUDE.md §1, §12).
- Sau mỗi step: Muscle verify exercise-driven (lessons "Service listening ≠ healthy", "Báo Done mà không restart"), Brain audit + APPEND `05_progress.md` + viết `report_phase_*.md`.

## 9. File vật lý đã tạo (bằng chứng không bịa)

- `agent/memory/workspaces/feature-system-refactor-2026-05/00_context.md`
- `agent/memory/workspaces/feature-system-refactor-2026-05/05_progress.md`
- `agent/memory/workspaces/feature-system-refactor-2026-05/10_gap_analysis.md`
- `agent/memory/workspaces/feature-system-refactor-2026-05/report_initial_scan_20260504.md` (file này)

## 10. Pre-flight CLAUDE.md §14

- §1: ✓ Brain chỉ Chairman, chưa code.
- §7: ✓ Workspace + 4 file vật lý.
- §11: ✓ `05_progress.md` APPEND only.
- §12: ✓ Brain 0 source code edit.
- §13: Lesson `L-input-fallback-pattern` ĐÃ identify, sẽ append vào `lessons.md` SAU khi user chốt scope (không append vội nếu user re-prioritise).
- §14: ✓ đang quét rule trước khi kết thúc.
