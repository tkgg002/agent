# Master Gap Analysis — CDC Integration Plans Review

> **Date**: 2026-04-17
> **Reviewer**: Brain (claude-opus-4-7)
> **Plans reviewed**:
>   - `02_plan_observability_final.md`
>   - `02_plan_data_integrity_final.md`
> **Reference detail docs**:
>   - `10_gap_analysis_data_integrity_review.md` (20 sections, 15 critical/high/medium issues)
>   - `10_gap_analysis_observability_review.md` (21 sections, 15 critical/high/medium issues)

---

## 1. Executive Summary (≤ 1 page)

**Bối cảnh**: 2 plan trên được Muscle (claude-sonnet-4-6) soạn. Khung kiến trúc đúng (Core/Agent, OTel, DLQ, Version-aware Heal, Merkle). Nhưng nhiều chi tiết fatal ở scale prod (ước tính 50M records/bảng lớn, 200+ bảng, 20+ topics, ~500 events/s).

**Severity matrix**:

|  | Data Integrity | Observability |
|:--|:-:|:-:|
| **CRITICAL** | 3 | 2 |
| **HIGH** | 4 | 4 |
| **MEDIUM** | 6 | 5 |
| **LOW** | 2 | 4 |
| **Total** | 15 | 15 |

**Verdict**:
- **Observability**: v1 đã chạy (T1-T12 runtime verified), nhưng có **silent bug** (T10 percentile sai semantics) + nhiều rủi ro scale (T1 không cache, T13 OOM khi SigNoz down, cardinality không bounded). **Không pass Staff Review**.
- **Data Integrity**: chưa implement. Nếu implement theo plan hiện tại → **disaster tại Tier 2/3** (full scan 50M), compact policy sai, heal semantics sai. **MUST rewrite trước khi code**.

**Rủi ro không fix**:
- Observability: alert sai → team ra quyết định sai capacity, spike SigNoz/Prom bill, Worker OOM.
- Data Integrity: Recon run làm degrade production DB (Mongo primary), heal ghi đè data mới, DLQ leak event.

---

## 2. Top 10 Issues — Must-Fix Before Next Implementation

| # | Plan | Issue | Impact | Fix |
|:--|:-----|:------|:-------|:----|
| 1 | Both | Không có "scale calculation" trong plan (memory/network/DB load ở 50M) | Plan ngầm giả định 1M → mọi giải pháp sai lệch × 50 | Bắt buộc thêm mục "Scale budget" đầu mỗi plan |
| 2 | DI | Tier 2 "ID Set batch 10K" không rõ chiến lược → dễ hiểu fetch full | RAM 2GB, network 1.2GB/bảng | Window-based + XOR-hash aggregate, không transfer full ID |
| 3 | DI | "Merkle Tree" = flat-chunk MD5, không hierarchical | 5000 hash compares mỗi run, instability khi insert | Time-partitioned hash + bucketed-hash (256 bucket ổn định) |
| 4 | DI | `cleanup.policy=compact` blanket | Mất ordering/delete → data lệch tại downstream | `retention.ms=14d` + alert lag > 70% retention |
| 5 | DI | Version-aware Heal so `_synced_at` (sai field) | Heal ghi đè sai version | Thêm `_source_ts`, OCC `WHERE _source_ts < EXCLUDED` |
| 6 | DI | Agent không throttle, không read-replica | Recon đấm Mongo primary | `readPreference=secondary`, rate limiter (token bucket) |
| 7 | Obs | `/api/system/health` blocking aggregate 5 service call | Cascade timeout, amplify user tab × 5 calls | Background collector → Redis cache → handler O(1) read |
| 8 | Obs | T10 percentile "compute từ activity_log" (silent bug) | Stats sai → mislead alert/capacity | Dùng Prom `histogram_quantile(0.95, ...)` |
| 9 | Obs | OTel logs `sampleRatio=1.0` không backpressure | OOM Worker khi SigNoz down | Severity-aware sample + memory_limiter + fallback console |
| 10 | Both | Destructive action (restart, reset, heal) không RBAC/audit/idempotency | Misclick = prod impact | Role gate + Idempotency-Key + audit table |

---

## 3. Flow-by-Flow Review — Điểm chưa ổn

### 3.1. Observability flows

| Flow | Điểm chưa ổn | Detail ref |
|:-----|:-------------|:-----------|
| GET /api/system/health | Sync 5 external, no timeout, no cache | Obs §1 |
| Kafka consumer batch log | Per-topic flusher × 20 → contention | Obs §3 |
| NATS command result log | OK concept, cần batch + TTL | Obs §16.3 |
| E2E latency measurement | Histogram đúng, nhưng consume sai (T10) | Obs §2 |
| Debezium poll + restart | Poll cần cache, restart cần RBAC + rate limit | Obs §1, §8 |
| OTel log pipeline | Sample 100% + no buffer bound → OOM risk | Obs §5 |
| Trace context Kafka | Producer không inject W3C → trace đứt | Obs §11 |

### 3.2. Data Integrity flows

| Flow | Điểm chưa ổn | Detail ref |
|:-----|:-------------|:-----------|
| Recon Tier 1 (count) | 200 bảng × 5min = 2400 count/h → PG replica quá tải | DI §16.1 |
| Recon Tier 2 (ID set) | Transfer full ID → RAM/network blow up | DI §1 |
| Recon Tier 3 (Merkle) | Full scan 50M mỗi 24h → 8-12 phút/bảng × N | DI §3 |
| Heal flow | Per-ID FindOne, sai field version | DI §5, §7 |
| DLQ write | Không rõ ACK-before-insert — risk leak | DI §16.4 |
| Debezium signal | Không throttle chunk, không filter range | DI §11 |
| Schedule/cron | Không lock, không jitter, không leader election | DI §9, §16.5 |

---

## 4. Cross-cutting Concerns (cả 2 plan chung)

1. **Scale calculation** — Plan không tính memory/network/storage → patterns không phù hợp quy mô thực.
2. **Security/RBAC** — Destructive action (restart, reset, heal) thiếu auth + audit.
3. **Retention/bloat** — `activity_log`, `failed_sync_logs`, `audit_log`, `recon_report` — không plan partition/TTL.
4. **Idempotency** — API heal, retry, snapshot trigger không có Idempotency-Key → double-click race.
5. **Observability của chính Recon** — Recon có metrics để user biết nó đang chạy/healthy không? Chưa có.
6. **SLO/error budget** — Threshold alert hiện dựa đoán, không gắn với SLO business → alert fatigue.
7. **Capacity planning** — Đã verify với DevOps/DBA: PG replica, Mongo secondary, Prom storage, SigNoz storage có đủ cho traffic thực tế?

---

## 5. Unverified Assumptions — cần xác nhận trước v3

Plan hiện tại ngầm định các điều sau, **chưa verify**:

| Assumption | Cần verify qua | Impact nếu sai |
|:-----------|:--------------|:---------------|
| Có MongoDB secondary replica | Check connection string, Mongo topology | Recon đấm primary |
| Có PG read-replica | Check infra config | Recon load PG primary |
| Có Prometheus (cạnh SigNoz) | Check monitoring stack | T10 không dùng được `histogram_quantile` |
| Debezium dùng JSON converter (không Schema Registry) | Check Kafka Connect config | T13 validation strategy sai |
| NATS dùng Core hay JetStream | Check Worker/CMS code `nc.Publish` vs `js.Publish` | `/jsz` endpoint có thể sai |
| `updated_at` index có trên cả Mongo + PG | Check schema | Window-based Recon full-scan |
| Bảng CDC có cột `_source_ts` | Check migration 001-007 | OCC heal cần thêm migration |
| Worker có dùng OTel Kafka instrumentation | Check Worker main.go | Trace context đứt |
| FE có React Query/SWR | Check package.json | Phải refactor data fetching |
| Có Redis available cho cache | Check infra | Background collector phải fallback storage |

---

## 6. Roadmap đề xuất (3 phase rewrite)

### Phase A — Pre-flight verification (Brain + Muscle, 1 ngày)
- Checklist §5 — Muscle verify từng assumption qua code/infra, báo cáo Brain.
- Output: `10_gap_analysis_assumptions_verified.md`.

### Phase B — Plan v3 rewrite (Brain, 1 ngày)
- Brain rewrite:
  - `02_plan_observability_final.md` → `02_plan_observability_v3.md`
  - `02_plan_data_integrity_final.md` → `02_plan_data_integrity_v3.md`
- Kèm bảng **Scale budget** đầu mỗi plan.
- Kèm **change log** so với v2.
- User approve v3 trước khi Muscle thực thi.

### Phase C — Implementation (Muscle, 2-4 tuần tùy scope)

#### C.1 Quick wins (tuần 1):
- Obs: T10 fix silent bug (dùng Prom hoặc HdrHistogram in-memory).
- Obs: T1 background collector + Redis cache.
- Obs: T13 OTel memory_limiter + severity sample.
- DI: Migration `_source_ts` + retention policy cho `failed_sync_logs`.

#### C.2 Core rewrite (tuần 2-3):
- DI: Recon Tier 2/3 chuyển sang window + XOR hash.
- DI: Heal OCC + batch `$in`.
- DI: Agent rate limiter + secondary read.
- Obs: Activity log single flusher multi-topic + partition/TTL.
- Obs: Alert state machine + SLO định nghĩa.

#### C.3 Hardening (tuần 4):
- RBAC + audit cho mọi destructive button.
- Idempotency-Key middleware.
- Kafka retention config + lag alert rule.
- Load test Recon với dataset mirror production.

---

## 7. Tài liệu rà soát — bộ hoàn chỉnh

| Tài liệu | Nội dung | Trạng thái |
|:---------|:---------|:-----------|
| `10_gap_analysis_data_integrity_review.md` | Review chi tiết plan DI — 21 section, 15 issue, propose 10 task mới + 5 rewrite | ✅ Done |
| `10_gap_analysis_observability_review.md` | Review chi tiết plan Obs — 21 section, 15 issue, propose 10 task mới + 3 rewrite | ✅ Done |
| `10_gap_analysis_master_summary.md` | Tài liệu này — executive summary + roadmap | ✅ Done |
| `09_tasks_solution_review_action_items.md` | Action item cụ thể để Muscle thực thi sau khi v3 approve | ✅ Done (file kế tiếp) |
| `02_plan_data_integrity_v3.md` | Plan rewrite tích hợp fixes | ✅ Done (2026-04-17) |
| `02_plan_observability_v3.md` | Plan rewrite tích hợp fixes | ✅ Done (2026-04-17) |
| `10_gap_analysis_assumptions_verified.md` | Kết quả verify 10 assumption + 10 additional facts | ✅ Done (2026-04-17) |

---

## 8. Ngôn ngữ kết luận cho User

**Có duyệt plan hiện tại không?**
> KHÔNG. Plan DI có 7 critical/high issue sẽ sập ở production. Plan Obs đã chạy v1 nhưng có 2 critical (T10 silent bug, T1 cascade fail) + 4 high issue phải fix trước khi scale.

**Next step đề xuất**:
1. User đọc 2 file `10_gap_analysis_*_review.md` chi tiết.
2. Thống nhất với Brain: đồng ý/bỏ/thay đổi issue nào.
3. Muscle chạy Phase A (verify assumptions) — báo cáo ngắn.
4. Brain viết plan v3.
5. User approve v3.
6. Muscle thực thi theo Roadmap §6.

---

## 9. Lesson rút ra

Đã append 2 lesson vào `agent/memory/global/lessons.md`:
1. **Scale calculation mandatory** — Plan data system phải có scale budget đầu doc.
2. **Runtime verified ≠ semantic correct** — Metric "chạy không crash" không chứng minh đúng semantics. Cần double-check với source-of-truth độc lập.
