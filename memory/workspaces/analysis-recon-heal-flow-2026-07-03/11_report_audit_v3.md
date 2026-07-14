# Audit Report v3 — Đối Chiếu Code Thực Tế vs Plan

> Ngày audit: 2026-07-06T13:17:00+07:00  
> Phạm vi: Đối chiếu `08_tasks_gap_fix.md` + `10_gap_analysis.md` vs code thực tế trên 3 repo  
> Build verify: ✅ Go build OK (cả 2 repo) + tsc --noEmit OK (FE)

---

## I. Đối Chiếu Từng Task vs Code

### Fix 1: Race Condition Guard 🔴

**Plan (08_tasks 1.1):** Thêm `ClaimForHealing(ctx, id)` + `ReleaseHealClaim()` vào `reconciliation_report_repo.go`

| Yêu cầu | Code thực tế | File:Line | Khớp? |
|---|---|---|---|
| `ClaimForHealing(ctx, id)` trả `(report, bool, error)` | ✅ Có | `reconciliation_report_repo.go:98-115` | ✅ |
| Atomic `UPDATE ... SET status='healing' WHERE ... NOT IN ('healing','healed')` | ✅ Đúng SQL pattern | Line 102: `.Where("id = ? AND status NOT IN ('healing', 'healed')", id).Update("status", "healing")` | ✅ |
| Return `(nil, false, nil)` nếu bị claim | ✅ Có | Line 108 | ✅ |
| Fetch full row sau claim | ✅ Có | Line 111: `First(&row)` | ✅ |
| `ReleaseHealClaim(ctx, id, prevStatus)` | ✅ Có | Line 119-127 | ✅ |
| Default prevStatus = "drift" | ✅ Có | Line 121 | ✅ |

**Plan (08_tasks 1.2):** Sửa `executeHeal()` dùng ClaimForHealing

| Yêu cầu | Code thực tế | File:Line | Khớp? |
|---|---|---|---|
| Gọi ClaimForHealing trước mỗi report | ✅ Có | `recon_execute_heal.go:105` | ✅ |
| Claim fail → log warn + skip | ✅ Có | Line 112-114 | ✅ |
| Lỗi/unknown segment → ReleaseHealClaim | ✅ Có | Line 125 (registry not found), Line 134 (unknown segment) | ✅ |

**Pattern check:** `ClaimForHealing` dùng `r.db.WithContext(ctx).Model().Where().Update()` — cùng pattern với `UpdateByID` (line 36). ✅

---

### Fix 2: Chunk SegA IDs 🟡

**Plan (08_tasks 2.1):** `segAChunkSize = 1000`  
**Code:** `recon_execute_heal.go:21` → `segAChunkSize = 1000` ✅

**Plan (08_tasks 2.2):** Helper `fetchAndWriteChunked()`, sửa `executeHealSegA` dùng helper

| Yêu cầu | Code thực tế | File:Line | Khớp? |
|---|---|---|---|
| `fetchAndWriteChunked()` helper | ✅ Có | `recon_execute_heal.go:203-227` | ✅ |
| Chunk loop `start += segAChunkSize` | ✅ Có | Line 205 | ✅ |
| Error handling per chunk (log + continue) | ✅ Có | Line 211-217 | ✅ |
| Progress logging | ✅ Có | Line 219-224 | ✅ |
| `executeHealSegA` dùng helper thay call trực tiếp | ✅ Có | Line 174 (mismatched), Line 181 (missing_dest) | ✅ |

---

### Fix 3: Safety Gate Interactive Heal 🔴

**Plan (08_tasks 3.1-3.7):**

| # | Yêu cầu | Code thực tế | File:Line | Khớp? |
|---|---|---|---|---|
| 3.1 | `ForceHeal bool` trong `executeHealOpts` | ✅ | `recon_execute_heal.go:37` | ✅ |
| 3.2 | `interactiveHealMaxIDs = 50000` | ✅ | `recon_execute_heal.go:27` | ✅ |
| 3.3 | Tính tổng IDs → block nếu > threshold + !ForceHeal | ✅ | `recon_execute_heal.go:82-98` | ✅ |
| 3.4 | `ForceHeal bool` trong `ExecuteHealCommand` | ✅ | `recon_async.go:43` | ✅ |
| 3.5 | ForceHeal mapping trong `TriggerExecuteHeal` | ✅ | `reconciliation_handler_execute_heal.go:37,58` | ✅ |
| 3.6 | `force_heal` trong `ExecuteHealPayload` interface | ✅ | `useReconStatus.ts:92` | ✅ |
| 3.7 | Confirmation dialog khi threshold error | ✅ | `ExecuteHealModal.tsx:83-98` | ✅ |

**FE error extraction (fix A2 từ audit lần 1):**
- `axiosErr?.response?.data?.error` ✅ (Line 80-81)
- Fallback `err.message` nếu không có response ✅

---

### Verify (08_tasks 4.1-4.3)

| # | Yêu cầu | Kết quả thực tế |
|---|---|---|
| 4.1 | `go build ./internal/...` centralized-data-service | ✅ OK |
| 4.2 | `go build ./internal/... ./cmd/...` cdc-cms-service | ✅ OK |
| 4.3 | `tsc --noEmit` cdc-cms-web | ✅ OK |

---

## II. Đối Chiếu Architecture & Pattern

| Kiểm tra | Kết quả | Chi tiết |
|---|---|---|
| GORM pattern | ✅ | `ClaimForHealing` dùng `Model().Where().Update()` giống `UpdateByID` (line 36) |
| Naming Go | ✅ | PascalCase exports (`ClaimForHealing`), camelCase private (`fetchAndWriteChunked`) |
| Naming JSON | ✅ | snake_case: `force_heal`, `report_ids`, `heal_mismatched` |
| Naming TS | ✅ | Interface `ExecuteHealPayload` + snake_case fields (khớp API contract) |
| Constants scope | ✅ | File-level (`segAChunkSize`, `interactiveHealMaxIDs`), không global |
| Error handling | ✅ | Chunk fail → log + continue (không crash toàn bộ heal) |
| Logging | ✅ | Dùng `zap` structured logging (giống toàn bộ codebase) |
| Observability | ✅ | `observability.ExtractNATSHeader` + `ChildSpan` (line 44-46) |
| No workaround | ✅ | Giải pháp core: atomic UPDATE status machine |

---

## III. Files Đã Thay Đổi

| File | Repo | Dòng thay đổi | Mô tả |
|---|---|---|---|
| `reconciliation_report_repo.go` | centralized-data-service | +37 lines (93-129) | `ClaimForHealing` + `ReleaseHealClaim` |
| `recon_execute_heal.go` | centralized-data-service | ~+200 lines (17-327) | Constants, safety gate, claim guard, chunking, publishTransmuteChunked |
| `recon_async.go` | cdc-cms-service | +1 line (43) | `ForceHeal bool` field |
| `reconciliation_handler_execute_heal.go` | cdc-cms-service | +2 lines (37, 58) | ForceHeal parse + mapping |
| `useReconStatus.ts` | cdc-cms-web | +2 lines (92, 296) | `force_heal` field trong interface + mutation |
| `ExecuteHealModal.tsx` | cdc-cms-web | +20 lines (78-99) | Error extraction fix + threshold confirmation dialog |

---

## IV. Sai Sót & Thiếu Sót

### ✅ Không phát hiện sai sót code

Tất cả 7 sub-tasks trong plan (1.1, 1.2, 2.1, 2.2, 3.1-3.7) đều implement đúng đặc tả. Build pass, pattern khớp.

### ⚠️ Thiếu sót đã biết (documented, chưa trong scope fix hiện tại)

| # | Thiếu sót | Nguồn | Lý do chưa fix |
|---|---|---|---|
| 1 | Background Heal (`recon_heal_v4.go`) chưa dùng `ClaimForHealing` | Audit v1 #1 | Pattern khác — BG Heal tự RunTier2 + heal, cần task riêng |
| 2 | Background Heal SegA chưa dùng `fetchAndWriteChunked` | Audit v1 #5 | Gom cùng task riêng cho `recon_heal_v4.go` |
| 3 | `prune_missing_src` là placeholder (log only, chưa `UPDATE _deleted=true`) | Code line 189, 284 | TODO đã ghi rõ trong code — cần thiết kế riêng |
| 4 | Không có TTL-based revert cho reports stuck ở `healing` status | 13_analysis note | Cần cơ chế cleanup riêng (cronjob hoặc TTL) |
| 5 | `10_gap_analysis.md` chưa cập nhật trạng thái "sau fix" | Audit v1 #4 | Hygiene task |

---

## V. Tổng Kết

- **Plan hoàn thành:** 7/7 sub-tasks ✅
- **Code đúng đặc tả:** ✅ 
- **Architecture/Pattern:** ✅ Không phá vỡ
- **Build:** ✅ Cả 3 repo
- **Thiếu sót:** 5 items đã documented — nằm ngoài scope fix hiện tại, cần task riêng
