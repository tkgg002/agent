# 04_decisions — Audit Snapshot Zero Records

## D-1: Chọn Plan A (Plumb error+count) thay vì Plan B (Refactor sang sync upsert)

### Plan A — Plumb `(int, error)` qua Flush chain
**Pros:**
- Minimal-impact: 4 patch site, ~30 LOC.
- Không refactor BatchBuffer async semantics — timer loop tiếp tục async.
- Surface error cho snapshot path mà KHÔNG break ingestion path khác.
- §6 GEMINI Simplicity First / Demand Elegance: dùng signature change tối thiểu.

**Cons:**
- BatchBuffer giờ có 2 contract: void cho timer (ignore return), value cho sync flush (consume return).
- Vẫn còn batching → snapshot path có async delay giữa Add và Flush.

### Plan B — Refactor BatchBuffer sang sync per-record cho snapshot path
**Pros:**
- True end-to-end sync, zero ambiguity.
- BatchBuffer.WriteRecordSync line 87-125 đã có sẵn — chỉ cần wire vào processEvent.

**Cons:**
- Throughput drop: per-record SQL roundtrip = 10-100× slower batch.
- Phá vỡ pattern "snapshot.v2 dùng batch để efficient" — đụng performance baseline.
- Đụng nhiều: phải tách `processEvent` thành 2 mode (kafka batch vs snapshot sync) — rủi ro regression.
- §6 — over-engineer cho 1 bug observability.

### Quyết định
**Chọn Plan A.** Plumb `(written, err)` qua Flush chain. Plan B để dành cho refactor riêng nếu sau này cần.

## D-2: `batchUpsert` đếm RowsAffected hay đếm len(chunk)?

**Option 1**: Đếm `tx.Exec(...).RowsAffected` cộng dồn → chính xác PG-side, nhưng `BuildBatchUpsertSQL` build 1 query multi-row VALUES → RowsAffected = số rows INSERT-or-UPDATE thành công.

**Option 2**: Đếm `len(chunk)` nếu tx return nil → đơn giản, nhưng nếu ON CONFLICT skip 1 row (do OCC older-wins WHERE clause), vẫn count.

**Quyết định**: Dùng **Option 1** (RowsAffected). Bảo đảm counter đo persist thực tế.

- Tx path: `tx.Exec(query, values...).RowsAffected` per query → cộng dồn.
- Sequential fallback: per-row `db.Exec(...).RowsAffected` chỉ cộng khi err == nil.

## D-3: Final flush khác biệt với per-batch flush?

- Per-batch flush (line 516): adjust `rowsTotal` MỖI batch dựa trên `persisted` (không phải `batchWritten`).
- Final flush (line 550): cộng thêm `persisted` cuối cùng cho tail records còn trong buffer.

**Quyết định**: Cả 2 đều consume `(persisted, err)`. Nếu err → `markProgressError`. Nếu persisted < expected → log warn + adjust counter xuống.

## D-4: Có nên panic / hard-fail khi Flush err?

**Không.** Pattern khác (kafka consumer timer loop) cần graceful continue. Snapshot path cụ thể:
- Snapshot path: convert err thành `markProgressError` → frontend hiển thị status=error đúng sự thật.
- Timer loop: ignore return giữ nguyên behavior cũ (chỉ log).

## D-5: Order of patch (atomic vs incremental)

**Atomic apply**, single commit-equivalent edit batch:
1. SOL-1 batchUpsert (innermost) trước.
2. SOL-2 Flush.
3. SOL-3 FlushBatchBuffer.
4. SOL-4 runSnapshot lines 516 + 550.
5. SOL-5 timer loop callers (nếu có) bổ sung `_, _ =` để không break compile.

Lý do: signature change lan từ trong ra ngoài; nếu apply lẻ tẻ build sẽ fail giữa chừng. Apply hết rồi build 1 phát.

## D-6: Backward compat

`Flush()` vốn là method public của `BatchBuffer`. Đổi signature từ `()` → `(int, error)` = breaking. Grep usage:
- timer loop (`batch_buffer.go` line 141-156) — tự gọi mình → fix bên trong.
- `event_handler.go:62` `FlushBatchBuffer` wrap — fix.
- Test / mock nếu có — phải check.

Nếu có callers khác ngoài 2 chỗ trên → bổ sung `_, _ = bb.Flush()` để giữ compile.
