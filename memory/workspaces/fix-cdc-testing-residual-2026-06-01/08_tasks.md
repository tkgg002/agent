# 08_tasks — Fix CDC Testing Residual (2026-06-01)

## Wave 1 — P0 (Pre-compaction, đã verify build/vet/test)
- [x] G5-RES: 2 FAIL cms mapping_rule assertion (test file edit)
- [x] G8-RES: APPEND path correction + FAKE log vào plan workspace
- [x] G3-RES: metrics_callback_test.go NEW 72 LOC, 2/2 PASS
- [x] G1-RES: ConsumerOffset gauge + Set() call (build/vet PASS)

## Wave 2 — P1
- [ ] G4-RES: k6 CDC data path script + xk6-sql build
- [ ] G2-RES: WAL auto snapshot resume service + unit test

## Wave 3 — P2
- [ ] G7-RES: Adaptive batch throttle-down when dest unhealthy
- [ ] G6-RES: Chaos pumba thay iptables

## Verification gates
- [ ] `go build ./...` PASS (centralized-data-service + cdc-cms-service)
- [ ] `go vet ./...` PASS
- [ ] `go test ./...` PASS (unit)
- [ ] k6 run thresholds GREEN
- [ ] /security-agent gate
- [ ] Report final: `report_fix_residual_2026-06-01.md`
