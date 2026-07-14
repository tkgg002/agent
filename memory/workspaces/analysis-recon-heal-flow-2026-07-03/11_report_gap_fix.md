# Report — Fix Rủi Ro Vận Hành Recon & Heal Pipeline

> Ngày: 2026-07-06 | Branch: `recon-heal` trên cả 3 repo

---

## Tổng Quan Thay Đổi

| Repo | Files thay đổi | Dòng thêm | Dòng xóa |
|---|---|---|---|
| **centralized-data-service** (Worker) | 2 files | ~95 | ~15 |
| **cdc-cms-service** (API Gateway) | 2 files | ~3 | 0 |
| **cdc-cms-web** (FE) | 2 files | ~25 | ~5 |
| **Tổng** | **6 files** | **~123** | **~20** |

---

## Chi Tiết Theo Repo

### 1. centralized-data-service (Worker)

#### [reconciliation_report_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go)
- **+** `ClaimForHealing(ctx, id)` — atomic UPDATE status='healing' WHERE NOT IN ('healing','healed')
- **+** `ReleaseHealClaim(ctx, id, prevStatus)` — revert status khi heal fail

#### [recon_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go)
- **+** Constants: `segAChunkSize = 1000`, `interactiveHealMaxIDs = 50000`
- **+** `ForceHeal bool` field trong `executeHealOpts`
- **Δ** `executeHeal()`: Thêm Safety Gate (tính tổng IDs → block nếu > 50K + !ForceHeal) + Race Condition Guard (ClaimForHealing trước xử lý, ReleaseHealClaim khi lỗi)
- **Δ** `executeHealSegA()`: Dùng `fetchAndWriteChunked()` thay vì call trực tiếp `FetchAndWriteByIDs()`
- **+** `fetchAndWriteChunked()` — helper chunk IDs thành batches 1000 trước khi gọi MongoDB

### 2. cdc-cms-service (API Gateway)

#### [recon_async.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_async.go)
- **+** `ForceHeal bool` trong `ExecuteHealCommand` struct

#### [reconciliation_handler_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_execute_heal.go)
- **+** `ForceHeal` trong request parsing + command dispatch mapping

### 3. cdc-cms-web (Frontend)

#### [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
- **+** `force_heal?: boolean` trong `ExecuteHealPayload` interface
- **Δ** `useExecuteHealMutation()` — destructure + truyền `force_heal` vào body

#### [ExecuteHealModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ExecuteHealModal.tsx)
- **Δ** `handleOk()` — bắt error threshold → hiển thị `Modal.confirm()` cho user xác nhận force heal
- **+** Truyền `force_heal: forceHeal` vào mutation call

---

## Verification

| Check | Result |
|---|---|
| `go build ./internal/...` (Worker) | ✅ PASS |
| `go build ./internal/... ./cmd/...` (API) | ✅ PASS |
| `tsc --noEmit` (FE) | ✅ PASS |

---

## Post-Audit Fixes (2026-07-06T10:55)

> Phát hiện qua Audit tự rà soát code vs plan

### Fix A1: Xóa `healErr` dead code
**File:** `recon_execute_heal.go` (Worker)
- **-** `var healErr error` (khai báo nhưng KHÔNG BAO GIỜ gán)
- **-** `if healErr != nil { ReleaseHealClaim(...) }` (dead code branch)
- **Lý do:** `executeHealSegA()` và `executeHealSegB()` trả `int`, không trả error → `healErr` luôn nil → ReleaseHealClaim dead code

### Fix A2: Sửa FE error extraction
**File:** `ExecuteHealModal.tsx` (FE)
- **Δ** `const errMsg = err.message` → `axiosErr?.response?.data?.error || err.message`
- **Lý do:** Axios error `.message` là generic "Request failed with status code 500", text "execute-heal blocked" nằm trong `.response.data.error`

### Remaining (chưa fix, cần task riêng)
| # | Mô tả | File | Lý do chưa fix |
|---|---|---|---|
| A3 | Background Heal thiếu ClaimForHealing | `recon_heal_v4.go` | Luồng BG Heal khác pattern (tự RunTier2 trước), cần thiết kế riêng |
| A4 | Background Heal SegA thiếu chunking | `recon_heal_v4.go` | Cùng file, nên gom fix A3+A4 |
| A5 | 10_gap_analysis chưa cập nhật trạng thái | `10_gap_analysis.md` | Hygiene, ưu tiên thấp |
