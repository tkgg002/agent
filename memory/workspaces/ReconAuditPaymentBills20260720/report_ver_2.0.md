# 📋 Report Audit — Recon `payment_bills` · v2.0

> **Ngày:** 2026-07-20 | **Version:** 2.0 (updated after User confirm)
> **Agent:** Claude Sonnet 4.6 (Thinking)
> **Workspace:** `ReconAuditPaymentBills20260720`
> **Trạng thái:** ✅ ROOT CAUSE CONFIRMED — 🔧 Chờ approve fix

---

## 📝 Changelog v2.0

| Mục | Trạng thái cũ | Trạng thái mới |
|-----|--------------|----------------|
| P1 — Timezone drift | 🔴 CRITICAL (nghi ngờ) | ✅ **CLOSED** — Đã fix dynamic detection |
| P2+P3 — MongoDB index | 🟡 HIGH (chưa confirm) | 🔴 **CONFIRMED ROOT CAUSE** |
| P4 — Granularity mismatch | 🟠 MEDIUM | 🟠 MEDIUM (giữ nguyên) |

**User confirmed:**
> *"anh có làm vụ tự detect timezone trên db xong chuyển về number để so sánh rồi"* → P1 đã xử lý.
> *"db mongo thiếu lastupdatedat đó"* → P2+P3 confirmed.

---

## Flow Diagram (Updated)

```
POST /api/reconciliation/check (2h range)
│
├─ pick_scan_range (2.48s)
│   ├─ 🔴 MongoDB MAX(lastUpdatedAt) → 2.46s  ← COLLSCAN (no index)
│   └─ ✅ Postgres MAX("lastUpdatedAt") → 2ms
│
├─ verify_global_range: HashWindow(lo, hi)
│   ├─ 🔴 MongoDB hash_window (2h) → 5.14s   ← COLLSCAN (no index)
│   └─ ✅ Postgres hash_window     → 9.52ms
│   → HASH KHÔNG KHỚP → vào window_loop (do COLLSCAN không stable?)
│
└─ window_loop (8 windows × 15min)
    └─ 8/8 windows đều drill_down
        ├─ 🔴 MongoDB ListIDTsInWindow → 5.3s × 8 = 42.4s  ← COLLSCAN
        └─ ✅ Postgres ListIDTsInWindow → 5ms × 8 = 40ms

TOTAL: ~90s | KẾT QUẢ: 1,952 → 1,952 (diff=0) ✅
```

**47% thời gian** (42.4s) là từ MongoDB COLLSCAN `ListIDTsInWindow`.
**Root cause = thiếu index `lastUpdatedAt` trên MongoDB.**

---

## Findings (Updated)

### ✅ P1 — CLOSED: Timezone drift

**Trạng thái:** Đã xử lý.

User đã implement: **dynamic timezone detection** — kết nối DB, query `SHOW TIMEZONE`, convert sang `*time.Location`, dùng để normalize timestamp về epoch number trước khi so sánh/hash.

→ Không cần action thêm cho P1.

---

### 🔴 P2+P3 — CONFIRMED ROOT CAUSE: MongoDB thiếu index `lastUpdatedAt`

**Confirmed bởi User:** `"db mongo thiếu lastUpdatedAt đó"`

**Impact đo được:**

| Query | Thực tế | Với index |
|-------|---------|-----------|
| `MAX(lastUpdatedAt)` (pick_scan_range) | **2.46s** | < 5ms |
| `find({ lastUpdatedAt: {$gte,$lt} })` per window | **5.3s** | < 50ms |
| Tổng 8 windows | **42.4s** | **< 0.5s** |
| **Tổng toàn bộ run** | **~90s** | **< 10s** |

**Fix:**
```javascript
// Chạy trên MongoDB production (background = không lock collection):
db.payment_bills.createIndex(
  { "lastUpdatedAt": 1 },
  { background: true, name: "idx_lastUpdatedAt" }
)

// Verify sau khi tạo:
db.payment_bills.getIndexes()
```

> ⚠️ **Lưu ý production:** `background: true` tạo index không block write ops.
> Với collection lớn có thể mất vài phút. Monitor qua `db.currentOp()`.

---

### 🟠 P4 — MEDIUM: Granularity hash ↔ diff không nhất quán

**Trạng thái:** Còn nguyên, chưa fix.

```go
// recon_dest_hash.go:126 — hash dùng millisecond
xorAcc ^= hashIDPlusTsMs(id, ts.UnixMilli())

// recon_tier_a.go:1059 — diff so sánh giây
} else if (dstTs / 1000) != (it.Ts / 1000) {
```

**Hệ quả:** Nếu có bất kỳ drift millis nào (dù không đáng kể) → hash sẽ fail → drill_down không cần thiết → tốn thêm ~5s/window.

**Sau khi fix P2+P3:** nếu vẫn còn 8/8 windows false drift → sẽ cần fix P4.

**Fix đề xuất (cần Brain approve trước):**
```go
// Thay vì chia 1000 (so sánh giây):
} else if (dstTs / 1000) != (it.Ts / 1000) {

// Dùng tolerance 1 giây (vẫn dùng ms, rõ ràng hơn):
} else if abs(dstTs-it.Ts) > 1000 {
```

---

## Ước tính hiệu năng sau fix

| Giai đoạn | Hiện tại | Sau P2+P3 | Sau P2+P3+P4 |
|-----------|----------|-----------|--------------|
| MaxWindowTs (MongoDB) | 2.46s | **< 5ms** | < 5ms |
| HashWindow global (MongoDB) | 5.14s | **< 0.5s** | < 0.5s |
| ListIDTsInWindow × 8 | 42.4s | **< 0.5s** | 0 (skip nếu hash khớp) |
| **Tổng** | **~90s** | **< 8s** | **< 3s** |

---

## Action Items

| # | Priority | Action | Status |
|---|----------|--------|--------|
| A1 | ✅ Done | Confirm timezone P1 | CLOSED |
| A2 | ✅ Done | Confirm MongoDB thiếu index | CONFIRMED |
| **A3** | 🔴 **Approve ngay** | Tạo index MongoDB `lastUpdatedAt` (background) | ⏳ Chờ approve |
| A4 | 🟠 Sau | Review `diffIDTsSegmentA` P4 granularity | ⏳ Sau A3 |

---

## Workspace Files

| File | Link |
|------|------|
| Requirements | [01_requirements_audit.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/01_requirements_audit.md) |
| Progress Log | [05_progress.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/05_progress.md) |
| Analysis | [13_analysis_audit.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/13_analysis_audit.md) |
| Report v1.0 | [11_report_audit.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/11_report_audit.md) |
| **Report v2.0** | [report_ver_2.0.md](file:///Users/trainguyen/Documents/work/agent/memory/workspaces/ReconAuditPaymentBills20260720/report_ver_2.0.md) |
