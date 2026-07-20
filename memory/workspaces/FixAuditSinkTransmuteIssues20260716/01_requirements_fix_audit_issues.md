# Requirements — Fix Audit Sink & Transmute Issues

## Nguồn
- Báo cáo audit: `audit_sink_transmute_risks.md` (40 rủi ro, 7 Critical)
- Workspace audit: `AuditSinkTransmuteRisks20260716/`

## Yêu cầu
1. Fix 5 issues P0 — chặn data loss production ngay
2. Fix 5 issues P1 — sprint tiếp theo
3. Fix 4 issues P2 — kiến trúc (sau)

## Scope
- **In scope:** Fix code trong `internal/handler/shadow/`, `internal/service/master/`, `internal/handler/master/`
- **Out of scope:** Tách layer handler/shadow (P2 — để sau theo yêu cầu anh)

## Definition of Done
- [ ] P0: SINK-C1 + SINK-C2 fix → deploy không mất data khi restart
- [ ] P0: 4 silent drop points có metrics → monitor được
- [ ] P0: transmute panic → recover, không stuck schedule
- [ ] P1: bulkUpsert retry → transient error không mất data
- [ ] P1: QueueSubscribe → không duplicate processing
- [ ] Tests pass, không regression
