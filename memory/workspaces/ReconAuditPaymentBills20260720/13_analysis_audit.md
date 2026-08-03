# 13 — Phân tích Kỹ thuật: Audit Recon payment_bills

> Tạo: 2026-07-20T10:17:00+07:00 | Session: Phiên audit đầu tiên

---

## 1. Mapping Trace Log → Code

| Trace Span | Code hàm | File | Thời gian |
|---|---|---|---|
| `cdc.recon.pick_scan_range` | `pickScanRangeWithLag()` | recon_tier_a.go:317 | 2.48s |
| `recon.source.max_window_ts` | `sourceAgent.MaxWindowTs()` | MongoDB | 2.46s ⚠️ |
| `pg.max_window_ts` | `destAgent.MaxWindowTs()` | recon_dest_query.go:489 | 2ms ✅ |
| `cdc.recon.verify_global_range` | Global HashWindow check | recon_tier_a.go:667 | 5.15s |
| `recon.source.hash_window` | `sourceAgent.HashWindow()` | MongoDB | 5.14s ⚠️ |
| `pg.hash_window` | `destAgent.HashWindow()` | recon_dest_hash.go:23 | 9.52ms ✅ |
| `cdc.recon.window_loop` | Window loop 8 windows | recon_tier_a.go:756 | 1.38min |
| `cdc.recon.drift_drill_down` × 8 | Per-window drill-down | recon_tier_a.go:776 | 5.3s × 8 |
| `recon.source.list_idts_in_window` | MongoDB ListIDTs | MongoDB | 5.3s × 8 ⚠️ |
| `pg.list_idts_in_window` | `destAgent.ListIDTsInWindow()` | recon_dest_query.go:372 | 5ms × 8 ✅ |

## 2. Tính toán Window Count

```
HotWindowLookback = 2h (default, recon_engine.go:112)
WindowSize = 15min (default, recon_engine.go:74)
Số windows = 2h / 15min = 8 windows
```

→ 8 windows đều drift → xuống drill_down hết = bất thường

## 3. Phân tích Root Cause P1: False Drift do TIMESTAMP Column

### Code path khi dstTS = "lastUpdatedAt" (TIMESTAMP):

**HashWindow** (`recon_dest_hash.go:97-138`):
```go
// Nhánh domain timestamp (TIMESTAMP column):
sql = SELECT _id::text, "lastUpdatedAt"
      FROM payment_bills
      WHERE "lastUpdatedAt" >= ? AND "lastUpdatedAt" < ?
// Sau đó:
it.Ts = parsePostgresTimestampWithLocation(ts, da.getDBLocation()).UnixMilli()
xorAcc ^= hashIDPlusTsMs(id, ts_in_millis)  // Hash theo MILLISECOND
```

**diffIDTsSegmentA** (`recon_tier_a.go:1059`):
```go
} else if (dstTs / 1000) != (it.Ts / 1000) {  // So sánh theo GIÂY
    mismatched = append(mismatched, it.ID)
}
```

**Kết quả**: XOR hash tính bằng millis → nếu timezone shift → hash sai.
Nhưng diff `/1000` → giây → vẫn khớp → `diff=0`.

Đây giải thích tại sao `hash không khớp` nhưng `ListIDTsInWindow` cho `diff=0`.

### Kiểm tra `parsePostgresTimestampWithLocation` (recon_dest_hash.go:126):

Với TIMESTAMP column (không có timezone info):
- pgx driver trả `time.Time` với `.Location() = UTC`
- `parsePostgresTimestampWithLocation(ts, location)` với location = DB timezone

Nếu `da.getDBLocation()` = `Asia/Ho_Chi_Minh` (UTC+7) và column là TIMESTAMP:
→ timestamp có thể bị xử lý sai → millis lệch → XOR hash khác nhau.

Nhưng nếu MongoDB source cũng trả về epoch millis UTC → hash MongoDB ≠ hash Postgres → false drift.

## 4. Phân tích Root Cause P2+P3: MongoDB Index Missing

**MaxWindowTs** (2.46s):
- Query: `MAX(lastUpdatedAt)` trên MongoDB collection `payment_bills`
- Nếu không có index `{ lastUpdatedAt: 1 }` → COLLSCAN (collection scan toàn bộ)
- Với production collection lớn → 2-5s là normal với COLLSCAN

**ListIDTsInWindow** (5.3s × 8 = 42.4s):
- Query: `find({ lastUpdatedAt: { $gte, $lt } })` mỗi window 15 phút
- Không có index `{ lastUpdatedAt: 1 }` → COLLSCAN mỗi lần
- 8 windows × 5.3s = 42.4s chỉ để query MongoDB

**Với index đúng:**
- MaxWindowTs: 2.46s → < 10ms (index lookup)
- ListIDTsInWindow: 5.3s → < 100ms mỗi window

## 5. Phân tích `effectiveLookback` vs manual range

Trace log cho thấy custom range 2h được truyền qua context:
```go
// recon_tier_a.go:644
if customStart, customEnd, ok := GetReconTimeRange(ctx); ok {
    lo = customStart
    hi = customEnd
}
```

→ User truyền range 2h qua API → override effectiveLookback.

## 6. Tính toán thời gian thực tế

```
pick_scan_range:         2.48s
verify_global_range:     5.15s  (global hash check → DRIFT)
window_loop (8 windows):
  - hash per window:     ~0 (chỉ lấy hash, fast vì đã skip trace)
  - drill_down × 8:      8 × 10s = 80s
    (MongoDB: 5.3s + Postgres: 5ms + overhead)

TỔNG ≈ 2.5 + 5.2 + 80 ≈ 87.7s ≈ 90s ✅ (khớp với báo cáo)
```

## 7. Câu hỏi mở cần xác nhận

1. `SHOW TIMEZONE` trên Shadow Postgres DB → xác nhận timezone drift
2. `db.payment_bills.getIndexes()` → xác nhận thiếu index MongoDB
3. `da.getDBLocation()` trả ra gì trong production config?
