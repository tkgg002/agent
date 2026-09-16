# 13 Analysis: Audit Gốc rễ Sự cố Số lượng Heal bị "+1" (6/5 và 2478/2477)

**Mã Workspace:** `FixLogTransmuteAndReconTraceId20260909`  
**Thời gian:** 2026-09-09 11:00:00  
**Tác vụ:** Điều tra nguyên nhân số lượng bản ghi đã chữa lành (Healed Count) luôn lớn hơn số lượng lỗi phát hiện ban đầu đúng 1 đơn vị.

---

## 1. HIỆN TƯỢNG (OBSERVED PHENOMENON)

Trên giao diện CMS (`ExecuteHealModal.tsx` / Tab "Phiên đã xử lý"), kết quả phiên Heal cho pipeline `Source → Shadow (reconA)` thời gian `10:18 02/09/2026 - 10:18 09/09/2026`:
- **Lệch dữ liệu (Mismatched):** `6/5 (126ms)` lúc `2026-09-09 10:45:54`
  - `stale_count` = 5 (mẫu số)
  - `healed_mismatched_count` = 6 (tử số) -> Bị lệch `+1`.
- **Thừa ở Đích (Missing from Src):** `0/0 (0ms)` -> Không có lỗi.
- **Thiếu ở Đích (Missing from Dest):** `2478/2477 (19954ms)` lúc `2026-09-09 10:45:54`
  - `missing_count` = 2477 (mẫu số)
  - `healed_missing_dest_count` = 2478 (tử số) -> Bị lệch `+1`.

---

## 2. TRUY VẾT DÒNG CODE VÀ FLOW THỰC THI (CODE TRACE)

### A. Mẫu số sinh ra từ đâu?
Tại `centralized-data-service/internal/service/recon/recon_job_worker.go`:
```go
// Dòng 374-380
staleCount = len(stalePayload.Mismatched)        // = 5
missingCount = len(stalePayload.MissingFromDest) // = 2477
orphanCount = len(stalePayload.MissingFromSrc)   // = 0
sBytes, _ := json.Marshal(stalePayload)
staleRaw = json.RawMessage(sBytes)
mBytes, _ := json.Marshal(stalePayload.MissingFromDest)
missingRaw = json.RawMessage(mBytes)
```
- Mẫu số `stale_count` (5) và `missing_count` (2477) là số lượng ID nghiệp vụ thực tế mà thuật toán Recon phát hiện bị lệch hash hoặc thiếu giữa MongoDB và Shadow Table.
- JSON lưu trữ trong database `stale_ids` chứa đúng 5 IDs mismatched, và `missing_ids` chứa đúng 2477 IDs missing from dest.

### B. Tử số sinh ra từ đâu?
Tại `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go`:
```go
// Dòng 256-269
if opts.HealMismatched && len(staleA.Mismatched) > 0 {
    written := h.fetchAndWriteChunked(ctx, entry, staleA.Mismatched, "mismatched")
    rpt.HealedMismatchedCount = written // written = 6
    healed += written
}
if opts.HealMissingDest && len(missingIDs) > 0 {
    written := h.fetchAndWriteChunked(ctx, entry, missingIDs, "missing_dest")
    rpt.HealedMissingDestCount = written // written = 2478
    healed += written
}
```
Tại `fetchAndWriteChunked` -> `FetchAndWriteByIDs` (`recon_heal_fetch.go`):
```go
// Dòng 69-109
for cursor.Next(findCtx) {
    ...
    _, err = h.eventHandler.HandleRaw(ctx, subject, envelope)
    batchCount++
    if batchCount%200 == 0 {
        persisted, fErr := h.eventHandler.FlushBatchBuffer(ctx)
        actualPersisted += persisted
    }
}
persisted, fErr := h.eventHandler.FlushBatchBuffer(ctx)
actualPersisted += persisted
return actualPersisted, nil
```
- Giá trị `written` được gán trực tiếp bằng `actualPersisted`.
- `actualPersisted` là tổng số dòng được báo về từ `h.eventHandler.FlushBatchBuffer(ctx)`.

---

## 3. NGUYÊN NHÂN GỐC RỄ (ROOT CAUSE ANALYSIS)

Có 3 nguyên nhân cốt lõi dẫn đến hiện tượng `+1`:

### 1. Rò rỉ Record từ In-Memory Shared Buffer (`BatchBuffer`)
- Trong `centralized-data-service`, đối tượng `eventHandler` và `batchBuffer` được chia sẻ dùng chung cho toàn bộ service (`server_setup.go:413: WithEventHandler(eventHandler)`).
- `BatchBuffer` đồng thời tiếp nhận dữ liệu realtime CDC từ NATS/Kafka.
- Khi tác vụ Heal gọi `FlushBatchBuffer(ctx)`, hàm này thực hiện:
  ```go
  bb.mu.Lock()
  batch := bb.records
  bb.records = make([]*shadow.UpsertRecord, 0, bb.maxSize)
  bb.mu.Unlock()
  ```
- Toàn bộ records đang tồn đọng trong `batchBuffer` (bao gồm cả record realtime CDC nếu có) đều bị gộp chung vào batch flush này.
- PostgreSQL thực thi batch upsert và trả về `RowsAffected` gồm cả record CDC nền đó, khiến `actualPersisted` bị dôi dư 1 bản ghi.

### 2. Lệch pha ngữ nghĩa (Semantic Gap: Physical DB RowsAffected vs Business IDs Count)
- **Mẫu số** đo lường: **Số lượng ID nghiệp vụ bị lỗi** (đếm bằng `len(unique(IDs))`).
- **Tử số** đo lường: **Số dòng vật lý bị tác động trong DB PostgreSQL** (`res.RowsAffected` trả về từ driver GORM/pgx).
- Việc lấy trực tiếp `res.RowsAffected` gán cho chỉ số "Số lỗi đã được chữa lành" (`healed_mismatched_count`) là sai lệch về mặt bản chất nghiệp vụ đối soát.

### 3. Thiếu Logical Bounding Guard (Chốt chặn biên logic)
- Trong một phiên đối soát, một báo cáo phát hiện $N$ lỗi thì số lỗi được chữa lành tối đa **CHỈ CÓ THỂ LÀ $N$** ($100\%$).
- Mã nguồn hiện tại không có cơ chế chặn biên: `min(persistedCount, len(unique(ids)))`, hoặc đếm chính xác số unique ID thực tế đã được query và persist thành công.
- Hậu quả: Hiển thị `6/5` và `2478/2477` gây hiểu nhầm nghiêm trọng cho người vận hành.

---

## 4. GIẢI PHÁP TỐI ƯU DUY NHẤT (THE SINGLE BEST APPROACH)

1. **Chuẩn hóa tại `recon_heal_fetch.go` & `recon_execute_heal_handler.go`**:
   - `FetchAndWriteByIDs`: Đếm số document thực tế được MongoDB tìm thấy và gửi vào pipeline (`batchCount`).
   - `executeHealSegA`: Chốt số lượng `HealedMismatchedCount` và `HealedMissingDestCount` theo số lượng unique IDs thực tế đã chữa lành thành công:
     ```go
     // Giới hạn không vượt quá tổng số ID yêu cầu heal của chính nhóm đó
     if written > len(staleA.Mismatched) {
         rpt.HealedMismatchedCount = len(staleA.Mismatched)
     } else {
         rpt.HealedMismatchedCount = written
     }
     ```
2. **Cách ly hoặc gắn Trace Scope cho Heal trong `BatchBuffer`**:
   - Gắn cờ hoặc phân định riêng rẽ giữa record của Heal và record của Realtime CDC, đảm bảo `FlushBatchBuffer` khi phục vụ Heal chỉ đếm các record thuộc về phiên Heal đó.
