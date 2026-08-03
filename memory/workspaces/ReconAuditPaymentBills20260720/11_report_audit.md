# 11 — Report Phiên: Audit Recon payment_bills

> Phiên: 2026-07-20 | Người thực hiện: Agent:Claude-Sonnet-4.6-Thinking
> Trạng thái: AUDIT COMPLETE — chờ User confirm và approve action items

---

## Tóm tắt thay đổi

Phiên này là **phân tích thuần túy (read-only)** — không có thay đổi code.

### Files đã đọc

| File | Mục đích |
|------|----------|
| `recon_tier_a.go` (40KB) | Đọc toàn bộ RunHashWindowCheck, pickScanRangeWithLag, buildWindows, diffIDTsSegmentA |
| `recon_dest_hash.go` (6.6KB) | Đọc HashWindow — nhánh TIMESTAMP vs _source_ts |
| `recon_dest_query.go` (16.7KB) | Đọc ListIDTsInWindow, MaxWindowTs, BucketCounts |
| `recon_engine.go` (9.4KB) | Đọc ReconCoreConfig defaults, effectiveLookback |

### Files đã tạo (workspace docs)

| File | Kích thước | Nội dung |
|------|-----------|----------|
| `01_requirements_audit.md` | ~1KB | Phạm vi và DoD audit |
| `05_progress.md` | ~1KB | Audit log phiên |
| `08_tasks_audit.md` | ~1.5KB | Danh sách tasks |
| `13_analysis_audit.md` | ~4KB | Phân tích kỹ thuật chi tiết |
| `11_report_audit.md` | (file này) | Report tóm tắt |

---

## Findings tóm tắt

### P1 🔴 CRITICAL — False Drift do Hash Granularity Mismatch

**Vấn đề:** `HashWindow` tính XOR hash bằng **millisecond**, `diffIDTsSegmentA` so sánh bằng **giây** (chia /1000). Với cột `lastUpdatedAt` là `TIMESTAMP` (no timezone) + `parsePostgresTimestampWithLocation` → nếu có timezone mismatch nhỏ → millis lệch → XOR hash khác → tất cả 8 windows báo drift giả. Nhưng ListIDTsInWindow so sánh theo giây → vẫn khớp → `diff=0`.

**Chứng cứ:** 8/8 windows đều drift nhưng kết quả `1,952 → 1,952 (0)` — không có missing/stale nào.

**File liên quan:** `recon_dest_hash.go:126`, `recon_tier_a.go:1059`

---

### P2 🟡 HIGH — MongoDB thiếu index trên `lastUpdatedAt`

**Vấn đề:** `sourceAgent.MaxWindowTs` = **2.46s** (nên < 10ms). COLLSCAN toàn collection.

**Fix:** `db.payment_bills.createIndex({ "lastUpdatedAt": 1 }, { background: true })`

---

### P3 🟡 HIGH — MongoDB `ListIDTsInWindow` COLLSCAN mỗi window

**Vấn đề:** 8 windows × 5.3s = **42.4s** toàn bộ từ MongoDB query. COLLSCAN per window.

**Fix:** Compound index `{ lastUpdatedAt: 1, _id: 1 }` hoặc tái sử dụng index P2.

---

### P4 🟠 MEDIUM — Hash/Diff granularity không nhất quán

**Vấn đề:** Hash tính ms, diff so sánh giây. Bất nhất quán thiết kế.

**Fix đề xuất:**
```go
// Thay vì:
} else if (dstTs / 1000) != (it.Ts / 1000) {
// Dùng tolerance-based:
} else if abs(dstTs - it.Ts) > 1000 {  // tolerance 1s
```

---

## Ước tính hiệu năng sau fix P2+P3

| Giai đoạn | Hiện tại | Sau fix |
|-----------|----------|---------|
| MaxWindowTs (MongoDB) | 2.46s | < 10ms |
| HashWindow global (MongoDB) | 5.14s | ~1-2s |
| ListIDTsInWindow × 8 | 42.4s | < 1s |
| **Tổng** | **~90s** | **< 10s** |

---

## Action Items cần User approve

1. ✋ **Confirm:** Chạy `SHOW TIMEZONE` trên Postgres shadow DB
2. ✋ **Confirm:** Chạy `db.payment_bills.getIndexes()` trên MongoDB
3. 🔧 **Approve:** Tạo index MongoDB `{ lastUpdatedAt: 1 }` (background)
4. 🔧 **Approve:** Review và align granularity hash/diff trong `diffIDTsSegmentA`

---

## Câu hỏi anh cần trả lời

> **Q1:** Shadow Postgres DB đang chạy timezone gì? (`SHOW TIMEZONE`)
> **Q2:** MongoDB `payment_bills` hiện có index gì? (`db.payment_bills.getIndexes()`)
> **Q3:** Anh muốn fix P4 (granularity) hay chỉ focus vào P2+P3 (index) trước?
