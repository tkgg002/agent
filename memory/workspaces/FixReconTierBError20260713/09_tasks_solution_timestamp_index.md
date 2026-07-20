# Giải pháp kỹ thuật chi tiết - Tự động tạo và đề xuất index trên Timestamp Field

## 1. Sửa đổi `internal/handler/shadow/schema_ddl_handler.go`

### Bổ sung các hàm trợ giúp (helpers)
```go
func getTargetTSColumn(reg *source.TableRegistry, rules []mastermodel.MappingRuleV2) string {
	if reg == nil {
		return ""
	}
	primary := "updated_at"
	if reg.TimestampField != nil && *reg.TimestampField != "" {
		primary = *reg.TimestampField
	}

	// 1. Tìm trong MappingRuleV2
	for _, rule := range rules {
		if rule.IsActive && rule.SourceField == primary {
			return rule.TargetColumn
		}
	}

	// 2. Mặc định trả về primary
	return primary
}

func camelToSnake(s string) string {
	var res strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			res.WriteRune('_')
		}
		res.WriteRune(r)
	}
	return strings.ToLower(res.String())
}
```

### Thêm logic trong `HandleCreateDefaultColumns`
Sau khối log `"ALTER TABLE summary"`, thêm đoạn mã:
```go
		h.Logger.Info("ALTER TABLE summary",
			zap.String("table", payload.TargetTable),
			zap.Int("rules_total", len(rules)),
			zap.Int("columns_added", columnsAdded),
			zap.Int("columns_altered_type", columnsAlteredType),
			zap.Int("columns_already_exist", columnsAlreadyExist),
			zap.Int("columns_skipped", columnsSkipped),
		)

		// Bổ sung Index trên Timestamp Field nếu thiếu
		if reg := metadata.ResolveTableConfigByID(ctx, h.metadataRegistry, h.registryRepo, uint(effectiveID)); reg != nil {
			tsCol := getTargetTSColumn(reg, rules)
			if tsCol != "" {
				if _, exists := existingColsWithType[strings.ToLower(tsCol)]; exists {
					idxName := fmt.Sprintf("idx_%s_%s", payload.TargetTable, camelToSnake(tsCol))
					if len(idxName) > 63 {
						idxName = idxName[:63]
					}
					createIdxSQL := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s.%s(%s)`,
						sqlutil.QuoteIdent(idxName),
						sqlutil.QuoteIdent(schemaName),
						sqlutil.QuoteIdent(payload.TargetTable),
						sqlutil.QuoteIdent(tsCol),
					)
					if err := h.DB.WithContext(ctx).Exec(createIdxSQL).Error; err != nil {
						h.Logger.Error("failed to create index on timestamp field",
							zap.String("table", payload.TargetTable),
							zap.String("column", tsCol),
							zap.Error(err),
						)
					} else {
						h.Logger.Info("ensured index on timestamp field",
							zap.String("table", payload.TargetTable),
							zap.String("column", tsCol),
							zap.String("index", idxName),
						)
					}
				}
			}
		}
```

---

## 2. Sửa đổi `internal/service/governance/index_manager.go`

### Bổ sung helper `camelToSnake` ở cuối file
```go
func camelToSnake(s string) string {
	var res strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			res.WriteRune('_')
		}
		res.WriteRune(r)
	}
	return strings.ToLower(res.String())
}
```

### Thêm khuyến nghị index trong `GetRecommendations`
Ở cuối hàm `GetRecommendations` (trước câu lệnh `return recs`), bổ sung logic:
```go
	// 3. Kiểm tra index trên Timestamp Field để tối ưu MaxWindowTs cho Recon
	type tableRegistry struct {
		ID             uint
		TimestampField *string
	}
	var reg tableRegistry
	err := db.WithContext(ctx).Table("cdc_system.cdc_table_registry").
		Select("id, timestamp_field").
		Where("target_table = ? AND is_active = true", table).
		Limit(1).
		Scan(&reg).Error
	if err == nil && reg.ID > 0 {
		tsField := "updated_at"
		if reg.TimestampField != nil && *reg.TimestampField != "" {
			tsField = *reg.TimestampField
		}

		// Xác định target_column tương ứng từ mapping_rule_v2
		targetCol := tsField
		type mappingRuleV2 struct {
			TargetColumn string `gorm:"column:target_column"`
		}
		var rule mappingRuleV2
		errRule := db.WithContext(ctx).Table("cdc_system.mapping_rule_v2").
			Select("target_column").
			Where("source_object_id = ? AND source_field = ? AND is_active = true", reg.ID, tsField).
			Limit(1).
			Scan(&rule).Error
		if errRule == nil && rule.TargetColumn != "" {
			targetCol = rule.TargetColumn
		}

		// Kiểm tra xem index đã tồn tại hay chưa
		hasTsIndex := false
		colPattern1 := fmt.Sprintf(`(%s)`, targetCol)
		colPattern2 := fmt.Sprintf(`("%s")`, targetCol)
		for _, idx := range indexes {
			if strings.Contains(idx.IndexDef, colPattern1) || strings.Contains(idx.IndexDef, colPattern2) {
				hasTsIndex = true
				break
			}
		}

		if !hasTsIndex {
			idxName := fmt.Sprintf("idx_%s_%s", table, camelToSnake(targetCol))
			if len(idxName) > 63 {
				idxName = idxName[:63]
			}
			recs = append(recs, IndexRecommendation{
				IndexName:   idxName,
				Columns:     []string{targetCol},
				IsUnique:    false,
				IsPartial:   false,
				Description: fmt.Sprintf("Tối ưu hóa MaxWindowTs: Tạo index trên cột %s (Timestamp Field) để tối ưu hóa truy vấn đối soát thời gian cho Recon.", targetCol),
			})
		}
	}
```
