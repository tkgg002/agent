# Giải pháp Kỹ thuật Chi tiết - ReconSelfHealing

## 1. Source Code Modification (`transmuter.go`)
```go
// Tích hợp gom và soft-delete orphan master records
deletedInShadow := make([]string, 0)
for _, row := range shadowRows {
    existingMap[row.SourceID] = true
    if row.Deleted {
        deletedInShadow = append(deletedInShadow, row.SourceID)
    }
}

orphanMasterIDs := make([]string, 0)
for _, id := range onlySourceIDs {
    if !existingMap[id] {
        orphanMasterIDs = append(orphanMasterIDs, id)
    }
}

toSoftDelete := append(orphanMasterIDs, deletedInShadow...)
if len(toSoftDelete) > 0 {
    masterDB, errDB := t.connMgr.GetMasterDB(ctx, masterRow.MasterConnectionKey)
    if errDB == nil {
        nowMs := time.Now().UnixMilli()
        sqlText := fmt.Sprintf(`UPDATE %s SET _deleted = true, _source_ts = ?, _updated_at = NOW() WHERE _source_id = ANY(?)`,
            quoteTransmuteQualified(masterRow.MasterSchema, masterRow.MasterTable))
        _ = masterDB.WithContext(ctx).Exec(sqlText, nowMs, toSoftDelete).Error
    }
}
```

## 2. Test Implementation (`transmuter_orphan_test.go`)
Mô phỏng 3 trạng thái của record (`id1`, `id2`, `id3`) trên SQLite DB để xác minh transmuter soft-delete đúng rule và nâng `_source_ts` timestamp.
