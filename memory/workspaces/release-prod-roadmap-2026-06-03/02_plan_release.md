# 02_plan_release.md — Lộ trình & Timeline Release Prod

> **Workspace**: `release-prod-roadmap-2026-06-03` | **Ngày**: 2026-06-03 (T0)
> **Mốc**: T0 = 2026-06-03 (Thứ Tư). Ước lượng theo ngày làm việc, 1 Muscle thi công tuần tự,
> Brain audit/plan song song phase kế tiếp (overlap ~1 ngày giữa các phase).

---

## Tổng quan 6 Phase → Prod

```
Phase A  Master (Transmute) finalize ........ 06-03 → 06-05  (3d)  [đã audit, chỉ execute+test]
Phase B  SystemHealth audit + fix ........... 06-05 → 06-09  (3-4d)
Phase C  Reconcile audit + fix ............... 06-09 → 06-13  (4d)
Phase D  Monitor audit + fix ................. 06-13 → 06-16  (3d)
Phase E  Auth-debt + E2E + Hardening + Staging 06-16 → 06-22  (6d)
Phase F  Prod readiness + Release ............ 06-22 → 06-25  (3d)
                                              ─────────────────────
                                   TARGET GO-LIVE:  2026-06-25 (±2d buffer → 06-27)
```

> **Tổng**: ~22 ngày lịch (≈ 3 tuần + buffer). Có overlap Brain↔Muscle nên rút từ ~26 → ~22.

---

## Phase A — Master / Transmute (F2) → prod-ready
**06-03 → 06-05 (3 ngày)** · Owner thi công: Muscle · Input: `feature-masters-page-audit-2026-06-02` (verify đã xong)

| Task | Mô tả | DoD |
|------|-------|-----|
| A1 | Execute **P1** FE Sync Modal trên `/masters` + **fix bug `.find`** (lọc theo `master_table===row.master_name`, tránh run-now nhầm master) | `npm run build` PASS; modal 3 mode hiện đúng |
| A2 | Execute **P3** tooltip TransmuteSchedules (trivial, thêm `Tooltip`+`InfoCircleOutlined`) | build PASS |
| A3 | **Revise doc P2** (bỏ phần wire `DB` vào Config/worker_server — SAI, DB đã wire ở `cmd/sinkworker/main.go`). Chỉ giữ `hasPostIngestSchedule()` + guard trước `publishTransmuteTrigger` | doc P2 revised lưu workspace cũ |
| A4 | Execute **P2** sinkworker post_ingest gate (verify tên cột JOIN `shadow_binding`/`master_binding` lúc code) | `go build ./...` + `go vet` + `go test ./internal/sinkworker/...` PASS |
| A5 | **E2E test 3 mode**: run_now → worker log `transmute complete`; cron `* * * * *` → /schedules có row; realtime → Kafka event → log `transmute-shadow` + `transmute complete` | evidence log đủ 3 mode |
| A6 | **Security gate** `/security-agent` cho diff Phase A | no HIGH/CRITICAL |

**Risk**: P2 đụng SinkWorker hot-path (NATS publish) → gate phải fail-open hợp lý, tránh chặn nhầm realtime.

---

## Phase B — SystemHealth (F3) audit + fix
**06-05 → 06-09 (3-4 ngày)** · Brain audit 06-05/06; Muscle fix 06-07→09

| Task | Mô tả |
|------|-------|
| B1 | Audit endpoint cms `/api/system/health` + worker liveness/readiness probes + DB ping 4 PG |
| B2 | Audit slow-SQL guards (nền `bug-cms-slow-sql-probes`) — đảm bảo không regress; check probe timeout/cardinality |
| B3 | Audit dependency health: Kafka Connect (18083), Schema Registry (18081), NATS, Redis, Debezium connector status |
| B4 | Gap analysis (severity) → fix HIGH/MED → build + vet + test |
| B5 | E2E: tắt 1 dependency (vd NATS) → health endpoint phản ánh đúng degraded/down + không cascade crash |
| B6 | Security gate |

**DoD**: health endpoint phản ánh trung thực trạng thái 4 PG + Kafka + NATS + Redis + Debezium; có evidence degraded-path.

---

## Phase C — Reconcile (F4) audit + fix
**06-09 → 06-13 (4 ngày)** · Brain audit 06-09/10; Muscle fix 06-11→13

| Task | Mô tả |
|------|-------|
| C1 | Audit 3-tier hash window: source vs dest hashing, window boundary, off-by-one |
| C2 | Audit healer: OCC upsert, Mongo read ÉP `primary` (R4), idempotency heal |
| C3 | Verify fix `bug-reconcile-mongodb-not-configured` còn đứng vững trên V2 (ReconCore init không gate bằng legacy MongoDB.URL) |
| C4 | Audit scheduler reconcile cycle (60s), fencing, không double-run; multi-source coverage |
| C5 | Gap analysis → fix → build + vet + test |
| C6 | E2E: cố ý tạo drift (xóa N row ở dest) → reconcile detect missing → heal → verify khớp |
| C7 | Security gate (heal path không leak PII qua log/NATS) |

**DoD**: reconcile detect + heal đúng trên ≥1 Mongo source + ≥1 PG source, có evidence trước/sau.

---

## Phase D — Monitor (F5) audit + fix
**06-13 → 06-16 (3 ngày)** · Brain audit 06-13; Muscle fix 06-14→16

| Task | Mô tả |
|------|-------|
| D1 | Audit JobMonitor close-loop: subscribe `cdc.evt.transmute.completed` → UPDATE `transmute_schedule.last_status` (không miss/dup) |
| D2 | Audit Dashboard V2 monitoring (nền `feature-dashboard-v2-monitoring`) — số liệu khớp DB thật |
| D3 | Audit snapshot monitor + activity log metrics (RowsAffected, materialization) |
| D4 | Audit trace aggregation 5 luồng (nền `feature-all-flows-trace-aggregation`) + OTel collector DNS fix (log spam) |
| D5 | Gap analysis → fix → build + test |
| D6 | E2E: trigger transmute → dashboard cập nhật last_status đúng; trace span đủ 5 luồng |
| D7 | Security gate |

**DoD**: monitor phản ánh đúng close-loop + dashboard số liệu khớp + trace aggregate hiển thị.

---

## Phase E — Auth-debt + E2E + Hardening + Staging
**06-16 → 06-22 (6 ngày)** · Muscle chính, Brain điều phối

| Task | Mô tả |
|------|-------|
| E1 | **Đóng nợ `cdc-auth-service`**: viết test cơ bản (login/jwt/bcrypt) + chạy được local | prod blocker R3 |
| E2 | **E2E xuyên suốt 5 luồng**: operator login → register source → approve master → schedule run → ingest → transmute → reconcile → monitor. Dùng `/qa-agent` (Playwright) cho FE path |
| E3 | **Validate Staging hiện có** (D-2: staging ĐÃ CÓ) + seed data + verify 13 container/4 PG/Kafka/NATS lành mạnh (~1d, không phải dựng mới) | — |
| E4 | **Migration review**: thứ tự + idempotent + rollback script cho toàn bộ migration DB |
| E5 | **Load/perf smoke** theo SLA (Q3) — throughput ingest + latency shadow→master |
| E6 | **Observability hardening**: OTel DNS fix, alert rules cơ bản |
| E7 | **Security review toàn repo** `/security-review` — PII masking, token strong, DLQ sanitize, body-size cap |

**DoD**: 1 lần chạy E2E full-pipeline PASS trên staging có evidence; auth có test; security no HIGH.

---

## Phase F — Prod readiness + Release
**06-22 → 06-25 (3 ngày)**

| Task | Mô tả |
|------|-------|
| F1 | **Runbook**: start/stop, scale, common failure → recovery (Kafka/NATS/PG down) |
| F2 | **Rollback plan**: binary + migration rollback, feature toggle |
| F3 | **Go/No-Go checklist** review với R2 + R3 (đủ 6 tiêu chí mỗi luồng) |
| F4 | **Deploy prod** (canary/blue-green nếu hạ tầng cho phép) + smoke prod |
| F5 | **Post-release watch** 24-48h: dashboard + DLQ + reconcile drift = 0 |

**DoD**: Go-live, smoke prod PASS, watch window không có CRITICAL.

---

## Đường găng (Critical Path) & Song song hoá
- **Critical path**: A → B → C → D → E2(E2E) → F4(deploy). E1(auth test) chạy **song song** từ Phase B vì độc lập.
- **Có thể rút ngắn** nếu thêm Muscle thứ 2: B/C/D audit độc lập → fan-out 3 luồng song song (tiết kiệm ~5-6 ngày → go-live ~06-19).
- **Brain–Muscle overlap**: Brain audit phase N+1 trong khi Muscle fix phase N.

## Rủi ro & Giảm thiểu
| Risk | Mức | Giảm thiểu |
|------|-----|-----------|
| ~~Chưa có Staging~~ → ĐÃ CÓ (D-2) | ~~HIGH~~ ✅ gỡ | E3 chỉ validate + seed (~1d) |
| Big-bang: gap muộn ở luồng cuối trượt cả release (D-3) | MED | Buffer ±2d + escalation re-plan nếu gap > 1 luồng (§8); audit sớm để lộ gap trước |
| `cdc-auth-service` 0 test | HIGH | E1 ưu tiên, chạy song song sớm |
| Audit lộ gap lớn ngoài dự kiến (vd Reconcile) | MED | Buffer ±2d; nếu gap > 1 luồng → re-plan (§3, §8 escalation) |
| SinkWorker gate (P2) chặn nhầm realtime | MED | fail-open + test degraded |
| OTel DNS spam che mất alert thật | LOW | E6 fix |
| Mongo ObjectId cast bigint (G-13 cũ defer) | MED | xác minh có còn ảnh hưởng transmute prod không |

## Verb chờ User
- `chốt timeline` — Brain ghi nhận, bắt đầu điều phối Phase A.
- `execute phase A` — Muscle thi công Master (đã có plan verified sẵn).
- `parallel` — bật chế độ 2-Muscle fan-out B/C/D (rút ~06-19).
- `revise` — điều chỉnh scope/thứ tự/ngày.
- Trả lời Q1–Q4 (`01_requirements §R5`) để chốt timeline chính xác.
