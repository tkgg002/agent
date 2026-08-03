# 12 — Implementation Plan: Audit Recon payment_bills

> Tạo: 2026-07-20T10:19:00+07:00 | Agent: Claude-Sonnet-4.6-Thinking
> Loại task: Hotfix/Analysis (Audit Phase)

---

## Mục tiêu

Audit luồng Recon Tier A cho bảng `payment_bills` trên production đang chạy 2h window.
Phiên này tập trung **phân tích thuần túy** — không thay đổi code.

---

## Kế hoạch thực hiện

### Phase 1 — Research (DONE ✅)

**Skills khai báo:** Golang patterns, PostgreSQL, CDC Data Pipeline, Debugger Agent

1. Đọc trace log từ user → map sang code flow
2. Đọc toàn bộ code liên quan (4 files chính)
3. Xác định bottleneck theo từng span

### Phase 2 — Analysis (DONE ✅)

1. P1: Phân tích false drift — hash ms ↔ diff giây mismatch + TIMESTAMP timezone
2. P2+P3: Phân tích MongoDB index missing
3. P4: Phân tích granularity mismatch

### Phase 3 — Action (PENDING — chờ User)

Sau khi User xác nhận:
1. **A1** Chạy `SHOW TIMEZONE` + `getIndexes()` để confirm root cause
2. **A2** Tạo MongoDB index (background, không lock production)
3. **A3** (tùy) Patch `diffIDTsSegmentA` nếu cần

---

## Files được đọc (read-only)

```
centralized-data-service/internal/service/recon/
├── recon_tier_a.go          (40KB — RunHashWindowCheck, pickScanRangeWithLag)
├── recon_dest_hash.go       (6.6KB — HashWindow TIMESTAMP branch)
├── recon_dest_query.go      (16.7KB — ListIDTsInWindow, MaxWindowTs)
└── recon_engine.go          (9.4KB — ReconCoreConfig defaults)
```

## Files được tạo (workspace docs)

```
agent/memory/workspaces/ReconAuditPaymentBills20260720/
├── 01_requirements_audit.md
├── 05_progress.md
├── 08_tasks_audit.md
├── 11_report_audit.md
├── 12_implementation_plan_audit.md  (file này)
└── 13_analysis_audit.md
```

---

## Verification Plan

- [x] Đọc code thực tế (không suy đoán)
- [x] Cross-check trace log với code
- [x] Tính toán số liệu thời gian (2h/15min = 8 windows, 8×5.3s = 42.4s)
- [ ] User confirm `SHOW TIMEZONE` + MongoDB indexes
- [ ] Chạy governance linter pass
