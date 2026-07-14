# Giải pháp kỹ thuật chi tiết - Khắc phục transmute safety gate batchSize

## 1. Các file cần thay đổi

### File: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter.go`

Thay thế hàm `Run`:

```go
func (t *TransmuterModule) Run(ctx context.Context, masterName string, onlySourceIDs []string) (res TransmuteResult, err error) {
	ctx, span := observability.ChildSpan(ctx, "cdc.service.transmute",
		attribute.String("transmute.master_table", masterName),
		attribute.Int("transmute.source_ids_count", len(onlySourceIDs)),
	)
	defer func() {
		observability.EndSpan(span, &err)
	}()

	start := time.Now()
	res = TransmuteResult{Master: masterName}

	var masterRow *masterBindingRuntime
	masterRow, err = t.loadMaster(ctx, masterName)
	if err != nil {
		t.markRuntimeFailure(ctx, 0, err)
		return res, fmt.Errorf("master lookup: %w", err)
	}
	if !masterRow.IsActive || masterRow.SchemaStatus != "approved" {
		res.ActiveGate = fmt.Sprintf("master gate: is_active=%v schema_status=%s", masterRow.IsActive, masterRow.SchemaStatus)
		t.markRuntimeSkipped(ctx, masterRow.ID, res.ActiveGate)
		return res, nil
	}
	res.Source = masterRow.ShadowTable

	if ok, reason := t.shadowActive(masterRow); !ok {
		res.ActiveGate = fmt.Sprintf("shadow gate: %s", reason)
		t.markRuntimeSkipped(ctx, masterRow.ID, res.ActiveGate)
		return res, nil
	}
	if t.ddlEnsurer != nil {
		t.mu.RLock()
		isEnsured := t.ensuredMasters[masterRow.MasterTable]
		t.mu.RUnlock()

		if !isEnsured {
			t.logger.Info("transmuter: executing hot-path DDL EnsureMaster (AccessExclusiveLock required)",
				zap.String("master", masterRow.MasterTable))

			if err = t.ddlEnsurer.EnsureMaster(ctx, masterRow.MasterTable); err != nil {
				t.markRuntimeFailure(ctx, masterRow.ID, err)
				return res, fmt.Errorf("ensure master destination: %w", err)
			}

			t.mu.Lock()
			t.ensuredMasters[masterRow.MasterTable] = true
			t.mu.Unlock()
		}
	}

	var rules []mappingRuleRow
	rules, err = t.loadRules(ctx, masterRow)
	if err != nil {
		t.markRuntimeFailure(ctx, masterRow.ID, err)
		return res, fmt.Errorf("rule load: %w", err)
	}
	if len(rules) == 0 {
		res.DurationMs = time.Since(start).Milliseconds()
		t.markRuntimeSuccess(ctx, masterRow.ID, res)
		return res, nil
	}

	// Phân chia onlySourceIDs thành các lô nhỏ hơn hoặc bằng t.batchSize
	var idChunks [][]string
	if len(onlySourceIDs) > 0 {
		for i := 0; i < len(onlySourceIDs); i += t.batchSize {
			end := i + t.batchSize
			if end > len(onlySourceIDs) {
				end = len(onlySourceIDs)
			}
			idChunks = append(idChunks, onlySourceIDs[i:end])
		}
	} else {
		idChunks = [][]string{nil}
	}

	for _, chunkIDs := range idChunks {
		var lastGpayID int64
		if len(chunkIDs) == 0 && t.runtimeRepo != nil {
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
			shadowRows, err = t.fetchShadowBatch(ctx, masterRow, lastGpayID, chunkIDs)
			if err != nil {
				t.markRuntimeFailure(ctx, masterRow.ID, err)
				return res, fmt.Errorf("fetch shadow batch: %w", err)
			}

			// Xử lý Orphan Master: so sánh chunkIDs với shadowRows để tìm ra các bản ghi mồ côi (đã bị xóa ở Shadow)
			// và tiến hành soft-delete trực tiếp trên Master table sử dụng SQL = ANY(?) và gán _source_ts = time.Now().UnixMilli()
			if len(chunkIDs) > 0 && lastGpayID == 0 {
				existingMap := make(map[string]bool)
				deletedInShadow := make([]string, 0)
				for _, row := range shadowRows {
					existingMap[row.SourceID] = true
					if row.Deleted {
						deletedInShadow = append(deletedInShadow, row.SourceID)
					}
				}

				orphanMasterIDs := make([]string, 0)
				for _, id := range chunkIDs {
					if !existingMap[id] {
						orphanMasterIDs = append(orphanMasterIDs, id)
					}
				}

				toSoftDelete := append(orphanMasterIDs, deletedInShadow...)
				if len(toSoftDelete) > 0 {
					masterDB, errDB := t.connMgr.GetMasterDB(ctx, masterRow.MasterConnectionKey)
					if errDB != nil {
						t.logger.Error("transmuter: get master DB for orphan prune failed", zap.Error(errDB))
					} else {
						nowMs := time.Now().UnixMilli()
						sqlText := fmt.Sprintf(`UPDATE %s SET _deleted = true, _source_ts = ?, _updated_at = NOW() WHERE _source_id = ANY(?)`,
							quoteTransmuteQualified(masterRow.MasterSchema, masterRow.MasterTable))
						errDel := masterDB.WithContext(ctx).Exec(sqlText, nowMs, toSoftDelete).Error
						if errDel != nil {
							t.logger.Error("transmuter: soft-delete orphan master failed", zap.String("master", masterRow.MasterTable), zap.Error(errDel))
						} else {
							t.logger.Info("transmuter: soft-deleted orphan master rows successfully",
								zap.String("master", masterRow.MasterTable),
								zap.Int("count", len(toSoftDelete)),
								zap.Int("physical_orphans", len(orphanMasterIDs)),
								zap.Int("marked_deleted", len(deletedInShadow)))
						}
					}
				}
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
			if len(chunkIDs) == 0 && t.runtimeRepo != nil && lastGpayID > 0 {
				cursorBytes, _ := json.Marshal(map[string]any{"last_gpay_id": lastGpayID})
				t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
					item.LastCursorJSON = cursorBytes
				})
			}

			// Nếu là Incremental Sync, thoát ngay sau batch đầu tiên của chunk hiện tại
			if len(chunkIDs) > 0 {
				break
			}
		}
	}

	res.DurationMs = time.Since(start).Milliseconds()

	// Reset checkpoint sau khi đồng bộ Full Sync hoàn tất thành công
	if len(onlySourceIDs) == 0 && t.runtimeRepo != nil {
		t.persistRuntimeState(ctx, masterRow.ID, func(item *mastermodel.SyncRuntimeState, now time.Time) {
			item.LastCursorJSON = []byte(`{}`)
		})
	}

	if res.Scanned > 0 && res.Inserted == 0 && res.Updated == 0 {
		if res.Skipped > 0 && res.OccSkipped == res.Skipped && res.RuleMisses == 0 && res.TypeErrors == 0 {
			t.markRuntimeSuccess(ctx, masterRow.ID, res)
			return res, nil
		}
		degraded := fmt.Errorf("transmute degraded: scanned=%d nhưng 0 dòng ghi master (skipped=%d occ_skipped=%d rule_misses=%d type_errors=%d) — kiểm tra transform_spec/mapping rules",
			res.Scanned, res.Skipped, res.OccSkipped, res.RuleMisses, res.TypeErrors)
		t.markRuntimeFailure(ctx, masterRow.ID, degraded)
		return res, degraded
	}
	t.markRuntimeSuccess(ctx, masterRow.ID, res)
	return res, nil
}
```

### File: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/master/transmuter_orphan_test.go`

Thêm test case `TestTransmuter_OrphanMasterChunking`:

```go
func TestTransmuter_OrphanMasterChunking(t *testing.T) {
	// Initialize sqlite in-memory database for Shadow and Master with Info logger
	dataDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	require.NoError(t, err)

	// GORM callback on SQLite dataDB to dynamically adapt PG-specific syntax to SQLite compatible syntax
	fixFn := func(d *gorm.DB) {
		sql := d.Statement.SQL.String()
		if sql == "" {
			return
		}
		// Replace pg-specific syntax
		sql = strings.ReplaceAll(sql, "::bigint", "")
		sql = strings.ReplaceAll(sql, "::jsonb", "")
		sql = strings.ReplaceAll(sql, "NOW()", "datetime('now')")
		sql = strings.ReplaceAll(sql, "= ANY(", "IN (")
		sql = strings.ReplaceAll(sql, " = ANY (", " IN (")
		sql = strings.ReplaceAll(sql, " = ANY(", " IN (")
		sql = strings.ReplaceAll(sql, `"public".`, "")
		sql = strings.ReplaceAll(sql, `"main".`, "")
		sql = strings.ReplaceAll(sql, "(xmax = 0)", "1")

		d.Statement.SQL.Reset()
		d.Statement.SQL.WriteString(sql)
	}
	err = dataDB.Callback().Query().Before("gorm:query").Register("sqlite_fix_chunking", fixFn)
	require.NoError(t, err)
	err = dataDB.Callback().Row().Before("gorm:row").Register("sqlite_fix_chunking", fixFn)
	require.NoError(t, err)
	err = dataDB.Callback().Raw().Before("gorm:raw").Register("sqlite_fix_chunking", fixFn)
	require.NoError(t, err)

	// Initialize sqlmock for systemDB
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mockDB.Close()

	sysDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: mockDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	require.NoError(t, err)

	// Set up sqlmock expectations for systemDB
	// 1. loadMaster query
	mock.ExpectQuery(`SELECT mb\.id, mb\.source_object_id, mb\.shadow_binding_id,.*FROM cdc_system\.master_binding mb.*`).
		WithArgs("master_chunk_test").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "source_object_id", "shadow_binding_id", "master_connection_key",
			"master_schema", "master_table", "physical_table_fqn", "transform_type",
			"transform_spec", "is_active", "schema_status", "shadow_connection_key",
			"shadow_schema", "shadow_table", "shadow_is_active", "source_profile_status",
			"source_pk", "shadow_pk",
		}).AddRow(
			1, 10, 100, "default",
			"public", "master_chunk_test", "public.master_chunk_test", "copy",
			"{}", true, "approved", "default",
			"public", "shadow_chunk_test", true, "active",
			"id", "_gpay_id",
		))

	// 2. loadRules query (mapping_rule_master JOIN mapping_rule_v2)
	mock.ExpectQuery(`SELECT m\.id,.*FROM cdc_system\.mapping_rule_master m.*`).
		WithArgs(10, 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "source_object_id", "master_binding_id", "source_field", "target_column",
			"data_type", "source_format", "source_path", "transform_fn", "is_nullable", "default_value",
		}).AddRow(
			1, 10, 1, "id", "_source_id",
			"TEXT", "", "", "", true, nil,
		))

	// Create mock connection manager using reflection to set DB instances in registry
	cfg := &config.AppConfig{}
	cfg.ShadowDB.Host = "localhost"
	cfg.ShadowDB.Database = "shadow"
	cfg.MasterDB.Host = "localhost"
	cfg.MasterDB.Database = "dest"

	reg := database.NewRegistry(cfg)

	val := reflect.ValueOf(reg).Elem()
	gormDBsField := val.FieldByName("gormDBs")

	ptr := unsafe.Pointer(gormDBsField.UnsafeAddr())
	*(*map[string]*gorm.DB)(ptr) = map[string]*gorm.DB{
		"cdc":    sysDB,
		"shadow": dataDB,
		"dest":   dataDB,
	}

	connMgr := NewConnectionManagerWithRegistry(cfg, zap.NewNop(), reg)

	// Migrate test tables on sqlite shadow and master
	err = dataDB.Exec(`
		CREATE TABLE shadow_chunk_test (
			_gpay_id INTEGER PRIMARY KEY,
			_source_id TEXT,
			_raw_data TEXT,
			_source_ts INTEGER,
			_deleted BOOLEAN
		)
	`).Error
	require.NoError(t, err)

	err = dataDB.Exec(`
		CREATE TABLE master_chunk_test (
			_gpay_id INTEGER PRIMARY KEY,
			_source_id TEXT,
			_source_ts INTEGER,
			_deleted BOOLEAN,
			_updated_at TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	// Insert 5 rows into shadow
	for i := 1; i <= 5; i++ {
		idStr := fmt.Sprintf("id%d", i)
		err = dataDB.Exec(`INSERT INTO shadow_chunk_test (_gpay_id, _source_id, _raw_data, _source_ts, _deleted) VALUES (?, ?, ?, ?, ?)`,
			i, idStr, fmt.Sprintf(`{"id":"%s"}`, idStr), 1783564316564, false).Error
		require.NoError(t, err)
	}

	tm := NewTransmuterModule(sysDB, connMgr, nil, nil, shadow.NewTypeResolver(zap.NewNop()), zap.NewNop())

	// Sử dụng unsafe/reflect để set batchSize = 2 cho transmuter
	valTransmuter := reflect.ValueOf(tm).Elem()
	batchSizeField := valTransmuter.FieldByName("batchSize")
	ptrBatch := unsafe.Pointer(batchSizeField.UnsafeAddr())
	*(*int)(ptrBatch) = 2

	// Run transmuter with 5 onlySourceIDs, which is greater than batchSize = 2
	res, err := tm.Run(context.Background(), "master_chunk_test", []string{"id1", "id2", "id3", "id4", "id5"})
	require.NoError(t, err)

	// Verify results:
	// Scanned should be 5, Inserted should be 5 because we have 5 shadow rows.
	require.Equal(t, int64(5), res.Scanned)
	require.Equal(t, int64(5), res.Inserted)

	// Verify records in Master
	var count int64
	err = dataDB.Table("master_chunk_test").Count(&count).Error
	require.NoError(t, err)
	require.Equal(t, int64(5), count)
}
```
