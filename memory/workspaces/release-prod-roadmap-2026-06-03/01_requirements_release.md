# 01_requirements_release.md — Điều kiện Release Prod

> **Workspace**: `release-prod-roadmap-2026-06-03` | **Ngày**: 2026-06-03

## R1. Phạm vi audit còn lại (User chỉ định)
Audit + đóng gap + test cho 4 luồng: **Master, SystemHealth, Reconcile, Monitor**.
Master "vừa được audit" → chỉ cần thi công + test (không audit lại từ đầu).

## R2. Definition of Done — toàn hệ thống (Release Gate)
Một luồng chỉ được coi là "prod-ready" khi đủ **6 tiêu chí**:
1. **Audit doc** đầy đủ (00→10 prefix) + gap analysis có severity.
2. **Fix** mọi gap HIGH (MED tùy rủi ro, LOW có thể defer + ticket).
3. **Build PASS**: `go build ./...` + `go vet` (BE) / `npm run build` + `tsc` (FE).
4. **Test PASS**: unit + ít nhất 1 happy-path E2E thủ công có evidence (log/screenshot).
5. **Security gate**: `/security-agent` chạy, không còn lỗ hổng HIGH/CRITICAL.
6. **Verification before Done** (§3): "Một Staff Engineer có duyệt PR này không?" — có evidence CI/log.

## R3. Release Gate toàn cục (ngoài per-flow)
- [ ] `cdc-auth-service` có test cơ bản + chạy được (đóng nợ B1).
- [ ] E2E xuyên suốt 5 luồng trên môi trường gần-prod (staging) với evidence.
- [ ] Observability: OTel collector DNS fix (hiện log spam cosmetic), trace 5 luồng aggregate.
- [ ] Migration DB review: thứ tự migration idempotent, rollback script.
- [ ] Runbook + Rollback plan + Go/No-Go checklist.
- [ ] Security review toàn repo (PII masking, token strong, DLQ sanitize) — không HIGH.
- [ ] Load/perf smoke: pipeline chịu được throughput mục tiêu (cần Boss cung cấp số).

## R4. Ràng buộc nghiệp vụ phải giữ (không được phá khi fix)
- `master_binding.is_active` chỉ bật khi `schema_status='approved'` (CHECK constraint).
- Transmute gate chain: master active+approved ∧ shadow active+profile_active ∧ ≥1 approved rule.
- OCC theo `_source_ts older` — không overwrite dữ liệu mới hơn.
- Mongo healing read ÉP `primary` (tránh replication lag).
- DLQ + NATS alert: KHÔNG phát tán PII thô; mask trước persist/publish.

## R5. Câu hỏi mở cần Boss chốt (ảnh hưởng timeline)
- Q1: Có **deadline cứng** release không? (đổi mức song song hoá / cắt scope)
- Q2: **Staging** đã có chưa, hay phải dựng trong roadmap? (ảnh hưởng Phase E ~3-5 ngày)
- Q3: Throughput/SLA mục tiêu prod (rows/s, latency shadow→master) để định nghĩa load test.
- Q4: Có chấp nhận **release theo từng luồng** (incremental) hay phải **big-bang** đủ 5 luồng?
