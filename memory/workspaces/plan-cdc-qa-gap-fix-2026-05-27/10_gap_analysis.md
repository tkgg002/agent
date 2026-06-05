# 10_gap_analysis — Mapping Gap → Fix → Score Impact

> Reference audit cũ: `agent/memory/workspaces/audit-cdc-qa-process-2026-05-26/10_gap_analysis.md`.

## Baseline 16 criterion (audit 2026-05-26)

| # | Group | Criterion | Rating cũ | Weight |
|---|---|---|---|---|
| 1.1 | Functional | Data Reconciliation | L2 | 4 |
| 1.2 | Functional | Failover/Restart | L2 | 4 |
| 1.3 | Functional | Event Ordering | L3 | 4 |
| 1.4 | Functional | Schema Drift | L3 | 4 |
| 2.1 | Stability | WAL Slot Expire | L1 | 4 |
| 2.2 | Stability | Network Flicker | L2 | 4 |
| 2.3 | Stability | LSN Tracking | L3 | 4 |
| 2.4 | Stability | DLQ Spike | L1 | 4 |
| 3.1 | Performance | Data Lag | L3 | 4 |
| 3.2 | Performance | TPS | L3 | 4 |
| 3.3 | Performance | Backlog/Burst | L3 | 4 |
| 4.1 | Resource | Memory Leak | L2 | 4 |
| 4.2 | Resource | Concurrency | L3 | 4 |
| 4.3 | Resource | Pool Sizing | L2 | 4 |
| 5.1 | Metric | Exporter Coverage | L0 | 4 |
| 5.2 | Metric | Trace | L0 | 4 |

**Tổng**: (2+2+3+3 + 1+2+3+1 + 3+3+3 + 2+3+2 + 0+0) × 1 = 35/64 (L4 max = 4 mỗi tiêu chí × 16 = 64).

## Sau Phase P0 — score +9 → 44/64

| Criterion | Trước | Fix | Sau | Delta |
|---|---|---|---|---|
| 5.1 Metric Exporter | L0 | G-1+G-3 | L4 | +4 |
| 5.2 Trace | L0 | G-2 | L3 | +3 |
| 2.4 DLQ Spike | L1 | G-4 | L4 | +3 |
| **Subtotal** | | | | **+10** |

Tổng cộng +10, làm tròn về **+9** vì L2.4 thực tế L1→L3 (G-4 đủ cho L3 vì chưa có chaos test xác minh L4) — trong file `02_plan.md` chốt +9.

> Note: Score delta thực tế khi Muscle thực hiện sẽ re-audit chính xác sau khi PASS verify command.

## Sau Phase P1 — score +7 → 51/64

| Criterion | Trước | Fix | Sau | Delta |
|---|---|---|---|---|
| 1.2 Failover | L2 | G-5 | L4 | +2 |
| 1.3 Event Ordering | L3 | G-8 | L4 | +1 |
| 1.4 Schema Drift | L3 | G-9 | L4 | +1 |
| 2.1 WAL Slot | L1 | G-6 | L3 | +2 |
| 4.1 Memory Leak | L2 | G-7 | L3 | +1 |
| **Subtotal** | | | | **+7** |

## Sau Phase P2 — score +5 → 56/64

| Criterion | Trước | Fix | Sau | Delta |
|---|---|---|---|---|
| 3.2 TPS | L3 | G-11 + G-16 | L4 | +1 |
| 3.3 Backlog/Burst | L3 | G-12 | L4 | +1 |
| 4.2 Concurrency | L3 | G-13 | L4 | +1 |
| 2.2 Network Flicker | L2 | G-15 | L3 | +1 |
| 3.1 Data Lag | L3 | G-16 | L4 | +1 |
| **Subtotal** | | | | **+5** |

> G-10 (Tier3 config) refine 1.1 nhưng vẫn L4 cap. G-14 (runbook) hỗ trợ 2.3 LSN nhưng L3 đã đạt ceiling effective.

## Gap còn lại sau P0+P1+P2 (8/64 chưa đạt L4)

| Criterion | Rating sau | Lý do chưa L4 |
|---|---|---|
| 1.1 Reconcile | L4 cap | Đã L4, nhưng cần dài hạn data quality dashboard |
| 2.1 WAL Slot | L3 | L4 cần auto-recover slot (out of scope hiện tại) |
| 2.2 Network | L3 | L4 cần multi-DC failover (out of scope) |
| 2.3 LSN Tracking | L3 | Tương tự |
| 4.1 Memory Leak | L3 | L4 cần long-soak 7 ngày, hiện 1h |
| 4.3 Pool Sizing | L2 | Không touch trong plan này — Future work |
| 5.2 Trace | L3 | L4 cần sampling adaptive (future) |

→ **Backlog future** (không trong scope plan này): pool sizing auto-tune, multi-DC failover, long-soak 7d test, trace adaptive sampling.

## UI Phase impact
- Không thay đổi score (visibility-only).
- Giá trị: UI là single-source-of-truth cho 16 gap state, operator follow-up tốt hơn.

## Composite formula
```
composite = Σ(rating_value × weight) / Σ(max_rating × weight)
         = Σ(L_i × 4) / (4 × 4 × 16)
         = Σ(L_i) / 64
```

| Phase | Σ L_i | Composite | % |
|---|---|---|---|
| Baseline | 35 | 35/64 | 54.7% |
| +P0 | 44 | 44/64 | 68.75% |
| +P1 | 51 | 51/64 | 79.7% |
| +P2 | 56 | 56/64 | 87.5% |
