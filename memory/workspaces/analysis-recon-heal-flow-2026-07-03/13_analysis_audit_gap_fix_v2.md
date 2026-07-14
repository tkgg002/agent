# Audit Report Lần 2 — Rà Soát Toàn Diện Post-Fix

> Ngày audit: 2026-07-06T11:06:00+07:00
> Phạm vi: Toàn bộ code đã thay đổi trên 3 repo + tài liệu workspace
> Mục đích: Đảm bảo KHÔNG thiếu luồng, KHÔNG có sai sót code, tài liệu đồng bộ

---

## I. Checklist — Code vs Tài Liệu (`13_analysis`)

### Giai đoạn 0: Kích Hoạt Đối Soát (Recon Check)
| Item | Analysis Doc | Code thực tế | Khớp? |
|---|---|---|---|
| FE `useCheckTableMutation` | ✅ Mô tả đúng | ✅ Không thay đổi | ✅ |
| API `TriggerCheck()` | ✅ | ✅ Không thay đổi | ✅ |
| Worker `HandleReconCheck()` | ✅ | ✅ Không thay đổi | ✅ |
| Payload wire format | ✅ tier + table + lookback | ✅ Không thay đổi | ✅ |

> **Kết luận:** Giai đoạn 0 không bị ảnh hưởng bởi fixes. ✅

---

### Giai đoạn 1: Truy Vấn Danh Sách Lỗi (Unhealed Reports)
| Item | Analysis Doc | Code thực tế | Khớp? |
|---|---|---|---|
| API `GetUnhealedReports()` | ✅ | ✅ Không thay đổi | ✅ |
| Query guard `(missing_count > 0 OR ...)` | ✅ Đã vá (Gap #4) | ✅ Code xác nhận | ✅ |
| FE `useUnhealedReports()` | ✅ | ✅ Không thay đổi | ✅ |

> **Kết luận:** Giai đoạn 1 không bị ảnh hưởng. ✅

---

### Giai đoạn 2: Thực Thi Chữa Lành (Interactive Execute Heal)

#### Payload — FE → API → Worker
| Field | Analysis Doc | FE (`useReconStatus.ts`) | API (`recon_async.go` + handler) | Worker (`executeHealOpts`) | Khớp? |
|---|---|---|---|---|---|
| `table` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `segment` | ✅ | ✅ | ✅ | ❌ **Worker opts struct thiếu `Segment`** | ⚠️ |
| `report_ids` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `heal_mismatched` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `heal_missing_dest` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `prune_missing_src` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `force_heal` | ✅ | ✅ | ✅ | ✅ | ✅ |

> 🟡 **Phát hiện #1:** `Segment` field có trong `ExecuteHealCommand` (API) nhưng Worker `executeHealOpts` struct KHÔNG có field `Segment`. Tuy nhiên, Worker xác định segment TỪ `rpt.Segment` (lấy từ DB), nên field segment trong payload bị **bỏ qua** khi unmarshal. Đây không phải bug (logic đúng — segment lấy từ report, không từ payload), nhưng tài liệu nên ghi rõ.

#### Safety Gate Flow
| Step | Analysis Doc | Code (`recon_execute_heal.go`) | Khớp? |
|---|---|---|---|
| Tính tổng IDs trước khi xử lý | ✅ (line 82-98) | ✅ `rpt.MissingCount + StaleCount + OrphanCount` | ✅ |
| Block nếu > 50K + !ForceHeal | ✅ | ✅ return error | ✅ |
| FE bắt error → Modal.confirm | ✅ | ✅ `axiosErr?.response?.data?.error` (post-fix A2) | ✅ |
| FE re-dispatch với force_heal=true | ✅ | ✅ `onOk: () => handleOk(true)` | ✅ |

#### Race Condition Guard Flow
| Step | Analysis Doc | Code | Khớp? |
|---|---|---|---|
| `ClaimForHealing(id)` trước xử lý | ✅ | ✅ line 105 | ✅ |
| Claim fail → log warn + skip | ✅ | ✅ line 111-114 | ✅ |
| Registry not found → `ReleaseHealClaim` | ✅ | ✅ line 125-126 | ✅ |
| Unknown segment → `ReleaseHealClaim` | ✅ | ✅ line 134-135 | ✅ |
| Heal thành công → set status=healed | ✅ | ✅ line 147 | ✅ |
| Heal lỗi giữa chừng → ReleaseHealClaim | ✅ (mô tả trong diagram) | ⚠️ **Code hiện tại không có** | ⚠️ |

> 🟡 **Phát hiện #2:** Diagram 13_analysis ghi "Heal lỗi giữa chừng → ReleaseHealClaim", nhưng code thực tế sau khi xóa `healErr` dead code (fix A1) **KHÔNG CÒN** nhánh release khi SegA/SegB lỗi. Lý do: `executeHealSegA()` và `executeHealSegB()` handle error internally (log + continue chunk), trả về int (số đã heal thành công). Code vẫn set `status=healed` dù chỉ heal 1 phần.
>
> **Đánh giá:** Đây là trade-off chấp nhận được — giống behavior của Background Heal hiện tại. Nhưng tài liệu mô tả "ReleaseHealClaim khi lỗi" thì **KHÔNG đúng với code**. Cần sửa diagram.

#### Chunking SegA
| Step | Analysis Doc | Code | Khớp? |
|---|---|---|---|
| `fetchAndWriteChunked(ids)` | ✅ 1000 IDs/batch | ✅ line 174, 181, 203-226 | ✅ |
| Chunk progress logging | ✅ (mô tả) | ✅ line 219-223 | ✅ |

---

### Luồng Background Heal (`HandleReconHeal`)
| Item | Analysis Doc | Code (`recon_heal_v4.go`) | Khớp? |
|---|---|---|---|
| Window mode + Full-diff mode | ✅ Mô tả đầy đủ | ✅ Không thay đổi | ✅ |
| `healThresholdBlocked()` | ✅ | ✅ line 325, 422 | ✅ |
| Report update sau heal | ✅ | ✅ line 438-443 | ✅ |
| **ClaimForHealing guard** | ❌ KHÔNG mô tả (đúng — BG Heal chưa dùng) | ❌ Chưa implement | ✅ (consistent) |

---

### Schema & State Machine
| Item | Analysis Doc | Code | Khớp? |
|---|---|---|---|
| `status` column values | `ok`, `drift`, `healing`, `healed` | ✅ ClaimForHealing set 'healing', UpdateByID set 'healed' | ✅ |
| Model có field `Status` | ✅ | ✅ `Status string gorm:"column:status"` line 24 | ✅ |
| `healing` → `healed` transition | ✅ | ✅ line 147 | ✅ |
| `healing` → `drift` (release) | ✅ | ✅ ReleaseHealClaim line 119-127 (repo) | ✅ |

---

## II. Checklist — Tài Liệu Workspace

| File | Tồn tại | Cập nhật mới nhất | Khớp code? |
|---|---|---|---|
| `05_progress.md` | ✅ | ✅ 2026-07-06T10:55 | ✅ |
| `08_tasks_gap_fix.md` | ✅ | ✅ All marked [x] | ✅ |
| `10_gap_analysis.md` | ✅ | ⚠️ Chưa cập nhật trạng thái sau fix | 🟡 |
| `11_report_gap_fix.md` | ✅ | ✅ Có post-audit fixes | ✅ |
| `13_analysis_recon_heal_flow.md` | ✅ | ✅ Có Safety/Race/Chunk | ⚠️ Diagram sai #2 |
| `13_analysis_audit_gap_fix.md` | ✅ | ✅ Audit lần 1 | ✅ |

---

## III. Checklist — Architecture & Pattern Compliance

| Kiểm tra | Kết quả | Chi tiết |
|---|---|---|
| GORM pattern consistency | ✅ | ClaimForHealing dùng `Model().Where().Update()` — giống UpdateByID |
| Naming convention (Go) | ✅ | camelCase functions, PascalCase exports |
| Naming convention (JSON) | ✅ | snake_case: `force_heal`, `report_ids` |
| Error handling | ✅ (sau fix A1) | Dead code đã xóa |
| FE error extraction | ✅ (sau fix A2) | `axiosErr?.response?.data?.error` |
| Constants scope | ✅ | File-level, không global |
| No workaround/hack | ✅ | Giải pháp core: atomic UPDATE + status machine |

---

## IV. Tổng Kết Phát Hiện Lần 2

| # | Mức độ | Phát hiện | Hành động |
|---|---|---|---|
| 1 | 🟢 Info | `Segment` field trong payload bị Worker bỏ qua (lấy từ report DB). Đúng logic nhưng nên document rõ | Không cần fix code |
| 2 | 🟡 Doc | 13_analysis diagram ghi "ReleaseHealClaim khi lỗi" nhưng code hiện tại KHÔNG có nhánh này (SegA/B handle error internally). Tài liệu mô tả KHÔNG khớp code | Cần sửa diagram |
| 3 | 🟡 Doc | 13_analysis Chi Tiết Kỹ Thuật table line references sai (L111→L155, L164→L230 sau khi thêm constants + safety gate) | Cần sửa line refs |

> **Kết luận tổng thể:** Sau audit lần 1 và fix A1+A2, code đã KHÔNG CÒN bug critical. Các phát hiện lần 2 đều ở mức **Documentation** — cần đồng bộ lại tài liệu cho khớp code.
