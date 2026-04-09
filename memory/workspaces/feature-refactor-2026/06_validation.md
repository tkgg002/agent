# 06_validation.md - Quality Assurance

## Success Metrics

| Phase | Definition of Done |
|-------|--------------------|
| **GĐ 1** | Restart service -> Zero 502 errors. No "Socket Hang up". |
| **GĐ 2** | Kill Disbursement (Go) -> Retry success on restart. |
| **GĐ 3** | Network Cut -> Queue persists. Saga auto-refunds on failure. |
| **GĐ 4** | User never sees "Unknown Error". |

## Test Scenarios
- [ ] Load Test: Restart 10% of pods during 1000 RPS. Verify Error Rate < 0.01%.
- [ ] Chaos Test: Kill DB connection. Verify Sweeper picks up stuck transactions.
