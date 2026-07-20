# Kế hoạch triển khai: Ép số lượng source về shadow khi diff == 0 trong recon_smoke.go

## Mục tiêu
Khắc phục lỗi cú pháp toán tử ba ngôi (không hợp lệ trong Go) tại dòng 387-388 của `internal/service/recon/recon_smoke.go` và ép giá trị SourceTotal và SourceActive về dstActiveClean (Shadow Active) khi diff == 0.

## Thay đổi đề xuất

### [MODIFY] [recon_smoke.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_smoke.go)
Sửa đổi hàm `RunTotalOnlyA`:

```go
	var srcTotalPtr *int64
	var srcActivePtr *int64
	if diff == 0 {
		srcTotalPtr = &dstActiveClean
		srcActivePtr = &dstActiveClean
	} else {
		srcTotalPtr = &srcEstClean
		srcActivePtr = &srcEstClean
	}

	var diffTimeJSON []byte
	result := &recon.SmokeResult{
		RunID:        entry.RunID,
		Segment:      "source_shadow",
		SourceType:   ptr(entry.SourceType),
		SourceHost:   ptr(extractHost(entry.SourceURL)), // ĐÃ CHE CREDENTIALS
		SourceDB:     ptr(entry.SourceDB),
		SourceTable:  ptr(entry.SourceTable),
		SourceTotal:  srcTotalPtr,
		SourceActive: srcActivePtr,
		ShadowSchema: ptr(entry.ShadowSchema),
		ShadowTable:  ptr(entry.TargetTable),
		ShadowTotal:  &dstTotalClean,  //&dstTotal,
		ShadowActive: &dstActiveClean, //&dstActive,
		Diff:         diff,
		DiffTime:     json.RawMessage(diffTimeJSON),
		Status:       statusStr,
		DurationMs:   &dur,
		CheckedAt:    fromTime,
	}
```

## Kế hoạch kiểm chứng
- Thực hiện build thử để xác nhận biên dịch thành công.
- Run các unit test liên quan của recon.
