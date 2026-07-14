# Audit Report — Quá Trình Fix Gap Analysis

> Ngày audit: 2026-07-06T10:54:00+07:00
> Đối chiếu: `10_gap_analysis.md` + `08_tasks_gap_fix.md` + `13_analysis_recon_heal_flow.md` vs Code thực tế

---

## I. Tổng Quan — Fix đã Implement vs Plan

| # | Gap Analysis Item | Trạng thái Plan | Code thực tế | Đúng Plan? |
|---|---|---|---|---|
| 1 | Race Condition Guard | ✅ ClaimForHealing + ReleaseHealClaim | ✅ Implement ở Interactive Heal | ⚠️ **THIẾU** — Background Heal CHƯA dùng |
| 2 | Chunk SegA IDs | ✅ fetchAndWriteChunked 1000/batch | ✅ Implement đúng | ✅ Đúng |
| 3 | Safety Gate 50K | ✅ interactiveHealMaxIDs + ForceHeal | ✅ Implement đúng | ⚠️ **BUG FE** — Error parsing sai |
| 4 | Query Unhealed (đã vá) | ❌ Không cần fix | ❌ Không fix | ✅ Đúng |
| 5 | Partial Failure | ❌ Chấp nhận idempotent | ❌ Không fix | ✅ Đúng (ưu tiên thấp) |

---

## II. Phát Hiện Sai Sót / Thiếu Sót

### 🔴 Sai sót #1: Background Heal KHÔNG dùng ClaimForHealing

**Mô tả:** `ClaimForHealing()` chỉ được gọi tại `executeHeal()` (Interactive Heal, line 105 trong `recon_execute_heal.go`). Luồng Background Heal (`healSegmentA` + `healSegmentB` trong `recon_heal_v4.go`) vẫn thao tác trực tiếp `GetLatestByTable()` + `UpdateByID()` **KHÔNG QUA** claim.

**Hậu quả:** Nếu Background Heal (cronjob) chạy cùng lúc Interactive Heal trên cùng report → race condition VẪN XẢY RA.

**Root Cause phân tích:**
- Background Heal (`healSegmentA`/`healSegmentB`) hoạt động theo mô hình khác: nó TỰ CHẠY đối soát (`RunTier2`/`RunSegmentBFor`) rồi heal report MỚI (chưa có trong DB), hoặc dùng report đã có.
- Tại line 438 (`recon_heal_v4.go`): `h.reportRepo.UpdateByID(ctx, report.ID, ...)` — set `healed_at`, `status=healed` KHÔNG check nếu report đang bị Interactive Heal claim.
- Tương tự tại line 207 (healSegmentB).

**Khuyến nghị sửa:** Thêm `ClaimForHealing()` guard TRƯỚC khi `healSegmentA`/`healSegmentB` gọi `UpdateByID()` set healed. Hoặc ít nhất thêm `WHERE status != 'healing'` guard tại `UpdateByID()` cho background heal.

---

### 🔴 Sai sót #2: `healErr` trong executeHeal() LUÔN = nil (Dead Code)

**Mô tả:** Tại `recon_execute_heal.go` line 120:
```go
var healErr error          // <-- khai báo
switch rpt.Segment {
case "source_shadow", "":
    totalProcessed += h.executeHealSegA(...)  // trả int, KHÔNG gán healErr
case "shadow_master":
    totalProcessed += h.executeHealSegB(...)  // trả int, KHÔNG gán healErr
}

if healErr != nil {        // <-- LUÔN false vì không ai gán healErr
    _ = h.reportRepo.ReleaseHealClaim(ctx, id, prevStatus)  // <-- DEAD CODE
    continue
}
```

**Hậu quả:** Nếu heal fail (FetchAndWriteByIDs lỗi mạng, MongoDB timeout), `healErr` vẫn nil → ReleaseHealClaim **KHÔNG** được gọi → report bị stuck ở status `healing` vĩnh viễn.

**Khuyến nghị:** Sửa `executeHealSegA()` và `executeHealSegB()` trả `(int, error)` thay vì chỉ `int`, hoặc gán `healErr` từ kết quả.

---

### 🟡 Sai sót #3: FE catch AxiosError message sai

**Mô tả:** Tại `ExecuteHealModal.tsx` line 79:
```tsx
const errMsg = err instanceof Error ? err.message : String(err);
if (errMsg.includes('safety threshold') || errMsg.includes('execute-heal blocked')) { ... }
```

**Vấn đề:** Axios error message là generic: `"Request failed with status code 500"`. Text "execute-heal blocked" nằm trong `err.response.data.error`, KHÔNG nằm trong `err.message`.

**Chuỗi error propagation thực tế:**
```
Worker → NATS reply: {error: "execute-heal blocked: 60000 IDs exceeds safety threshold 50000"}
API Gateway → Fiber 500: {"error": "execute-heal blocked: ..."}
FE Axios → catch(err) → err.message = "Request failed with status code 500"
                        → err.response.data.error = "execute-heal blocked: ..."
```

**Khuyến nghị:** Sửa FE để extract error từ `err.response?.data?.error`:
```tsx
const axiosErr = err as any;
const errMsg = axiosErr?.response?.data?.error || (err instanceof Error ? err.message : String(err));
```

---

### 🟡 Thiếu sót #4: `10_gap_analysis.md` chưa cập nhật trạng thái SAU fix

**Mô tả:** File `10_gap_analysis.md` vẫn ghi trạng thái gốc `"KHÔNG có lock/guard"`, `"KHÔNG có threshold check"` — chưa phản ánh fix đã implement. 

**Khuyến nghị:** Append "Trạng thái sau fix" vào mỗi rủi ro trong 10_gap_analysis.md.

---

### 🟡 Thiếu sót #5: Background Heal SegA thiếu chunking

**Mô tả:** `healSegmentA()` tại line 429 (`recon_heal_v4.go`):
```go
written, dispatchErr := h.FetchAndWriteByIDs(ctx, entry, healIDs)
```
Gọi trực tiếp `FetchAndWriteByIDs` **KHÔNG** qua `fetchAndWriteChunked()`. Chunking chỉ apply cho Interactive Heal SegA.

**Hậu quả:** Background Heal SegA vẫn gửi toàn bộ array `healIDs` làm 1 câu MongoDB `$in` query — cùng rủi ro OOM/query degradation đã mô tả trong Gap #2.

**Khuyến nghị:** Sửa `healSegmentA()` dùng `fetchAndWriteChunked()` thay vì call trực tiếp.

---

## III. Đối Chiếu Architecture & Pattern

| Kiểm tra | Kết quả |
|---|---|
| Pattern nhất quán với hệ thống | ✅ ClaimForHealing dùng GORM style giống UpdateByID, GetByID |
| Naming convention | ✅ camelCase cho Go functions, snake_case cho JSON/DB |
| Error handling | 🔴 `healErr` dead code — vi phạm "no silent failure" |
| FE error extraction | 🟡 Không match pattern Axios → cần `err.response.data.error` |
| Constants placement | ✅ Đúng file scope, không global |
| Tài liệu đồng bộ | 🟡 13_analysis cập nhật, nhưng 10_gap_analysis chưa |

---

## IV. Danh Sách Hành Động Cần Làm

| # | Hành động | Mức độ | File |
|---|---|---|---|
| A1 | Fix `healErr` dead code → sửa `executeHealSegA/B` trả `(int, error)` | 🔴 Critical | `recon_execute_heal.go` |
| A2 | Fix FE error extraction: dùng `err.response?.data?.error` | 🔴 Critical | `ExecuteHealModal.tsx` |
| A3 | Thêm ClaimForHealing guard cho Background Heal | 🟡 Important | `recon_heal_v4.go` |
| A4 | Thêm chunking cho Background Heal SegA | 🟡 Important | `recon_heal_v4.go` |
| A5 | Cập nhật 10_gap_analysis.md trạng thái sau fix | 🟢 Hygiene | `10_gap_analysis.md` |
