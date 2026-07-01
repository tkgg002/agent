# Detailed Code Solution - Strict Audit Fixes

Tài liệu này đặc tả chi tiết mã nguồn cần thay đổi để giải quyết 3 lỗi logic nghiêm trọng và 1 lỗi log nhỏ được chỉ ra trong đợt Strict Audit.

---

## 1. Sửa đổi file `recon_handler_run.go`
* **Vị trí sửa**: Hàm `HandleReconCheck` trong [recon_handler_run.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_handler_run.go).
* **Mục tiêu**: Loại bỏ khối `else` tự động gán fallback range khi không truyền tham số để bảo vệ cơ chế Adaptive Freeze Margin ở tầng dưới, chống bão lệch giả.

### Code thay đổi (Diff):
```diff
-	} else {
-		// Fallback an toàn: end_time = now, start_time = now - 24h
-		endMs := time.Now().UnixMilli()
-		startMs := endMs - 24*3600*1000
-		startT := time.UnixMilli(startMs)
-		endT := time.UnixMilli(endMs)
-		ctx = servicerecon.WithReconTimeRange(ctx, startT, endT)
-	}
```

---

## 2. Sửa đổi file `recon_heal_v4.go`
* **Vị trí sửa**: Hàm `healSegmentB` trong [recon_heal_v4.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go).
* **Mục tiêu**:
  - Gộp `staleObj.OrphanInMaster` vào mảng `gpayIDs` gửi xuống Transmuter để xử lý soft-delete.
  - Di chuyển lệnh ghi log NATS dispatch từ ngoài vòng lặp vào trong vòng lặp batching để tăng cường observability trên Kibana.

### Code thay đổi (Diff):
```diff
-	gpayIDs := append(append([]string{}, missingGpayIDs...), staleObj.StaleIDs...)
+	gpayIDs := append(append(append([]string{}, missingGpayIDs...), staleObj.StaleIDs...), staleObj.OrphanInMaster...)
```

```diff
 	// Re-trigger transmute theo chunk — đi đúng pipeline chuẩn, OCC bảo vệ.
 	dispatched := 0
 	for start := 0; start < len(sourceIDs); start += healChunkSize {
 		if start > 0 {
 			time.Sleep(healDelayMs)
 		}
 		end := start + healChunkSize
 		if end > len(sourceIDs) {
 			end = len(sourceIDs)
 		}
 		payload, _ := json.Marshal(map[string]any{
 			"master_table": table,
 			"_source_ids":  sourceIDs[start:end],
 			"triggered_by": "recon-heal-b",
 		})
 		if err := h.natsPub.Publish("cdc.cmd.transmute", payload); err != nil {
 			h.logActivity(op, table, "error", int64(dispatched), err)
 			h.respondErr(msg, fmt.Errorf("publish transmute chunk: %w", err))
 			return
 		}
 		dispatched += end - start
+
+		h.logger.Info("recon heal-b dispatched re-transmute batch",
+			zap.String("master_table", table), zap.Int("dispatched_chunk", end-start), zap.Int("total_so_far", dispatched))
 	}
 
 	// Đánh dấu đã dispatch (healed thực sự xác nhận ở lần recon B kế tiếp).
 	now := time.Now().UTC()
 	_ = h.reportRepo.UpdateByID(ctx, report.ID, map[string]any{"healed_at": now, "healed_count": dispatched})
 
-	h.logger.Info("recon heal-b dispatched re-transmute",
-		zap.String("master_table", table), zap.Int("source_ids", dispatched),
-		zap.Int("orphan_in_master_pending_operator", len(staleObj.OrphanInMaster)))
 	h.logActivity(op, table, "dispatched", int64(dispatched), nil)
 	if msg.Reply != "" {
 		res, _ := json.Marshal(map[string]any{
 			"status": "dispatched", "segment": "shadow_master",
 			"retransmute_ids": dispatched,
-			// Orphan ở master KHÔNG tự xoá (an toàn dữ liệu) — surface cho operator.
-			"orphan_in_master": len(staleObj.OrphanInMaster),
-			"note":             "verify bằng recon-check segment=shadow_master sau khi transmute chạy xong",
+			"note":             "verify bằng recon-check segment=shadow_master sau khi transmute chạy xong và sync",
 		})
 		msg.Respond(res)
 	}
```

---

## 3. Sửa đổi file `transmuter.go`
* **Vị trí sửa**: Hàm `Run` trong [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go).
* **Mục tiêu**:
  - Thêm chốt chặn an toàn `len(onlySourceIDs) > t.batchSize` ngay đầu hàm `Run`.
  - Loại bỏ khối `if len(onlySourceIDs) > 0 { break }` để cho phép phân trang tự nhiên (pagination) mà không gây xoá oan bản ghi.

### Code thay đổi (Diff):
```diff
 func (t *TransmuterModule) Run(ctx context.Context, masterName string, onlySourceIDs []string) (res TransmuteResult, err error) {
+	if len(onlySourceIDs) > t.batchSize {
+		return res, fmt.Errorf("transmute safety gate: len(onlySourceIDs) = %d vượt quá batchSize = %d, hãy chia lô nhỏ hơn", len(onlySourceIDs), t.batchSize)
+	}
+
 	ctx, span := observability.ChildSpan(ctx, "cdc.service.transmute",
 		attribute.String("transmute.master_table", masterName),
 		attribute.Int("transmute.source_ids_count", len(onlySourceIDs)),
 	)
```

```diff
 		batchRes := t.processBatch(ctx, masterRow, rules, shadowRows)
 		res.Scanned += batchRes.scanned
 		res.Inserted += batchRes.inserted
 		res.Updated += batchRes.updated
 		res.Skipped += batchRes.skipped
 		res.OccSkipped += batchRes.occSkipped
 		res.RuleMisses += batchRes.ruleMisses
 		res.TypeErrors += batchRes.typeErrors
 		lastGpayID = batchRes.lastGpayID
-		if len(onlySourceIDs) > 0 {
-			break
-		}
 	}
 	res.DurationMs = time.Since(start).Milliseconds()
```
