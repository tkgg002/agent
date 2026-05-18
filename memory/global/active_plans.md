# Active Plans Registry

> **Maintained by**: Brain (Antigravity)
> **Last Updated**: 2026-03-16
> **Purpose**: Registry để Brain biết workspace nào đang active → load đúng context khi bắt đầu phiên mới. KHÔNG phải cơ chế agent communication.

| Workspace | Project | Status | Last Active |
|-----------|---------|--------|-------------|
| upgrade-core-system | Upgrade Core Brain/Muscle System | ✅ Done | 2026-02-25 |
| feature-refactor-2026 | GooPay Core Refactor 2026 | ✅ Done — sẵn sàng tiếp tục | 2026-02-25 |
| optimize-brain-muscle-models | Tối ưu hóa model cho Brain/Muscle | ✅ Done (V2 Quota & Multi-Muscle) | 2026-02-25 |
| compare-disbursement-export | So sánh logic DisbursementTicketExport | ⏸ Paused | 2026-02-25 |
| compare-disbursement-trans-his-export | So sánh logic DisbursementTransHisExport | ✅ Done | 2026-02-27 |
| feature-merchant-export-activation-info | Bổ sung thông tin kích hoạt Merchant Export | ✅ Done | 2026-03-12 |
| feature-id-expired-notification-log-export | Tạo IDExpiredNotificationLogExport type | ✅ Done | 2026-02-27 |
| feature-fee-configuration | Cấu hình phí dịch vụ (Fee Configuration) | ⏸ Paused | 2026-03-03 |
| feature-cdc-integration | CDC Integration (Debezium-only sau commit 8ef7d71 remove airbyte) — Phase F (F1+F3) Done 2026-05-04 | ✅ Done | 2026-05-04 |
| feature-system-refactor-2026-05 | System Refactor 2026-05 — bucket B1+B2 (hygiene + tooling), 4 service local smoke | 🟡 Active | 2026-05-04 |
| feature-export-driver-search | Driver Info & Approximate Search in Exports | ✅ Done | 2026-03-24 |
| upgrade-agent-infrastructure | Nâng cấp hạ tầng Agent v1.10.0 (Brain/Muscle) | ✅ Done | 2026-04-06 |
| feature-trans-his-collection-export | Export TransHis Collection | 🟡 Active | 2026-04-09 |
| feature-multi-pg-isolation-e2e | Tách 4 PG containers (auth/cdc/dest/source) + E2E auto-pipeline | 🟡 Active | 2026-04-28 |
| feature-cdc-system-refactor | Task #19 service-tier drainage — Đợt G/H/I closed by max → Đợt J handed off to x2 (cms-lane locked) | 🟡 Active | 2026-05-07 |


---

## Notes
- **Active** 🟡: Đang làm trong phiên hiện tại hoặc phiên gần nhất
- **Paused** ⏸: Tạm dừng, sẽ tiếp tục sau
- **Done** ✅: Hoàn thành, archived
- Khi bắt đầu phiên mới: Brain đọc bảng này → load workspace có status Active đầu tiên
|-----------|---------|--------|-------------|

## 2026-04-27 Updates

- `feature-cms-fe-overhaul`: đã hoàn thành Phase 2 FE Navigation Refactor trong `cdc-cms-web`; bước kế tiếp là refactor data model/API usage của các page sang semantics V2.
- `feature-cms-fe-overhaul`: đã hoàn thành Phase 8 `worker-schedule` contract refactor; `cms-fe` hiện được chốt theo mô hình 2 luồng, trong đó operator-flow (monitoring / backup / retry / reconcile) phải được giữ lại nhưng làm gọn API surface.

## 2026-05-07 Updates

- `feature-cdc-system-refactor`: Task #19 service-tier drainage — Đợt G+H+I closed (cms `b4a3461`, agent `c141012`). Cluster A (alert_manager + approval_service + provisioning_orchestrator + state_machine + master_swap + shadow_automator + source_object_v2_sync + registry_repo + mapping_rule_repo) → `infra/persistence/`. Đợt J (cluster A* system_health_* + cluster C probes/) → `infra/observability/{,probes/}` đã plan + tasks (agent `dd21443`), x2 thi công.
- Lane lock effective from `b4a3461`: max owns worker (`centralized-data-service/`) + workspace docs; x2 owns cms (`cdc-cms-service/`).
- Pending Task #19 closure: x2 lands Đợt J commit + APPEND `05_progress.md` → Task #19 marked CLOSED.
- Worker-lane sub-issues queued for max post-Đợt-J: P3 prune residue 1 row legacy_1, duplicate close-loop log dedup, Track E (Mongo CDC) plan kickoff.

## 2026-05-07 09:48 ICT Updates (post-Đợt-J)

- `feature-cdc-system-refactor`: **Task #19 CLOSED** ✅ — Đợt J landed by x2 (cms `b453d36`, agent `57d1b2a`). 10 đợt A→J done; `internal/service/` removed entirely; cms hexagonal-aligned (`app/{commands,queries,ports}` + `domain/` + `infra/{cache,http,messaging,persistence,observability}` + `api/` + `server/` + `middleware/` + `router/` + `model/`). Q3 (rebuild + restart cms-server) ALSO done by x2: PID 52079 `/tmp/cdc-cms-service-postJ` post-`b453d36` binary, smoke test 3 endpoint PASS (200 trên `/health`, `/api/system/health`, `/api/v1/source-objects/registry/1/dispatch-status`).
- Workspace `feature-cdc-system-refactor`: status row trong table có thể đọc là 🟡 Active (audit trail) — actual phase: ⏸ Paused (Task #19 đóng, chờ next directive).
- Next priorities cho max-Brain (post-Task-#19): plan Track E (Mongo CDC, blocked on Boss brief — lesson L-1436 không bịa scope), P3 prune residue 1 row legacy_1 investigation (worker-lane), duplicate close-loop log dedup investigation (worker-lane).
- max output 2026-05-07 ICT: `report_flow1_connect_source_2026-05-07.md` (overview Wizard step 1–5 manual operator flow per Boss directive).

## 2026-05-07 16:30+ ICT Updates (Flow 1 push iter#46–#72)

- **Workspace re-active 🟡**: `feature-cdc-system-refactor` từ Paused → Active vì Boss directive "**bằng mọi giá phải lên đc flow1 này**" (multi-iter /loop session).
- **Path A → Path B migration LIVE**: cms binary `/tmp/cdc-cms-service-flow1` PID 43919 (mtime 11:21, swap 13:54) chạy commit `0eddad0 feat(cms): support hybrid shadow db configuration (A3)`. Log evidence: `PostgreSQL (shadow data plane) connected port=5436 cdc_shadow`. Path B (5436 cdc_shadow) có 7 schemas + 11 shadow tables LIVE.
- **G-11 master/shadow hyphen**: ✅ CLOSED (iter#68 evidence). master_binding id=37 + shadow_binding id=62 cho src 44 (`src_mongodb_payment_bill_service_refund_requests`) đều có underscore. SQL backfill skip.
- **G-12 (NEW) worker A3 hybrid**: ⏳ OPEN — `/tmp/cdc-worker-host` PID 90006 vẫn chạy binary May 5 09:39 (pre-A3-hybrid). Working tree đã có shadowDB *gorm.DB* + adapter swap nhưng UNCOMMITTED + UNBUILT. Worker query Path A `cdc_dw` thay vì Path B → ERROR `column "_id" does not exist` + `relation does not exist`. Cần verb `commit a3-worker` → `ship g11`.
- **G-13 (NEW) Mongo PK cast bigint**: ⏳ OPEN (defer post-Flow-1). Transmuter hardcoded `"_id"::bigint` không cast được Mongo ObjectId. Recommend: dùng PG source cho Flow 1 P1 happy-path để né.
- **Scope divergence x2**: x2 (Antigravity:gemini-1.5-pro) đã pivot Phase 2 P3 (CQRS refactor + FE polling) — `08_tasks_phase2_p3.md` + `09_tasks_solution_phase2_p3.md` + master_registry_handler thin-adapter. Phase 2 P3 KHÔNG unblock Flow 1; recommend Boss verb `defer phase2, focus flow1`.
- **Brain output post-iter#46**: `report_flow1_loop_iter46_2026-05-07.md` + `report_flow1_loop_iter68_2026-05-07.md` + (đang) `report_flow1_loop_iter72_decision_pack.md`. 05_progress.md đã APPEND iter#46 + iter#47 + iter#68 (215487 bytes / 2700+ lines).
- **Halt status**: Brain đã hoàn thành scope plan + document + coordinate. Block còn lại đều cần Boss explicit verb (CLAUDE.md §1, §11, §12 + Auto Mode rule #5). Verb dictionary: `commit a3-worker` | `ship g11` | `smoke flow1 pg` | `defer phase2, focus flow1` | (or) `max switch muscle` để Brain cross-line execute.

## 2026-05-08 Audit Update

- `audit-flow1-3repos-2026-05-08`: workspace audit mới để re-check `cdc-cms-service`, `cdc-cms-web`, `centralized-data-service`, và `flow1` trên CMS.
- Kết quả chính:
  - `cdc-cms-service` test PASS.
  - `cdc-cms-web` build FAIL ở `src/pages/flow1/*`.
  - `centralized-data-service` binary build PASS nhưng test FAIL (`scratch/`, `internal/handler`, `internal/service`).
  - `flow1` NOT READY do FE build lỗi + contract drift FE↔CMS + runtime drift dấu hiệu stale CMS binary.
