# Bug Fix Report — feat-recon-hardening (Code Review)

> **Date**: 2026-06-25T09:38 +07:00
> **Reviewer**: User (manual code review)
> **Fixer**: Brain/Antigravity
> **Service**: `centralized-data-service`
> **Scope**: 3 bugs phát hiện sau khi deploy hardening plan v3

---

## Kết quả Verification

| Check | Result |
|-------|--------|
| `go build ./internal/... ./pkgs/... ./cmd/...` | ✅ PASS |
| `go test -race ./internal/service/recon/... -timeout 120s` | ✅ PASS (2.064s) |
| Race detector | ✅ CLEAN — no data races |

---

## Bug 1 — Context Leak + Đứt OTel TraceID (🔴 Critical)

### Mô tả
Hai lỗi context xảy ra cùng lúc:
1. `CheckAll` (`recon_engine_run.go:221`) wrap `RunTier1` bằng `tableCtx` 45 giây.
2. Trong `RunTier1` (`recon_tier_a.go:508`), `drillCtx` được tạo từ `context.Background()` thay vì kế thừa `ctx`.

**Hậu quả**:
- OTel `TraceID` bị đứt tại `drillCtx` → `BucketCounts` không xuất hiện trên SigNoz/Jaeger.
- Khi `tableCtx` 45s hết hạn, goroutine `RunTier1` vẫn tiếp tục ngầm (do `drillCtx` detached) → goroutine leak.
- `finishRun(ctx, ...)` và `alertOnReport(ctx, ...)` dùng `ctx` gốc đã hết hạn → ném `context deadline exceeded` → không ghi được status vào DB.

### Files thay đổi

**`recon_engine_run.go` (L221-223)**
```diff
- tableCtx, cancelTable := context.WithTimeout(ctx, 45*time.Second)
- defer cancelTable()
- report := rc.RunTier1(tableCtx, e)
+ // Timeout được quản lý bên trong RunTier1:
+ //   - fastCtx (10s) cho EstimatedCount/EstimatedCountRows
+ //   - drillCtx (8m, kế thừa ctx) cho BucketCounts aggregate
+ report := rc.RunTier1(ctx, e)
```

**`recon_tier_a.go` (L444-465, L506-509)**
```diff
+ // fastCtx kế thừa OTel TraceID từ ctx gốc
+ fastCtx, cancelFast := context.WithTimeout(ctx, 10*time.Second)
+ defer cancelFast()
+ srcEst, errE := rc.sourceAgent.EstimatedCount(fastCtx, ...)
+ dstTotal, errD := rc.destAgent.EstimatedCountRows(fastCtx, ...)

- drillCtx, cancelDrill := context.WithTimeout(context.Background(), 8*time.Minute)
+ // drillCtx kế thừa ctx gốc — giữ TraceID, hỗ trợ cancel từ ngoài
+ drillCtx, cancelDrill := context.WithTimeout(ctx, 8*time.Minute)
```

---

## Bug 2 — `pg_class.reltuples = -1` trên PostgreSQL 14+ (🔴 Critical @ Scale)

### Mô tả
`recon_dest_query.go:60` dùng `COALESCE(c.reltuples::bigint, 0)`.

Từ PostgreSQL 14+, bảng mới tạo chưa qua `VACUUM`/`ANALYZE` có `reltuples = -1` (không phải `NULL`, không phải `0`). `COALESCE` chỉ xử lý `NULL` → trả về `-1`.

Caller ở `recon_tier_a.go:456` check `dstTotal == 0` không bắt được `-1` → lấy `-1` so sánh với `srcEst` (số dương) → **False Drift chắc chắn** cho mọi bảng mới tạo.

### Files thay đổi

**`recon_dest_query.go` (L60)**
```diff
- SELECT COALESCE(c.reltuples::bigint, 0)
+ SELECT GREATEST(COALESCE(c.reltuples::bigint, 0), 0)
```

**`recon_tier_a.go` (L456)**
```diff
- if errD != nil || dstTotal == 0 {
+ // Bug-2 fix: dstTotal <= 0 bắt cả -1 (PG14+ chưa ANALYZE)
+ if errD != nil || dstTotal <= 0 {
```

---

## Bug 3 — Hot/Cold Lookback không được wire vào runtime (🟡 High)

### Mô tả
`recon_engine.go` khai báo `HotWindowLookback` và `RunMode` trong `ReconCoreConfig` và `applyDefaults()`, nhưng `pickScanRangeWithLag` (`recon_tier_a.go:233`) vẫn hardcode:

```go
lower := upper.Add(-rc.cfg.WindowLookback)  // Luôn dùng 7d, bỏ qua HotWindowLookback
```

**Hậu quả**: Cron job hot recon chạy 15 phút/lần nhưng vẫn quét 7 ngày data → 200 tables × 50M records × 168 buckets = 70 tỷ row-scans/cycle thay vì 200 tables × 2h data.

### Files thay đổi

**`recon_engine.go`** — Thêm method `effectiveLookback()`:
```go
// effectiveLookback returns the actual scan window based on RunMode.
// RunMode="" or "hot" → HotWindowLookback (2h)
// RunMode="cold"      → WindowLookback    (7d)
func (rc *ReconCore) effectiveLookback() time.Duration {
    if rc.cfg.RunMode == "cold" && rc.cfg.WindowLookback > 0 {
        return rc.cfg.WindowLookback
    }
    if rc.cfg.HotWindowLookback > 0 {
        return rc.cfg.HotWindowLookback
    }
    return rc.cfg.WindowLookback
}
```

**`recon_tier_a.go` (L233)**
```diff
- lower := upper.Add(-rc.cfg.WindowLookback)
+ lower := upper.Add(-rc.effectiveLookback())
```

---

## Summary — Files đã thay đổi

| File | Dòng | Bug |
|------|------|-----|
| `recon_engine_run.go` | L221-225 | Bug 1 — xóa tableCtx 45s |
| `recon_tier_a.go` | L444-465 | Bug 1 — thêm fastCtx |
| `recon_tier_a.go` | L506-509 | Bug 1 — drillCtx kế thừa ctx |
| `recon_tier_a.go` | L456 | Bug 2 — `<= 0` thay `== 0` |
| `recon_dest_query.go` | L60 | Bug 2 — `GREATEST(COALESCE(...),0)` |
| `recon_engine.go` | L177-194 | Bug 3 — thêm `effectiveLookback()` |
| `recon_tier_a.go` | L233 | Bug 3 — dùng `effectiveLookback()` |
