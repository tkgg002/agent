# Plan: Wiring Fix — Nối tất cả flows end-to-end

> Date: 2026-04-16
> Priority: P0 CRITICAL — features đang là facade
> Rule: Mỗi fix xong → trace FE → API → NATS → Worker → DB → FE

---

## Gaps phải fix

| # | Gap | CMS sends | Worker missing |
|:--|:----|:----------|:---------------|
| 1 | Recon Check | `cdc.cmd.recon-check` | Subscribe + handler (call reconCore.RunTier1/2/3) |
| 2 | Recon Heal | `cdc.cmd.recon-heal` | Subscribe + handler (call reconCore.Heal) |
| 3 | Retry Failed | `cdc.cmd.retry-failed` | Subscribe + handler (re-upsert from raw_json) |
| 4 | Debezium Signal | `cdc.cmd.debezium-signal` | Subscribe + handler (insert MongoDB signal) |
| 5 | Debezium Snapshot | `cdc.cmd.debezium-snapshot` | Subscribe + handler (insert MongoDB signal) |
| 6 | ReconCore unused | `_ = reconCore` | Wire vào handlers + schedule |
| 7 | Redis health fake | Returns "up" always | Actually ping Redis |
| 8 | Activity Log filters | Missing cmd-*, recon-* | Add all operation types |

## Tasks

- [ ] W1: Worker subscribe 5 NATS commands + handlers
- [ ] W2: Wire reconCore into handlers (remove `_ = reconCore`)
- [ ] W3: Fix Redis health check (actual ping)
- [ ] W4: Fix Activity Log FE filters
- [ ] W5: Trace EACH flow E2E: button → DB → display
- [ ] W6: Update 11_flow_testing docs
