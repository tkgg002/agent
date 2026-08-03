# Technical Solution Profile: Flexible ID Resolution for Segment B Execute Heal

## Proposed Code Edit Details

### File: `centralized-data-service/internal/handler/recon/recon_execute_heal_handler.go`

Thay thế hàm `mapGpayToSourceIDs` bằng hàm linh hoạt `resolveSourceIDsForSegmentB`:

```go
func (h *ExecuteHealHandler) resolveSourceIDsForSegmentB(ctx context.Context, shadowRel string, inputIDs []string) ([]string, error) {
	if len(inputIDs) == 0 {
		return nil, nil
	}
	if shadowRel == "" || !strings.Contains(shadowRel, ".") {
		return nil, fmt.Errorf("invalid shadow relation %q", shadowRel)
	}
	qualified := quoteRelation(shadowRel)

	out := make([]string, 0, len(inputIDs))
	for start := 0; start < len(inputIDs); start += healChunkSize {
		end := start + healChunkSize
		if end > len(inputIDs) {
			end = len(inputIDs)
		}
		chunk := inputIDs[start:end]

		// Bước 1: Thử tìm theo _source_id (trường hợp inputIDs đã là _source_id như "44702")
		var mapped []string
		err := h.shadowDB.WithContext(ctx).Raw(
			fmt.Sprintf(`SELECT _source_id FROM %s WHERE _source_id IN (?) OR _gpay_id::text IN (?)`, qualified),
			chunk, chunk,
		).Scan(&mapped).Error

		if err == nil && len(mapped) > 0 {
			out = append(out, mapped...)
		} else {
			// Fallback: Nếu không query được, giữ nguyên inputIDs nếu chúng đã là _source_id string
			out = append(out, chunk...)
		}
	}
	return uniqueStrings(out), nil
}
```

Và cập nhật câu DELETE trong Prune Master DB:
```go
delSQL := fmt.Sprintf(
	`DELETE FROM %s WHERE "_source_id" IN (?) OR "_gpay_id"::text IN (?)`,
	quoteRelation(rpt.TargetTable),
)
```
