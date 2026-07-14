# Walkthrough - Refactor Tier sang TypeRecon trong Centralized Data Service

Walkthrough này mô tả cụ thể các thay đổi và kết quả chạy test.

## 🔍 Changes Made

### 1. Core Service `recon_tier_a.go`
```diff
-func (rc *ReconCore) RunTier1(ctx context.Context, entry source.TableRegistry) *recon.ReconciliationReport {
+func (rc *ReconCore) RunSmokeCheck(ctx context.Context, entry source.TableRegistry) *recon.ReconciliationReport {
```
```diff
-func (rc *ReconCore) RunTier2(ctx context.Context, entry source.TableRegistry) *recon.ReconciliationReport {
+func (rc *ReconCore) RunHashWindowCheck(ctx context.Context, entry source.TableRegistry) *recon.ReconciliationReport {
```
```diff
-func (rc *ReconCore) RunTier3(ctx context.Context, entry source.TableRegistry) *recon.ReconciliationReport {
+func (rc *ReconCore) RunDeepCheck(ctx context.Context, entry source.TableRegistry) *recon.ReconciliationReport {
```

### 2. API Switch Case in `recon_check_handler.go`
```diff
 		switch payload.TypeRecon {
 		case "deep_check":
-			report = h.reconCore.RunTier3(ctx, *entry)
+			report = h.reconCore.RunDeepCheck(ctx, *entry)
 		case "smoke":
-			report = h.reconCore.RunTier1(ctx, *entry)
+			report = h.reconCore.RunSmokeCheck(ctx, *entry)
 		case "hash_window":
 			fallthrough
 		default:
 			tier2Ctx := context.WithValue(ctx, "manual_lookback", true)
 			if payload.Lookback == "cold" {
 				tier2Ctx = context.WithValue(tier2Ctx, "cold_lookback", true)
 			}
-			report = h.reconCore.RunTier2(tier2Ctx, *entry)
+			report = h.reconCore.RunHashWindowCheck(tier2Ctx, *entry)
 		}
```

---

## 🧪 Chạy Test
```bash
go test ./internal/handler/recon/... ./internal/service/recon/...
```
Kết quả:
```text
ok  	centralized-data-service/internal/handler/recon	1.567s
ok  	centralized-data-service/internal/service/recon	1.706s
```
