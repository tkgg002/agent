# Hồ sơ giải pháp kỹ thuật cụ thể - Tối ưu hóa Transmuter

Tài liệu này chứa chi tiết các thay đổi mã nguồn trong dự án `centralized-data-service`.

## 1. Tối ưu hóa truy vấn và Checkpoint trong `transmuter.go`

Đường dẫn: [transmuter.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go)

### Thay đổi 1: Nhận checkpoint từ DB khi chạy Full Sync và thoát sớm khi chạy Incremental Sync
Trong hàm `Run`:
```go
	// Lấy checkpoint cho Full Sync từ SyncRuntimeState
	var lastGpayID int64
	if len(onlySourceIDs) == 0 && t.runtimeRepo != nil {
		state, errState := t.runtimeRepo.GetByMasterBinding(ctx, masterRow.ID)
		if errState == nil && state != nil && len(state.LastCursorJSON) > 0 {
			var cursorMap map[string]any
			if errJson := json.Unmarshal(state.LastCursorJSON, &cursorMap); errJson == nil {
				if val, ok := cursorMap["last_gpay_id"]; ok {
					if fVal, ok := val.(float64); ok {
						lastGpayID = int64(fVal)
						t.logger.Info("transmuter: resuming from checkpoint",
							zap.String("master", masterName),
							zap.Int64("last_gpay_id", lastGpayID))
					}
				}
			}
		}
	}

	var shadowRows []shadowBatchRow
	for {
		shadowRows, err = t.fetchShadowBatch(ctx, masterRow, lastGpayID, onlySourceIDs)
		if err != nil {
			t.markRuntimeFailure(ctx, masterRow.ID, err)
			return res, fmt.Errorf("fetch shadow batch: %w", err)
		}

		if len(shadowRows) == 0 {
			break
		}
		batchRes := t.processBatch(ctx, masterRow, rules, shadowRows)
		res.Scanned += batchRes.scanned
		res.Inserted += batchRes.inserted
		res.Updated += batchRes.updated
		res.Skipped += batchRes.skipped
		res.OccSkipped += batchRes.occSkipped
		res.RuleMisses += batchRes.ruleMisses
		res.TypeErrors += batchRes.typeErrors
		lastGpayID = batchRes.lastGpayID

		// Lưu checkpoint sau mỗi lô (chỉ áp dụng cho Full Sync)
		if len(onlySourceIDs) == 0 && t.runtimeRepo != nil && lastGpayID > 0 {
			cursorBytes, _ := json.Marshal(map[string]any{"last_gpay_id": lastGpayID})
			t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
				item.LastCursorJSON = cursorBytes
			})
		}

		// Nếu là Incremental Sync, thoát ngay sau batch đầu tiên vì đã lấy toàn bộ ID yêu cầu
		if len(onlySourceIDs) > 0 {
			break
		}
	}

	// Reset checkpoint sau khi đồng bộ Full Sync hoàn tất thành công
	if len(onlySourceIDs) == 0 && t.runtimeRepo != nil {
		t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
			item.LastCursorJSON = []byte(`{}`)
		})
	}
```

### Thay đổi 2: Tách biệt truy vấn incremental và tự động tạo index CONCURRENTLY
Trong hàm `fetchShadowBatch` và thêm hàm phụ trợ:
```go
func (t *TransmuterModule) fetchShadowBatch(ctx context.Context, row *masterBindingRuntime, cursor int64, onlyIDs []string) ([]shadowBatchRow, error) {
	shadowDB, err := t.connMgr.GetShadowDB(ctx, row.ShadowConnectionKey)
	if err != nil {
		return nil, err
	}
	var rows []shadowBatchRow
	pkIdent := quoteTransmuteIdent(row.ShadowPK)
	pkSelect := pkIdent
	if row.ShadowPK != "_gpay_id" {
		pkSelect = fmt.Sprintf("%s::bigint AS _gpay_id", pkIdent)
	}

	// Tách biệt truy vấn incremental/heal: không ORDER BY PK, không LIMIT
	if len(onlyIDs) > 0 {
		// Đảm bảo có index trên _source_id
		t.ensureShadowSourceIDIndex(ctx, shadowDB, row)

		qt := fmt.Sprintf(`SELECT %s, _source_id, _raw_data, _source_ts, _deleted FROM %s WHERE _source_id IN (?)`,
			pkSelect, quoteTransmuteQualified(row.ShadowSchema, row.ShadowTable))
		err = shadowDB.WithContext(ctx).Raw(qt, onlyIDs).Scan(&rows).Error
		return rows, err
	}

	qt := fmt.Sprintf(`SELECT %s, _source_id, _raw_data, _source_ts, _deleted FROM %s WHERE %s > ?`,
		pkSelect, quoteTransmuteQualified(row.ShadowSchema, row.ShadowTable), fmt.Sprintf("(%s)::bigint", pkIdent))

	args := []any{cursor}
	qt += ` ORDER BY 1 LIMIT ?`
	args = append(args, t.batchSize)
	err = shadowDB.WithContext(ctx).Raw(qt, args...).Scan(&rows).Error
	return rows, err
}

func (t *TransmuterModule) ensureShadowSourceIDIndex(ctx context.Context, shadowDB *gorm.DB, row *masterBindingRuntime) {
	indexName := fmt.Sprintf("idx_%s_source_id", row.ShadowTable)
	var count int64
	errIdx := shadowDB.WithContext(ctx).Raw(`
		SELECT COUNT(*) 
		FROM pg_indexes 
		WHERE schemaname = ? AND tablename = ? AND indexname = ?`,
		row.ShadowSchema, row.ShadowTable, indexName).Scan(&count).Error
	if errIdx == nil && count == 0 {
		// Tạo index CONCURRENTLY bất đồng bộ dưới nền để không block transmuter
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			sqlText := fmt.Sprintf(`CREATE INDEX CONCURRENTLY IF NOT EXISTS %s ON %s (_source_id)`,
				quoteTransmuteIdent(indexName),
				quoteTransmuteQualified(row.ShadowSchema, row.ShadowTable))
			t.logger.Info("transmuter: creating missing non-partial index on _source_id concurrently",
				zap.String("schema", row.ShadowSchema),
				zap.String("table", row.ShadowTable),
				zap.String("index", indexName))
			if errCreate := shadowDB.WithContext(bgCtx).Exec(sqlText).Error; errCreate != nil {
				t.logger.Error("transmuter: failed to create concurrent index on _source_id",
					zap.String("table", row.ShadowTable),
					zap.Error(errCreate))
			} else {
				t.logger.Info("transmuter: successfully created concurrent index on _source_id",
					zap.String("table", row.ShadowTable))
			}
		}()
	}
}
```

---

## 2. Bất đồng bộ hóa trong `transmute_handler.go`

Đường dẫn: [transmute_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/master/transmute_handler.go)

### Thay đổi 3: Đẩy luồng chạy sang goroutine bất đồng bộ với timeout context tương ứng
Trong hàm `HandleTransmute`:
```go
func (h *TransmuteHandler) HandleTransmute(msg *nats.Msg) {
	// ... (parse request, start span ban đầu và log activity)
	
	// Xác định timeout: 30 phút cho incremental/heal, 24 giờ cho full sync
	timeout := 24 * time.Hour
	if len(req.SourceIDs) > 0 {
		timeout = 30 * time.Minute
	}

	// Đẩy luồng đồng bộ chạy bất đồng bộ dưới nền
	go func() {
		// Tạo context cô lập (detached) từ Background để không bị hủy khi NATS connection/request đóng
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Propagate tracing context từ parent span nếu cần, hoặc tạo mới span độc lập để quản lý trace
		ctx, span := h.tracer.Start(ctx, "cdc.worker.transmute.process")
		defer span.End()

		res, err := h.svc.Run(ctx, req.MasterTable, req.SourceIDs)
		
		// ... (xử lý kết quả, cập nhật activity log, publish kết quả cdc.evt.transmute.completed)
	}()

	// Trả lời NATS Msg ngay lập tức nếu cần (hoặc ghi nhận lệnh đã nhận)
	h.reply(msg, TransmuteResponse{Success: true, Message: "Transmute job started in background"})
}
```
