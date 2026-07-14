# Gap Analysis — Rủi Ro Vận Hành Hệ Thống Recon & Heal

> Ngày phân tích: 2026-07-06
> Phiên bản phân tích: dựa trên code thực tế đối chiếu từng file

---

## Tổng Kết

| # | Rủi ro | Mức độ | Trạng thái code | Cần hành động? |
|---|---|---|---|---|
| 1 | Race Condition (2 luồng heal đồng thời) | 🔴 Cao | **KHÔNG có lock/guard** | ✅ CẦN VÁ |
| 2 | OOM tại Segment A (thiếu chunking) | 🟡 Trung bình | **Mongo cursor đã batch 200**, nhưng `$in` query chưa chunk | ⚠️ CẦN CẢI THIỆN |
| 3 | Partial Failure Idempotency | 🟡 Trung bình | **Chấp nhận được** — FetchAndWrite là Upsert, nhưng lãng phí I/O | ⚠️ NÊN CẢI THIỆN |
| 4 | Query Unhealed trả về report "sạch" | 🟢 ĐÃ VÁ | **Query đã có guard** `(missing_count > 0 OR stale_count > 0 OR orphan_count > 0)` | ❌ KHÔNG CẦN |
| 5 | Interactive Heal thiếu Safety Gate | 🔴 Cao | **KHÔNG có threshold check** | ✅ CẦN VÁ |

---

## Rủi Ro 1: Race Condition — 2 Luồng Heal Đồng Thời

### Phân tích code

**Hai handler tồn tại song song:**
- `HandleReconHeal()` → [recon_handler_run.go:206](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_handler_run.go#L206) — subscribe `cdc.cmd.recon-heal`
- `HandleExecuteHeal()` → [recon_execute_heal.go:29](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go#L29) — subscribe `cdc.cmd.execute-heal`

**Kiểm tra:** Không tìm thấy bất kỳ cơ chế nào:
- ❌ Không có `FOR UPDATE SKIP LOCKED` khi đọc report (`GetByID` tại [repo:26](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go#L26) — plain `SELECT ... First()`)
- ❌ Không có cột `status = 'healing'` để đánh dấu report đang xử lý
- ❌ Không có distributed lock (Redis/DB advisory lock) trên table scope
- ❌ `UpdateByID()` ([repo:35](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go#L35)) là plain `UPDATE` không có optimistic lock (version check)

### Hậu quả thực tế
- Segment A: 2 luồng cùng gọi `FetchAndWriteByIDs()` → double I/O trên MongoDB + Shadow DB (Upsert = không sai data nhưng gấp đôi tải)
- Segment B: 2 luồng cùng publish `cdc.cmd.transmute` → batch duplicate messages trên NATS → Master DB chịu double write
- `healed_count` bị ghi đè chéo: luồng A ghi `healed_at = T1`, luồng B ghi đè `healed_at = T2` — số liệu report sai

### Đề xuất vá
```go
// Trong executeHeal() hoặc HandleReconHeal(), trước khi xử lý:
tx := h.db.Begin()
var rpt modelrecon.ReconciliationReport
err := tx.Raw("SELECT * FROM cdc_reconciliation_report WHERE id = ? FOR UPDATE SKIP LOCKED", id).Scan(&rpt).Error
if err != nil || rpt.ID == 0 {
    tx.Rollback()
    continue // Luồng khác đang xử lý, skip
}
// ... xử lý heal ...
tx.Commit()
```

Hoặc thêm cột `heal_status ENUM('pending','healing','healed','failed')` và guard:
```sql
UPDATE cdc_reconciliation_report 
SET heal_status = 'healing' 
WHERE id = :id AND heal_status = 'pending'
-- affected rows = 0 → skip
```

---

## Rủi Ro 2: OOM tại Segment A (Thiếu Chunking)

### Phân tích code

**`executeHealSegA()`** ([recon_execute_heal.go:111](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go#L111)):
```go
// Line 130 — toàn bộ staleA.Mismatched đưa thẳng vào:
written, err := h.FetchAndWriteByIDs(ctx, entry, staleA.Mismatched)
// Line 140 — tương tự cho missingIDs:
written, err := h.FetchAndWriteByIDs(ctx, entry, missingIDs)
```

**`FetchAndWriteByIDs()`** ([recon_heal_fetch.go:44](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_fetch.go#L44)):
```go
// Line 87-88: MongoDB Find với $in query
cursor, err := coll.Find(findCtx, bson.M{"_id": bson.M{"$in": oids}},
    options.Find().SetBatchSize(200),
)
// Line 123-128: Cursor iteration có flush mỗi 200 docs ✅
if batchCount%200 == 0 {
    h.eventHandler.FlushBatchBuffer()
}
```

### Đánh giá thực tế

| Điểm | Trạng thái |
|---|---|
| MongoDB cursor batch | ✅ **SetBatchSize(200)** — cursor streaming, KHÔNG load toàn bộ vào RAM |
| Flush batch buffer | ✅ **Flush mỗi 200 docs** — bounded memory |
| `$in` query size | ⚠️ **Chưa chunk** — nếu 50,000 IDs → 1 câu `{_id: {$in: [50000 elements]}}` |
| MongoDB `$in` limit | MongoDB hỗ trợ $in lớn nhưng **query plan degradation** khi > 10,000 elements |
| Shadow write pipeline | ✅ Streaming qua `HandleRaw()` + `FlushBatchBuffer()` |

**So sánh với Segment B:** `executeHealSegB()` ([recon_execute_heal.go:164](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go#L164)) dùng `publishTransmuteChunked()` — chunk 200 IDs + delay 200ms ✅

### Kết luận
- **KHÔNG phải OOM risk trên Worker** (cursor streaming đã xử lý)
- **RỦI RO thực tế là MongoDB** — câu `$in` quá lớn gây query plan chậm, lock time cao
- Cần chunk `oids` array TRƯỚC khi gọi `coll.Find()`, hoặc chunk tại lớp caller (`executeHealSegA`)

### Đề xuất vá
```go
// Trong executeHealSegA, chia IDs thành chunk trước khi gọi FetchAndWriteByIDs:
const segAChunkSize = 1000
for start := 0; start < len(ids); start += segAChunkSize {
    end := min(start+segAChunkSize, len(ids))
    written, err := h.FetchAndWriteByIDs(ctx, entry, ids[start:end])
    // ...
}
```

---

## Rủi Ro 3: Partial Failure — Worker Crash Giữa Chừng

### Phân tích code

**`executeHeal()`** ([recon_execute_heal.go:67](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go#L67)):
```go
// Line 95-105: UPDATE report chỉ chạy SAU KHI xong cả 3 nhánh
now := time.Now().UTC()
_ = h.reportRepo.UpdateByID(ctx, rpt.ID, map[string]any{
    "healed_at": now,
    "status":    "healed",
    // ... counts ...
})
```

**Kịch bản crash:**
1. Worker heal `mismatched` xong (đã write Shadow DB)
2. Pod restart trước khi heal `missing_dest`
3. `healed_at` vẫn NULL → Admin thấy chưa heal → bấm heal lại
4. Worker lôi lại TOÀN BỘ `mismatched` + `missing_dest` → heal `mismatched` lại lần 2

### Đánh giá
- **Data integrity:** ✅ Không sai — `FetchAndWriteByIDs` dùng Upsert pipeline (`HandleRaw` → shadow apply)
- **Lãng phí I/O:** ⚠️ Có — re-fetch từ MongoDB + re-write Shadow cho các IDs đã heal xong
- **JSONB cleanup:** ❌ Worker KHÔNG xóa IDs đã heal khỏi `missing_ids`/`stale_ids` JSONB — full array vẫn nằm nguyên

### Đề xuất (ưu tiên thấp — data đúng, chỉ lãng phí I/O)
- Option A: Sau mỗi nhánh heal, UPDATE JSONB xóa IDs đã heal thành công
- Option B: Thêm cờ `heal_progress` JSON (`{mismatched: "done", missing_dest: "pending"}`)
- Option C: Chấp nhận idempotent waste — Upsert không gây hại, skip fix

---

## Rủi Ro 4: Query Unhealed Trả Về Report "Sạch"

### Phân tích code

**CMS Service repo** — [recon_read_repo_gorm.go:429-441](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go#L429):
```go
q := r.db.WithContext(ctx).
    Table("cdc_system.cdc_reconciliation_report").
    Where("(shadow_table = ? OR master_table = ?)", table, table).
    Where("healed_at IS NULL").
    Where("(missing_count > 0 OR stale_count > 0 OR orphan_count > 0)")  // ✅ ĐÃ CÓ GUARD!
```

**Worker repo** — [reconciliation_report_repo.go:84-92](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go#L84):
```go
Where("target_table = ? AND healed_at IS NULL AND (missing_count > 0 OR stale_count > 0 OR orphan_count > 0)", targetTable).
```

### Kết luận: 🟢 **ĐÃ VÁ** — Cả 2 repo (CMS + Worker) đều có điều kiện `(missing_count > 0 OR stale_count > 0 OR orphan_count > 0)`. Report "sạch" (status=ok, count=0) KHÔNG bị lọt vào danh sách unhealed.

---

## Rủi Ro 5: Interactive Heal Thiếu Safety Gate

### Phân tích code

**Background Heal** — [recon_heal_v4.go:48-77](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go#L48):
```go
func (h *ReconHandler) healThresholdBlocked(...) bool {
    // healAutoMaxIDs = 1000
    // healAutoMaxDriftPct = 5.0%
    // healSmallTableFloor = 100
    if mismatch <= healSmallTableFloor { return false }
    if mismatch <= healAutoMaxIDs && driftPct <= healAutoMaxDriftPct { return false }
    // → BLOCKED + alert
}
```
→ ✅ Background Heal có safety gate

**Interactive Heal** — [recon_execute_heal.go:67-108](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go#L67):
```go
func (h *ReconHandler) executeHeal(ctx context.Context, opts executeHealOpts) (int, error) {
    for _, id := range opts.ReportIDs {
        rpt, err := h.reportRepo.GetByID(ctx, id)
        // ...
        switch rpt.Segment {
        case "source_shadow", "":
            totalProcessed += h.executeHealSegA(ctx, rpt, entry, opts) // ← KHÔNG có threshold check
        case "shadow_master":
            totalProcessed += h.executeHealSegB(ctx, rpt, opts)       // ← KHÔNG có threshold check
        }
    }
}
```
→ ❌ **KHÔNG CÓ** `healThresholdBlocked()` call — bất kỳ lượng IDs nào cũng chạy thẳng

### Kịch bản nguy hiểm
1. Source DB rớt data → Recon quét ra 5 triệu records missing
2. Report ghi nhận `missing_count = 5,000,000` + `missing_ids = [5M elements]`
3. Admin không để ý, bấm "Thực thi chữa lành" trên UI
4. Worker chạy `FetchAndWriteByIDs(5M_ids)` → MongoDB Find 5M → Shadow Upsert 5M
5. Hệ thống CDC tắc nghẽn hoàn toàn

### Đề xuất vá
```go
// Trong executeHeal(), TRƯỚC vòng for:
totalIDs := 0
for _, id := range opts.ReportIDs {
    rpt, _ := h.reportRepo.GetByID(ctx, id)
    totalIDs += rpt.MissingCount + rpt.StaleCount + rpt.OrphanCount
}
if totalIDs > interactiveHealMaxIDs { // 50,000
    return 0, fmt.Errorf("execute-heal blocked: %d IDs exceeds safety threshold %d — use force_heal=true or split reports", totalIDs, interactiveHealMaxIDs)
}
```

Hoặc thêm `force_heal` flag trong `ExecuteHealCommand` + UI confirmation dialog.

---

## Tham Chiếu Files

| File | Repo | Mục đích |
|---|---|---|
| [recon_execute_heal.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_execute_heal.go) | Worker | Interactive Heal handler |
| [recon_heal_v4.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go) | Worker | Background Heal + safety gate |
| [recon_heal_fetch.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_fetch.go) | Worker | FetchAndWriteByIDs implementation |
| [reconciliation_report_repo.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/repository/recon/reconciliation_report_repo.go) | Worker | Report repo (GetByID, UpdateByID) |
| [recon_read_repo_gorm.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/infra/persistence/recon/recon_read_repo_gorm.go) | CMS Service | ListUnhealedReports query |
| [recon_handler_run.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_handler_run.go) | Worker | HandleReconHeal entry point |
