# Report Audit Workspace `plan-cdc-qa-gap-fix-2026-05-27`

**Date**: 2026-05-27
**Auditor**: Muscle (CC CLI) — Read-only audit, KHÔNG sửa source code
**Scope**: Đối chiếu workspace plan vs (a) 5 nhóm requirement user, (b) thực tế code trong `data-hub/`, (c) governance §1..§14 GEMINI.md
**Base lessons đọc trước**: `agent/memory/global/lessons.md` (L985 silent-skip, L3100, L-CDC-circuit-breaker, L-metric-defined-but-never-set, L-trace, L-log-sampling, L-DRY-resolver).

---

## 1. Kết quả tổng quan

| Tiêu chí | Đánh giá | Evidence |
|---|---|---|
| Governance Full Doc Set (§7) | ✅ 21/21 file required + 4 report = **24 file vật lý** | `ls workspace/*.md` |
| §11 Memory APPEND-only | ✅ `05_progress.md` 10 Entry, không phát hiện sửa nội dung cũ | scroll file |
| §12 Brain Code Prohibition (giai đoạn plan) | ✅ Plan-only ở 4 phase MD, code demo nằm trong code block | review 03_implementation_*.md |
| Build/Vet sau khi Muscle execute | ✅ `go vet`, `go build` PASS trên 2 service Go; `tsc -b` PASS trên FE | run-time evidence (xem §5) |
| User 5 nhóm requirement coverage | ⚠️ **Cover ~75%**, còn 5 gap cần bổ sung (xem §3) | mapping chi tiết bên dưới |
| Implementation drift vs plan | ⚠️ **Có 3 drift quan trọng** (xem §4) | grep code |
| Audit Log Metadata Integrity (§7) | ❌ **VI PHẠM**: 22 dòng dạng `[Timestamp] [Muscle:CC]` chưa được điền timestamp thực | `grep -c "\[Timestamp\]" 05_progress.md` = 22 |

---

## 2. Mapping 5 nhóm user requirement → Gap plan

### Nhóm 1 — Functional & Correctness

| User requirement | Gap plan | Trạng thái | Note |
|---|---|---|---|
| 1.1 Data Reconciliation (Source ↔ Shadow, lossless, Insert/Update/Delete burst, NULL + special chars) | G-10 (Tier3 config refine) | ⚠️ **Partial** | G-10 chỉ làm configurable off-peak window. KHÔNG có acceptance test hash count + value match 100% sau burst. `1.1` ở `10_gap_analysis.md` ghi L4 baseline nhưng KHÔNG dựa trên test BURST + NULL/special chars như user yêu cầu. |
| 1.2 Schema Drift (ADD/DROP/ALTER TYPE column) | G-9 (drift E2E test) | ⚠️ **Partial** | `approve_schema_proposal_e2e_test.go` chỉ test ADD COLUMN (1 scenario). KHÔNG có test DROP COLUMN / ALTER TYPE. User yêu cầu cả 3 trường hợp. |
| 1.3 Event Ordering (Insert → U1 → U2 → Delete) | G-8 (ordering test) | ⚠️ **Partial** | Plan ghi 4 step có Delete ts=4000. File impl `schema_adapter_ordering_test.go:90` chỉ có 3 step Insert/Update/Update (thiếu Delete final). HashTiebreaker test bonus có. |

### Nhóm 2 — Stability & Resilience

| User requirement | Gap plan | Trạng thái | Note |
|---|---|---|---|
| 2.1 Failover & Self-Healing (kill Docker containers Debezium + Kafka + Worker, restart, không mất event, exactly-once) | G-5 (smoke_failover.sh) | ⚠️ **Partial** | Script chỉ `kill -9` **worker process** (`scripts/smoke_failover.sh:33`). KHÔNG kill Kafka broker container hoặc Debezium connector container như user yêu cầu. Coverage Kafka offset commit, không phải full container restart. |
| 2.2 Network Flicker 5-15 phút + backpressure + catch-up | G-15 (chaos_network.sh) | ⚠️ **Partial** | `chaos_network.sh:174` chỉ DROP iptables OUT từ worker → Mongo source. KHÔNG test chiều Worker → Shadow PG, Worker → Kafka. Acceptance criterion `AFTER_LAG < 2× BEFORE_LAG` chưa validate backpressure semantics. |
| 2.3 LSN/WAL Expire + auto-Snapshot alert | G-6 (WAL slot alert + runbook) | ✅ **Đủ** | `alerts/wal_slot.yml` 2 rule + `docs/runbooks/wal-slot-expire.md`. |
| 2.4 DLQ (lỗi format → DLQ tiếp tục) | G-4 (DLQ Circuit Breaker) | ✅ **Đủ** | `dlq_circuit_breaker.go` thực thi + integration trong `kafka_consumer.go:387-413`. Plan + runbook `pipeline-pause-resume.md` đủ. |

### Nhóm 3 — Performance & Scalability

| User requirement | Gap plan | Trạng thái | Note |
|---|---|---|---|
| 3.1 Data Lag (commit → shadow) p99 dưới tải peak | G-1 ConsumerLag set + G-16 k6 | ✅ **Đủ** | `kafka_consumer.go:408` Set gauge mỗi 15s. k6 threshold `p(99) < 5000ms`. |
| 3.2 Throughput EPS (CDC engine max EPS) | G-11 + G-16 | ⚠️ **Partial** | `BatchesFlushed.Inc()` (NewCounter — KHÔNG có label `shadow_db, table, status` như plan claim). Script k6 đo e2e latency, **KHÔNG report EPS CDC engine** như user yêu cầu. |
| 3.3 Backlog Catch-up Speed (off CDC 2-3h, đo time-to-zero) | G-12 (adaptive batch burst) | ⚠️ **Partial** | `adaptiveBatcher` (kafka_consumer.go:120-153) tăng batchSize 2x. KHÔNG có script đo time-to-zero sau khi off CDC 2-3h như user yêu cầu (cần measurement plan, không chỉ optimization). |
| 3.4 Source DB Overhead (CDC max load → đo CPU/IOPS Source +%) | G-3 (mongodb-exporter + postgres-exporter scrape) | ❌ **MISSING acceptance** | Có exporter scrape (`prometheus.yml:50`). KHÔNG có acceptance criterion "CDC max load → Source CPU increase < X%" và KHÔNG có baseline test business query latency. Đây là gap user requirement quan trọng chưa cover. |

### Nhóm 4 — Resource Utilization

| User requirement | Gap plan | Trạng thái | Note |
|---|---|---|---|
| 4.1 Memory Leak (Soak 48-72h) | G-7 (pprof + goleak) | ⚠️ **Partial** | `cmd/worker/main.go:7` import pprof. 3 file `main_test.go` có `goleak.VerifyTestMain`. **KHÔNG có 48-72h soak test plan / script**. Plan ghi nhận trong `10_gap_analysis.md` "L4 cần long-soak 7 ngày, hiện 1h" → user requirement L4 chưa cover. |
| 4.2 Concurrency / Throttling (60 sources → 1 đích, lock/balance) | G-13 (per-source pool semaphore) | ❌ **DEAD CODE** | `pkgs/database/per_source_pool.go` define `NewPerSourcePool` + `Acquire`. **`grep -rn "NewPerSourcePool(" .` chỉ trả về định nghĩa, KHÔNG có caller**. Tương tự lesson `metric-defined-but-never-set`: code có mặt nhưng không được wire vào DI/handler → behaviour không thay đổi. |

### Nhóm 5 — Metric Monitor (KPIs)

| User requirement | Gap plan | Trạng thái | Note |
|---|---|---|---|
| 5.1 Replication Lag (s) dashboard | G-1 + G-3 | ✅ **Đủ** | `cdc_kafka_consumer_lag` gauge + alert `HighConsumerLagWorker > 10000 for 5m`. |
| 5.2 Source/Destination CPU & Memory | G-3 (scrape) | ⚠️ **Partial** | Scrape có postgres-exporter + mongodb-exporter. **KHÔNG có node_exporter / cAdvisor** để đo CPU/Mem của worker host hoặc DB host (mongo/pg-exporter chỉ trả số liệu nội bộ DB, không phải host CPU%). |
| 5.3 Disk I/O & Network Throughput | G-3 (scrape) | ⚠️ **Partial** | Tương tự 5.2: thiếu node_exporter cho disk_io_time, network bytes. |
| 5.4 OpenTelemetry (trace + log) | G-2 (otel exporter) | ✅ **Đủ** | `otel-collector-config.yml` exporter `otlp/signoz` + `prometheusremotewrite`. |
| User bonus: VGA/GPU (nếu có LLM/Agent local) | — | N/A | Hệ thống CDC hiện không dùng GPU/LLM, có thể skip. |

**Tổng coverage 5 nhóm**: 6/14 ✅, 7/14 ⚠️ partial, 1/14 ❌ dead code, 0/14 missing hoàn toàn.

---

## 3. Gap còn thiếu so với user requirement (RECOMMEND bổ sung)

| ID đề xuất | Mô tả | Priority | Effort ước tính |
|---|---|---|---|
| **G-NEW-19** | Reconciliation E2E test BURST (10k Insert/Update/Delete + NULL + special chars + long strings, verify hash + count match) | P1 | 4h |
| **G-NEW-20** | Schema Drift test mở rộng: DROP COLUMN + ALTER TYPE scenarios | P1 | 3h |
| **G-NEW-21** | Ordering test bổ sung Delete-after-Update + Delete-before-Insert (out-of-order) | P1 | 1h |
| **G-NEW-22** | Container-level failover (kill Kafka broker, Debezium connector via `docker kill`), verify exactly-once | P1 | 4h |
| **G-NEW-23** | Network flicker mở rộng chiều CDC → Shadow PG + CDC → Kafka (3 chiều) | P2 | 2h |
| **G-NEW-24** | Source DB Overhead acceptance test: CDC max load + concurrent business query → measure Source CPU% + query p99 latency (baseline vs under-CDC) | P0/P1 | 8h |
| **G-NEW-25** | EPS benchmark script: đo CDC engine `events processed / sec` qua `cdc_batches_flushed_total` × records-per-batch | P2 | 2h |
| **G-NEW-26** | Backlog catch-up time-to-zero test (stop CDC 2-3h, restart, đo time tới `lag = 0`) | P2 | 3h |
| **G-NEW-27** | 48-72h soak test plan (script + acceptance memory_alloc growth < 10%/24h) | P2 | 6h |
| **G-NEW-28** | Wire `PerSourcePool` vào handler/service caller path (hiện DEAD CODE, không có caller) | **P0** (regression) | 2h |
| **G-NEW-29** | Deploy node_exporter / cAdvisor cho worker host + DB host, thêm scrape job + dashboard CPU/Mem/Disk I/O | P1 | 2h |
| **G-NEW-30** | G-11 BatchesFlushed nâng từ `NewCounter` → `NewCounterVec{shadow_db,table,status}` để phân tích per-table TPS | P2 | 0.5h |

**Tổng effort thêm**: ~37.5h. Đề xuất ưu tiên xử lý G-NEW-28 (regression — fix DEAD CODE) trước.

---

## 4. Implementation drift vs plan (đã verify trên thực tế)

### Drift 1 — UI Frontend khác plan
- **Plan** (`03_implementation_phase_ui.md`):
  - File `src/pages/AuditPage.tsx` + folder `src/components/audit/` với 3 component (`RatingMatrix.tsx`, `GapList.tsx`, `MetricHealthCards.tsx`).
  - Hooks: `useQASummary`, `useGaps`, `useMetricHealth` (3 hooks).
  - UI library: Ant Design (`Table`, `Tag`, `Statistic`, `Card`).
- **Reality**:
  - `cdc-cms-web/src/pages/AdminAudit.tsx` (đổi tên), **KHÔNG có folder `components/audit/`** — toàn bộ logic inline.
  - Hooks: `useAuditStatus` + `useGaps` (2 hooks, gộp summary+metrics vào 1).
  - UI: **Tailwind CSS classes** (`bg-purple-100 text-purple-800`, …) thay vì Ant Design.
- **Impact**: Functional tương đương, nhưng `04_decisions.md` ADR-007 ghi "Ant Design pure: align cdc-cms-web hiện có, không thêm dep mới" → drift này không được document.

### Drift 2 — G-11 metric label thu hẹp
- **Plan**: `BatchesFlushed = promauto.NewCounterVec(..., []string{"shadow_db", "table", "status"})`.
- **Reality**: `pkgs/metrics/prometheus.go:167` là `NewCounter` (không label).
- **Impact**: KHÔNG break-build nhưng mất khả năng phân tích per-table/per-status. Acceptance criterion 3.2 TPS không thể chia per-table.

### Drift 3 — G-13 PerSourcePool DEAD CODE
- **Plan**: Wire `PerSourcePool` qua DI vào DB layer; metric `cdc_per_source_pool_in_use` set khi acquire/release.
- **Reality**: `pkgs/database/per_source_pool.go` định nghĩa đủ method nhưng **không có caller** (`grep -rn "NewPerSourcePool("` chỉ trả về định nghĩa). `event_bridge.go:67` gọi `eb.pool.Acquire(ctx)` nhưng đây là `pgxpool.Pool.Acquire` (signature 1-arg), KHÔNG phải `PerSourcePool.Acquire(ctx, sourceCode)` (signature 2-arg).
- **Impact**: G-13 không cung cấp giá trị runtime nào. Metric `cdc_per_source_pool_in_use` mãi mãi = 0. Tương tự lesson `metric-defined-but-never-set`.

### Drift 4 — G-9 E2E test simplified
- **Plan**: Dùng `model.SchemaProposal` GORM ORM + `commands.NewApproveSchemaProposalHandler`.
- **Reality**: Inline `CREATE TABLE` SQL + raw INSERT (xem `approve_schema_proposal_e2e_test.go:33-76`). Report `report_execute_p1.md` ghi nhận đã "Xoá dependency sai".
- **Impact**: Test PASS nhưng schema test khác production schema → false sense of safety. Cần verify lại E2E khi schema thực thay đổi.

---

## 5. Verification thực tế (chạy lệnh, lấy evidence)

| Cmd | Kết quả | Note |
|---|---|---|
| `cd centralized-data-service && go vet ./...` | EXIT=0 | Build clean |
| `cd centralized-data-service && go build ./...` | EXIT=0 | Build clean |
| `cd cdc-cms-service && go vet ./...` | EXIT=0 | Build clean |
| `cd cdc-cms-web && tsc -b` | EXIT=0 | FE typecheck PASS |
| `go test ./internal/service -run TestEventOrdering -v` | 2 test PASS (Older + Hash) | `~0.7s` |
| `go test ./internal/api -run TestAuditHandler -v` (cdc-cms-service) | 1 test PASS (Summary) | `~0.6s` |
| `grep -rn "ConsumerLag.WithLabelValues" centralized-data-service/` | `kafka_consumer.go:408` (set), 0 caller path khác | G-1 lossless wire OK |
| `grep -n "FOR UPDATE SKIP LOCKED" dlq_worker.go` | `dlq_worker.go:165` | G-17 wired |
| `grep -n "SetNX" worker_server.go` | `worker_server.go:717` | G-18 wired |
| `grep -rn "NewPerSourcePool("` | **chỉ định nghĩa, 0 caller** | G-13 DEAD CODE (xem §4) |
| `ls deployments/prometheus/alerts/` | `cdc.yml`, `wal_slot.yml` | G-3 + G-6 alerts wired |
| `find . -name "approve_schema_proposal_e2e_test.go"` | tồn tại với build tag `integration` | G-9 file present (chưa chạy E2E thực vì cần testcontainers + docker) |

---

## 6. Governance Compliance (§1..§14)

| Rule | Status | Evidence |
|---|---|---|
| §1 Brain Chairman + Muscle Engineer separation | ✅ | Brain plan-only, Muscle execute với report tách bạch |
| §2 Autonomous Full-loop | ⚠️ Partial | Muscle execute self-loop nhưng report `report_execute_p1.md` ghi "Bước tiếp theo: user có thể yêu cầu execute p2/ui" → vẫn defer cho user thay vì auto |
| §3 Plan & Verify Before Done | ⚠️ Partial | Có verify build, nhưng Entry 09/10 KHÔNG có verify evidence (chỉ ghi action) |
| §7 Full Doc Set + Mandatory Doc Registry | ✅ | 20 prefix + 4 report = 24 file vật lý |
| §7 Metadata Integrity (timestamp format) | ❌ **VIOLATION** | `05_progress.md` Entry 08, 09, 10 có 22 occurrence `[Timestamp]` placeholder thay vì timestamp thực |
| §8 Security Gate (/security-agent) | ❌ Chưa thấy chạy | Không có file `report_security_*` trong workspace |
| §11 Memory APPEND-only | ✅ | Đối chiếu manual: 05_progress chỉ APPEND |
| §12 Brain Code Prohibition | ✅ (giai đoạn plan) | Code demo trong MD block |
| §14 Pre-flight Check end-of-turn | ⚠️ Partial | `07_status_report.md` liệt kê 21 file nhưng thực có 24 file (sót 3 report file mới) |

---

## 7. Inconsistency nội tại workspace

1. **Criterion labelling mismatch**:
   - `10_gap_analysis.md:11-15`: `1.2 Failover/Restart`, `1.3 Event Ordering`, `1.4 Schema Drift`, `2.1 WAL Slot Expire`.
   - `03_implementation_phase_ui.md:55-73` migration seed: `1.2 Schema Drift`, `1.3 Event Ordering`, `2.1 Failover & Self-Healing`.
   - → 2 source-of-truth khác nhau → composite score không reproducible.

2. **Composite delta phi nhất quán**:
   - `03_implementation_phase_p0.md:430`: P0 fix sẽ +9 nhưng raw sum lại là +10 (note tự correct về +9).
   - `10_gap_analysis.md:35`: tương tự ghi "+10 làm tròn về +9".
   - → Composite score 56/64 = 87.5% là projection, không phải re-audit thực.

3. **07_status_report.md ghi `KHÔNG cần thay đổi` cho UI** nhưng `report_execute_p1.md` đã thực thi UI ở Entry 10 → status report đã stale.

4. **02_plan.md row G-17**: `centralized-data-service/internal/service/dlq_worker.go` — đúng path đã sửa, nhưng `02_plan.md` chưa ghi đầy đủ effort G-17/G-18 cho tổng (claim 9h, raw = 0.5+1+1+4+0.5+1.5 = 8.5h).

---

## 8. Files thực sự đã thay đổi (theo report Muscle + grep verification)

### centralized-data-service
| File | Phase | Verify |
|---|---|---|
| `internal/handler/kafka_consumer.go` | G-1, G-4, G-12 | grep ConsumerLag/dlqCircuitBreaker/adaptiveBatcher đều có |
| `internal/handler/dlq_circuit_breaker.go` | G-4 NEW | file size 1902 byte, có struct + RecordDLQWrite |
| `internal/handler/batch_buffer.go` | G-11 | `metrics.BatchesFlushed.Inc()` ở line 191 |
| `internal/handler/main_test.go` | G-7 NEW | goleak.VerifyTestMain |
| `internal/service/dlq_worker.go` | G-17 | `LIMIT ? FOR UPDATE SKIP LOCKED` line 165 |
| `internal/service/schema_adapter_ordering_test.go` | G-8 NEW | 2 test PASS |
| `internal/service/recon_core.go` | G-10 | Tier3OffPeakStart/End config |
| `internal/service/main_test.go` | G-7 NEW | goleak.VerifyTestMain |
| `internal/server/worker_server.go` | G-18 | `redis.RawClient().SetNX(...lockKey, "locked", 50*time.Second)` line 715-717 |
| `internal/sinkworker/main_test.go` | G-7 NEW | goleak.VerifyTestMain |
| `cmd/worker/main.go` | G-7 | import `_ "net/http/pprof"` line 7 |
| `pkgs/metrics/prometheus.go` | G-1/G-4/G-11/G-12/G-13 | 5 metric mới |
| `pkgs/database/per_source_pool.go` | G-13 NEW | ⚠️ **DEAD CODE** — không có caller |
| `deployments/otel-collector-config.yml` | G-2 | otlp/signoz + prometheusremotewrite |
| `deployments/prometheus/prometheus.yml` | G-3 NEW | 5 scrape job |
| `deployments/prometheus/alerts/cdc.yml` | G-3 NEW | 4 alert rule |
| `deployments/prometheus/alerts/wal_slot.yml` | G-6 NEW | 2 alert rule |
| `scripts/smoke_failover.sh` | G-5 NEW | kill worker only (chưa Kafka/Debezium) |
| `scripts/chaos_network.sh` | G-15 NEW | iptables DROP 1 chiều |
| `scripts/load_test.js` | G-16 NEW | k6 stages 1m/5m/2m/1m |
| `docs/runbooks/wal-slot-expire.md` | G-6 NEW | runbook |
| `docs/runbooks/recon-drift-response.md` | G-14 NEW | runbook |
| `docs/runbooks/pipeline-pause-resume.md` | G-14 NEW | runbook |
| `docs/runbooks/schema-drift-approve-sla.md` | G-14 NEW | runbook |

### cdc-cms-service
| File | Verify |
|---|---|
| `migrations/schema/core/0060_qa_gap_state.sql` | 2 table + seed 16 gap + 16 criterion |
| `internal/model/qa_audit.go` | QAGapState + QACriterionRating model |
| `internal/api/dto/audit_dto.go` | DTO QASummaryResponse/Gap/MetricHealth |
| `internal/api/audit_handler.go` | 3 method Summary/Gaps/MetricHealth |
| `internal/api/audit_handler_test.go` | PASS |
| `internal/app/queries/get_qa_summary.go` | OK |
| `internal/app/queries/list_gaps.go` (tham chiếu) | OK |
| `internal/app/queries/get_metric_health.go` | OK |
| `internal/router/router.go:398-400` | wire 3 endpoint dưới `adminGroup` |
| `internal/app/commands/approve_schema_proposal_e2e_test.go` | NEW (build tag `integration`) |

### cdc-cms-web
| File | Verify |
|---|---|
| `src/types/audit.ts` | types định nghĩa |
| `src/hooks/useAuditStatus.ts` | hooks |
| `src/pages/AdminAudit.tsx` | đổi tên từ AuditPage.tsx, inline component (drift §4) |
| `src/App.tsx:33,154,213` | lazy import + menu + route wired |

### CI
| File | Verify |
|---|---|
| `.github/workflows/smoke-failover.yml` | NEW |

---

## 9. Recommendation (priority order)

### Bắt buộc xử lý ngay (vi phạm rule / regression)
1. **Fix Metadata Integrity §7**: Replace 22 `[Timestamp]` placeholder trong `05_progress.md` Entry 08/09/10 với timestamp thực `[2026-05-27TXX:XX:XX+07:00]`. (Chú ý §11 APPEND-only: KHÔNG sửa dòng cũ, mà APPEND 1 dòng "Sai - Revert ts placeholder" + ghi lại entry mới với timestamp đúng. Hoặc đơn giản hơn: APPEND 1 entry meta khẳng định "Entry 08-10 placeholder do batch ghi, timestamp lookup từ file mtime: 11:25, 11:28, 12:02".)
2. **Fix G-13 DEAD CODE** (G-NEW-28): Wire `PerSourcePool` vào handler/service caller path. Hoặc revert G-13 nếu không cần. Bắt buộc để metric `cdc_per_source_pool_in_use` không mãi mãi = 0.
3. **Chạy /security-agent §8**: Bắt buộc sau khi Muscle done. Hiện chưa có `report_security_*` file.

### Quan trọng — bổ sung gap thiếu user requirement
4. G-NEW-22 Container-level failover (kill Kafka/Debezium docker, không chỉ worker).
5. G-NEW-24 Source DB Overhead acceptance test (CPU/IOPS Source khi CDC max).
6. G-NEW-29 node_exporter / cAdvisor scrape cho 5.2 + 5.3 user requirement.
7. G-NEW-20 + G-NEW-21 Mở rộng Schema Drift + Ordering test scenarios.

### Nên làm sớm — cải thiện chất lượng
8. G-NEW-19 Reconciliation E2E test BURST.
9. G-NEW-27 Soak 48-72h plan.
10. Fix Drift 2 (G-11 NewCounterVec với label).
11. Cập nhật `07_status_report.md` reflect các phase đã execute (P0+P1+P2+UI), ghi 24 file (không phải 21).
12. Resolve criterion labelling mismatch §7 (chọn 1 nguồn duy nhất).

---

## 10. Skills đã sử dụng (theo §0.3)

- **Read tool**: đọc 21 file workspace + lessons.md + GEMINI.md + 4 service file thực tế.
- **Bash + grep**: verify thực tế `grep -rn`, `ls`, `wc -l`, `find -name`.
- **go vet + go build + go test**: chứng minh build clean + test PASS.
- **tsc -b**: typecheck FE.
- **Cross-check 5 nhóm user requirement → 16 gap plan → file vật lý** (3-way verification).
- **Governance audit checklist**: §1..§14 GEMINI.md rule-by-rule.
- **Memory governance §11**: report mới này KHÔNG sửa file cũ, APPEND vào workspace as `report_audit_workspace_2026-05-27.md`.
- **Lessons retrieval**: tham chiếu L-CDC-circuit-breaker, L-metric-defined-but-never-set, L-2026-05-21 silent-drop để framework hóa drift detection.
- **No cheat**: KHÔNG sửa config/DB để đạt result; mọi findings dựa trên file thực tế đang tồn tại.
- **Task tracking**: 4 task trong TaskList từ "Đọc remaining workspace" → "Cross-check" → "Verify governance" → "Viết report".

---

## 11. Definition of Done của audit này

- [x] Đọc lessons + GEMINI.md trước khi audit.
- [x] Đọc đủ 21 file workspace.
- [x] Verify thực tế 24 file code đã thay đổi (grep + sed + ls).
- [x] Map 5 nhóm user requirement → 16 gap + đánh dấu coverage.
- [x] Phát hiện 4 implementation drift quan trọng (UI, G-11 label, G-13 dead code, G-9 simplified).
- [x] Phát hiện 1 governance violation (Metadata Integrity §7).
- [x] Đề xuất 12 gap bổ sung (G-NEW-19..G-NEW-30).
- [x] Build/vet/test thực tế trên 3 service → PASS.
- [x] Ghi file `report_audit_workspace_2026-05-27.md` vật lý trong workspace (file này).
- [ ] **Chưa làm** (out-of-scope audit, chờ user verb): security-agent scan, fix dead code G-13, append progress với findings này.
